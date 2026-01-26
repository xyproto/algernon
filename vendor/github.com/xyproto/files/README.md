# files

Functions for querying files and paths.

## Function signatures

```
func Exists(path string) bool
func File(path string) bool
func Symlink(path string) bool
func FileOrSymlink(path string) bool
func Dir(path string) bool
func Which(executable string) string
func WhichCached(executable string) string
func PathHas(executable string) bool
func PathHasCached(executable string) bool
func BinDirectory(filename string) bool
func DataReadyOnStdin() bool
func Binary(filename string) bool
func FilterOutBinaryFiles(filenames []string) []string
func TimestampedFilename(filename string) string
func ShortPath(path string) string
func FileHas(path, what string) bool
func ReadString(filename string) string
func CanRead(filename string) bool
func Relative(path string) string
func Touch(filename string) error
func ExistsCached(path string) bool
func ClearCache()
func RemoveFile(path string) error
func DirectoryWithFiles(path string) (bool, error)
func Executable(path string) bool
func ExecutableCached(path string) bool
func Empty(path string) bool
func RealPath(path string) bool
```

## Running commands

```
// Run a command without using a shell, only return nil if it went well
func Run(command string) error
// Run a command with /bin/sh and return the combined and trimmed output
func Shell(command string) (string, error)
// Run a command with /bin/bash (or bash from the PATH) and return the combined and trimmed output
func Bash(command string) (string, error)
// Run a command with /usr/bin/fish (or fish from the PATH) and return the combined and trimmed output
func Fish(command string) (string, error)
```

## General info

* Version: 1.10.2
* License: BSD-3
* Author: Alexander F. Rødseth &lt;xyproto@archlinux.org&gt;
