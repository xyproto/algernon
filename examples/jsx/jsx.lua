-- Set the Content-Type. This file will be served as JavaScript.
content("text/javascript")

-- Passing the message in the URL, as a demonstration.
-- It could just as well have been defined in data.lua.
local formatted = asciiArtUpper(formdata()["msg"])

-- Output JavaScript
jprint([[React.render(
  <]] .. tag .. ">" .. formatted .. "</" .. tag .. [[>,
  document.getElementById('example')
);]])
