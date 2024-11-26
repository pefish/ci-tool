package task

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/pefish/ci-tool/pkg/db"
	"github.com/pefish/ci-tool/pkg/global"
	"github.com/pefish/ci-tool/pkg/util"
	go_format_slice "github.com/pefish/go-format/slice"
	i_logger "github.com/pefish/go-interface/i-logger"
	t_mysql "github.com/pefish/go-interface/t-mysql"
)

type WatchContainer struct {
	logger i_logger.ILogger
}

func NewWatchContainer(logger i_logger.ILogger) *WatchContainer {
	w := &WatchContainer{}
	w.logger = logger.CloneWithPrefix(w.Name())
	return w
}

func (t *WatchContainer) Init(ctx context.Context) error {
	return nil
}

func (t *WatchContainer) Run(ctx context.Context) error {
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

	// 删除的项目从 deadProjects 中移除
	for _, deadProject := range go_format_slice.DeepCopy(global.GlobalData.DeadProjects) {
		shouldCheck := false
		for _, project := range projects {
			containerName := fmt.Sprintf("%s-prod", project.Name)
			if strings.EqualFold(containerName, deadProject) && project.Status == 1 {
				shouldCheck = true
				break
			}
		}
		if shouldCheck {
			continue
		}
		t.logger.InfoF("<%s> 从 deadProjects 中移除", deadProject)
		global.GlobalData.DeadProjects = slices.DeleteFunc(global.GlobalData.DeadProjects, func(containerName_ string) bool {
			return containerName_ == deadProject
		})
	}

	// 检查需要监控的项目
	aliveProjects, err := util.ListAllAliveContainers(t.logger)
	if err != nil {
		return err
	}

	for _, project := range projects {
		containerName := fmt.Sprintf("%s-prod", project.Name)
		if project.Status == 0 {
			continue
		}
		isAlive := false
		for _, aliveProject := range aliveProjects {
			if strings.EqualFold(containerName, aliveProject) {
				isAlive = true
				break
			}
		}
		if isAlive {
			if slices.Contains(global.GlobalData.DeadProjects, containerName) {
				t.logger.InfoF("<%s> 复活，从 deadProjects 中移除", containerName)
				global.GlobalData.DeadProjects = slices.DeleteFunc(global.GlobalData.DeadProjects, func(containerName_ string) bool {
					return containerName_ == containerName
				})
				util.AlertNoError(t.logger, fmt.Sprintf(`
项目 <%s> 已复活
`, containerName))
			}
			continue
		}

		if !slices.Contains(global.GlobalData.DeadProjects, containerName) {
			t.logger.InfoF("<%s> 意外终止，下次检查如果还处于终止状态，则会报警", containerName)
			global.GlobalData.DeadProjects = append(global.GlobalData.DeadProjects, containerName)

			// 记录错误信息
			errorMsg, err := util.FetchErrorMsgFromContainer(t.logger, containerName)
			if err != nil {
				t.logger.InfoF("docker logs 命令执行出错：%+v", err)
			} else {
				_, err = global.MysqlInstance.Update(
					&t_mysql.UpdateParams{
						TableName: "project",
						Update: map[string]interface{}{
							"last_error": errorMsg,
						},
						Where: map[string]interface{}{
							"id": project.Id,
						},
					},
				)
				if err != nil {
					t.logger.Info(err)
				}
			}

			if project.IsAutoRestart == 1 {
				err = util.StartContainer(t.logger, containerName)
				if err != nil {
					return err
				}
				util.AlertNoError(t.logger, fmt.Sprintf(`
项目 <%s> 已重启，请关注错误信息
`, containerName))
			}
			continue
		}

		if time.Since(global.GlobalData.LastNotifyTime[containerName]) < 10*time.Minute {
			t.logger.InfoF("<%s> 短时间内报警过，略过报警", containerName)
			continue
		}

		// 警报
		util.AlertNoError(t.logger, fmt.Sprintf(`
项目 <%s> 意外终止，请检查
`, containerName))
		global.GlobalData.LastNotifyTime[containerName] = time.Now()
	}
	return nil
}

func (t *WatchContainer) Stop() error {
	return nil
}

func (t *WatchContainer) Name() string {
	return "WatchContainer"
}

func (t *WatchContainer) Interval() time.Duration {
	return time.Minute
}

func (t *WatchContainer) Logger() i_logger.ILogger {
	return t.logger
}
