#### Functions

RGB Functions
- [x] rgb($red, $green, $blue)
- [x] rgba($red, $green, $blue, $alpha)
- [x] red($color)
- [x] green($color)
- [x] blue($color)
- [x] mix($color1, $color2, [$weight])

HSL Functions
- [ ] hsl($hue, $saturation, $lightness)
- [ ] hsla($hue, $saturation, $lightness, $alpha)
- [ ] hue($color)
- [ ] saturation($color)
- [ ] lightness($color)
- [ ] adjust-hue($color, $degrees)
- [ ] lighten($color, $amount)
- [ ] darken($color, $amount)
- [ ] saturate($color, $amount)
- [ ] desaturate($color, $amount)
- [ ] grayscale($color)
- [ ] complement($color)
- [ ] invert($color)

Opacity Functions
- [ ] alpha($color)
- [ ] opacity($color)
- [x] rgba($color, $alpha)
- [ ] opacify($color, $amount) / fade-in($color, $amount)
- [ ] transparentize($color, $amount) / fade-out($color, $amount)

Other Color Functions
- [ ] adjust-color($color, [$red], [$green], [$blue], [$hue], [$saturation], [$lightness], [$alpha])
- [ ] scale-color($color, [$red], [$green], [$blue], [$saturation], [$lightness], [$alpha])
- [ ] change-color($color, [$red], [$green], [$blue], [$hue], [$saturation], [$lightness], [$alpha])

Changes one or more properties of a color.
- [ ] ie-hex-str($color)

String Functions
- [x] unquote($string)
- [ ] quote($string)
- [ ] str-length($string)
- [ ] str-insert($string, $insert, $index)

Inserts $insert into $string at $index.
- [ ] str-index($string, $substring)
- [ ] str-slice($string, $start-at, [$end-at])

Extracts a substring from $string.
- [ ] to-upper-case($string)
- [ ] to-lower-case($string)

Number Functions
- [ ] percentage($number)
- [ ] round($number)
- [ ] ceil($number)
- [ ] floor($number)
- [ ] abs($number)
- [ ] min($numbers…)
- [ ] max($numbers…)
- [ ] random([$limit])

List Functions
- [ ] length($list)
- [ ] nth($list, $n)
- [ ] set-nth($list, $n, $value)

Replaces the nth item in a list.
- [ ] join($list1, $list2, [$separator])
- [ ] Joins together two lists into one.
- [ ] append($list1, $val, [$separator])
- [ ] Appends a single value onto the end of a list.
- [ ] zip($lists…)

Combines several lists into a single multidimensional list.
- [ ] index($list, $value)
- [ ] list-separator($list)

Map Functions
- [ ] map-get($map, $key)
- [ ] map-merge($map1, $map2)
- [ ] map-remove($map, $keys…)

Returns a new map with keys removed.
- [ ] map-keys($map)
- [ ] map-values($map)

Returns a list of all values in a map.
- [ ] map-has-key($map, $key)
- [ ] keywords($args)

Selector Functions
- [ ] selector-nest($selectors…)
- [ ] selector-append($selectors…)
- [ ] selector-extend($selector, $extendee, $extender)
- [ ] selector-replace($selector, $original, $replacement)
- [ ] selector-unify($selector1, $selector2)
- [ ] is-superselector($super, $sub)
- [ ] simple-selectors($selector)
- [ ] selector-parse($selector)

Introspection Functions
- [ ] feature-exists($feature)
- [ ] variable-exists($name)
- [ ] global-variable-exists($name)
- [ ] function-exists($name)
- [ ] mixin-exists($name)
- [ ] inspect($value)
- [x] type-of($value)
- [ ] unit($number)
- [ ] unitless($number)
- [ ] comparable($number1, $number2)
- [ ] call($name, $args…)

Miscellaneous Functions
- [ ] if($condition, $if-true, $if-false)
- [ ] unique-id()
