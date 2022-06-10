-- utility module to make the Lua 5.1 standard libraries behave more like Lua 5.2

if _VERSION == "Lua 5.1" then
   local _type, _select, _unpack, _error = type, select, unpack, error

   bit32 = require("bit32")

   -- detect LuaJIT (including LUAJIT_ENABLE_LUA52COMPAT compilation flag)
   -- note that string.dump panics in glua so we short-circuit the below check
   local is_luajit = false and (string.dump(function() end) or ""):sub(1, 3) == "\027LJ"
   local is_luajit52 = is_luajit and
      #setmetatable({}, { __len = function() return 1 end }) == 1

   local weak_meta = { __mode = "kv" }
   -- table that maps each running coroutine to the coroutine that resumed it
   -- this is used to build complete tracebacks when "coroutine-friendly" pcall
   -- is used.
   local pcall_previous = setmetatable({}, weak_meta)
   local pcall_callOf = setmetatable({}, weak_meta)
   local xpcall_running = setmetatable({}, weak_meta)
   local coroutine_running = coroutine.running

   -- the most powerful getmetatable we can get (preferably from debug)
   local sudo_getmetatable = getmetatable

   if _type(debug) == "table" then

      if _type(debug.getmetatable) == "function" then
         sudo_getmetatable = debug.getmetatable
      end

      if not is_luajit52 then
         local _G, package = _G, package
         local debug_setfenv = debug.setfenv
         debug.setuservalue = function(obj, value)
            if _type(obj) ~= "userdata" then
               _error("bad argument #1 to 'setuservalue' (userdata expected, got "..
                     _type(obj)..")", 2)
            end
            if value == nil then value = _G end
            if _type(value) ~= "table" then
               _error("bad argument #2 to 'setuservalue' (table expected, got "..
                     _type(value)..")", 2)
            end
            return debug_setfenv(obj, value)
         end

         local debug_getfenv = debug.getfenv
         debug.getuservalue = function(obj)
            if _type(obj) ~= "userdata" then
               return nil
            else
               local v = debug_getfenv(obj)
               if v == _G or v == package then
                  return nil
               end
               return v
            end
         end

         local debug_setmetatable = debug.setmetatable
         if _type(debug_setmetatable) == "function" then
            debug.setmetatable = function(value, tab)
               debug_setmetatable(value, tab)
               return value
            end
         end
      end -- not luajit with compat52 enabled

      if not is_luajit then
         local debug_getinfo = debug.getinfo
         local function calculate_trace_level(co, level)
            if level ~= nil then
               for out = 1, 1/0 do
                  local info = (co==nil) and debug_getinfo(out, "") or debug_getinfo(co, out, "")
                  if info == nil then
                     local max = out-1
                     if level <= max then
                        return level
                     end
                     return nil, level-max
                  end
               end
            end
            return 1
         end

         local stack_pattern = "\nstack traceback:"
         local stack_replace = ""
         local debug_traceback = debug.traceback
         debug.traceback = function (co, msg, level)
            local lvl
            local nilmsg
            if _type(co) ~= "thread" then
               co, msg, level = coroutine_running(), co, msg
            end
            if msg == nil then
               msg = ""
               nilmsg = true
            elseif _type(msg) ~= "string" then
               return msg
            end
            if co == nil then
               msg = debug_traceback(msg, level or 1)
            else
               local xpco = xpcall_running[co]
               if xpco ~= nil then
                  lvl, level = calculate_trace_level(xpco, level)
                  if lvl then
                     msg = debug_traceback(xpco, msg, lvl)
                  else
                     msg = msg..stack_pattern
                  end
                  lvl, level = calculate_trace_level(co, level)
                  if lvl then
                     local trace = debug_traceback(co, "", lvl)
                     msg = msg..trace:gsub(stack_pattern, stack_replace)
                  end
               else
                  co = pcall_callOf[co] or co
                  lvl, level = calculate_trace_level(co, level)
                  if lvl then
                     msg = debug_traceback(co, msg, lvl)
                  else
                     msg = msg..stack_pattern
                  end
               end
               co = pcall_previous[co]
               while co ~= nil do
                  lvl, level = calculate_trace_level(co, level)
                  if lvl then
                     local trace = debug_traceback(co, "", lvl)
                     msg = msg..trace:gsub(stack_pattern, stack_replace)
                  end
                  co = pcall_previous[co]
               end
            end
            if nilmsg then
               msg = msg:gsub("^\n", "")
            end
            msg = msg:gsub("\n\t%(tail call%): %?", "\000")
            msg = msg:gsub("\n\t%.%.%.\n", "\001\n")
            msg = msg:gsub("\n\t%.%.%.$", "\001")
            msg = msg:gsub("(%z+)\001(%z+)", function(some, other)
               return "\n\t(..."..#some+#other.."+ tail call(s)...)"
            end)
            msg = msg:gsub("\001(%z+)", function(zeros)
               return "\n\t(..."..#zeros.."+ tail call(s)...)"
            end)
            msg = msg:gsub("(%z+)\001", function(zeros)
               return "\n\t(..."..#zeros.."+ tail call(s)...)"
            end)
            msg = msg:gsub("%z+", function(zeros)
               return "\n\t(..."..#zeros.." tail call(s)...)"
            end)
            msg = msg:gsub("\001", function(zeros)
               return "\n\t..."
            end)
            return msg
         end
      end -- is not luajit
   end -- debug table available

   if not is_luajit52 then
      local _pairs = pairs
      pairs = function(t)
         local mt = sudo_getmetatable(t)
         if _type(mt) == "table" and _type(mt.__pairs) == "function" then
            return mt.__pairs(t)
         else
            return _pairs(t)
         end
      end

      local _ipairs = ipairs
      ipairs = function(t)
         local mt = sudo_getmetatable(t)
         if _type(mt) == "table" and _type(mt.__ipairs) == "function" then
            return mt.__ipairs(t)
         else
            return _ipairs(t)
         end
      end
   end -- not luajit with compat52 enabled

   if not is_luajit then
      local function check_mode(mode, prefix)
         local has = { text = false, binary = false }
         for i = 1,#mode do
            local c = mode:sub(i, i)
            if c == "t" then has.text = true end
            if c == "b" then has.binary = true end
         end
         local t = prefix:sub(1, 1) == "\27" and "binary" or "text"
         if not has[t] then
            return "attempt to load a "..t.." chunk (mode is '"..mode.."')"
         end
      end

      local _setfenv = setfenv
      local _load, _loadstring = load, loadstring
      load = function(ld, source, mode, env)
         mode = mode or "bt"
         local chunk, msg
         if _type( ld ) == "string" then
            if mode ~= "bt" then
               local merr = check_mode(mode, ld)
               if merr then return nil, merr end
            end
            chunk, msg = _loadstring(ld, source)
         else
            local ld_type = _type(ld)
            if ld_type ~= "function" then
               _error("bad argument #1 to 'load' (function expected, got "..ld_type..")", 2)
            end
            if mode ~= "bt" then
               local checked, merr = false, nil
               local function checked_ld()
                  if checked then
                     return ld()
                  else
                     checked = true
                     local v = ld()
                     merr = check_mode(mode, v or "")
                     if merr then return nil end
                     return v
                  end
               end
               chunk, msg = _load(checked_ld, source)
               if merr then return nil, merr end
            else
               chunk, msg = _load(ld, source)
            end
         end
         if not chunk then
            return chunk, msg
         end
         if env ~= nil then
            _setfenv(chunk, env)
         end
         return chunk
      end

      loadstring = load

      local _loadfile = loadfile
      local io_open = io.open
      loadfile = function(file, mode, env)
         mode = mode or "bt"
         if mode ~= "bt" then
            local f = io_open(file, "rb")
            if f then
               local prefix = f:read(1)
               f:close()
               if prefix then
                  local merr = check_mode(mode, prefix)
                  if merr then return nil, merr end
               end
            end
         end
         local chunk, msg = _loadfile(file)
         if not chunk then
            return chunk, msg
         end
         if env ~= nil then
            _setfenv(chunk, env)
         end
         return chunk
      end
   end -- not luajit

   if not is_luajit52 then
      function rawlen(v)
         local t = _type(v)
         if t ~= "string" and t ~= "table" then
            _error("bad argument #1 to 'rawlen' (table or string expected)", 2)
         end
         return #v
      end
   end -- not luajit with compat52 enabled

   local gc_isrunning = true
   local _collectgarbage = collectgarbage
   local math_floor = math.floor
   collectgarbage = function(opt, ...)
      opt = opt or "collect"
      local v = 0
      if opt == "collect" then
         v = _collectgarbage(opt, ...)
         if not gc_isrunning then _collectgarbage("stop") end
      elseif opt == "stop" then
         gc_isrunning = false
         return _collectgarbage(opt, ...)
      elseif opt == "restart" then
         gc_isrunning = true
         return _collectgarbage(opt, ...)
      elseif opt == "count" then
         v = _collectgarbage(opt, ...)
         return v, (v-math_floor(v))*1024
      elseif opt == "step" then
         v = _collectgarbage(opt, ...)
         if not gc_isrunning then _collectgarbage("stop") end
      elseif opt == "isrunning" then
         return gc_isrunning
      elseif opt ~= "generational" and opt ~= "incremental" then
         return _collectgarbage(opt, ...)
      end
      return v
   end

   if not is_luajit52 then
      local os_execute = os.execute
      local bit32_rshift = bit32.rshift
      os.execute = function(cmd)
         local code = os_execute(cmd)
         -- Lua 5.1 does not report exit by signal.
         if code == 0 then
            return true, "exit", code
         else
            return nil, "exit", bit32_rshift(code, 8)
         end
      end
   end -- not luajit with compat52 enabled

   if not is_luajit52 then
      table.pack = function(...)
         return { n = _select('#', ...), ... }
      end

      table.unpack = _unpack
   end -- not luajit with compat52 enabled


   local main_coroutine = coroutine.create(function() end)

   local _assert = assert
   local _pcall = pcall
   local coroutine_create = coroutine.create
   coroutine.create = function (func)
      local success, result = _pcall(coroutine_create, func)
      if not success then
         _assert(_type(func) == "function", "bad argument #1 (function expected)")
         result = coroutine_create(function(...) return func(...) end)
      end
      return result
   end

   local pcall_mainOf = setmetatable({}, weak_meta)

   if not is_luajit52 then
      coroutine.running = function()
         local co = coroutine_running()
         if co then
            return pcall_mainOf[co] or co, false
         else
            return main_coroutine, true
         end
      end
   end

   local coroutine_yield = coroutine.yield
   coroutine.yield = function(...)
      local co, flag = coroutine_running()
      if co and not flag then
         return coroutine_yield(...)
      else
         _error("attempt to yield from outside a coroutine", 0)
      end
   end

   if not is_luajit then
      local coroutine_resume = coroutine.resume
      coroutine.resume = function(co, ...)
         if co == main_coroutine then
            return false, "cannot resume non-suspended coroutine"
         else
            return coroutine_resume(co, ...)
         end
      end

      local coroutine_status = coroutine.status
      coroutine.status = function(co)
         local notmain = coroutine_running()
         if co == main_coroutine then
            return notmain and "normal" or "running"
         else
            return coroutine_status(co)
         end
      end

      local function pcall_results(current, call, success, ...)
         if coroutine_status(call) == "suspended" then
            return pcall_results(current, call, coroutine_resume(call, coroutine_yield(...)))
         end
         if pcall_previous then
            pcall_previous[call] = nil
            local main = pcall_mainOf[call]
            if main == current then current = nil end
            pcall_callOf[main] = current
         end
         pcall_mainOf[call] = nil
         return success, ...
      end
      local function pcall_exec(current, call, ...)
         local main = pcall_mainOf[current] or current
         pcall_mainOf[call] = main
         if pcall_previous then
            pcall_previous[call] = current
            pcall_callOf[main] = call
         end
         return pcall_results(current, call, coroutine_resume(call, ...))
      end
      local coroutine_create52 = coroutine.create
      local function pcall_coroutine(func)
         if _type(func) ~= "function" then
            local callable = func
            func = function (...) return callable(...) end
         end
         return coroutine_create52(func)
      end
      pcall = function (func, ...)
         local current = coroutine_running()
         if not current then return _pcall(func, ...) end
         return pcall_exec(current, pcall_coroutine(func), ...)
      end

      local _tostring = tostring
      local function xpcall_catch(current, call, msgh, success, ...)
         if not success then
            xpcall_running[current] = call
            local ok, result = _pcall(msgh, ...)
            xpcall_running[current] = nil
            if not ok then
               return false, "error in error handling (".._tostring(result)..")"
            end
            return false, result
         end
         return true, ...
      end
      local _xpcall = xpcall
      xpcall = function(f, msgh, ...)
         local current = coroutine_running()
         if not current then
            local args, n = { ... }, _select('#', ...)
            return _xpcall(function() return f(_unpack(args, 1, n)) end, msgh)
         end
         local call = pcall_coroutine(f)
         return xpcall_catch(current, call, msgh, pcall_exec(current, call, ...))
      end
   end -- not luajit

   if not is_luajit then
      local math_log = math.log
      math.log = function(x, base)
         if base ~= nil then
            return math_log(x)/math_log(base)
         else
            return math_log(x)
         end
      end
   end

   local package = package
   if not is_luajit then
      local io_open = io.open
      local table_concat = table.concat
      package.searchpath = function(name, path, sep, rep)
         sep = (sep or "."):gsub("(%p)", "%%%1")
         rep = (rep or package.config:sub(1, 1)):gsub("(%%)", "%%%1")
         local pname = name:gsub(sep, rep):gsub("(%%)", "%%%1")
         local msg = {}
         for subpath in path:gmatch("[^;]+") do
            local fpath = subpath:gsub("%?", pname)
            local f = io_open(fpath, "r")
            if f then
               f:close()
               return fpath
            end
            msg[#msg+1] = "\n\tno file '" .. fpath .. "'"
         end
         return nil, table_concat(msg)
      end
   end -- not luajit

   local p_index = { searchers = package.loaders }
   local _rawset = rawset
   setmetatable(package, {
      __index = p_index,
      __newindex = function(p, k, v)
         if k == "searchers" then
            _rawset(p, "loaders", v)
            p_index.searchers = v
         else
            _rawset(p, k, v)
         end
      end
   })

   local string_gsub = string.gsub
   local function fix_pattern(pattern)
      return (string_gsub(pattern, "%z", "%%z"))
   end

   local string_find = string.find
   function string.find(s, pattern, ...)
      return string_find(s, fix_pattern(pattern), ...)
   end

   local string_gmatch = string.gmatch
   function string.gmatch(s, pattern)
      return string_gmatch(s, fix_pattern(pattern))
   end

   function string.gsub(s, pattern, ...)
      return string_gsub(s, fix_pattern(pattern), ...)
   end

   local string_match = string.match
   function string.match(s, pattern, ...)
      return string_match(s, fix_pattern(pattern), ...)
   end

   if not is_luajit then
      local string_rep = string.rep
      function string.rep(s, n, sep)
         if sep ~= nil and sep ~= "" and n >= 2 then
            return s .. string_rep(sep..s, n-1)
         else
            return string_rep(s, n)
         end
      end
   end -- not luajit

   if not is_luajit then
      local _tostring = tostring
      local string_format = string.format
      do
         local addqt = {
            ["\n"] = "\\\n",
            ["\\"] = "\\\\",
            ["\""] = "\\\""
         }

         local function addquoted(c, d)
            return (addqt[c] or string_format(d~= "" and "\\%03d" or "\\%d", c:byte()))..d
         end

         function string.format(fmt, ...)
            local args, n = { ... }, _select('#', ...)
            local i = 0
            local function adjust_fmt(lead, mods, kind)
               if #lead % 2 == 0 then
                  i = i + 1
                  if kind == "s" then
                     args[i] = _tostring(args[i])
                  elseif kind == "q" then
                     args[i] = '"'..string_gsub(args[i], "([%z%c\\\"\n])(%d?)", addquoted)..'"'
                     return lead.."%"..mods.."s"
                  end
               end
            end
            fmt = string_gsub(fmt, "(%%*)%%([%d%.%-%+%# ]*)(%a)", adjust_fmt)
            return string_format(fmt, _unpack(args, 1, n))
         end
      end
   end -- not luajit

   local io_open = io.open
   local io_write = io.write
   local io_output = io.output
   function io.write(...)
      local res, msg, errno = io_write(...)
      if res then
         return io_output()
      else
         return nil, msg, errno
      end
   end

   if not is_luajit then
      local lines_iterator
      do
         local function helper( st, var_1, ... )
            if var_1 == nil then
               if st.doclose then st.f:close() end
               if (...) ~= nil then
                  _error((...), 2)
               end
            end
            return var_1, ...
         end

         function lines_iterator(st)
            return helper(st, st.f:read(_unpack(st, 1, st.n)))
         end
      end

      local valid_format = { ["*l"] = true, ["*n"] = true, ["*a"] = true }

      local io_input = io.input
      function io.lines(fname, ...)
         local doclose, file, msg
         if fname ~= nil then
            doclose, file, msg = true, io_open(fname, "r")
            if not file then _error(msg, 2) end
         else
            doclose, file = false, io_input()
         end
         local st = { f=file, doclose=doclose, n=_select('#', ...), ... }
         for i = 1, st.n do
            if _type(st[i]) ~= "number" and not valid_format[st[i]] then
              _error("bad argument #"..(i+1).." to 'for iterator' (invalid format)", 2)
            end
         end
         return lines_iterator, st
      end

      do
         local io_stdout = io.stdout
         local io_type = io.type
         local file_meta = sudo_getmetatable(io_stdout)
         if _type(file_meta) == "table" and _type(file_meta.__index) == "table" then
            local file_write = file_meta.__index.write
            file_meta.__index.write = function(self, ...)
               local res, msg, errno = file_write(self, ...)
               if res then
                  return self
               else
                  return nil, msg, errno
               end
            end

            file_meta.__index.lines = function(self, ...)
               if io_type(self) == "closed file" then
                  _error("attempt to use a closed file", 2)
               end
               local st = { f=self, doclose=false, n=_select('#', ...), ... }
               for i = 1, st.n do
                  if _type(st[i]) ~= "number" and not valid_format[st[i]] then
                     _error("bad argument #"..(i+1).." to 'for iterator' (invalid format)", 2)
                  end
               end
               return lines_iterator, st
            end
         end
      end
   end -- not luajit

end
