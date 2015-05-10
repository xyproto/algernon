content("text/event-stream")
setheader("cache-control", "no-cache")
setheader("connection", "keep-alive")

function event(message)
  log("EVENT: " .. message)
  print("data: " .. message .. "\n")
  print("\n")
  flush()
end

while true do
  event("style.gcss")
  sleep(1.5)
  event("main.html")
  sleep(1.5)
  event("main.js")
  sleep(1.5)
  event("index.lua")
  sleep(1.5)
  log("LOOPING!")
end

error("IMPOSSIBRU")
