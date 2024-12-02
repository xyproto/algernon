# randomstring

Generate random strings.

These are the exported function signatures:

```go
func PickLetter() rune
func PickVowel() rune
func PickCons() rune
func Seed()
func String(length int) string
func EnglishFrequencyString(length int) string
func HumanFriendlyString(length int) string
func CookieFriendlyString(length int) string
func CookieFriendlyBytes(length int) []byte
func HumanFriendlyEnglishString(length int) string
```

Used by [cookie](https://github.com/xyproto/cookie) and [alienpdf](https://github.com/xyproto/alienpdf/).

### General info

* Version: 1.2.0
* License: BSD-3
