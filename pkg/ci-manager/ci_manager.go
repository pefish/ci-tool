package ci_manager

import (
	"context"
	"github.com/pefish/ci-tool/pkg/constant"
	go_best_type "github.com/pefish/go-best-type"
	go_logger "github.com/pefish/go-logger"
)

type CiManagerType struct {
	go_best_type.BaseBestType
}

func NewCiManager(ctx context.Context) *CiManagerType {
	return &CiManagerType{
		BaseBestType: *go_best_type.NewBaseBestType(ctx, 10),
	}
}

func (c CiManagerType) ProcessAsk(ask *go_best_type.AskType, bts map[string]go_best_type.IBestType) {
	switch ask.Action {
	case constant.ActionType_CI:
		go_logger.Logger.InfoF("CI 请求收到, params: %#v", ask.Data.(map[string]interface{}))
	}
}

func (c CiManagerType) Exited() {

}
