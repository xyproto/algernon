content("text/html")
print[[<!doctype html><html><head><title>Info</title></head><body style="background: #202020; color: white; font-family: sans-serif; margin: 4em;">]]
mprint("# " .. version())
mprint("#### Request information:")
mprint("* HTTP method: " .. method())
mprint("* URL path: " .. urlpath())
--- The HTTP body will only be read once, since it's streamed
local body = body()
if body ~= "" then
  mprint("* Request body: " .. body)
end
mprint("#### Header table:<br>")
for k, v in pairs(headers()) do
  mprint("* " .. k .. " = " .. v)
end
print[[</body></html>]]
