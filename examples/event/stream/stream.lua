-- Stream Server-Sent Events (SSE)
content("text/event-stream;charset=utf-8")
setheader("Cache-Control", "no-cache")
setheader("Connection", "keep-alive")
setheader("Access-Control-Allow-Origin", "*")

function event(message)
  log("EVENT: " .. message)
  print("data: " .. message .. "\n")
end

function finish()
  log("Done streaming events")
  print("\n")
end

-- Must take < 10 seconds, before the request times out

event("one event")
sleep(2)
event("a second event")
sleep(2)
event("a third event")
sleep(2)
event("yet another event")
sleep(2)
event("eventorama!")
sleep(2)
finish()
