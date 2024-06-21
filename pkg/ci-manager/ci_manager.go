package ci_manager

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"github.com/pefish/ci-tool/pkg/constant"
	go_best_type "github.com/pefish/go-best-type"
	go_logger "github.com/pefish/go-logger"
	go_shell "github.com/pefish/go-shell"
	tg_sender "github.com/pefish/tg-sender"
	"github.com/pkg/errors"
)

type CiManagerType struct {
	go_best_type.BaseBestType
	logs sync.Map // map[string]string
}

func NewCiManager(ctx context.Context, name string) *CiManagerType {
	c := &CiManagerType{}
	c.BaseBestType = *go_best_type.NewBaseBestType(
		c,
		name,
	)
	return c
}

func (c *CiManagerType) ProcessOtherAsk(exitChan <-chan go_best_type.ExitType, ask *go_best_type.AskType) error {
	data := ask.Data.(map[string]interface{})
	switch ask.Action {
	case constant.ActionType_CI:
		env := data["env"].(string)
		repo := data["repo"].(string)
		fetchCodeKey := data["fetch_code_key"].(string)
		gitUsername := data["git_username"].(string)
		projectName := data["project_name"].(string)
		srcPath := data["src_path"].(string)
		config := data["config"].(string)
		port := data["port"].(uint64)
		alertTgToken := data["alert_tg_token"].(string)
		alertTgGroupId := data["alert_tg_group_id"].(string)
		lokiUrl := data["loki_url"].(string)
		dockerNetwork := data["docker_network"].(string)
		go func() {
			logger := go_logger.Logger.CloneWithPrefix(projectName)
			logger.InfoF("<%s> running...\n", projectName)
			c.logs.Delete(projectName)
			err := c.startCi(
				logger,
				env,
				repo,
				fetchCodeKey,
				gitUsername,
				srcPath,
				config,
				projectName,
				port,
				lokiUrl,
				dockerNetwork,
			)
			if err != nil {
				c.logs.Store(projectName, err.Error())
				if alertTgGroupId != "" {
					tg_sender.NewTgSender(alertTgToken).
						SetLogger(go_logger.Logger).
						SendMsg(&tg_sender.MsgStruct{
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
					SendMsg(&tg_sender.MsgStruct{
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

	select {
	case <-exitChan:
	}

	return nil
}

func (c *CiManagerType) Start(exitChan <-chan go_best_type.ExitType, ask *go_best_type.AskType) error {
	return nil
}

func (c *CiManagerType) startCi(
	logger go_logger.InterfaceLogger,
	env,
	repo,
	fetchCodeKey,
	gitUsername,
	srcPath,
	config,
	projectName string,
	port uint64,
	lokiUrl string,
	dockerNetwork string,
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

	script := fmt.Sprintf(
		`
#!/bin/bash
set -euxo pipefail

src="`+srcPath+`"

# 检查源代码目录是否存在
if [ ! -d "$src" ]; then
    echo "源代码目录 '$src' 不存在，正在克隆仓库..."
    git clone --config core.sshCommand="ssh -i `+fetchCodeKey+`" "`+repo+`" "$src"
    
    if [ $? -eq 0 ]; then
        echo "克隆成功！"
    else
        echo "克隆失败，请检查 Git 仓库 URL 和网络连接。"
        exit 1
    fi
fi

cd ${src}
git config core.sshCommand "ssh -i `+fetchCodeKey+`"
git reset --hard && git clean -d -f . && git pull && git checkout `+branch+` && git pull

imageName="`+gitUsername+`-`+projectName+`:$(git rev-parse --short HEAD)"

if [[ "$(sudo docker images -q ${imageName} 2> /dev/null)" == "" ]]; then
  sudo docker build --build-arg APP_ENV=`+env+` -t ${imageName} .
fi

containerName="`+gitUsername+`-`+projectName+`-`+env+`"

sudo docker stop ${containerName} && sudo docker rm ${containerName}

# 创建一个临时文件
TEMP_FILE=$(mktemp)

echo "`+config+`" > "$TEMP_FILE"

sudo docker run --name ${containerName} --env-file "$TEMP_FILE" -d %s%s%s ${imageName}

# 删除临时文件
rm "$TEMP_FILE"

`,
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
		func() string {
			if dockerNetwork == "" {
				return ""
			} else {
				return fmt.Sprintf(` --network %s`, dockerNetwork)
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
