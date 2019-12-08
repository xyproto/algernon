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
	htmlData, err := ioutil.ReadFile("input.html")
	if err != nil {
		panic(err)
	}

	// Highlight the source code in the HTML with the monokai style
	htmlBytes, err := splash.Splash(htmlData, "monokai")
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile("output.html", htmlBytes, 0644)
	if err != nil {
		panic(err)
	}
}
```

## Available syntax highlighting styles

See the [Style Gallery](https://xyproto.github.io/splash/docs/) for a full overview of available styles and how they may appear.

## General information

* Version: 1.1.2
* License: MIT
* Author: Alexander F Rødseth &lt;xyproto@archlinux.org&gt;
