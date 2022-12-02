[![ci test status](https://img.shields.io/github/workflow/status/DataDog/gostackparse/Go?label=tests)](https://github.com/DataDog/gostackparse/actions/workflows/go.yml?query=branch%3Amain)
[![documentation](http://img.shields.io/badge/godoc-reference-blue.svg)](https://pkg.go.dev/github.com/DataDog/gostackparse)

# gostackparse

Package gostackparse parses goroutines stack traces as produced by [`panic()`](https://golang.org/pkg/builtin/#panic) or [`debug.Stack()`](https://golang.org/pkg/runtime/debug/#Stack) at ~300 MiB/s.

Parsing this data can be useful for [Goroutine Profiling](https://github.com/DataDog/go-profiler-notes/blob/main/goroutine.md#feature-matrix) or analyzing crashes from log files.

## Usage

The package provides a simple [Parse()](https://pkg.go.dev/github.com/DataDog/gostackparse#Parse) API. You can use it like [this](./example):

```go
import "github.com/DataDog/gostackparse"

func main() {
	// Get a text-based stack trace
	stack := debug.Stack()
	// Parse it
	goroutines, _ := gostackparse.Parse(bytes.NewReader(stack))
	// Ouptut the results
	json.NewEncoder(os.Stdout).Encode(goroutines)
}
```

The result is a simple list of [Goroutine](https://pkg.go.dev/github.com/DataDog/gostackparse#Goroutine) structs:

```
[
  {
    "ID": 1,
    "State": "running",
    "Wait": 0,
    "LockedToThread": false,
    "Stack": [
      {
        "Func": "runtime/debug.Stack",
        "File": "/usr/local/Cellar/go/1.16/libexec/src/runtime/debug/stack.go",
        "Line": 24
      },
      {
        "Func": "main.main",
        "File": "/home/go/src/github.com/DataDog/gostackparse/example/main.go",
        "Line": 18
      }
    ],
    "FramesElided": false,
    "CreatedBy": null
  }
]
```

## Design Goals

1. Safe: No panics should be thrown.
2. Simple: Keep this pkg small and easy to modify.
3. Forgiving: Favor producing partial results over no results, even if the input data is different than expected.
4. Efficient: Parse several hundred MiB/s.

## Testing

gostackparse has been tested using a combination of hand picked [test-fixtures](./test-fixtures), [property based testing](https://github.com/DataDog/gostackparse/search?q=TestParse_PropertyBased), and [fuzzing](https://github.com/DataDog/gostackparse/search?q=Fuzz).

## Comparsion to panicparse

[panicparse](https://github.com/maruel/panicparse) is a popular library implementing similar functionality.

gostackparse was created to provide a subset of the functionality (only the parsing) using ~10x less code while achieving > 100x faster performance. If you like fast minimalistic code, you might prefer it. If you're looking for more features and a larger community, use panicparse.

## Benchmarks

gostackparse includes a small benchmark that shows that it can parse [test-fixtures/waitsince.txt](./test-fixtures/waitsince.txt) at ~300 MiB/s and how that compares to panicparse.

```
$ cp panicparse_test.go.disabled panicparse_test.go
$ go get -t .
$ go test -bench .
goos: darwin
goarch: amd64
pkg: github.com/DataDog/gostackparse
cpu: Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz
BenchmarkGostackparse-12   45456    26275 ns/op 302.34 MiB/s   17243 B/op    306 allocs/op
BenchmarkPanicparse-12     76    15943320 ns/op   0.50 MiB/s 5274247 B/op 116049 allocs/op
PASS
ok  	github.com/DataDog/gostackparse	3.634s
```

## License

This work is dual-licensed under Apache 2.0 or BSD3. See [LICENSE](./LICENSE).
