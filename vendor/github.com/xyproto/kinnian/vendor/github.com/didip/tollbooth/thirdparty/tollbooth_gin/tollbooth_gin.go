package tollbooth_gin

import (
	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth/config"
	"github.com/gin-gonic/gin"
)

func LimitHandler(limiter *config.Limiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		httpError := tollbooth.LimitByRequest(limiter, c.Request)
		if httpError != nil {
			c.String(httpError.StatusCode, httpError.Message)
			c.Abort()
		} else {
			c.Next()
		}
	}
}
