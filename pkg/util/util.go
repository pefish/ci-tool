package util

import (
	"github.com/pefish/ci-tool/pkg/global"
	i_logger "github.com/pefish/go-interface/i-logger"
	tg_sender "github.com/pefish/tg-sender"
)

func Alert(logger i_logger.ILogger, token string, chatId string, msg string) {
	if token == "" || chatId == "" {
		tg_sender.NewTgSender(logger, global.GlobalConfig.AlertToken).
			SendMsg(&tg_sender.MsgStruct{
				ChatId: global.GlobalConfig.AlertChatId,
				Msg:    msg,
				Ats:    nil,
			}, 0)
	} else {
		tg_sender.NewTgSender(logger, token).
			SendMsg(&tg_sender.MsgStruct{
				ChatId: chatId,
				Msg:    msg,
				Ats:    nil,
			}, 0)
	}
}
