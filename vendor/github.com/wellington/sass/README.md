[![Circle CI](https://circleci.com/gh/wellington/sass/tree/master.svg?style=svg)](https://circleci.com/gh/wellington/sass/tree/master)
[![Report Card](http://goreportcard.com/badge/wellington/sass)](http://goreportcard.com/report/wellington/sass)

# sass
Pure Go sass scanner, ast, and parser

Cross platform compiler for Sass

# Usage

``` shell
go get -u github.com/wellington/sass/sass
```

``` shell
sass compile [-o file.css] input.scss
```


This project is currently in alpha, and contains no compiler. A scanner and parser are being developed to support a future compiler.

To help, check out [parser](https://github.com/wellington/sass/tree/master/parser). This project contains tests that iterate through sass-spec running the parser against example inputs. Errors detected by the parser are reported. However, you could also set the Parser mode to `Trace` and verify proper ast trees are being built from the input. As the parser matures, output can automatically be verified by the example outputs in these directories.

Glossary
- [ ] Partial Support :question:
- [x] Full Support
- [ ] No Support

### Compiler Status
Passing 34 of the basic Sass tests in [sass-spec](https://github.com/sass/sass-spec)

### Function Status
See [functions](functions.md)


### Parser Status
- [x] Nested Rules
- [x] Referencing Parent Selectors: &
- [x] Nested Properties
- [ ] Placeholder Selectors: %foo
- [x] Comments: /* */ and //
- SassScript :question:
- Variables: $ :question:
- Data Types :question:
- [x] Strings
- [x] Lists (space and comma delimited)
- [ ] Maps
- [x] Colors
- Operations
  - [x] Number Operations
  - [x] Division and /
  - [x] Subtraction, Negative Numbers, and -
  - [x] Color Operations
  - [x] String Operations
  - [ ] Boolean Operations
  - [ ] List Operations
  - Parentheses :question:
- [x] Functions
- [x] Keyword Arguments
- [x] Interpolation: #{} (there are still edge cases with support)
- [x] & in SassScript
- [ ] Variable Defaults: !default
- @-Rules and Directives
  - [x] @import
  - [x] @media
  - [ ] @extend
    - [ ] Extending Complex Selectors
    - [ ] Multiple Extends
    - [ ] Chaining Extends
- [ ] Selector Sequences
- [ ] Merging Selector Sequences
- [ ] @extend-Only Selectors
- [ ] The !optional Flag
- [ ] @extend in Directives
- [ ] @at-root
- [ ] @at-root (without: ...) and @at-root (with: ...)
- [ ] @debug
- [ ] @warn
- [ ] @error
- Control Directives & Expressions
  - [ ] if()
  - [x] @if
    - @else if :question:
    - [x] @else
  - [ ] @for
  - [x] @each
  - [ ] Multiple Assignment
  - [ ] @while
  - [x] url(/local/path)
  - [x] url(http://remote/path)
- Mixin Directives
  - [x] Defining a Mixin: @mixin
  - [x] Including a Mixin: @include
- Arguments
  - [x] Literal arguments foo(one, two)
  - [x] Keyword Arguments foo($y: two, $x: one)
  - [ ] Variable Arguments :question:
- [x] Passing Content Blocks to a Mixin
- [x] Variable Scope and Content Blocks
- Function Directives :question:
- [ ] Extending Sass
- [ ] Defining Custom Sass Functions
