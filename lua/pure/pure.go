// Package pure provides Lua functions for running commands and listing files
package pure

import (
	log "github.com/sirupsen/logrus"
	"github.com/xyproto/gopher-lua"
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
  local output = {}
  for k, v in pairs(t) do
	table.insert(output, string.format("%-16s\t->\t%s", tostring(k), tostring(v)))
  end
  return table.concat(output, "\n")
end
`

// Load makes functions for running commands, python code or listing files to
// the given Lua state struct: py, run and dir
func Load(L *lua.LState) {
	if err := L.DoString(luacode); err != nil {
		log.Errorf("Could not load extra Lua functions: %s", err)
	}
}
