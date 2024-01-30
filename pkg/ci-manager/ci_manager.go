package ci_manager

import (
	"context"
	"fmt"
	"github.com/pefish/ci-tool/pkg/constant"
	go_best_type "github.com/pefish/go-best-type"
	go_file "github.com/pefish/go-file"
	go_logger "github.com/pefish/go-logger"
	go_shell "github.com/pefish/go-shell"
	tg_sender "github.com/pefish/tg-sender"
	"github.com/pkg/errors"
	"os/exec"
	"strings"
	"sync"
)

type CiManagerType struct {
	go_best_type.BaseBestType
	logs sync.Map // map[string]string
}

func NewCiManager(ctx context.Context) *CiManagerType {
	c := &CiManagerType{
		BaseBestType: *go_best_type.NewBaseBestType(ctx, 0),
	}
	return c
}

func (c *CiManagerType) ProcessAsk(ask *go_best_type.AskType, bts map[string]go_best_type.IBestType) {
	data := ask.Data.(map[string]interface{})
	switch ask.Action {
	case constant.ActionType_CI:
		env := data["env"].(string)
		srcPath := data["src_path"].(string)
		projectName := data["project_name"].(string)
		port := data["port"].(uint64)
		configPath := data["config_path"].(string)
		alertTgToken := data["alert_tg_token"].(string)
		alertTgGroupId := data["alert_tg_group_id"].(string)
		lokiUrl := data["loki_url"].(string)
		go func() {
			logger := go_logger.Logger.CloneWithPrefix(projectName)
			logger.InfoF("<%s> running...\n", projectName)
			c.logs.Delete(projectName)
			err := c.startCi(
				logger,
				env,
				srcPath,
				projectName,
				port,
				configPath,
				lokiUrl,
			)
			if err != nil {
				c.logs.Store(projectName, err.Error())
				if alertTgGroupId != "" {
					tg_sender.NewTgSender(alertTgToken).
						SetLogger(go_logger.Logger).
						SendMsg(tg_sender.MsgStruct{
							ChatId: alertTgGroupId,
							Msg:    fmt.Sprintf("[ERROR] <%s> <%s> 环境发布失败。\n%+v", projectName, env, err),
							Ats:    nil,
						}, 0)
				}
				logger.ErrorF("<%s> failed!!! %+v", projectName, err)
				return
			}

			if alertTgGroupId != "" {
				tg_sender.NewTgSender(alertTgToken).
					SetLogger(go_logger.Logger).
					SendMsg(tg_sender.MsgStruct{
						ChatId: alertTgGroupId,
						Msg:    fmt.Sprintf("[INFO] <%s> <%s> 环境发布成功。", projectName, env),
						Ats:    nil,
					}, 0)
			}

			logger.InfoF("<%s> done!!!", projectName)
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
	projectName string,
	port uint64,
	configPath string,
	lokiUrl string,
) error {
	if env != "test" && env != "prod" {
		return errors.New("Env is illegal.")
	}

	branch := "test"
	if env == "prod" {
		branch = "main"
	}

	if strings.HasPrefix(srcPath, "~") {
		srcPath = "${HOME}" + srcPath[1:]
	}

	if strings.HasPrefix(configPath, "~") {
		configPath = "${HOME}" + configPath[1:]
	}

	if configPath != "" {
		// 校验 config 文件夹是否存在
		if !go_file.FileInstance.Exists(configPath) {
			return errors.New(fmt.Sprintf("Config <%s> not be found!", configPath))
		}
	}

	script := fmt.Sprintf(
		`
#!/bin/bash
set -euxo pipefail

src="%s"
projectName="%s"


cd ${src}
git reset --hard && git pull && git checkout %s && git pull

imageName="${projectName}:$(git rev-parse --short HEAD)"

if [[ "$(sudo docker images -q ${imageName} 2> /dev/null)" == "" ]]; then
  sudo docker build -t ${imageName} .
fi

containerName="${projectName}-%s"

sudo docker stop ${containerName} && sudo docker rm ${containerName}

sudo docker run --name ${containerName} -d %s%s%s ${imageName}

`,
		srcPath,
		projectName,
		branch,
		env,
		func() string {
			if configPath == "" {
				return ""
			} else {
				return fmt.Sprintf(" -v %s:/app/config", configPath)
			}
		}(),
		func() string {
			if port == 0 {
				return ""
			} else {
				return fmt.Sprintf(" -p %d:8000", port)
			}
		}(),
		func() string {
			if lokiUrl == "" {
				return ""
			} else {
				return fmt.Sprintf(` --log-driver=loki --log-opt loki-url="%s" --log-opt loki-retries=5 --log-opt loki-batch-size=400`, lokiUrl)
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
	err := go_shell.ExecForResultLineByLine(cmd, resultChan)
	if err != nil {
		return err
	}

	return nil
}
