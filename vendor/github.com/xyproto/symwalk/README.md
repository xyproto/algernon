# symwalk [![Build Status](https://travis-ci.com/xyproto/symwalk.svg?branch=master)](https://travis-ci.com/xyproto/symwalk) [![GoDoc](https://godoc.org/github.com/xyproto/symwalk?status.svg)](http://godoc.org/github.com/xyproto/symwalk)

Concurrently search directories while also following symlinks.

## Fork and license info

* `walker.go` and the tests are based on [powerwalk](https://github.com/stretchr/powerwalk) (MIT license), but with added support for traversing symlinks that points to directories too.
* `modwalk.go` is based on `path/filepath` from the Go standard library (BSD license).
* The modifications to these files and the rest of this project are licensed under the MIT license.

## Requirements

* Go 1.10 or later.

## Example use

This passes in a function to `symwalk.Walk`, which is called for every encountered file or directory:

```go
import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/xyproto/symwalk"
)

func main() {
	var mut sync.Mutex
	symwalk.Walk(".", func(p string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return nil
		}
		filename := filepath.Base(p)
		mut.Lock()
		fmt.Printf("%s\n", filename)
		mut.Unlock()
		return nil
	})
}
```

## General info

* Version: 1.1.0
