## tollbooth_httprouter

[httprouter](https://github.com/julienschmidt/httprouter) middleware for rate limiting HTTP requests.


## Five Minutes Tutorial

```
package main

import (
    "time"
    "log"

    "github.com/didip/tollbooth"
    "github.com/didip/tollbooth/thirdparty/tollbooth_httprouter"
    "github.com/julienschmidt/httprouter"
)

Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
    fmt.Fprint(w, "Welcome!\n")
}

func Hello(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
    fmt.Fprintf(w, "hello, %s!\n", ps.ByName("name"))
}

func main() {
    router := httprouter.New()

    // Create a limiter struct.
    limiter := tollbooth.NewLimiter(1, time.Second)

    // Index route without limiting.
    router.GET("/", Index)

    // Hello route with limiting.
    router.GET("/hello/:name",
        tollbooth_httprouter.LimitHandler(Hello, limiter),
    )

    log.Fatal(http.ListenAndServe(":8080", router))
}
```
