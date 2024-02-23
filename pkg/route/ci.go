package route

import (
	"github.com/pefish/ci-tool/pkg/controller"
	"github.com/pefish/go-core/api"
	"github.com/pefish/go-http/gorequest"
)

var CiRoute = []*api.Api{
	api.NewApi(&api.NewApiParamsType{
		Path:           "/v1/ci-start",
		Method:         gorequest.POST,
		Params:         controller.CiStartParams{},
		ControllerFunc: controller.CiController.CiStart,
	}),
	api.NewApi(&api.NewApiParamsType{
		Path:           "/v1/ci-log",
		Method:         gorequest.GET,
		Params:         controller.CiLogParams{},
		ControllerFunc: controller.CiController.CiLog,
	}),
}
