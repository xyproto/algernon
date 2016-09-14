package tollbooth_httprouter

import (
	"net/http"

	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth/config"
	"github.com/julienschmidt/httprouter"
)

// RateLimit is a rate limiting middleware
func LimitHandler(handler httprouter.Handle, limiter *config.Limiter) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		httpError := tollbooth.LimitByRequest(limiter, r)
		if httpError != nil {
			http.Error(w, httpError.Message, httpError.StatusCode)
			return
		}

		handler(w, r, ps)
	}
}
