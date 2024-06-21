package controller

import (
	"fmt"
	"path"
	"strings"

	"github.com/pefish/ci-tool/pkg/constant"
	"github.com/pefish/ci-tool/pkg/db"
	"github.com/pefish/ci-tool/pkg/global"
	go_best_type "github.com/pefish/go-best-type"
	_type "github.com/pefish/go-core-type/api-session"
	go_error "github.com/pefish/go-error"
	go_logger "github.com/pefish/go-logger"
	go_mysql "github.com/pefish/go-mysql"
	tg_sender "github.com/pefish/tg-sender"
)

type CiControllerType struct {
}

var CiController = CiControllerType{}

type CiStartParams struct {
	Env            string `json:"env" validate:"required"`
	Repo           string `json:"repo" validate:"required"`
	FetchCodeKey   string `json:"fetch_code_key" validate:"required"`
	Port           uint64 `json:"port"`
	AlertTgToken   string `json:"alert_tg_token"`
	AlertTgGroupId string `json:"alert_tg_group_id"`
	LokiUrl        string `json:"loki_url"`
	DockerNetwork  string `json:"docker_network"`
}

func (c *CiControllerType) CiStart(apiSession _type.IApiSession) (interface{}, *go_error.ErrorInfo) {
	var params CiStartParams
	err := apiSession.ScanParams(&params)
	if err != nil {
		go_logger.Logger.ErrorF("Read params error. %+v", err)
		return nil, go_error.INTERNAL_ERROR
	}

	atPos := strings.Index(params.Repo, "@")
	if atPos == -1 {
		if params.AlertTgToken != "" && params.AlertTgGroupId != "" {
			tg_sender.NewTgSender(params.AlertTgToken).
				SetLogger(go_logger.Logger).
				SendMsg(&tg_sender.MsgStruct{
					ChatId: params.AlertTgGroupId,
					Msg:    fmt.Sprintf("[ERROR] error: --repo [%s] is illegal.", params.Repo),
					Ats:    nil,
				}, 0)
		}
		return nil, go_error.WrapWithStr("Repo is illegal..")
	}
	colonPos := strings.Index(params.Repo, ":")
	slashPos := strings.Index(params.Repo, "/")
	username := params.Repo[colonPos+1 : slashPos]
	projectName := params.Repo[slashPos+1 : len(params.Repo)-4]

	// 检查数据库中是否有这个项目
	var project db.Project
	notFound, err := go_mysql.MysqlInstance.SelectFirst(
		&project,
		&go_mysql.SelectParams{
			TableName: "project",
			Select:    "*",
			Where:     "status = 1 and name = ?",
		},
		projectName,
	)
	if err != nil {
		go_logger.Logger.Error(err)
		return nil, go_error.INTERNAL_ERROR
	}
	if notFound {
		if params.AlertTgToken != "" && params.AlertTgGroupId != "" {
			tg_sender.NewTgSender(params.AlertTgToken).
				SetLogger(go_logger.Logger).
				SendMsg(&tg_sender.MsgStruct{
					ChatId: params.AlertTgGroupId,
					Msg:    fmt.Sprintf("[ERROR] <%s> CI 被禁用。", projectName),
					Ats:    nil,
				}, 0)
		}

		return nil, go_error.WrapWithStr("Project disabled.")
	}

	global.CiManager.Ask(&go_best_type.AskType{
		Action: constant.ActionType_CI,
		Data: map[string]interface{}{
			"env":            params.Env,
			"repo":           params.Repo,
			"fetch_code_key": params.FetchCodeKey,
			"git_username":   username,
			"project_name":   projectName,
			"src_path":       path.Join(global.GlobalConfig.SrcDir, username, projectName),
			"config": func() string {
				if project.Config == nil {
					return ""
				} else {
					return *project.Config
				}
			}(),
			"port":              params.Port,
			"alert_tg_token":    params.AlertTgToken,
			"alert_tg_group_id": params.AlertTgGroupId,
			"loki_url":          params.LokiUrl,
			"docker_network":    params.DockerNetwork,
		},
	})

	return true, nil
}

type CiLogParams struct {
	ProjectName string `json:"project_name" validate:"required"`
}

func (c *CiControllerType) CiLog(apiSession _type.IApiSession) (interface{}, *go_error.ErrorInfo) {
	var params CiLogParams
	err := apiSession.ScanParams(&params)
	if err != nil {
		go_logger.Logger.ErrorF("Read params error. %+v", err)
		return nil, go_error.INTERNAL_ERROR
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
