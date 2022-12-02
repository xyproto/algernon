# fgtrace - The Full Go Tracer

[![ci test status](https://img.shields.io/github/workflow/status/felixge/fgtrace/Go?label=tests)](https://github.com/felixge/fgtrace/actions/workflows/go.yml?query=branch%3Amain)
[![documentation](http://img.shields.io/badge/godoc-reference-blue.svg)](https://pkg.go.dev/github.com/felixge/fgtrace)

fgtrace is an experimental profiler/tracer that is capturing wallclock timelines for each goroutine. It's very similar to the Chrome profiler.

⚠️ fgtrace may cause noticeable stop-the-world pauses in your applications. It is intended for dev and testing environments for now.

<img src="./assets/fgtrace-example.png"/>

## Quick Start

To capture an fgtrace of your program, simply add the one-liner shown below. This will cause the creation of a `fgtrace.json` file in the current working directory that you can view by opening it in the [Perfetto UI](https://ui.perfetto.dev/).

```go
package main

import "github.com/felixge/fgtrace"

func main() {
	defer fgtrace.Config{Dst: fgtrace.File("fgtrace.json")}.Trace().Stop()

	// <code to trace>
}
```

Alternatively you can configure fgtrace as a `http.Handler` and request traces on-demand by hitting `http://localhost:1234/debug/fgtrace?seconds=30&hz=100`.

```go
package main

import (
	"net/http"
	"github.com/felixge/fgtrace"
)

func main() {
	http.DefaultServeMux.Handle("/debug/fgtrace", fgtrace.Config{})
	http.ListenAndServe(":1234", nil)
}
```

For more advanced use cases, have a look at the [API Documentation](https://pkg.go.dev/github.com/felixge/fgtrace#Config).

## Comparison with Similar Tools

Below is a [simple program](./testdata/readme/) that spends its time sleeping, requesting a website, capturing the response body and then hashing it a few times.

```go
for i := 0; i < 10; i++ {
	time.Sleep(10 * time.Millisecond)
}

res, err := http.Get("https://github.com/")
if err != nil {
	panic(err)
}
defer res.Body.Close()

var buf bytes.Buffer
if _, err := io.Copy(&buf, res.Body); err != nil {
	panic(err)
}

for i := 0; i < 1000; i++ {
	sha1.Sum(buf.Bytes())
}
```

Now let's have a look at how fgtrace and other tools allow you to understand the performance of such a program.

### fgtrace

Looking at our main goroutine (G1), we can easily recognize the operations of the program, their order, and how long they are taking (~100ms `time.Sleep`, ~65ms `http.Get`, ~30ms `io.Copy`ing the response and ~300ms calling `sha1.Sum` to hash it).

However, it's important to note that this data is captured by sampling goroutine stack traces rather than actual tracing. Therefore fgtrace does not know that there were ten `time.Sleep()` function calls lasting `10ms` each. Instead it just merges its samples into one big `time.Sleep()` call that appears to take `100ms`.

Another detail are the virtual goroutine state indicators on top, e.g. `sleep`, `select`, `sync.Cond.Wait` and `running/runnable`. These are not part of the real stack traces and meant to help understanding On-CPU activity (`running/runnable`) vs Off-CPU states. You can disable them via configuration.

<img src="./assets/fgtrace-example.png"/>

To break down the latency of our main goroutine, we can also look at other goroutines used by the program. E.g. below is a closer look on how the `http.Get` operation is broken down into resolving the IP address, connecting to it, and performing a TLS handshake.

<img src="./assets/fgtrace-example2.png"/>

So as you can see, fgtrace offers an intuitive, yet powerful way to understand the operation of Go programs. However, since it always captures the activity of all goroutines and has no information about how they communicate with each other, it may create overwhelming amounts of data in some cases.

### fgprof

You can think of [fgprof](https://github.com/felixge/fgprof) as a more simplified version of fgtrace. Instead of capturing a timeline for each goroutine, it aggregates the same data into a single profile as shown in the flame graph below.

<img src="./assets/fgprof-example.png"/>

This means that the x-axis represents duration rather than time, so function calls are ordered alphabetically rather than chronologically. E.g. notice how `time.Sleep` is shown after `sha1.Sum` in the graph above even so it's the first operation completed by our program.

Additionally the data of all goroutines ends up in the same graph which can be difficult to read without having a good understanding of the underlaying code and number of goroutines that are involved.

Despite these disadvantages, fgprof may still be useful in certain situations where the detail provided by the timeline may be overwhelming and a simpler view of the average program behavior is desirable. Additionally fgprof under Go 1.19 has less [negative impact](https://go-review.googlesource.com/c/go/+/387415) on the performance of the profiled program than fgtrace.

### runtime/trace

The `runtime/trace` package is a true execution tracer that is capable of capturing even more detailed information than fgtrace. However, it's mostly designed to understand the decisions made by the Go scheduler. So the default timeline is focused on how goroutines are scheduled onto the CPU (processors). This means only the `sha1.Sum` operation stands out in green, and full stack traces can only be seen by clicking on the individual scheduler activities.

<img src="./assets/runtime-example.png"/>

The goroutine analysis view offers a more useful breakdown. Here we can see that our goroutine is spending `271ms` in `Execution` on CPU, but it's not clear from this view alone that this is the `sha1.Sum` operation. Our networking activity (`http.Get` and `io.Copy`) gets grouped into `Sync block` rather than `Network wait` because the networking is done through channels via other goroutines. And our `time.Sleep` activity is shown as a grey component of the bar diagram, but not explicitly listed in the table. So while a lot of information is available here, it's difficult to interpret for casual users.

<img src="./assets/runtime-example2.png"/>

Last but not least it's possible to click on the goroutine id in the view above in order to see a timeline for the individual goroutine, as well as the other goroutines it is communicating with. However, the view is also CPU-centric, so remains difficult to understand the sleep and networking operations of our program.

<img src="./assets/runtime-example3.png"/>

That being said, some of the limitations of `runtime/trace` could probably be resolved with changes to the UI or converting the traces into a format that [Perfetto UI](https://ui.perfetto.dev/) can understand which might be a fun project for another time.

## How it Works

The current implementation of fgtrace is incredibly hacky. It calls [`runtime.Stack()`](https://pkg.go.dev/runtime#Stack) on a regular frequency (default 100 Hz) to capture textual stack traces of all goroutines and parses them using the [gostackparse](https://github.com/DataDog/gostackparse) package. Each call to `runtime.Stack()` is a blocking stop-the-world operation, so it scales very poorly to programs using ten thousand or more goroutines.

After the data is captured, it is converted into the [Trace Event Format](https://docs.google.com/document/d/1CvAClvFfyA5R-PhYUmn5OOQtYMH4h6I0nSsKchNAySU/preview) which is one of the data formats understood by [Perfetto UI](https://ui.perfetto.dev/).

## The Future

fgtrace is mostly a ["Do Things that Don't Scale"](http://paulgraham.com/ds.html) kind of project. If enough people like it, it will motivate me and perhaps others to invest into putting it on a solid technical foundation.

The Go team has previously [declined](https://github.com/golang/go/issues/41324#issuecomment-703796820) the idea of adding wallclock profiling capabilities similar to fgprof (which is similar to fgtrace) to the Go project and is more likely to invest in `runtime/trace` going forward.

That being said, I still think fgtrace can help by:

1. Showing the usefulness of stack-trace/wallclock focused timeline views in addition to the CPU-centric views used by `runtime/trace` to perhaps implement the future developement of the runtime tracer.
2. Starting a conversation (link to GH issue will follow ...) to offer more powerful goroutine profiling APIs to allow user-space tooling like this to thrive without having to hack around the [existing APIs](https://github.com/DataDog/go-profiler-notes/blob/main/goroutine.md#feature-matrix) while reducing their overhead.



## License

fgtrace is licensed under the MIT License.
