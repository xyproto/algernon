goja-babel
==========

Uses github.com/dop251/goja to run babel.js within Go.

**WARNING:** This is largely untested and the exposed API may change at any time.

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
			"transform-es2015-block-scoping",
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
var foo = 1;
React.createElement(
	"div",
	null,
	"Hello JSX! The value of foo is ",
	foo,
	"."
);
```

## Benchmarks

```
$ go test -bench Transform -benchmem
BenchmarkTransformString-8                    	     200	   6042202 ns/op	  925350 B/op	   15779 allocs/op
BenchmarkTransformStringWithSingletonPool-8   	     200	   5976874 ns/op	  927350 B/op	   15809 allocs/op
BenchmarkTransformStringWithLargePool-8       	     300	   5892891 ns/op	  926572 B/op	   15799 allocs/op
PASS
ok  	github.com/jvatic/goja-babel	20.346s
```
