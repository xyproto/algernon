# Env ![Build](https://github.com/xyproto/env/workflows/Build/badge.svg) [![GoDoc](https://godoc.org/github.com/xyproto/env?status.svg)](http://godoc.org/github.com/xyproto/env) [![Go Report Card](https://goreportcard.com/badge/github.com/xyproto/env)](https://goreportcard.com/report/github.com/xyproto/env)

Makes fetching and interpreting environment variables easy and safe.

Being able to provide default values when retrieving environment variables often makes program logic shorter and more readable.

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

### DurationSeconds

Takes a default int64 value, for the number of seconds, interprets the environment variable as the number of seconds and returns a `time.Duration`.

### DurationMinutes

Takes a default int64 value, for the number of minutes, interprets the environment variable as the number of seconds and returns a `time.Duration`.

### DurationHours

Takes a default int64 value, for the number of hours, interprets the environment variable as the number of seconds and returns a `time.Duration`.

### Contains

Checks if the given environment variable contains the given string.

### Is

Checks if the given environment variable is the given value, with leading and trailing spaces trimmed before comparing both values.

### HomeDir

Returns the home directory of the current user, or `/tmp` if it is not available. `/home/$LOGNAME` or `/home/$USER` are returned if `$HOME` is not set.

### ExpandUser

Replaces `~` or `$HOME` at the start of a string with the home directory of the current user.

### File

Does the same as the `Str` function, but replaces a leading `~` or `$HOME` with the home directory of the current user.

### Dir

Does the same as the `Str` function, but replaces a leading `~` or `$HOME` with the home directory of the current user.

### Path

Returns the current `$PATH` as a slice of strings.

### Other functions

There are also: `Float64`, `Float32`, `Uint64`, `Uint32`, `Uint16`, `Uint8`, `Int64`, `Int32`, `Int16` and `Int8` functions available.

## Example

```go
package main

import (
    "fmt"
    "github.com/xyproto/env"
)

func main() {
    fmt.Println(env.DurationSeconds("REQUEST_TIMEOUT", 1800))
}
```

Running the above program like this: `REQUEST_TIMEOUT=1200 ./main`, outputs:

    20m0s

## Cache

* If `Load()` is called, then cache is enabled and all environment variables are read from the system.
* The functions offered by this package will then read the values from the cache instead.
* If variables are set with `os.Setenv`, then `Load()` can be used to read the environment variables from the system again.
* `Unload()` can be called to stop using the cache and only rely on `os.Setenv` and `os.Getenv` again.
* The `Set` and `Unset` functions can be used to both set values with `os.Setenv` and also modify the cache, so that an additional call to `Load()` is not needed.

## General info

* Version: 1.9.1
* License: BSD-3
* Author: Alexander F. Rødseth &lt;xyproto@archlinux.org&gt;
