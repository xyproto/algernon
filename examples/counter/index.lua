content("text/html")
print[[<!doctype html><html><body style="background: #202020; color: white; font-family: sans-serif; margin: 4em;">]]

print("This is page view #" .. KeyValue("counter"):inc("number"))

print[[</body></html>]]
