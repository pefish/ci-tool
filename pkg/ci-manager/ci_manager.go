package ci_manager

import (
	"context"
	"fmt"
	"github.com/pefish/ci-tool/pkg/constant"
	go_best_type "github.com/pefish/go-best-type"
	go_logger "github.com/pefish/go-logger"
	go_shell "github.com/pefish/go-shell"
	"os/exec"
	"sync"
)

type CiManagerType struct {
	go_best_type.BaseBestType
	logs sync.Map // map[string]string
}

func NewCiManager(ctx context.Context) *CiManagerType {
	return &CiManagerType{
		BaseBestType: *go_best_type.NewBaseBestType(ctx, 0),
	}
}

func (c *CiManagerType) ProcessAsk(ask *go_best_type.AskType, bts map[string]go_best_type.IBestType) {
	data := ask.Data.(map[string]interface{})
	switch ask.Action {
	case constant.ActionType_CI:
		env := data["env"].(string)
		srcPath := data["src_path"].(string)
		scriptPath := data["script_path"].(string)
		projectName := data["project_name"].(string)
		port := data["port"].(uint64)
		configPath := data["config_path"].(string)
		go func() {
			logger := go_logger.Logger.CloneWithPrefix(projectName)
			logger.InfoF("<%s> 正在部署...\n", projectName)
			c.logs.Delete(projectName)
			c.logs.Store(projectName, "正在部署...\n")
			err := c.startCi(
				logger,
				env,
				srcPath,
				scriptPath,
				projectName,
				port,
				configPath,
			)
			if err != nil {
				c.logs.Store(projectName, err.Error())
			}
			logger.InfoF("<%s> 部署成功\n", projectName)
			c.logs.Store(projectName, "部署成功\n")
		}()
	case constant.ActionType_LOG:
		msg := data["msg"].(string)
		projectName := data["project_name"].(string)
		c.logs.Store(projectName, msg)
	case constant.ActionType_ReadLog:
		projectName := data["project_name"].(string)
		d, ok := c.logs.Load(projectName)
		if !ok {
			ask.AnswerChan <- ""
		} else {
			ask.AnswerChan <- d.(string)
		}

	}
}

func (c *CiManagerType) OnExited() {

}

func (c *CiManagerType) startCi(
	logger go_logger.InterfaceLogger,
	env,
	srcPath,
	scriptPath,
	projectName string,
	port uint64,
	configPath string,
) error {
	branch := "test"
	if env == "prod" {
		branch = "main"
	}
	script := fmt.Sprintf(
		`
#!/bin/bash
set -euxo pipefail
cd %s

projectName="%s"
port="%d"
configPath="%s"

git reset --hard && git pull && git checkout %s && git pull

imageName="${projectName}:`+`git rev-parse --short HEAD`+`"

if [[ "`+`sudo docker images -q ${imageName} 2> /dev/null`+`" == "" ]]; then
  sudo docker build -t ${imageName} .
fi

containerName="${projectName}-%s"

sudo docker stop ${containerName} && sudo docker rm ${containerName}

sudo docker run --name ${containerName} -d -v ${configPath}:/app/config%s ${imageName}

`,
		srcPath,
		scriptPath,
		port,
		configPath,
		branch,
		env,
		func() string {
			if port == 0 {
				return ""
			} else {
				return " -p ${port}:${port}"
			}
		}(),
	)
	cmd := exec.Command("/bin/bash", "-c", script)

	resultChan := make(chan string)
	go func() {
		for {
			select {
			case r := <-resultChan:
				logger.Info(r)
				d, ok := c.logs.Load(projectName)
				if !ok {
					c.logs.Store(projectName, r)
				} else {
					c.logs.Store(projectName, d.(string)+r+"\n")
				}
			}
		}
	}()
	err := go_shell.ExecResultLineByLine(cmd, resultChan)
	if err != nil {
		return err
	}

	return nil
}
