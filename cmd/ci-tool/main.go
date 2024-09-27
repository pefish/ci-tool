package main

import (
	"log"

	"github.com/pefish/ci-tool/cmd/ci-tool/command"
	"github.com/pefish/ci-tool/version"
	"github.com/pefish/go-commander"
)

func main() {
	commanderInstance := commander.NewCommander(version.AppName, version.Version, version.AppName+" 是一个模版，祝你玩得开心。作者：pefish")
	commanderInstance.RegisterDefaultSubcommand(&commander.SubcommandInfo{
		Desc:       "",
		Args:       nil,
		Subcommand: command.NewDefaultCommand(),
	})
	err := commanderInstance.Run()
	if err != nil {
		log.Fatal(err)
	}
}
