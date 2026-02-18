# Binary

Go module and command line utility for checking if the given file or data is likely to be **binary** or **text**.

* It does so by reading the first, middle and last 24 bytes of the file and trying to convert the data to utf8.
* If one of the 24 byte blocks can not be converted to utf8, it's considered to be a binary file.
* Also, if one of the blocks have more than 33% zero bytes, it's considered to be a binary file.
* If the first 24 bytes indicates that it's a binary file, the deduction is done, and no more seeking or reading will happen.
* If the file is empty, it's considered to be a text file.
* The `binary` utility has overlapping functionality with the `file` utility, but with a more limited focus.

## Installing the utility

With Go 1.24 or later:

    go install github.com/xyproto/binary/cmd/binary@latest

## Example use

* `binary /usr/bin/ls` returns `binary`.
* `binary /etc/fstab` returns `text`.

## Using the Go module

```go
package main

import (
    "fmt"
    "os"

    "github.com/xyproto/binary"
)

func main() {
    filename := os.Args[0]
    isBinary, err := binary.File(filename)
    if err != nil {
        fmt.Fprintf(os.Stderr, "%s\n", err)
        os.Exit(1)
    }
    fmt.Printf("%s is a binary file: %v\n", filename, isBinary)
}
```

The `binary.Data` function can be used to determine if a byte slice contains binary data or not.

## General info

* Version: 1.4.0
* License: BSD-3
* Author: Alexander F. Rødseth &lt;xyproto@archlinux.org&gt;
