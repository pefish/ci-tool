package global

import go_mysql "github.com/pefish/go-mysql"

type Config struct {
	ServerHost  string `json:"server-host" default:"127.0.0.1" usage:"Web server host."`
	ServerPort  int    `json:"server-port" default:"8000" usage:"Web server port."`
	DbHost      string `json:"db-host" default:"" usage:"Database host."`
	DbPort      int    `json:"db-port" default:"3306" usage:"Database port."`
	DbDatabase  string `json:"db-db" default:"" usage:"Database to connect."`
	DbUser      string `json:"db-user" default:"" usage:"Username to connect database."`
	DbPass      string `json:"db-pass" default:"" usage:"Password to connect database."`
	SrcDir      string `json:"src-dir" default:"~/src" usage:"Source code dir."`
	AlertToken  string `json:"alert-token" default:"" usage:"Default Alert token."`
	AlertChatId string `json:"alert-chat-id" default:"" usage:"Default Alert chat id."`
}

var GlobalConfig Config

var MysqlInstance *go_mysql.MysqlType
