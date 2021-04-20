# Splash

Highlight code embedded in HTML using the [chroma](https://github.com/alecthomas/chroma) package.

The generated output is tested by visual inspection in Chromium, Midori and Netsurf.

## Example usage

```go
package main

import (
	"github.com/xyproto/splash"
	"io/ioutil"
)

func main() {
	// Read "input.html"
	inputHTML, err := ioutil.ReadFile("input.html")
	if err != nil {
		panic(err)
	}

	// Highlight the source code in the HTML document with the monokai style
	outputHTML, err := splash.Splash(inputHTML, "monokai")
	if err != nil {
		panic(err)
	}

	// Write the highlighted HTML to "output.html"
	if err := ioutil.WriteFile("output.html", outputHTML, 0644); err != nil {
		panic(err)
	}
}
```

## Available syntax highlighting styles

See the [Style Gallery](https://xyproto.github.io/splash/docs/) for a full overview of available styles and how they may appear.

## General information

* Version: 1.1.4
* License: MIT
* Author: Alexander F. Rødseth &lt;xyproto@archlinux.org&gt;
