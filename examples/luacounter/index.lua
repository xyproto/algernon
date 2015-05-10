-- Set the HTTP Content-Type
content("text/html; charset=utf-8")

-- What happens?
flush()

-- Beginning of page
print[[<!doctype html><html><body style="background: #202020; color: white; font-family: sans-serif; margin: 4em;">]]

  -- Use the "pageview" KeyValue and increase the "counter" value with 1
print("This is page view #" .. KeyValue("pageview"):inc("counter"))

-- End of page
print[[</body></html>]]
