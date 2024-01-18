package global

import ci_manager "github.com/pefish/ci-tool/pkg/ci-manager"

type Config struct {
	ServerHost string `json:"server-host"`
	Token      string `json:"token"`
	ServerPort uint64 `json:"server-port"`
	//Db   struct {
	//	Db       string `json:"db"`
	//	Host     string `json:"host"`
	//	ReadHost string `json:"readHost"`
	//	User     string `json:"user"`
	//	Pass     string `json:"pass"`
	//} `json:"db"`
}

var GlobalConfig Config

var CiManager *ci_manager.CiManagerType
