## tollbooth_iris

[Iris](https://github.com/kataras/iris) middleware for rate limiting HTTP requests.


## Five Minutes Tutorial

```
package main

import (
    "time"

    "github.com/didip/tollbooth"
    "github.com/didip/tollbooth/thirdparty/tollbooth_iris"

    "gopkg.in/kataras/iris.v6"
    "gopkg.in/kataras/iris.v6/adaptors/httprouter"
)

func main() {
    app := iris.New()
    app.Adapt( httprouter.New() )
    
    // Create a limiter struct.
    limiter := tollbooth.NewLimiter(1, time.Second)

    app.Get("/", tollbooth_iris.LimitHandler(limiter), func(ctx *iris.Context) {
        ctx.WriteString("Hello, world!")
    })

    app.Listen(":8080")
}

```
