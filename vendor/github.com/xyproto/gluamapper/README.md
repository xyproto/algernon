# gluamapper

![Build](https://github.com/xyproto/gluamapper/workflows/Build/badge.svg) [![Go Report Card](https://goreportcard.com/badge/github.com/xyproto/gluamapper)](https://goreportcard.com/report/github.com/xyproto/gluamapper) [![License](https://img.shields.io/badge/license-MIT-green.svg?style=flat)](https://raw.githubusercontent.com/xyproto/gluamapper/main/LICENSE)

Maps a GopherLua table to a Go struct. This is a fork of [yuin/gluamapper](https://github.com/yuin/gluamapper).

gluamapper converts a GopherLua table to `map[string]interface{}`, and then converts it to a Go struct using [mitchellh/mapstructure](https://github.com/mitchellh/mapstructure/).

## Installation of the development version

    go get -u github.com/xyproto/gluamapper

## API

See [Go doc](http://godoc.org/github.com/xyproto/gluamapper).

## Usage

```go
type Role struct {
    Name string
}

type Person struct {
    Name      string
    Age       int
    WorkPlace string
    Role      []*Role
}

L := lua.NewState()
if err := L.DoString(`
person = {
  name = "Michel",
  age  = "31", -- weakly input
  work_place = "San Jose",
  role = {
    {
      name = "Administrator"
    },
    {
      name = "Operator"
    }
  }
}
`); err != nil {
    panic(err)
}
var person Person
if err := gluamapper.Map(L.GetGlobal("person").(*lua.LTable), &person); err != nil {
    panic(err)
}
fmt.Printf("%s %d", person.Name, person.Age)
```

## General info

* License: MIT
* Version: 1.1.0
