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
	srcPath,
	fullName string,
) {
	c.logs.Delete(fullName)
	logger := c.logger.CloneWithPrefix(fullName)
	logger.InfoF("<%s> running...\n", fullName)
	err := c.startCi(
		ctx,
		logger,
		project,
		srcPath,
		fullName,
	)
	if err != nil {
		c.logs.Store(fullName, err.Error())
		util.AlertNoError(
			c.logger,
			fmt.Sprintf("[ERROR] <%s> <%s> 环境发布失败。\n%+v", fullName, project.Params.Env, err),
		)
		logger.ErrorF("<%s> failed!!! %+v", fullName, err)
		return
	}

	err = util.Alert(
		c.logger,
		fmt.Sprintf("[INFO] <%s> <%s> 环境发布成功。", fullName, project.Params.Env),
	)
	if err != nil {
		logger.ErrorF("<%s> 发送通知失败!!! %+v", fullName, err)
	}

	logger.InfoF("<%s> done!!!", fullName)
}

func (c *CiManagerType) startCi(
	ctx context.Context,
	logger i_logger.ILogger,
	project *db.Project,
	srcPath,
	fullName string,
) error {
	resultChan := make(chan string)
	go func() {
		for {
			select {
			case r := <-resultChan:
				logger.Info(r)
				d, ok := c.logs.Load(fullName)
				if !ok {
					c.logs.Store(fullName, r)
				} else {
					c.logs.Store(fullName, d.(string)+r+"\n")
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

	if _, ok := global.GlobalData.StartLogTime[fullName]; !ok {
		global.GlobalData.StartLogTime[fullName] = time.Now()
	}

	logsPath := path.Join(global.Command.DataDir, "logs", fullName)
	err = go_file.AssertPathExist(logsPath)
	if err != nil {
		return err
	}

	imageName := ""
	if project.Image == nil || project.Image.Now == "" {
		shortCommitHash, err := util.GetGitShortCommitHash(srcPath)
		if err != nil {
			return err
		}
		imageName = fmt.Sprintf("%s:%s", fullName, shortCommitHash)
	} else {
		imageName = project.Image.Now
	}

	err = util.BuildImage(
		resultChan,
		project.Params.Env,
		imageName,
	)
	if err != nil {
		return err
	}
	if project.Image == nil || project.Image.Last2 != "" {
		err = util.RemoveImage(resultChan, project.Image.Last2)
		if err != nil {
			return err
		}
	}

	containerName := fmt.Sprintf("%s-%s", fullName, project.Params.Env)
	containerExists, err := util.ContainerExists(containerName)
	if err != nil {
		return err
	}
	if containerExists {
		err = util.StopContainer(containerName)
		if err != nil {
			return err
		}
		isPacked, err := util.BackupContainerLog(
			resultChan,
			logsPath,
			containerName,
			global.GlobalData.StartLogTime[fullName],
		)
		if err != nil {
			return err
		}
		if isPacked {
			global.GlobalData.StartLogTime[fullName] = time.Now()
		}

	} else {
		err = util.StartNewContainer(
			resultChan,
			imageName,
			envConfig,
			project.Port,
			project.Params.DockerNetwork,
			containerName,
		)
		if err != nil {
			return err
		}
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
