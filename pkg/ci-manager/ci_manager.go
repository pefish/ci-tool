package ci_manager

import (
	"context"
	"fmt"
	"github.com/pefish/ci-tool/pkg/constant"
	go_best_type "github.com/pefish/go-best-type"
)

type CiManagerType struct {
	go_best_type.BaseBestType
	logs map[string]string
}

func NewCiManager(ctx context.Context) *CiManagerType {
	return &CiManagerType{
		BaseBestType: *go_best_type.NewBaseBestType(ctx, 0),
		logs:         make(map[string]string),
	}
}

func (c CiManagerType) ProcessAsk(ask *go_best_type.AskType, bts map[string]go_best_type.IBestType) {
	data := ask.Data.(map[string]interface{})
	switch ask.Action {
	case constant.ActionType_CI:
		srcPath := data["src_path"].(string)
		scriptPath := data["script_path"].(string)
		projectName := data["project_name"].(string)
		c.logs[projectName] = fmt.Sprintf("CI 请求收到。srcPath: %s, scriptPath: %s, projectName: %s", srcPath, scriptPath, projectName)
	case constant.ActionType_LOG:
		msg := data["msg"].(string)
		projectName := data["project_name"].(string)
		c.logs[projectName] = msg
	case constant.ActionType_ReadLog:
		projectName := data["project_name"].(string)
		ask.AnswerChan <- c.logs[projectName]
	}
}

func (c CiManagerType) OnExited() {

}
