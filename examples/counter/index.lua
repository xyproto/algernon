content("text/html")

function count()
  local kv = KeyValue("counter")
  local count = kv:get("count")
  local number = ""
  if count == "" then
    number = "1"
  else
    number = tostring(tonumber(count)+1)
  end
  kv:set("count", number)
  return number
end

print[[<!doctype html><html><body style="background: #202020; color: white; font-family: sans-serif; margin: 4em;">]]

local number = count()
print("This is page view #"..number)

print[[</body></html>]]
