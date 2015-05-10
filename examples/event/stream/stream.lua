content("text/event-stream")
setheader("cache-control", "no-cache")
setheader("connection", "keep-alive")

function event(message)
  log("EVENT: " .. message)
  print("data: " .. message .. "\n")
  flush()
end

function done()
  print("\n")
  flush()
end

local x = 0
while x < 4 do
  log("LOOP #", x)
  event("style.gcss")
  sleep(1.5)
  event("main.html")
  sleep(1.5)
  event("main.js")
  sleep(1.5)
  event("index.lua")
  sleep(1.5)
  x = x + 1
end

done()
log("DONE STREAMING")
