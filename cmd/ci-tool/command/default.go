package command

import (
	"flag"
	"fmt"
	ci_manager "github.com/pefish/ci-tool/pkg/ci-manager"
	"github.com/pefish/ci-tool/pkg/constant"
	"github.com/pefish/ci-tool/pkg/global"
	"github.com/pefish/ci-tool/pkg/route"
	"github.com/pefish/ci-tool/version"
	"github.com/pefish/go-commander"
	go_config "github.com/pefish/go-config"
	"github.com/pefish/go-core/driver/logger"
	global_api_strategy "github.com/pefish/go-core/global-api-strategy"
	"github.com/pefish/go-core/service"
	go_logger "github.com/pefish/go-logger"
	task_driver "github.com/pefish/go-task-driver"
)

type DefaultCommand struct {
}

func NewDefaultCommand() *DefaultCommand {
	return &DefaultCommand{}
}

func (dc *DefaultCommand) DecorateFlagSet(flagSet *flag.FlagSet) error {
	flagSet.String("server-host", "127.0.0.1", "The host of web server.")
	flagSet.String("token", "", "The token of request.")
	flagSet.Int("server-port", 8000, "The port of web server.")
	return nil
}

func (dc *DefaultCommand) OnExited(data *commander.StartData) error {
	return nil
}

func (dc *DefaultCommand) Init(data *commander.StartData) error {
	service.Service.SetName(version.AppName)
	logger.LoggerDriverInstance.Register(go_logger.Logger)

	err := go_config.ConfigManagerInstance.Unmarshal(&global.GlobalConfig)
	if err != nil {
		return err
	}

	global.CiManager = ci_manager.NewCiManager(data.ExitCancelCtx)
	go func() {
		global.CiManager.Listen(global.CiManager, nil)
	}()
	fmt.Println(global.GlobalConfig.ServerPort)

	service.Service.SetHost(global.GlobalConfig.ServerHost)
	service.Service.SetPort(global.GlobalConfig.ServerPort)
	service.Service.SetPath(`/api`)
	global_api_strategy.ParamValidateStrategyInstance.SetErrorCode(constant.PARAM_ERROR)

	service.Service.SetRoutes(route.CiRoute)

	return nil
}

func (dc *DefaultCommand) Start(data *commander.StartData) error {

	taskDriver := task_driver.NewTaskDriver()
	taskDriver.Register(service.Service)

	taskDriver.RunWait(data.ExitCancelCtx)

	return nil
}
