--- Algernon Server Configuration

--- Set the host and port (can be overridden by commandline flags)
SetAddr(":443")

--- Debug mode (errors will be shown in the browser)
SetDebug(true)

--- Logging (will log to console if an empty string is given)
LogTo("/var/log/algernon.log")

--- Permissions, for URL prefixes (see https://github.com/xyproto/permissions2 for more info)
--ClearPermissions()
--AddAdminPrefix("/admin")
--AddUserPrefix("/data")

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
