# go-term-ansicolor

Go library that colors strings using ANSI escape sequences.

(Go porting from https://github.com/flori/term-ansicolor)

## Installation

### Using *go get*

    $ go get github.com/koyachi/go-term-ansicolor/ansicolor

After this command *go-term-ansicolor* is ready to use. its source will be in:

    $GOROOT/src/pkg/github.com/koyachi/go-term-ansicolor/ansicolor

You can use `go get -u -a` for update all installed packages.

### Using *git clone* command:

    $ git clone git://github.com/koyachi/go-term-ansicolor
    $ cd go-term-ansicolor/ansicolor
    $ go install

## Example

```go
package main

import (
	"fmt"
	"github.com/koyachi/go-term-ansicolor/ansicolor"
)

func main() {
	fmt.Println(ansicolor.Green("Hello, ") + ansicolor.Red("World!"))
}
```

## Sample Application: cdiff

Source:

- cdiff/cdiff.go

Install:

    $ cd go-term-ansicolor/cdiff
    $ go install

Usage:

    $ diff -u ./cdiff.go~ ./cdiff.go | cdiff

## See also:

- https://github.com/flori/term-ansicolor
- https://github.com/hotei/ansiterm

## Contributing

1. Fork it
2. Create your feature branch (`git checkout -b my-new-feature`)
3. Commit your changes (`git commit -am 'Add some feature'`)
4. Push to the branch (`git push origin my-new-feature`)
5. Create new Pull Request
