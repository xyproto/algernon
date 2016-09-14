package main

import (
	"time"

	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth/thirdparty/tollbooth_echo"
	"github.com/webx-top/echo"
	"github.com/webx-top/echo/engine/standard"
)

func main() {
	e := echo.New()

	// Create a limiter struct.
	limiter := tollbooth.NewLimiter(1, time.Second)

	e.Get("/", echo.HandlerFunc(func(c echo.Context) error {
		return c.String(200, "Hello, World!")
	}), tollbooth_echo.LimitHandler(limiter))

	e.Run(standard.New(":4444"))
}
