package route

import (
	"github.com/pefish/ci-tool/pkg/controller"
	"github.com/pefish/go-core/api"
	"github.com/pefish/go-http/gorequest"
)

var CiRoute = []*api.Api{
	{
		Path:       "/v1/ci-start",
		Method:     gorequest.POST,
		Params:     controller.CiStartParams{},
		Controller: controller.CiController.CiStart,
	},
}
