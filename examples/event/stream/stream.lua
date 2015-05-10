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
event("--- start ---")
sleep(4.2)
event("style.gcss")
sleep(4.2)
event("main.html")
sleep(4.2)
event("main.js")
sleep(4.2)
event("index.lua")
sleep(4.2)
event("--- end ---")
sleep(4.2)

done()
log("DONE STREAMING")
