-- For a larger project, redirecting after handling the form data is cleaner

-- Set Content-Type and begin HTML
content("text/html; charset=utf-8")
print([[<!doctype html><html><head><title>Lucky</title><style>
body { margin: 4em; background-color: #234; color: #eee; font-family: sans-serif; }
#green { color: #bfb; } hr { border: 1px dotted #345; }
</style></head><body>]])

mprint("# Lucky?")
print("<hr />")

-- Output the result from the form
if method() == "POST" then
  print([[<p>You said you were lucky: <font id="green">]]..formdata()["optionsRadios"]..[[</font></p>]])
end
print("<hr />")

-- Check if user is lucky (pretty high likelyhood)
require 'math'
mprint("#### Are you lucky?")
if math.random(10) == 7 then
  print("Yes, you are lucky.")
else
  print("Not yet.")
end

-- End HTML
print("</body></html>")
