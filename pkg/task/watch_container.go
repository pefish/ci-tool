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
	go_shell "github.com/pefish/go-shell"
	"github.com/pkg/errors"
)

type WatchContainer struct {
	logger         i_logger.ILogger
	deadProjects   []string
	lastNotifyTime map[string]time.Time
}

func NewWatchContainer(logger i_logger.ILogger) *WatchContainer {
	w := &WatchContainer{
		deadProjects:   make([]string, 0),
		lastNotifyTime: make(map[string]time.Time, 0),
	}
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
	for _, deadProject := range go_format_slice.DeepCopy(t.deadProjects) {
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
		t.deadProjects = slices.DeleteFunc(t.deadProjects, func(containerName_ string) bool {
			return containerName_ == deadProject
		})
	}

	// 检查需要监控的项目
	aliveProjects, err := ListAllAliveContainers()
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
			if slices.Contains(t.deadProjects, containerName) {
				t.logger.InfoF("<%s> 复活，从 deadProjects 中移除", containerName)
				t.deadProjects = slices.DeleteFunc(t.deadProjects, func(containerName_ string) bool {
					return containerName_ == containerName
				})
				util.Alert(t.logger, fmt.Sprintf(`
项目 <%s> 已复活
`, containerName))
			}
			continue
		}

		if !slices.Contains(t.deadProjects, containerName) {
			t.logger.InfoF("<%s> 意外终止，下次检查如果还处于终止状态，则会报警", containerName)
			t.deadProjects = append(t.deadProjects, containerName)

			// 记录错误信息
			errorMsg, err := FetchErrorMsgFromContainer(containerName)
			if err != nil {
				return err
			}
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
				return err
			}
			if project.IsAutoRestart == 1 {
				err = StartContainer(containerName)
				if err != nil {
					return err
				}
				util.Alert(t.logger, fmt.Sprintf(`
项目 <%s> 已重启，请关注错误信息
`, containerName))
			}
			continue
		}

		if time.Since(t.lastNotifyTime[containerName]) < 10*time.Minute {
			t.logger.InfoF("<%s> 短时间内报警过，略过报警", containerName)
			continue
		}

		// 警报
		util.Alert(t.logger, fmt.Sprintf(`
项目 <%s> 意外终止，请检查
`, containerName))
		t.lastNotifyTime[containerName] = time.Now()
	}
	return nil
}

func FetchErrorMsgFromContainer(containerName string) (string, error) {
	result, err := go_shell.ExecForResult(go_shell.NewCmd(fmt.Sprintf(`sudo docker logs %s --tail 200"`, containerName)))
	if err != nil {
		return "", err
	}
	return result, nil
}

func StartContainer(containerName string) error {
	result, err := go_shell.ExecForResult(go_shell.NewCmd(fmt.Sprintf(`sudo docker start %s"`, containerName)))
	if err != nil {
		return err
	}
	if strings.Contains(result, "Error") {
		return errors.New(result)
	}
	return nil
}

func ListAllAliveContainers() ([]string, error) {
	result, err := go_shell.ExecForResult(go_shell.NewCmd(`sudo docker ps --format "table {{.Names}}"`))
	if err != nil {
		return nil, err
	}
	lines := strings.Split(result, "\n")

	return lines[1 : len(lines)-1], nil
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
