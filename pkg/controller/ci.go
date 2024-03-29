package controller

import (
	"github.com/pefish/ci-tool/pkg/constant"
	"github.com/pefish/ci-tool/pkg/global"
	go_best_type "github.com/pefish/go-best-type"
	_type "github.com/pefish/go-core-type/api-session"
	go_error "github.com/pefish/go-error"
	go_logger "github.com/pefish/go-logger"
)

type CiControllerType struct {
}

var CiController = CiControllerType{}

type CiStartParams struct {
	Env            string `json:"env" validate:"required"`
	ProjectName    string `json:"project_name" validate:"required"`
	SrcPath        string `json:"src_path" validate:"required"`
	Token          string `json:"token" validate:"required"`
	Port           uint64 `json:"port"`
	ConfigPath     string `json:"config_path"`
	AlertTgToken   string `json:"alert_tg_token"`
	AlertTgGroupId string `json:"alert_tg_group_id"`
	LokiUrl        string `json:"loki_url"`
}

func (c *CiControllerType) CiStart(apiSession _type.IApiSession) (interface{}, *go_error.ErrorInfo) {
	var params CiStartParams
	err := apiSession.ScanParams(&params)
	if err != nil {
		go_logger.Logger.ErrorF("Read params error. %+v", err)
		return nil, go_error.INTERNAL_ERROR
	}

	if params.Token != global.GlobalConfig.Token {
		global.CiManager.Ask(&go_best_type.AskType{
			Action: constant.ActionType_LOG,
			Data: map[string]interface{}{
				"project_name": params.ProjectName,
				"msg":          "Token error.",
			},
		})
		return nil, go_error.WrapWithStr("Token error.")
	}

	global.CiManager.Ask(&go_best_type.AskType{
		Action: constant.ActionType_CI,
		Data: map[string]interface{}{
			"env":               params.Env,
			"project_name":      params.ProjectName,
			"src_path":          params.SrcPath,
			"port":              params.Port,
			"config_path":       params.ConfigPath,
			"alert_tg_token":    params.AlertTgToken,
			"alert_tg_group_id": params.AlertTgGroupId,
			"loki_url":          params.LokiUrl,
		},
	})

	return params, nil
}

type CiLogParams struct {
	ProjectName string `json:"project_name" validate:"required"`
	Token       string `json:"token" validate:"required"`
}

func (c *CiControllerType) CiLog(apiSession _type.IApiSession) (interface{}, *go_error.ErrorInfo) {
	var params CiLogParams
	err := apiSession.ScanParams(&params)
	if err != nil {
		go_logger.Logger.ErrorF("Read params error. %+v", err)
		return nil, go_error.INTERNAL_ERROR
	}

	if params.Token != global.GlobalConfig.Token {
		return nil, go_error.WrapWithStr("Token error.")
	}

	answer := global.CiManager.AskForAnswer(&go_best_type.AskType{
		Action: constant.ActionType_ReadLog,
		Data: map[string]interface{}{
			"project_name": params.ProjectName,
		},
	})

	apiSession.WriteText(answer.(string))

	return nil, nil
}
