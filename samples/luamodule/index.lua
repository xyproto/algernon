-- https://www.tutorialspoint.com/lua/lua_modules.htm
print[[<!doctype html><html><head><title>Lua Module</title><link rel="stylesheet" type="text/css" href="style.css"></head><body>]]

mymath = require("mymath")
mprint("## A simple Lua Module")
mprint("* 10 + 20 = " .. tostring(mymath.add(10,20)))
mprint("* 30 - 20 = " .. tostring(mymath.sub(30,20)))
mprint("* 10 * 20 = " .. tostring(mymath.mul(10,20)))
mprint("* 30 / 20 = " .. tostring(mymath.div(30,20)))

print[[</body></html>]]
