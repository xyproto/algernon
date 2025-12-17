# Splash

Syntax highlight code embedded in HTML with a splash of color by using the [chroma](https://github.com/alecthomas/chroma) package.

The generated output is tested by visual inspection in Chromium, Midori and Netsurf.

## Example usage

```go
package main

import (
    "os"

    "github.com/xyproto/splash"
)

func main() {
    // Read "input.html"
    inputHTML, err := os.ReadFile("input.html")
    if err != nil {
        panic(err)
    }

    // Highlight the source code in the HTML document with the monokai style
    outputHTML, err := splash.Splash(inputHTML, "monokai")
    if err != nil {
        panic(err)
    }

    // Write the highlighted HTML to "output.html"
    if err := os.WriteFile("output.html", outputHTML, 0644); err != nil {
        panic(err)
    }
}
```

## Available syntax highlighting styles

See the [Style Gallery](https://xyproto.github.io/splash/docs/) for a full overview of available styles and how they may appear.

## General information

* Version: 1.3.0
* License: BSD-3
* Author: Alexander F. Rødseth &lt;xyproto@archlinux.org&gt;
