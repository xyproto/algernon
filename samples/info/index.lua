content("text/html")
print[[<!doctype html><html><head><title>Info</title></head><body style="background: #202020; color: white; font-family: sans-serif; margin: 4em;">]]
mprint("# " .. version())
mprint("#### Headers from your browser:")
mprint("* HTTP method: " .. method())
mprint("* URL path: " .. urlpath())
--- The HTTP body will only be read once, since it's streamed
local body = body()
if body ~= "" then
  mprint("* Request body: " .. body)
end
mprint("* User agent:<br>" .. header("User-Agent"))
mprint("* Available compression algorithms: " .. header("Accept-Encoding"))
print[[</body></html>]]
