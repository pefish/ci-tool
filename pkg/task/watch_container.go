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
	i_logger "github.com/pefish/go-interface/i-logger"
	t_mysql "github.com/pefish/go-interface/t-mysql"
	go_shell "github.com/pefish/go-shell"
)

type WatchContainer struct {
	logger       i_logger.ILogger
	deadProjects []string
}

func NewWatchContainer(logger i_logger.ILogger) *WatchContainer {
	w := &WatchContainer{
		deadProjects: make([]string, 0),
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

	// 禁用的项目从 deadProjects 中移除
	for _, project := range projects {
		containerName := fmt.Sprintf("%s-prod", project.Name)
		if project.Status == 1 {
			continue
		}
		if !slices.Contains(t.deadProjects, containerName) {
			continue
		}
		t.logger.InfoF("<%s> 从 deadProjects 中移除", containerName)
		t.deadProjects = slices.DeleteFunc(t.deadProjects, func(containerName_ string) bool {
			return containerName_ == containerName
		})
	}

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
			continue
		}

		if !slices.Contains(t.deadProjects, containerName) {
			t.logger.InfoF("<%s> 意外终止，下次检查如果还处于终止状态，则会报警", containerName)
			t.deadProjects = append(t.deadProjects, containerName)
			continue
		}

		// 警报
		util.Alert(t.logger, fmt.Sprintf(`
项目 <%s> 意外终止，请检查
`, containerName))
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
	return 5 * time.Minute
}

func (t *WatchContainer) Logger() i_logger.ILogger {
	return t.logger
}
