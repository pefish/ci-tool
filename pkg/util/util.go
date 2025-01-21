package util

import (
	"fmt"
	"strings"
	"time"

	"github.com/pefish/ci-tool/pkg/global"
	go_http "github.com/pefish/go-http"
	i_logger "github.com/pefish/go-interface/i-logger"
	go_shell "github.com/pefish/go-shell"
	go_time "github.com/pefish/go-time"
	tg_sender "github.com/pefish/tg-sender"
	"github.com/pkg/errors"
)

func AlertNoError(logger i_logger.ILogger, msg string) {
	err := Alert(logger, msg)
	if err != nil {
		logger.ErrorF("发送通知失败!!! %+v", err)
	}
}

func Alert(logger i_logger.ILogger, msg string) error {
	switch global.GlobalConfig.AlertType {
	case "weixin":
		var httpResult struct {
			ErrCode uint64 `json:"errcode"`
			ErrMsg  string `json:"errmsg"`
		}
		_, _, err := go_http.NewHttpRequester(
			go_http.WithLogger(logger),
			go_http.WithTimeout(5*time.Second),
		).PostForStruct(
			&go_http.RequestParams{
				Url: fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=%s", global.GlobalConfig.AlertToken),
				Params: map[string]interface{}{
					"msgtype": "text",
					"text": map[string]interface{}{
						"content":        msg,
						"mentioned_list": []string{"@all"},
					},
				},
			},
			&httpResult,
		)
		if err != nil {
			return err
		}
		if httpResult.ErrCode != 0 {
			return errors.Errorf(httpResult.ErrMsg)
		}
	case "tg":
		err := tg_sender.NewTgSender(logger, global.GlobalConfig.AlertToken).
			SendMsg(&tg_sender.MsgStruct{
				ChatId: global.GlobalConfig.AlertChatId,
				Msg:    msg,
				Ats:    nil,
			}, 0)
		if err != nil {
			return err
		}
	default:
		return errors.Errorf("Alert type <%s> not be supported", global.GlobalConfig.AlertType)
	}

	return nil
}

func FetchErrorMsgFromContainer(containerName string) (string, error) {
	cmd := go_shell.NewCmd(`
#!/bin/bash
	
# 要检查的容器名称
container_name="%s"

# 检查容器是否存在
if ! sudo docker ps -a --filter "name=^${container_name}$" --format '{{.Names}}' | grep -q "^${container_name}$"; then
    echo "ERROR<CI-TOOL>: container not exist"
	exit 0
fi

sudo docker logs "${container_name}" --tail 200
		
	`, containerName)
	result, err := go_shell.ExecForResult(cmd)
	if err != nil {
		return "", err
	}
	if strings.Contains(result, "ERROR<CI-TOOL>") {
		return "", errors.New(result)
	}
	return result, nil
}

func StartContainer(containerName string) error {
	cmd := go_shell.NewCmd(`
#!/bin/bash

# 要检查的容器名称
container_name="%s"

# 检查容器是否存在
if ! sudo docker ps -a --filter "name=^${container_name}$" --format '{{.Names}}' | grep -q "^${container_name}$"; then
    echo "ERROR<CI-TOOL>: container not exist"
	exit 0
fi

# 检查容器是否存在且正在运行
if sudo docker ps --filter "name=^${container_name}$" --format '{{.Names}}' | grep -q "^${container_name}$"; then
    echo "ERROR<CI-TOOL>: running already"
    exit 0
fi

sudo docker start "${container_name}"
	
`, containerName)
	result, err := go_shell.ExecForResult(cmd)
	if err != nil {
		return err
	}
	if strings.Contains(result, "ERROR<CI-TOOL>") {
		return errors.New(result)
	}
	return nil
}

