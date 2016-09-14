## tollbooth_negroni

[Negroni](https://github.com/urfave/negroni) middleware for rate limiting HTTP requests.


## Five Minutes Tutorial

```
package main

import (
    "github.com/urfave/negroni"
    "github.com/didip/tollbooth"
    "github.com/didip/tollbooth/thirdparty/tollbooth_negroni"
    "net/http"
    "time"
)

func HelloHandler() http.Handler {
    handleFunc := func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Hello, world!"))
    }

    return http.HandlerFunc(handleFunc)
}

func main() {
    // Create a limiter struct.
    limiter := tollbooth.NewLimiter(1, time.Second)

    mux := http.NewServeMux()

    mux.Handle("/", negroni.New(
        tollbooth_negroni.LimitHandler(limiter),
        negroni.Wrap(HelloHandler()),
    ))

    n := negroni.Classic()
    n.UseHandler(mux)
    n.Run(":12345")
}
```