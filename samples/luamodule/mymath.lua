--
-- An example Lua module, based on https://www.tutorialspoint.com/lua/lua_modules.htm
--

local mymath =  {}

function mymath.add(a,b)
   return a + b
end

function mymath.sub(a,b)
   return a - b
end

function mymath.mul(a,b)
   return a * b
end

function mymath.div(a,b)
   return a / b
end

return mymath	
