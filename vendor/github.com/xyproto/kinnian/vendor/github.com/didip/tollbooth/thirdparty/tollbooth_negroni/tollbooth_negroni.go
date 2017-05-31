package tollbooth_negroni

import (
	"github.com/urfave/negroni"
	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth/config"
	"net/http"
)

func LimitHandler(limiter *config.Limiter) negroni.HandlerFunc {
	return negroni.HandlerFunc(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		httpError := tollbooth.LimitByRequest(limiter, r)
		if httpError != nil {
			w.Header().Add("Content-Type", limiter.MessageContentType)
			/* RHMOD Fix for error "http: multiple response.WriteHeader calls"
			     Reverse the sequence of the functions calls w.WriteHeader() and w.Write()
			*/
			w.WriteHeader(httpError.StatusCode)
			w.Write([]byte(httpError.Message))
			return

		} else {
			next(w, r)
		}

	})
}
