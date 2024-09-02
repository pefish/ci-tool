package controller

import (
	"fmt"
	"path"
	"strings"

	ci_manager "github.com/pefish/ci-tool/pkg/ci-manager"
	"github.com/pefish/ci-tool/pkg/db"
	"github.com/pefish/ci-tool/pkg/global"
	"github.com/pefish/ci-tool/pkg/util"
	i_core "github.com/pefish/go-interface/i-core"
	t_error "github.com/pefish/go-interface/t-error"
	t_mysql "github.com/pefish/go-interface/t-mysql"
	go_logger "github.com/pefish/go-logger"
)

type CiControllerType struct {
}

var CiController = CiControllerType{}

type CiStartParams struct {
	Env            string `json:"env" validate:"required"`
	Repo           string `json:"repo" validate:"required"`
	FetchCodeKey   string `json:"fetch_code_key" validate:"required"`
	AlertTgToken   string `json:"alert_tg_token"`
	AlertTgGroupId string `json:"alert_tg_group_id"`
	LokiUrl        string `json:"loki_url"`
	DockerNetwork  string `json:"docker_network"`
}

func (c *CiControllerType) CiStart(apiSession i_core.IApiSession) (interface{}, *t_error.ErrorInfo) {
	var params CiStartParams
	err := apiSession.ScanParams(&params)
	if err != nil {
		go_logger.Logger.ErrorF("Read params error. %+v", err)
		return nil, t_error.INTERNAL_ERROR
	}

	atPos := strings.Index(params.Repo, "@")
	if atPos == -1 {
		util.Alert(
			go_logger.Logger,
			params.AlertTgToken,
			params.AlertTgGroupId,
			fmt.Sprintf("[ERROR] error: --repo [%s] is illegal.", params.Repo),
		)
		return nil, t_error.WrapWithStr("Repo is illegal..")
	}
	colonPos := strings.Index(params.Repo, ":")
	slashPos := strings.Index(params.Repo, "/")
	gitUsername := params.Repo[colonPos+1 : slashPos]
	projectName := params.Repo[slashPos+1 : len(params.Repo)-4]
	fullName := strings.ToLower(fmt.Sprintf("%s-%s", gitUsername, projectName))

	// 检查数据库中是否有这个项目
	var project db.Project
	notFound, err := global.MysqlInstance.SelectFirst(
		&project,
		&t_mysql.SelectParams{
			TableName: "project",
			Select:    "*",
			Where:     "status = 1 and name = ?",
		},
		fullName,
	)
	if err != nil {
		go_logger.Logger.Error(err)
		return nil, t_error.INTERNAL_ERROR
	}
	if notFound {
		util.Alert(
			go_logger.Logger,
			params.AlertTgToken,
			params.AlertTgGroupId,
			fmt.Sprintf("[ERROR] <%s> CI 被禁用。", fullName),
		)

		return nil, t_error.WrapWithStr("Project disabled.")
	}

	go ci_manager.CiManager.StartCi(
		params.Env,
		params.Repo,
		params.FetchCodeKey,
		gitUsername,
		path.Join(global.GlobalConfig.SrcDir, gitUsername, projectName),
		func() string {
			if project.Config == nil {
				return ""
			} else {
				return *project.Config
			}
		}(),
		fullName,
		project.Port,
		params.LokiUrl,
		params.DockerNetwork,
		params.AlertTgToken,
		params.AlertTgGroupId,
	)

	return true, nil
}

type CiLogParams struct {
	FullName string `json:"name" validate:"required"`
}

func (c *CiControllerType) CiLog(apiSession i_core.IApiSession) (interface{}, *t_error.ErrorInfo) {
	var params CiLogParams
	err := apiSession.ScanParams(&params)
	if err != nil {
		go_logger.Logger.ErrorF("Read params error. %+v", err)
		return nil, t_error.INTERNAL_ERROR
	}

	apiSession.WriteText(ci_manager.CiManager.Logs(params.FullName))

	return nil, nil
}
