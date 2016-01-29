package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/yuin/gopher-lua"
)

// Extra Lua functions

const luacode = `
-- Given the name of a python script in the same directory,
-- return the outputted lines as a table
function py(filename)
  if filename == nil then
    return {}
  end
  local cmd = "python " .. scriptdir() .. "/" .. filename
  local f = assert(io.popen(cmd, 'r'))
  local a = {}
  for line in f:lines() do
    table.insert(a, line)
  end
  f:close()
  return a
end

-- Given the name of an executable (or executable script) in the same directory,
-- return the outputted lines as a table
function run(given_command)
  if given_command == nil then
    return {}
  end
  local cmd = "cd " .. scriptdir() .. "; " .. given_command
  local f = assert(io.popen(cmd, 'r'))
  local a = {}
  for line in f:lines() do
    table.insert(a, line)
  end
  f:close()
  return a
end

-- List a table
function dir(t)
  if t == nil then
    t = _G
  end
  for k, v in pairs(t) do
    print(string.format("%-16s\t->\t%s", tostring(k), tostring(v)))
  end
  return ""
end
`

// Lua function for converting a table to JSON (string or int)
func loadExtras(L *lua.LState) int {
	if err := L.DoString(luacode); err != nil {
		log.Error("Could not load Lua extras!")
		log.Error(err)
	}
	//L.Push(lua.LString("Loaded extra functions"))
	//return 1 // number of results
	return 0 // number of results
}

func exportExtras(L *lua.LState) {
	// Load extra Lua functions
	//L.SetGlobal("extras", L.NewFunction(loadExtras))
	loadExtras(L)
}
