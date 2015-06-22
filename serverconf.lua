--- Algernon Server Configuration
--- For use with the samples

--- Logging (will log to console if an empty string is given)
--LogTo("algernon.log")
--LogTo("/var/log/algernon.log")

--- Clear the URL prefixes for the access permissions
--- (see https://github.com/xyproto/permissions2 for an overview of the default paths)
ClearPermissions()

--- For the "bob" example, when running from this directory
AddAdminPrefix("/samples/bob/admin")
AddUserPrefix("/samples/bob/data")

--- For the "bob" example, when running from the "bob" directory
AddAdminPrefix("/admin")
AddUserPrefix("/data")

--- For the "chat" example, when running from this directory
AddUserPrefix("/samples/chat/chat")

--- For the "chat" example, when running from the "chat" directory
AddUserPrefix("/chat")

-- Output server configuration after parsing this file and commandline arguments
OnReady(function ()
  print(ServerInfo())
end)

-- Custom permission denied handler
DenyHandler(function ()
  content("text/html")
  print[[<!doctype html><html><head><title>Permission denied</title><link href='//fonts.googleapis.com/css?family=Lato:300' rel='stylesheet' type='text/css'></head><body style="background-color: #f0f0f0; color: #101010; font-family: 'Lato', sans-serif; font-weight: 300; margin: 4em; font-size: 2em;">]]
  print("<strong>HTTP "..method()..[[</strong> <font color="red">denied</font> for ]]..urlpath())
  print([[</body></html>]])
end)
