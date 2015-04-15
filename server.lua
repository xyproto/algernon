--- Algernon Server Configuration

--- Set the default host and port.
SetAddr(":3000")

--- Debug flag
--- If set to true, errors will be shown in the browsers,
--- and request will be buffered (which results in cookies not working, for now)
SetDebug(false)

--- Verbose flag
-- If set to true, log messages will be more frequent
SetVerbose(false)

--- Logging (will log to console if an empty string is given)
--LogTo("algernon.log")
--LogTo("/var/log/algernon.log")

--- Clear the URL prefixes for the access permissions
--- (see https://github.com/xyproto/permissions2 for an overview of the default paths)
ClearPermissions()

--- For the "bob" example, when running from this directory
AddAdminPrefix("/examples/bob/admin")
AddUserPrefix("/examples/bob/data")

--- For the "bob" example, when running from the "bob" directory
AddAdminPrefix("/admin")
AddUserPrefix("/data")

--- For the "chat" example, when running from this directory
AddUserPrefix("/examples/chat/chat")

--- For the "chat" example, when running from the "chat" directory
AddUserPrefix("/chat")

-- Output server configuration after parsing this file and commandline arguments
OnReady(function()
  print(ServerInfo())
end)

-- Custom permission denied handler
DenyHandler(function()
  content("text/html")
  print[[<!doctype html><html><head><title>Permission denied</title><link href='//fonts.googleapis.com/css?family=Lato:300' rel='stylesheet' type='text/css'></head><body style="background: #f0f0f0; color: #101010; font-family: 'Lato', sans-serif; font-weight: 300; margin: 4em; font-size: 2em;">]]
  print("<strong>HTTP "..method()..[[</strong> <font color="red">denied</font> for ]]..urlpath())
  error([[</body></html>]])
end)