func StopContainer(containerName string) error {
	cmd := go_shell.NewCmd(`
#!/bin/bash

# 要检查的容器名称
container_name="%s"

# 检查容器是否存在
if ! sudo docker ps -a --filter "name=^${container_name}$" --format '{{.Names}}' | grep -q "^${container_name}$"; then
	echo "ERROR<CI-TOOL>: container not exist"
	exit 0
fi
	
# 检查容器是否存在且处于停止状态
if docker ps -a --filter "name=^${container_name}$" --filter "status=exited" --format '{{.Names}}' | grep -q "^${container_name}$"; then
    echo "ERROR<CI-TOOL>: stopped already"
    exit 0
fi
	
sudo docker stop "${container_name}"
		
	`, containerName)
	result, err := go_shell.ExecForResult(cmd)
	if err != nil {
		return err
	}
	if strings.Contains(result, "ERROR<CI-TOOL>") {
		return errors.New(result)
	}
	return nil
}

func RemoveContainer(containerName string) error {
	cmd := go_shell.NewCmd(`
#!/bin/bash

# 要检查的容器名称
container_name="%s"

# 检查容器是否存在
if ! sudo docker ps -a --filter "name=^${container_name}$" --format '{{.Names}}' | grep -q "^${container_name}$"; then
	echo "ERROR<CI-TOOL>: container not exist"
	exit 0
fi
	
# 检查容器是否存在且处于运行状态
if docker ps -a --filter "name=^${container_name}$" --filter "status=running" --format '{{.Names}}' | grep -q "^${container_name}$"; then
    sudo docker stop "${container_name}"
fi

sudo docker rm "${container_name}"
		
	`, containerName)
	result, err := go_shell.ExecForResult(cmd)
	if err != nil {
		return err
	}
	if strings.Contains(result, "ERROR<CI-TOOL>") {
		return errors.New(result)
	}
	return nil
}

func RestartContainer(containerName string) error {
	cmd := go_shell.NewCmd(`
#!/bin/bash

# 要检查的容器名称
container_name="%s"

# 检查容器是否存在
if ! sudo docker ps -a --filter "name=^${container_name}$" --format '{{.Names}}' | grep -q "^${container_name}$"; then
	echo "ERROR<CI-TOOL>: container not exist"
	exit 0
fi
	
sudo docker restart "${container_name}"
		
`, containerName)
	result, err := go_shell.ExecForResult(cmd)
	if err != nil {
		return err
	}
	if strings.Contains(result, "ERROR<CI-TOOL>") {
		return errors.New(result)
	}
	return nil
}

func ListAllAliveContainers(logger i_logger.ILogger) ([]string, error) {
	cmd := go_shell.NewCmd(`
#!/bin/bash
set -euxo pipefail

sudo docker ps --format "table {{.Names}}"
`)
	logger.DebugF("Exec shell: <%s>", cmd.String())
	result, err := go_shell.ExecForResult(cmd)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(result, "\n")

	return lines[1 : len(lines)-1], nil
}

func GetGitShortCommitHash(srcPath string) (string, error) {
	shortCommitHash, err := go_shell.ExecForResult(go_shell.NewCmd(
		`
#!/bin/bash

src="` + srcPath + `"
cd ${src}
echo $(git rev-parse --short HEAD)
`,
	))
	if err != nil {
		return "", err
	}

	return strings.TrimSuffix(shortCommitHash, "\n"), nil
}

func GitPullSourceCode(
	resultChan chan string,
	srcPath string,
	repo string,
	fetchCodeKey string,
	branch string,
) error {
	err := go_shell.ExecForResultLineByLine(go_shell.NewCmd(
		`
#!/bin/bash
set -euxo pipefail

src="`+srcPath+`"

# 检查源代码目录是否存在
if [ ! -d "$src" ]; then
    echo "源代码目录 '$src' 不存在，正在克隆仓库..."
    git clone%s "`+repo+`" "$src"
    
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

`,
		func() string {
			if fetchCodeKey == "" {
				return ""
			} else {
				return fmt.Sprintf(` --config core.sshCommand="ssh -i %s"`, fetchCodeKey)
			}
		}(),
	), resultChan)
	if err != nil {
		return err
	}

	return nil
}

