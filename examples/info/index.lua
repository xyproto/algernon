content("text/html")
print([[<html><body style="background: #202020; color: white; font-family: sans-serif; margin: 4em;">]])
mprint(version().."\n====")
mprint("* HTTP method: "..method())
mprint("* URL path: "..urlpath())
if body() ~= "" then
  mprint("* Request body: "..body())
end
mprint("* User agent:<br>"..header("User-Agent"))
print([[</body></html>]])
