content("text/event-stream")
setheader("cache-control", "no-cache")
setheader("connection", "keep-alive")

function event(message)
  log("EVENT: " .. message)
  print("data: " .. message .. "\n")
  print("\n")
end

while true do
  event("style.gcss")
  sleep(3.5)
  event("main.html")
  sleep(3.5)
  event("main.js")
  sleep(3.5)
  event("index.lua")
  sleep(3.5)
end
