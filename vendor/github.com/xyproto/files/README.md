# files

Functions for querying files and paths.

## Function signatures

```
func Exists(path string) bool
func IsFile(path string) bool
func IsSymlink(path string) bool
func IsFileOrSymlink(path string) bool
func IsDir(path string) bool
func Which(executable string) string
func WhichCached(executable string) string
func PathHas(executable string) bool
func PathHasCached(executable string) bool
func BinDirectory(filename string) bool
func DataReadyOnStdin() bool
func IsBinary(filename string) bool
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
func IsExecutable(path string) bool
func IsExecutableCached(path string) bool
```

## General info

* Version: 1.9.0
* License: BSD-3
* Author: Alexander F. Rødseth &gt;xyproto@archlinux.org&lt;
