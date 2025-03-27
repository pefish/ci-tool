package command

import (
	"os"
	"strings"

	ci_manager "github.com/pefish/ci-tool/pkg/ci-manager"
	"github.com/pefish/ci-tool/pkg/constant"
	"github.com/pefish/ci-tool/pkg/global"
	"github.com/pefish/ci-tool/pkg/route"
	"github.com/pefish/ci-tool/pkg/task"
	"github.com/pefish/ci-tool/version"
	"github.com/pefish/go-commander"
	"github.com/pefish/go-core/driver/logger"
	global_api_strategy "github.com/pefish/go-core/global-api-strategy"
	"github.com/pefish/go-core/service"
	t_mysql "github.com/pefish/go-interface/t-mysql"
	go_mysql "github.com/pefish/go-mysql"
	task_driver "github.com/pefish/go-task-driver"
	"github.com/pkg/errors"
)

type DefaultCommand struct {
}

func NewDefaultCommand() *DefaultCommand {
	return &DefaultCommand{}
}

func (dc *DefaultCommand) Config() interface{} {
	return &global.GlobalConfig
}

func (dc *DefaultCommand) Data() interface{} {
	return &global.GlobalData
}

func (dc *DefaultCommand) OnExited(command *commander.Commander) error {
	global.MysqlInstance.Close()
	return nil
}

func (dc *DefaultCommand) Init(command *commander.Commander) error {
	global.MysqlInstance = go_mysql.NewMysqlInstance(command.Logger)
	err := global.MysqlInstance.ConnectWithConfiguration(t_mysql.Configuration{
		Host:     global.GlobalConfig.DbHost,
		Port:     global.GlobalConfig.DbPort,
		Username: global.GlobalConfig.DbUser,
		Password: global.GlobalConfig.DbPass,
		Database: global.GlobalConfig.DbDatabase,
	})
	if err != nil {
		return err
	}

	service.Service.SetName(version.AppName)
	logger.LoggerDriverInstance.Register(command.Logger)

	ci_manager.CiManager = ci_manager.NewCiManager(command.Logger)

	service.Service.SetHost(global.GlobalConfig.ServerHost)
	service.Service.SetPort(uint64(global.GlobalConfig.ServerPort))
	service.Service.SetPath(`/api`)
	global_api_strategy.ParamValidateStrategyInstance.SetErrorCode(constant.PARAM_ERROR)

	service.Service.SetRoutes(route.CiRoute)

	return nil
}

func (dc *DefaultCommand) Start(command *commander.Commander) error {
	taskDriver := task_driver.NewTaskDriver()
	taskDriver.Register(service.Service)
	taskDriver.Register(task.NewWatchContainer(command.Logger))
	taskDriver.Register(task.NewWatchProject(command.Logger))

	data, err := os.ReadFile("/etc/machine-id")
	if err != nil {
		return errors.Errorf("读取 /etc/machine-id 失败: %+v", err)
	}
	global.MachineID = strings.TrimSpace(string(data))

	command.Logger.InfoF("MachineID: %s", global.MachineID)
	if global.MachineID == "" {
		return errors.Errorf("MachineID 没有找到")
	}

	taskDriver.RunWait(command.Ctx)

	return nil
}
