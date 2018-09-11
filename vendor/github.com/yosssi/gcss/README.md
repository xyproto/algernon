# GCSS - Pure Go CSS Preprocessor

[![wercker status](https://app.wercker.com/status/4857161fd705e6c43df492e6a33ce87f/m "wercker status")](https://app.wercker.com/project/bykey/4857161fd705e6c43df492e6a33ce87f)
[![Build status](https://ci.appveyor.com/api/projects/status/ocbu6upgr3j0m3vc/branch/master)](https://ci.appveyor.com/project/yosssi/gcss/branch/master)
[![Coverage Status](https://img.shields.io/coveralls/yosssi/gcss.svg)](https://coveralls.io/r/yosssi/gcss?branch=master)
[![GoDoc](http://godoc.org/github.com/yosssi/gcss?status.svg)](http://godoc.org/github.com/yosssi/gcss)
[![Gitter](https://badges.gitter.im/Join Chat.svg)](https://gitter.im/yosssi/gcss?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)

## Overview

GCSS is a pure Go CSS preprocessor. This is inspired by [Sass](http://sass-lang.com/) and [Stylus](http://learnboost.github.io/stylus/).

## Syntax

### Variables

```scss
$base-font: Helvetica, sans-serif
$main-color: blue

body
  font: 100% $base-font
  color: $main-color
```

### Nesting

```scss
nav
  ul
    margin: 0
    padding: 0

a
  color: blue
  &:hover
    color: red
```

### Mixins

```scss
$border-radius($radius)
  -webkit-border-radius: $radius
  -moz-border-radius: $radius
  -ms-border-radius: $radius
  border-radius: $radius

.box
  $border-radius(10px)
```

## Installation

```sh
$ go get -u github.com/yosssi/gcss/...
```

## Compile from the Command-Line

```sh
$ gcss /path/to/gcss/file
```

or

```sh
$ cat /path/to/gcss/file | gcss > /path/to/css/file
```

## Compile from Go programs

You can compile a GCSS file from Go programs by invoking the `gcss.CompileFile` function.

```go
cssPath, err := gcss.CompileFile("path_to_gcss_file")

if err != nil {
	http.Error(w, err.Error(), http.StatusInternalServerError)
	return
}

http.ServeFile(w, r, cssPath)
```

You can invoke the `gcss.Compile` function instead of the `gcss.CompileFile` function. The `gcss.Compile` function takes `io.Writer` and `io.Reader` as a parameter, compiles the GCSS data which is read from the `io.Reader` and writes the result CSS data to the `io.Writer`. Please see the [GoDoc](http://godoc.org/github.com/yosssi/gcss) for the details.

```go
f, err := os.Open("path_to_gcss_file")

if err != nil {
	panic(err)
}

defer func() {
	if err := f.Close(); err != nil {
		panic(err)
	}
}()

n, err := gcss.Compile(os.Stdout, f)
```

## Documentation

* [GoDoc](http://godoc.org/github.com/yosssi/gcss)

## Syntax Highlightings

* [vim-gcss](https://github.com/yosssi/vim-gcss) - Vim syntax highlighting for GCSS
