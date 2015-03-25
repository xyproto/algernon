--- Algernon Server Configuration

--- Host and port
SetAddr(":3000")

--- Debug mode (source code and errors may be shown in the browser)
SetDebug(true)

--- Logging (will log to console unless filenames are provided)
AccessLog("/var/log/algernon_access.log")
ErrorLog("/var/log/algernon_error.log")

--- Clear the URL prefixes for the access permissions
--- (see https://github.com/xyproto/permissions2 for an overview
--- of the default paths)
--ClearPermissions()

-- URL prefixes for the "bob" example, if running from this directory
AddAdminPrefix("/examples/bob/admin")
AddUserPrefix("/examples/bob/data")

-- URL prefixes for the "chat" example, if running from this directory
AddUserPrefix("/examples/chat/chat")

-- Output server configuration
print(ServerInfo())

-- Custom permission denied handler
DenyHandler(function()
  content("text/html")
  print[[<!doctype html><html><body style="background: white; color: red; font-family: sans-serif; margin: 4em;">]]
  print("HTTP "..method().." denied for "..urlpath().."!")
  error([[</body></html>]])
end)
