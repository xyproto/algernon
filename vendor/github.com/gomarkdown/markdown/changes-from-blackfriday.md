## Changes from blackfriday

This library is derived from the blackfriday library. Here's a list of changes.

**Redesigned API**

- split into 3 separate packages: ast, parser and html (for the HTML renderer). This makes the API more manageable. It also separates e.g. parser options from renderer options
- changed how an AST node is represented from a union-like representation (manually keeping track of the type of the node) to using an interface (the Go way to combine an arbitrary value with its type)

**Allow reusing most of the HTML renderer logic**

You can implement your own renderer by implementing the `Renderer` interface.

Implementing a full renderer is a lot of work and often you just want to tweak HTML rendering of a few node types.

I've added a way to hook `Renderer.RenderNode` in the HTML renderer with a custom function that can take over rendering of specific nodes.

I use it myself to do syntax-highlighting of code snippets.

**Speed up go test**

Running `go test` was really slow (17 secs) because it did a poor man's version of fuzzing by feeding the parser all subsets of test strings in order to find panics
due to incorrect parsing logic.

That logic was removed from the default test run. Crashers live in `fuzz_crashes_test.go`. `FuzzParse` and `FuzzCommonMark` are `go test -fuzz` targets, and `s/fuzz.sh` runs them.

Now `go test` is blazing fast.
