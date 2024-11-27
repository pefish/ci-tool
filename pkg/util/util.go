package util

import (
	"fmt"
	"strings"
	"time"

	"github.com/pefish/ci-tool/pkg/global"
	go_http "github.com/pefish/go-http"
	i_logger "github.com/pefish/go-interface/i-logger"
	go_shell "github.com/pefish/go-shell"
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

func FetchErrorMsgFromContainer(logger i_logger.ILogger, containerName string) (string, error) {
	cmd := go_shell.NewCmd(`
#!/bin/bash
	
# 要检查的容器名称
container_name="%s"

# 检查容器是否存在
if ! sudo docker ps -a --filter "name=^${container_name}$" --format '{{.Names}}' | grep -q "^${container_name}$"; then
    echo "ERROR: container not exist"
	exit 1
fi

sudo docker logs "${container_name}" --tail 200
		
	`, containerName)
	result, err := go_shell.ExecForResult(cmd)
	if err != nil {
		return "", err
	}
	if strings.Contains(result, "ERROR") {
		return "", errors.New(result)
	}
	return result, nil
}

func StartContainer(logger i_logger.ILogger, containerName string) error {
	cmd := go_shell.NewCmd(`
#!/bin/bash

# 要检查的容器名称
container_name="%s"

# 检查容器是否存在
if ! sudo docker ps -a --filter "name=^${container_name}$" --format '{{.Names}}' | grep -q "^${container_name}$"; then
    echo "ERROR: container not exist"
	exit 1
fi

# 检查容器是否存在且正在运行
if sudo docker ps --filter "name=^${container_name}$" --format '{{.Names}}' | grep -q "^${container_name}$"; then
    echo "ERROR: running already"
    exit 1
fi

sudo docker start "${container_name}"
	
`, containerName)
	result, err := go_shell.ExecForResult(cmd)
	if err != nil {
		return err
	}
	if strings.Contains(result, "ERROR") {
		return errors.New(result)
	}
	return nil
}

func StopContainer(logger i_logger.ILogger, containerName string) error {
	cmd := go_shell.NewCmd(`
#!/bin/bash

# 要检查的容器名称
container_name="%s"

# 检查容器是否存在
if ! sudo docker ps -a --filter "name=^${container_name}$" --format '{{.Names}}' | grep -q "^${container_name}$"; then
	echo "ERROR: container not exist"
	exit 1
fi
	
# 检查容器是否存在且处于停止状态
if docker ps -a --filter "name=^${container_name}$" --filter "status=exited" --format '{{.Names}}' | grep -q "^${container_name}$"; then
    echo "ERROR: stopped already"
    exit 1
fi
	
sudo docker stop "${container_name}"
		
	`, containerName)
	result, err := go_shell.ExecForResult(cmd)
	if err != nil {
		return err
	}
	if strings.Contains(result, "Error") {
		return errors.New(result)
	}
	return nil
}

func RestartContainer(logger i_logger.ILogger, containerName string) error {
	cmd := go_shell.NewCmd(`
#!/bin/bash

# 要检查的容器名称
container_name="%s"

# 检查容器是否存在
if ! sudo docker ps -a --filter "name=^${container_name}$" --format '{{.Names}}' | grep -q "^${container_name}$"; then
	echo "ERROR: container not exist"
	exit 1
fi
	
sudo docker restart "${container_name}"
		
`, containerName)
	result, err := go_shell.ExecForResult(cmd)
	if err != nil {
		return err
	}
	if strings.Contains(result, "Error") {
		return errors.New(result)
	}
	return nil
}

func ListAllAliveContainers(logger i_logger.ILogger) ([]string, error) {
	cmd := go_shell.NewCmd(`sudo docker ps --format "table {{.Names}}"`)
	logger.DebugF("Exec shell: <%s>", cmd.String())
	result, err := go_shell.ExecForResult(cmd)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(result, "\n")

	return lines[1 : len(lines)-1], nil
}
