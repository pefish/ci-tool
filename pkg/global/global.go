package global

import (
	"time"

	go_mysql "github.com/pefish/go-mysql"
)

type Config struct {
	ServerHost  string `json:"server-host" default:"127.0.0.1" usage:"Web server host."`
	ServerPort  int    `json:"server-port" default:"8000" usage:"Web server port."`
	DbHost      string `json:"db-host" default:"" usage:"Database host."`
	DbPort      int    `json:"db-port" default:"3306" usage:"Database port."`
	DbDatabase  string `json:"db-db" default:"" usage:"Database to connect."`
	DbUser      string `json:"db-user" default:"" usage:"Username to connect database."`
	DbPass      string `json:"db-pass" default:"" usage:"Password to connect database."`
	SrcDir      string `json:"src-dir" default:"~/src" usage:"Source code dir."`
	AlertType   string `json:"alert-type" default:"weixin" usage:"Alert type. weixin/tg"`
	AlertToken  string `json:"alert-token" default:"" usage:"Alert token."`
	AlertChatId string `json:"alert-chat-id" default:"" usage:"Alert chat id."`
}

type Data struct {
	DeadProjects   []string             `json:"dead_projects"`
	LastNotifyTime map[string]time.Time `json:"last_notify_time"`
}

var GlobalConfig Config

var GlobalData Data = Data{
	DeadProjects:   make([]string, 0),
	LastNotifyTime: make(map[string]time.Time, 0),
}

var MysqlInstance *go_mysql.MysqlType
