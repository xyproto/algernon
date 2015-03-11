content("text/html")
print[[<!doctype html><html><body style="background: #202020; color: white; font-family: sans-serif; margin: 4em;">]]
mprint("# "..version())
mprint("* HTTP method: "..method())
mprint("* URL path: "..urlpath())
--- The HTTP body will only be read once, since it's streamed
body = body()
if body ~= "" then
  mprint("* Request body: "..body)
end
mprint("* User agent:<br>"..header("User-Agent"))
print[[</body></html>]]
