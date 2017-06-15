## tollbooth_iris

[Iris](https://github.com/kataras/iris) middleware for rate limiting HTTP requests.


## Five Minutes Tutorial

```
package main

import (
    "time"

    "github.com/didip/tollbooth"
    "github.com/didip/tollbooth/thirdparty/tollbooth_iris"

    "github.com/kataras/iris"
    "github.com/kataras/iris/context"
)

func main() {
    app := iris.New()

    // Create a limiter struct.
    limiter := tollbooth.NewLimiter(1, time.Second)

    app.GET("/", tollbooth_iris.LimitHandler(limiter), func(ctx context.Context) {
        ctx.WriteString("Hello, world!")
    })

    app.Run(iris.Addr(":8080"))
}

```