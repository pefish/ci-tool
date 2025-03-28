package main

import (
	"github.com/pefish/ci-tool/cmd/ci-tool/command"
	"github.com/pefish/ci-tool/pkg/global"
	"github.com/pefish/ci-tool/version"
	"github.com/pefish/go-commander"
)

func main() {
	global.Command = commander.NewCommander(version.AppName, version.Version, version.AppName+" 是一个模版，祝你玩得开心。作者：pefish")
	global.Command.RegisterDefaultSubcommand(&commander.SubcommandInfo{
		Desc:       "",
		Args:       nil,
		Subcommand: command.NewDefaultCommand(),
	})
	err := global.Command.Run()
	if err != nil {
		global.Command.Logger.Error(err)
	}
}
