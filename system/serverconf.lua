--- Algernon Server Configuration

-- Custom permission denied handler
DenyHandler(function()
  content("text/html")
  print[[<!doctype html><html><head><title>Permission denied</title><link href='//fonts.googleapis.com/css?family=Lato:300' rel='stylesheet' type='text/css'></head><body style="background-color: #f0f0f0; color: #101010; font-family: 'Lato', sans-serif; font-weight: 300; margin: 4em; font-size: 2em;">]] .. "<strong>HTTP " .. htmlescape(method()) .. [[</strong> <font color="red">denied</font> for ]] .. htmlescape(urlpath()) .. " (based on the current permission settings).</body></html>")
end)

--- LogTo can be used for logging to a different file than
--- /var/log/algernon.log, or to the console, if an empty string is given.
--LogTo("/var/log/algernon.log")
