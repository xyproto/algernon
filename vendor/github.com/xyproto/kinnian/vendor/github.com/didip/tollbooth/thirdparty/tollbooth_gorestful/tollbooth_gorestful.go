package tollbooth_gorestful

import (
	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth/config"
	"github.com/emicklei/go-restful"
)

func LimitHandler(handler restful.RouteFunction, limiter *config.Limiter) restful.RouteFunction {
	return func(request *restful.Request, response *restful.Response) {
		httpError := tollbooth.LimitByRequest(limiter, request.Request)
		if httpError != nil {
			response.WriteErrorString(429, "429: Too Many Requests")
			return
		}

		handler(request, response)
	}
}
