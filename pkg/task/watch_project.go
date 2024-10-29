package task

import (
	"context"
	"fmt"
	"time"

	"github.com/pefish/ci-tool/pkg/db"
	"github.com/pefish/ci-tool/pkg/global"
	"github.com/pefish/ci-tool/pkg/util"
	i_logger "github.com/pefish/go-interface/i-logger"
	t_mysql "github.com/pefish/go-interface/t-mysql"
)

type WatchProject struct {
	logger i_logger.ILogger
}

func NewWatchProject(logger i_logger.ILogger) *WatchProject {
	w := &WatchProject{}
	w.logger = logger.CloneWithPrefix(w.Name())
	return w
}

func (t *WatchProject) Init(ctx context.Context) error {
	return nil
}

func (t *WatchProject) Run(ctx context.Context) error {
	projects := make([]*db.Project, 0)
	err := global.MysqlInstance.Select(
		&projects,
		&t_mysql.SelectParams{
			TableName: "project",
			Select:    "*",
		},
	)
	if err != nil {
		return err
	}

	for _, project := range projects {
		containerName := fmt.Sprintf("%s-prod", project.Name)
		if project.Status == 0 {
			continue
		}

		if project.Start == 1 {
			err = util.StartContainer(t.logger, containerName)
			if err != nil {
				util.Alert(t.logger, fmt.Sprintf(`
项目 <%s> 启动失败 (%s)
				`, containerName, err.Error()))
			} else {
				util.Alert(t.logger, fmt.Sprintf(`
项目 <%s> 启动成功
				`, containerName))
			}
			global.MysqlInstance.Update(
				&t_mysql.UpdateParams{
					TableName: "project",
					Update: map[string]interface{}{
						"start": 0,
					},
					Where: map[string]interface{}{
						"id": project.Id,
					},
				},
			)
		}

		if project.Stop == 1 {
			err = util.StopContainer(t.logger, containerName)
			if err != nil {
				util.Alert(t.logger, fmt.Sprintf(`
项目 <%s> 停止失败 (%s)
				`, containerName, err.Error()))
			} else {
				util.Alert(t.logger, fmt.Sprintf(`
项目 <%s> 停止成功
				`, containerName))
			}
			global.MysqlInstance.Update(
				&t_mysql.UpdateParams{
					TableName: "project",
					Update: map[string]interface{}{
						"stop": 0,
					},
					Where: map[string]interface{}{
						"id": project.Id,
					},
				},
			)
		}

		if project.Restart == 1 {
			err = util.RestartContainer(t.logger, containerName)
			if err != nil {
				util.Alert(t.logger, fmt.Sprintf(`
项目 <%s> 重启失败 (%s)
				`, containerName, err.Error()))
			} else {
				util.Alert(t.logger, fmt.Sprintf(`
项目 <%s> 重启成功
				`, containerName))
			}
			global.MysqlInstance.Update(
				&t_mysql.UpdateParams{
					TableName: "project",
					Update: map[string]interface{}{
						"restart": 0,
					},
					Where: map[string]interface{}{
						"id": project.Id,
					},
				},
			)
		}

	}
	return nil
}

func (t *WatchProject) Stop() error {
	return nil
}

func (t *WatchProject) Name() string {
	return "WatchProject"
}

func (t *WatchProject) Interval() time.Duration {
	return 3 * time.Second
}

func (t *WatchProject) Logger() i_logger.ILogger {
	return t.logger
}
