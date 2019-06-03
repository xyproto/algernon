# go-httpclient 

[![Travis](https://img.shields.io/travis/ddliu/go-httpclient.svg?style=flat-square)](https://travis-ci.org/ddliu/go-httpclient)
[![godoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](https://godoc.org/github.com/ddliu/go-httpclient)
[![License](https://img.shields.io/badge/license-MIT-blue.svg?style=flat-square)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/ddliu/go-httpclient)](https://goreportcard.com/report/github.com/ddliu/go-httpclient)
[![cover.run](https://cover.run/go/github.com/ddliu/go-httpclient.svg?style=flat&tag=golang-1.10)](https://cover.run/go?tag=golang-1.10&repo=github.com%2Fddliu%2Fgo-httpclient)

Advanced HTTP client for golang.

## Features

- Chainable API
- Direct file upload
- Timeout
- HTTP Proxy
- Cookie
- GZIP
- Redirect Policy

## Installation

```bash
go get github.com/ddliu/go-httpclient
```

## Quick Start

```go
package main

import (
    "github.com/ddliu/go-httpclient"
)

func main() {
    httpclient.Defaults(httpclient.Map {
        httpclient.OPT_USERAGENT: "my awsome httpclient",
        "Accept-Language": "en-us",
    })

    res, err := httpclient.Get("http://google.com/search", map[string]string{
        "q": "news",
    })

    println(res.StatusCode, err)
}
```

## Usage

### Setup

Use `httpclient.Defaults` to setup default behaviors of the HTTP client.

```go
httpclient.Defaults(httpclient.Map {
    httpclient.OPT_USERAGENT: "my awsome httpclient",
    "Accept-Language": "en-us",
})
```

The `OPT_XXX` options define basic behaviours of this client, other values are 
default request headers of this request. They are shared between different HTTP 
requests.


### Sending Request

```go
// get
httpclient.Get("http://httpbin.org/get", map[string]string{
    "q": "news",
})

// get with url.Values
httpclient.Get("http://httpbin.org/get", url.Values{
    "q": []string{"news", "today"}
})

// post
httpclient.Post("http://httpbin.org/post", map[string]string {
    "name": "value"
})

// post file(multipart)
httpclient.Post("http://httpbin.org/multipart", map[string]string {
    "@file": "/tmp/hello.pdf",
})

// put json
httpclient.PutJson("http://httpbin.org/put", 
`{
    "name": "hello",
}`)

// delete
httpclient.Delete("http://httpbin.org/delete")

// options
httpclient.Options("http://httpbin.org")

// head
httpclient.Head("http://httpbin.org/get")
```

### Customize Request

Before you start a new HTTP request with `Get` or `Post` method, you can specify
temporary options, headers or cookies for current request.

```go
httpclient.
    WithHeader("User-Agent", "Super Robot").
    WithHeader("custom-header", "value").
    WithHeaders(map[string]string {
        "another-header": "another-value",
        "and-another-header": "another-value",
    }).
    WithOption(httpclient.OPT_TIMEOUT, 60).
    WithCookie(&http.Cookie{
        Name: "uid",
        Value: "123",
    }).
    Get("http://github.com")
```

### Response

The `httpclient.Response` is a thin wrap of `http.Response`.

```go
// traditional
res, err := httpclient.Get("http://google.com")
bodyBytes, err := ioutil.ReadAll(res.Body)
res.Body.Close()

// ToString
res, err = httpclient.Get("http://google.com")
bodyString, err := res.ToString()

// ReadAll
res, err = httpclient.Get("http://google.com")
bodyBytes, err := res.ReadAll()
```

### Handle Cookies

```go
url := "http://github.com"
httpclient.
    WithCookie(&http.Cookie{
        Name: "uid",
        Value: "123",
    }).
    Get(url)

for _, cookie := range httpclient.Cookies() {
    fmt.Println(cookie.Name, cookie.Value)
}

for k, v := range httpclient.CookieValues() {
    fmt.Println(k, v)
}

fmt.Println(httpclient.CookieValue("uid"))
```

### Concurrent Safe

If you want to start many requests concurrently, remember to call the `Begin` 
method when you begin:

```go
go func() {
    httpclient.
        Begin().
        WithHeader("Req-A", "a").
        Get("http://google.com")
}()
go func() {
    httpclient.
        Begin().
        WithHeader("Req-B", "b").
        Get("http://google.com")
}()

```

### Error Checking

You can use `httpclient.IsTimeoutError` to check for timeout error:

```go
res, err := httpclient.Get("http://google.com")
if httpclient.IsTimeoutError(err) {
    // do something
}
```

### Full Example

See `examples/main.go`

## Options

Available options as below:

- `OPT_FOLLOWLOCATION`: TRUE to follow any "Location: " header that the server sends as part of the HTTP header. Default to `true`.
- `OPT_CONNECTTIMEOUT`: The number of seconds to wait while trying to connect. Use 0 to wait indefinitely.
- `OPT_CONNECTTIMEOUT_MS`: The number of milliseconds to wait while trying to connect. Use 0 to wait indefinitely.
- `OPT_MAXREDIRS`: The maximum amount of HTTP redirections to follow. Use this option alongside `OPT_FOLLOWLOCATION`.
- `OPT_PROXYTYPE`: Specify the proxy type. Valid options are `PROXY_HTTP`, `PROXY_SOCKS4`, `PROXY_SOCKS5`, `PROXY_SOCKS4A`. Only `PROXY_HTTP` is supported currently. 
- `OPT_TIMEOUT`: The maximum number of seconds to allow httpclient functions to execute.
- `OPT_TIMEOUT_MS`: The maximum number of milliseconds to allow httpclient functions to execute.
- `OPT_COOKIEJAR`: Set to `true` to enable the default cookiejar, or you can set to a `http.CookieJar` instance to use a customized jar. Default to `true`.
- `OPT_INTERFACE`: TODO
- `OPT_PROXY`: Proxy host and port(127.0.0.1:1080).
- `OPT_REFERER`: The `Referer` header of the request.
- `OPT_USERAGENT`: The `User-Agent` header of the request. Default to "go-httpclient v{{VERSION}}".
- `OPT_REDIRECT_POLICY`: Function to check redirect.
- `OPT_PROXY_FUNC`: Function to specify proxy.
- `OPT_UNSAFE_TLS`: Set to `true` to disable TLS certificate checking.
- `OPT_DEBUG`: Print request info.

## Seperate Clients

By using the `httpclient.Get`, `httpclient.Post` methods etc, you are using a 
default shared HTTP client.

If you need more than one client in a single programme. Just create and use them
seperately.

```go
c1 := httpclient.NewHttpClient().Defaults(httpclient.Map {
    httpclient.OPT_USERAGENT: "browser1",
})

c1.Get("http://google.com/")

c2 := httpclient.NewHttpClient().Defaults(httpclient.Map {
    httpclient.OPT_USERAGENT: "browser2",
})

c2.Get("http://google.com/")

```
