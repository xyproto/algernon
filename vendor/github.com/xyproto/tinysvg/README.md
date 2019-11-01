# tinysvg [![Build Status](https://travis-ci.org/xyproto/tinysvg.svg?branch=master)](https://travis-ci.org/xyproto/tinysvg) [![Go Report Card](https://goreportcard.com/badge/github.com/xyproto/tinysvg)](https://goreportcard.com/report/github.com/xyproto/tinysvg) [![GoDoc](https://godoc.org/github.com/xyproto/tinysvg?status.svg)](https://godoc.org/github.com/xyproto/tinysvg)

This is the parts related to TinySVG 1.2, extracted from the [onthefly](https://github.com/xyproto/onthefly) package.

It can be used for generating and saving SVG images that follow the TinySVG 1.2 spec.

This package mainly uses `[]byte` slices instead of strings, and does not indent the generated SVG daata, for performance and compactness.

Requires Go 1.9 or later.

* Version: 0.3.0
* Author: Alexander F. Rødseth &lt;xyproto@archlinux.org&gt;
* License: MIT
