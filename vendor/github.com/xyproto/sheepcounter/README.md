# SheepCounter [![Build Status](https://travis-ci.org/xyproto/sheepcounter.svg?branch=master)](https://travis-ci.org/xyproto/sheepcounter) [![GoDoc](https://godoc.org/github.com/xyproto/sheepcounter?status.svg)](http://godoc.org/github.com/xyproto/sheepcounter) [![Report Card](https://img.shields.io/badge/go_report-A+-brightgreen.svg?style=flat)](http://goreportcard.com/report/xyproto/sheepcounter)

A `http.ResponseWriter` that can count the bytes written to the client so far.

# Why?

If you want to create an access log of how many bytes are sent to which clients, one method would be to write data to a buffer, count the bytes and then send the data to the client. This may be problematic for large files, since it eats up a lot of memory. It is also costly performance wise, since the data would then have to be counted while or after the data is sent to the client.

A better way is to store use the number returned by the `Write` function directly. This is not straightforward with `http.ResponseWriter` without wrapping it somehow, which is what this module does. A lightweight struct wraps both a `http.ResponseWriter` and an `uint64`, for keeping track of the written bytes.

# Examples

## Count the bytes sent, per response

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
        fmt.Println("COUNTED:", sc.Counter()) // Counts the bytes sent, for this response only
}

func main() {
        http.HandleFunc("/", handler)
        fmt.Println("Serving on port 8080")
        log.Fatal(http.ListenAndServe(":8080", nil))
}
~~~

## Count the total amount of bytes sent

~~~go
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/xyproto/sheepcounter"
)

const (
	title = "SheepCounter"
	style = `body { margin: 4em; background: wheat; color: black; font-family: terminus, "courier new", courier; font-size: 1.1em; } a:link { color: #403020; } a:visited { color: #403020; } a:hover { color: #605040; } a:active { color: #605040; } #counter { color: red; }`
	page  = "<!doctype html><html><head><style>%s</style><title>%s</title><body>%s</body></html>"
)

var totalBytesWritten uint64

func helloHandler(w http.ResponseWriter, r *http.Request) {
	sc := sheepcounter.New(w)
	body := `<p>Here are the <a href="/counter">counted bytes</a>.</p>`
	fmt.Fprintf(sc, page, style, title, body)
	written, err := sc.UCounter2()
	if err != nil {
		// Log an error and return
		log.Printf("error: %s\n", err)
		return
	}
	atomic.AddUint64(&totalBytesWritten, written)
	log.Printf("counted %d bytes\n", written)
}

func counterHandler(w http.ResponseWriter, r *http.Request) {
	sc := sheepcounter.New(w)
	body := fmt.Sprintf(`<p>Total bytes sent from the server (without counting this response): <span id="counter">%d</span></p><p><a href="/">Back</a></p>`, atomic.LoadUint64(&totalBytesWritten))
	fmt.Fprintf(sc, page, style, title, body)
	written, err := sc.UCounter2()
	if err != nil {
		// Log an error and return
		log.Printf("error: %s\n", err)
		return
	}
	atomic.AddUint64(&totalBytesWritten, written)
	log.Printf("counted %d bytes\n", written)
}

func main() {
	http.HandleFunc("/", helloHandler)
	http.HandleFunc("/counter", counterHandler)

	httpAddr := os.Getenv("HTTP_ADDR")
	if httpAddr == "" {
		httpAddr = ":4000"
	}

	log.Println("Serving on " + httpAddr)
	log.Fatal(http.ListenAndServe(httpAddr, nil))
}
~~~

# Requirements

* Go 1.8 or later

# General information

* Version: 1.6.0
* Alexander F. Rødseth &lt;xyproto@archlinux.org&gt;
* License: MIT
