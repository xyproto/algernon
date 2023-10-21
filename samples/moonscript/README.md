# Moonscript

Moonscript is a language that can be compiled into Lua.

This sample requires Make (`make`) and Moonscript (`moonc`) to be installed and working.

Just run `make` to build `hello.moon` and serve it with `algernon`.

Use `make index` to build `index.moon` and serve it with `algernon`.

* `hello.lua` is served as server configuration script that sets up a simple handle.
* `index.lua` is served as a regular Algernon handler.

Different Lua functions are available in the two modes.
