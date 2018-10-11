# SheepCounter [![Build Status](https://travis-ci.org/xyproto/sheepcounter.svg?branch=master)](https://travis-ci.org/xyproto/sheepcounter) [![GoDoc](https://godoc.org/github.com/xyproto/sheepcounter?status.svg)](http://godoc.org/github.com/xyproto/sheepcounter) [![Report Card](https://img.shields.io/badge/go_report-A+-brightgreen.svg?style=flat)](http://goreportcard.com/report/xyproto/sheepcounter)

A `http.ResponseWriter` that can count the bytes written to the client so far.

# Why?

If you want to create an access log of how many bytes are sent to which clients, one method would be to write data to a buffer, count the bytes and then send the data to the client. This may be problematic for large files, since it eats up a lot of memory. It is also costly performance wise, since the data would then have to be counted while or after the data is sent to the client.

A better way is to store use the number returned by the `Write` function directly. This is not straightforward with `http.ResponseWriter` without wrapping it somehow, which is what this module does. A lightweight struct wraps both a `http.ResponseWriter` and an `int64`, for keeping track of the written bytes.

# Example use

~~~go
package main

import (
        "fmt"
        "github.com/xyproto/sheepcounter"
        "log"
        "net/http"
)

func handler(w http.ResponseWriter, r *http.Request) {
        sc := sheepcounter.New(w)
        fmt.Fprintf(sc, "Hi %s!", r.URL.Path[1:])
        fmt.Println("COUNTED:", sc.Counter())
}

func main() {
        http.HandleFunc("/", handler)
        fmt.Println("Serving on port 8080")
        log.Fatal(http.ListenAndServe(":8080", nil))
}
~~~

# Requirements

* Go 1.8 or later

# General information

* Version: 1.2.0
* Alexander F. Rødseth &lt;xyproto@archlinux.org&gt;
* License: MIT

