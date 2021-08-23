goja-babel
==========

[![CI](https://github.com/jvatic/goja-babel/actions/workflows/ci.yml/badge.svg)](https://github.com/jvatic/goja-babel/actions/workflows/ci.yml)

Uses github.com/dop251/goja to run babel.js within Go.

## Usage

```go
package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jvatic/goja-babel"
)

func main() {
	babel.Init(4) // Setup 4 transformers (can be any number > 0)
	res, err := babel.Transform(strings.NewReader(`let foo = 1;
	<div>
		Hello JSX!
		The value of foo is {foo}.
	</div>`), map[string]interface{}{
		"plugins": []string{
			"transform-react-jsx",
			"transform-block-scoping",
		},
	})
	if err != nil {
		panic(err)
	}
	io.Copy(os.Stdout, res)
	fmt.Println("")
}
```

```js
$ go run main.go
var foo = 1;

/*#__PURE__*/
React.createElement("div", null, "Hello JSX! The value of foo is ", foo, ".");
```

## Benchmarks

```
go test -bench Transform -benchmem
goos: darwin
goarch: amd64
pkg: github.com/jvatic/goja-babel
cpu: Intel(R) Core(TM) i7-3615QM CPU @ 2.30GHz
BenchmarkTransformString-8                    	      81	  15642708 ns/op	 3069085 B/op	   37243 allocs/op
BenchmarkTransformStringWithSingletonPool-8   	      67	  15820676 ns/op	 3070920 B/op	   37244 allocs/op
BenchmarkTransformStringWithLargePool-8       	      78	  15497562 ns/op	 3070015 B/op	   37243 allocs/op
PASS
ok  	github.com/jvatic/goja-babel	4.993s
```