func BuildImage(
	resultChan chan string,
	srcPath string,
	env string,
	imageName string,
) error {
	err := go_shell.ExecForResultLineByLine(go_shell.NewCmd(
		`
#!/bin/bash
set -euxo pipefail

cd `+srcPath+`

if [[ "$(sudo docker images -q `+imageName+` 2> /dev/null)" == "" ]]; then
  sudo docker build --build-arg APP_ENV=`+env+` -t `+imageName+` .
fi
`,
	), resultChan)
	if err != nil {
		return err
	}

	return nil
}

func ContainerExists(containerName string) (bool, error) {
	r, err := go_shell.ExecForResult(go_shell.NewCmd(
		`
#!/bin/bash

if sudo docker ps -a --filter "name=^` + containerName + `$" --format '{{.Names}}' | grep -q "` + containerName + `"; then
	echo 1
	exit 0
fi

echo 0
`,
	))
	if err != nil {
		return false, err
	}

	return r == "1\n", nil
}

func ListProjectContainers(fullName string) ([]string, error) {
	r, err := go_shell.ExecForResult(go_shell.NewCmd(
		`
#!/bin/bash

echo $(sudo docker ps -a --filter "name=%s" --format '{{.Names}}')
`,
		fullName,
	))
	if err != nil {
		return nil, err
	}
	r = strings.TrimSuffix(r, "\n")
	if r == "" {
		return nil, nil
	}

	return strings.Split(r, " "), nil
}

func StartNewContainer(
	resultChan chan string,
	imageName string,
	envConfig string,
	port uint64,
	network string,
	containerName string,
) error {
	portStr := ""
	if port != 0 {
		portStr = fmt.Sprintf("-p %d:8000", port)
	}
	networkStr := ""
	if network != "" {
		networkStr = fmt.Sprintf(`--network %s`, network)
	}
	err := go_shell.ExecForResultLineByLine(go_shell.NewCmd(
		`
#!/bin/bash
set -euxo pipefail

# 创建一个临时文件
TEMP_FILE=$(mktemp)

echo "`+envConfig+`" > "$TEMP_FILE"

sudo docker run --name `+containerName+` --env-file "$TEMP_FILE" -d `+portStr+` `+networkStr+` `+imageName+`

# 删除临时文件
rm "$TEMP_FILE"
`,
	), resultChan)
	if err != nil {
		return err
	}

	return nil
}

func BackupContainerLog(
	resultChan chan string,
	logsPath string,
	containerName string,
	startLogTime time.Time,
) (isPacked_ bool, err_ error) {
	err := go_shell.ExecForResultLineByLine(go_shell.NewCmd(
		`
#!/bin/bash
set -euxo pipefail

containerId=$(sudo docker inspect `+containerName+` | grep '"Id"' | head -1 | awk -F '"' '{print $4}')

logPath="/var/lib/docker/containers/${containerId}/${containerId}-json.log"

sudo cat ${logPath} >> `+logsPath+`/current.log

echo "日志已备份"
`,
	), resultChan)
	if err != nil {
		return false, err
	}

	if time.Since(startLogTime) > 3*24*time.Hour {
		err = go_shell.ExecForResultLineByLine(go_shell.NewCmd(
			`
#!/bin/bash
set -euxo pipefail

mv `+logsPath+`/current.log `+logsPath+`/%s_%s.log

echo "日志已打包"
`,
			go_time.TimeToStr(startLogTime, "000000000000"),
			go_time.TimeToStr(time.Now(), "000000000000"),
		), resultChan)
		if err != nil {
			return false, err
		}
		return true, nil
	}

	return false, nil
}

func RemoveImage(
	resultChan chan string,
	imageName string,
) error {
	err := go_shell.ExecForResultLineByLine(go_shell.NewCmd(
		`
#!/bin/bash
set -euxo pipefail

sudo docker rmi "%s" || echo "Failed to delete image, continuing..."
`,
		imageName,
	), resultChan)
	if err != nil {
		return err
	}

	return nil
}
