package ci_manager

import (
	"context"
	"fmt"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/pefish/ci-tool/pkg/db"
	"github.com/pefish/ci-tool/pkg/global"
	"github.com/pefish/ci-tool/pkg/util"
	go_file "github.com/pefish/go-file"
	go_format "github.com/pefish/go-format"
	i_logger "github.com/pefish/go-interface/i-logger"
	t_mysql "github.com/pefish/go-interface/t-mysql"
	"github.com/pkg/errors"
)

var CiManager *CiManagerType

type CiManagerType struct {
	logs   sync.Map // map[string]string
	logger i_logger.ILogger
}

func NewCiManager(logger i_logger.ILogger) *CiManagerType {
	c := &CiManagerType{
		logger: logger,
	}
	return c
}

func (c *CiManagerType) Logs(fullName string) string {
	d, ok := c.logs.Load(fullName)
	if !ok {
		return ""
	} else {
		return d.(string)
	}
}

func (c *CiManagerType) StartCi(
	ctx context.Context,
	project *db.Project,
	srcPath string,
	name string,
) {
	c.logs.Delete(project.FullName)
	logger := c.logger.CloneWithPrefix(project.FullName)
	logger.InfoF("<%s> running...\n", project.FullName)
	err := c.startCi(
		ctx,
		logger,
		project,
		srcPath,
		name,
	)
	if err != nil {
		c.logs.Store(project.FullName, err.Error())
		util.AlertNoError(
			c.logger,
			fmt.Sprintf("[ERROR] <%s> <%s> 环境发布失败。\n%+v", project.FullName, project.Params.Env, err),
		)
		logger.ErrorF("<%s> failed!!! %+v", project.FullName, err)
		return
	}

	err = util.Alert(
		c.logger,
		fmt.Sprintf("[INFO] <%s> <%s> 环境发布成功。", project.FullName, project.Params.Env),
	)
	if err != nil {
		logger.ErrorF("<%s> 发送通知失败!!! %+v", project.FullName, err)
	}

	logger.InfoF("<%s> done!!!", project.FullName)
}

func (c *CiManagerType) startCi(
	ctx context.Context,
	logger i_logger.ILogger,
	project *db.Project,
	srcPath string,
	name string,
) error {
	resultChan := make(chan string)
	go func() {
		for {
			select {
			case r := <-resultChan:
				logger.Info(r)
				d, ok := c.logs.Load(project.FullName)
				if !ok {
					c.logs.Store(project.FullName, r)
				} else {
					c.logs.Store(project.FullName, d.(string)+r+"\n")
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	if project.Params.Env != "test" && project.Params.Env != "prod" {
		return errors.New("Env is illegal.")
	}

	envConfig := ""
	if project.Config != nil {
		envConfig = *project.Config
	}

	branch := "test"
	if project.Params.Env == "prod" {
		branch = "main"
	}

	if strings.HasPrefix(srcPath, "~") {
		srcPath = "${HOME}" + srcPath[1:]
	}

	logger.Info("开始 pull 代码...")

	err := util.GitPullSourceCode(
		resultChan,
		srcPath,
		project.Params.Repo,
		project.Params.FetchCodeKey,
		branch,
	)
	if err != nil {
		return err
	}

	logger.Info("pull 代码完成.")

	// fmt.Println("111", global.GlobalData.StartLogTime[fullName])
	if _, ok := global.GlobalData.StartLogTime[project.FullName]; !ok {
		global.GlobalData.StartLogTime[project.FullName] = time.Now()
	}

	logsPath := path.Join(global.Command.DataDir, "logs", project.FullName)
	err = go_file.AssertPathExist(logsPath)
	if err != nil {
		return err
	}

	imageName := ""
	if project.Image != nil && project.Image.Should != "" {
		imageName = project.Image.Should
	} else {
		shortCommitHash, err := util.GetGitShortCommitHash(srcPath)
		if err != nil {
			return err
		}
		imageName = fmt.Sprintf("%s:%s", project.FullName, shortCommitHash)
		logger.InfoF("开始构建镜像 <%s>...", imageName)
		err = util.BuildImage(
			resultChan,
			srcPath,
			project.Params.Env,
			imageName,
		)
		if err != nil {
			return err
		}
		logger.Info("构建镜像完成.")
	}

	if project.Image != nil && project.Image.Last2 != "" && project.Image.Last2 != project.Image.Now {
		logger.InfoF("开始删除镜像 <%s>...", project.Image.Last2)
		err = util.RemoveImage(resultChan, project.Image.Last2)
		if err != nil {
			return err
		}
		logger.Info("删除镜像完成.")
	}

	// 删除每一个容器
	containerNames, err := util.ListProjectContainers(fmt.Sprintf("%s-%s", project.FullName, project.Params.Env))
	if err != nil {
		return err
	}
	for _, containerName := range containerNames {
		logger.InfoF("开始停止容器 <%s>...", containerName)
		err = util.StopContainer(containerName)
		if err != nil {
			return err
		}
		logger.Info("停止容器完成.")

		logger.InfoF("开始备份容器 <%s> 日志...", containerName)
		// fmt.Println("global.GlobalData.StartLogTime[fullName]", fullName, global.GlobalData.StartLogTime[fullName])
		isPacked, err := util.BackupContainerLog(
			resultChan,
			logsPath,
			containerName,
			global.GlobalData.StartLogTime[project.FullName],
		)
		if err != nil {
			return err
		}
		if isPacked {
			global.GlobalData.StartLogTime[project.FullName] = time.Now()
			logger.Info("容器日志被打包.")
		}
		logger.Info("备份容器日志完成.")

		logger.InfoF("开始删除容器 <%s>...", containerName)
		err = util.RemoveContainer(containerName)
		if err != nil {
			return err
		}
		logger.Info("删除容器完成.")
	}

	ports := strings.Split(project.Port, ",")
	for i, portStr := range ports {
		port, _ := go_format.ToUint64(portStr)

		containerName := fmt.Sprintf("%s-%s%d", project.FullName, project.Params.Env, i)
		logger.InfoF("开始启动容器 <%s>...", containerName)
		err = util.StartNewContainer(
			resultChan,
			imageName,
			envConfig,
			port,
			project.Params.DockerNetwork,
			containerName,
			name,
		)
		if err != nil {
			return err
		}
		logger.Info("启动容器完成.")
	}

	newImageInfo := db.ImageInfo{
		Now: imageName,
	}
	if project.Image != nil && project.Image.Now != "" {
		newImageInfo.Last1 = project.Image.Now
	}
	if project.Image != nil && project.Image.Last1 != "" {
		newImageInfo.Last2 = project.Image.Last1
	}
	global.MysqlInstance.Update(&t_mysql.UpdateParams{
		TableName: "project",
		Update: map[string]interface{}{
			"image": newImageInfo,
		},
		Where: map[string]interface{}{
			"id": project.Id,
		},
	})

	return nil
}
