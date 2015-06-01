[pie](https://github.comnatefinch/pie) plugins for Algernon
===========================================================

Plugins must offer Lua.Code and Lua.Help functions.

Lua.Code serves Lua code, while Lua.Help serves help for the available Lua functions.
The Lua functions can call other plugin functions with the `CallPlugin` function in Algernon.

Plugins can be loaded with the `Plugin` function in Algernon.
