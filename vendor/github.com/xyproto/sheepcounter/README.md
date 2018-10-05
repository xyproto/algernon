# SheepCounter [![Build Status](https://travis-ci.org/xyproto/sheepcounter.svg?branch=master)](https://travis-ci.org/xyproto/sheepcounter) [![GoDoc](https://godoc.org/github.com/xyproto/sheepcounter?status.svg)](http://godoc.org/github.com/xyproto/sheepcounter) [![Report Card](https://img.shields.io/badge/go_report-A+-brightgreen.svg?style=flat)](http://goreportcard.com/report/xyproto/sheepcounter)

A http.ResponseWriter that can count the number of bytes written so far.

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
        sc := sheepcounter.NewSheepCounter(w)
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

* Version: 1.0.0
* Alexander F. Rødseth &lt;xyproto@archlinux.org&gt;
* License: MIT

