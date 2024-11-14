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
	go_mysql "github.com/pefish/go-mysql"
	go_shell "github.com/pefish/go-shell"
)

type CiControllerType struct {
}

var CiController = CiControllerType{}

func (c *CiControllerType) CiStart(apiSession i_core.IApiSession) (interface{}, *t_error.ErrorInfo) {
	var params db.CiParams
	err := apiSession.ScanParams(&params)
	if err != nil {
		apiSession.Logger().ErrorF("Read params error. %+v", err)
		return nil, t_error.INTERNAL_ERROR
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
		apiSession.Logger().Error(err)
		return nil, t_error.INTERNAL_ERROR
	}
	if notFound {
		util.Alert(
			apiSession.Logger(),
			fmt.Sprintf("[ERROR] <%s> CI 被禁用。", fullName),
		)

		return nil, t_error.WrapWithStr("Project disabled.")
	}

	_, err = global.MysqlInstance.Update(&t_mysql.UpdateParams{
		TableName: "project",
		Update: map[string]interface{}{
			"params": params,
		},
		Where: map[string]interface{}{
			"id": project.Id,
		},
	})
	if err != nil && err != go_mysql.ErrorNoAffectedRows {
		apiSession.Logger().Error(err)
		return nil, t_error.INTERNAL_ERROR
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
