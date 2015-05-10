content("text/event-stream;charset=utf-8")
setheader("cache-control", "no-cache")
setheader("connection", "keep-alive")

function event(message)
  log("EVENT: " .. message)
  print("data: " .. message .. "\n")
end

function done()
  log("DONE")
  print("\n")
end

-- Must take < 10 seconds
local x = 0
while x < 4 do
  log("LOOP #", x)
  event("style.gcss")
  sleep(0.2)
  event("main.html")
  sleep(0.2)
  event("main.js")
  sleep(0.2)
  event("index.lua")
  sleep(0.2)
  x = x + 1
end

done()
log("DONE STREAMING")
