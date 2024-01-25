package global

import ci_manager "github.com/pefish/ci-tool/pkg/ci-manager"

type Config struct {
	ServerHost string `json:"server-host"`
	Token      string `json:"token"`
	ServerPort uint64 `json:"server-port"`
}

var GlobalConfig Config

var CiManager *ci_manager.CiManagerType
