---
--- Algernon Application Configuration
---

-- After parsing this file and commandline arguments
OnReady(function()
  -- Store the Lua code from add3 in the code library
  c = CodeLib()
  c:set("lib", PluginCode("plugins/add3"))
  c:add("lib", [[print("lib loaded!")]])
end)

-- Custom permission denied handler
DenyHandler(function()
  content("text/html")
  print[[<!doctype html><html><head><title>Permission denied</title><link href='//fonts.googleapis.com/css?family=Lato:300' rel='stylesheet' type='text/css'></head><body style="background-color: #f0f0f0; color: #101010; font-family: 'Lato', sans-serif; font-weight: 300; margin: 4em; font-size: 2em;">]]
  print("<strong>HTTP "..method()..[[</strong> <font color="red">denied</font> for ]]..urlpath())
  print([[</body></html>]])
end)
