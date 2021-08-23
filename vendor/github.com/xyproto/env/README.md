# Env [![Build Status](https://travis-ci.com/xyproto/env.svg?branch=main)](https://travis-ci.com/xyproto/env) [![GoDoc](https://godoc.org/github.com/xyproto/env?status.svg)](http://godoc.org/github.com/xyproto/env) [![Go Report Card](https://goreportcard.com/badge/github.com/xyproto/env)](https://goreportcard.com/report/github.com/xyproto/env)

Get the benefit of supplying default values when fetching environment variables.

Also interpret many types of string values that could mean `true` or `false`.

## Functions

### func Str

`func Str(name string, optionalDefault ...string) string`

`Str` does the same as `os.Getenv`, but allows the user to provide a default value (optional).
Only the first optional value is used, if the environment variable value is empty or not set.

### func Bool

`func Bool(envName string) bool`

`Bool` returns the bool value of the given environment variable name. Returns `false` if it is not declared or empty.

### func Int

`func Int(envName string, defaultValue int) int`

`Int` returns the number stored in the environment variable, or the given default value.

### func AsBool

`func AsBool(s string) bool`

`AsBool` can be used to interpret a string value as either `true` or `false`. Examples of `true` values are "yes" and "1".

### func Has

`func Has(s string) bool`

`Has` return true if the given environment variable name is non-empty.

## General info

* Version: 1.1.0
* License: MIT
* Author: Alexander F. Rødseth &lt;xyproto@archlinux.org&gt;
