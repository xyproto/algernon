--- Algernon 0.48 Server Configuration

--- Host and port
SetAddr(":3000")

-- TO IMPLEMENT:
--- Debug mode (source code and errors may be shown in the browser)
--SetDebug(true)

-- TO IMPLEMENT:
--- Logging (will log to console unless filenames are provided)
--AccessLog("/var/log/algernon_access.log")
--ErrorLog("/var/log/algernon_error.log")

--- Clear the URL prefixes for the access permissions
--- (see https://github.com/xyproto/permissions2 for an overview of the default paths)
ClearPermissions()

-- URL prefixes for the "bob" example, if running from this directory
AddAdminPrefix("/examples/bob/admin")
AddUserPrefix("/examples/bob/data")

-- URL perfixes for the "bob" example, if running from the "bob" directory
AddAdminPrefix("/admin")
AddUserPrefix("/data")

-- URL prefixes for the "chat" example, if running from this directory
AddUserPrefix("/examples/chat/chat")

-- Output server configuration
print(ServerInfo())

-- Custom permission denied handler
DenyHandler(function()
  content("text/html")
  print[[<!doctype html><html><head><title>Permission denied</title><link href='//fonts.googleapis.com/css?family=Lato:300' rel='stylesheet' type='text/css'></head><body style="background: #f0f0f0; color: #101010; font-family: 'Lato', sans-serif; font-weight: 300; margin: 4em; font-size: 2em;">]]
  print("<strong>HTTP "..method()..[[</strong> <font color="red">denied</font> for ]]..urlpath())
  error([[</body></html>]])
end)
