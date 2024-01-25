package ci_manager

import (
	"context"
	"fmt"
	"github.com/pefish/ci-tool/pkg/constant"
	go_best_type "github.com/pefish/go-best-type"
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
	logs     sync.Map // map[string]string
	tgSender *tg_sender.TgSender
}

func NewCiManager(ctx context.Context, alertToken string) *CiManagerType {
	c := &CiManagerType{
		BaseBestType: *go_best_type.NewBaseBestType(ctx, 0),
	}
	if alertToken != "" {
		c.tgSender = tg_sender.NewTgSender(alertToken).SetLogger(go_logger.Logger)
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
				if c.tgSender != nil && alertTgGroupId != "" {
					c.tgSender.SendMsg(tg_sender.MsgStruct{
						ChatId: alertTgGroupId,
						Msg:    fmt.Sprintf("[ERROR] <%s> 发布失败。%+v", projectName, err),
						Ats:    nil,
					}, 0)
				}
				return
			}

			if c.tgSender != nil && alertTgGroupId != "" {
				// 发送通知
				c.tgSender.SendMsg(tg_sender.MsgStruct{
					ChatId: alertTgGroupId,
					Msg:    fmt.Sprintf("[INFO] <%s> 发布成功。", projectName),
					Ats:    nil,
				}, 0)
			}

			logger.InfoF("<%s> done!!!\n", projectName)
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

	script := fmt.Sprintf(
		`
#!/bin/bash
set -euxo pipefail
cd %s

projectName="%s"
port="%d"
configPath="%s"

git reset --hard && git pull && git checkout %s && git pull

imageName="${projectName}:$(git rev-parse --short HEAD)"

if [[ "$(sudo docker images -q ${imageName} 2> /dev/null)" == "" ]]; then
  sudo docker build -t ${imageName} .
fi

containerName="${projectName}-%s"

sudo docker stop ${containerName} && sudo docker rm ${containerName}

sudo docker run --name ${containerName} -d -v ${configPath}:/app/config%s%s ${imageName}

`,
		srcPath,
		projectName,
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
	err := go_shell.ExecResultLineByLine(cmd, resultChan)
	if err != nil {
		return err
	}

	return nil
}
