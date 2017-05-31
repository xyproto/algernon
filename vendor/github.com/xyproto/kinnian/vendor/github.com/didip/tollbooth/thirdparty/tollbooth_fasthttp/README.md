## tollbooth_fasthttp

[Fasthttp](https://github.com/valyala/fasthttp) middleware for rate limiting HTTP requests.


## Five Minutes Tutorial

```
package main

import (
	"time"

	"github.com/didip/tollbooth"
	"github.com/magicwrighter/tollbooth/thirdparty/tollbooth_fasthttp"
	"github.com/valyala/fasthttp"
)

func main() {
	requestHandler := func(ctx *fasthttp.RequestCtx) {
		switch string(ctx.Path()) {
		case "/hello":
			helloHandler(ctx)
		default:
			ctx.Error("Unsupporterd path", fasthttp.StatusNotFound)
		}
	}

	// Create a limiter struct.
	limiter := tollbooth.NewLimiter(1, time.Second)

	fasthttp.ListenAndServe(":4444", tollbooth_fasthttp.LimitHandler(requestHandler, limiter))
}

func helloHandler(ctx *fasthttp.RequestCtx) {
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBody([]byte("Hello, World!"))
}
```