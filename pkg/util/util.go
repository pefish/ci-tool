package util

import (
	"fmt"
	"time"

	"github.com/pefish/ci-tool/pkg/global"
	go_http "github.com/pefish/go-http"
	i_logger "github.com/pefish/go-interface/i-logger"
	tg_sender "github.com/pefish/tg-sender"
	"github.com/pkg/errors"
)

// 1769f0d0-fbbc-45bb-9f60-da2c04776e56
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
