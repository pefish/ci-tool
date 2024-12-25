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
	go_shell "github.com/pefish/go-shell"
)

type CiControllerType struct {
}

var CiController = CiControllerType{}

type CiStartParams struct {
	Name    string `json:"name" validate:"required"`
	OrgName string `json:"org_name" validate:"required"`
}

func (c *CiControllerType) CiStart(apiSession i_core.IApiSession) (interface{}, *t_error.ErrorInfo) {
	var params CiStartParams
	err := apiSession.ScanParams(&params)
	if err != nil {
		apiSession.Logger().ErrorF("Read params error. %+v", err)
		return nil, t_error.INTERNAL_ERROR
	}

	for _, name := range strings.Split(params.Name, ",") {
		fullName := fmt.Sprintf("%s-%s", params.OrgName, name)
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
			apiSession.Logger().Error(err)
			return nil, t_error.INTERNAL_ERROR
		}
		if notFound {
			util.AlertNoError(
				apiSession.Logger(),
				fmt.Sprintf("[ERROR] <%s> CI 被禁用。", fullName),
			)

			return nil, t_error.WrapWithStr("Project disabled.")
		}

		if project.Params == nil {
			util.AlertNoError(
				apiSession.Logger(),
				fmt.Sprintf("[ERROR] <%s> CI 参数没有配置。", fullName),
			)

			return nil, t_error.WrapWithStr("CI 参数没有配置.")
		}

		go ci_manager.CiManager.StartCi(
			global.Command.Ctx,
			&project,
			path.Join(global.GlobalConfig.SrcDir, params.OrgName, name),
			fullName,
		)
	}

	return true, nil
}

type CiLogParams struct {
	FullName string `json:"name" validate:"required"`
}

func (c *CiControllerType) CiLog(apiSession i_core.IApiSession) (interface{}, *t_error.ErrorInfo) {
	var params CiLogParams
	err := apiSession.ScanParams(&params)
	if err != nil {
		apiSession.Logger().ErrorF("Read params error. %+v", err)
		return nil, t_error.INTERNAL_ERROR
	}

	apiSession.WriteText(ci_manager.CiManager.Logs(params.FullName))

	return nil, nil
}

type DockerLogsParams struct {
	Name  string `json:"name" validate:"required"`
	Lines uint64 `json:"lines" default:"200"`
}

func (c *CiControllerType) DockerLogs(apiSession i_core.IApiSession) (interface{}, *t_error.ErrorInfo) {
	var params DockerLogsParams
	err := apiSession.ScanParams(&params)
	if err != nil {
		apiSession.Logger().ErrorF("Read params error. %+v", err)
		return nil, t_error.INTERNAL_ERROR
	}

	cmdStr := fmt.Sprintf("sudo docker logs %s --tail %d", params.Name, params.Lines)
	apiSession.Logger().Debug(cmdStr)
	cmd := go_shell.NewCmd(cmdStr)
	result, err := go_shell.ExecForResult(cmd)
	if err != nil {
		apiSession.Logger().ErrorF("Exec docker logs error. %+v", err)
		return nil, t_error.INTERNAL_ERROR
	}

	apiSession.WriteText(result)

	return nil, nil
}
