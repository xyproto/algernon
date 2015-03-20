-- Set the HTTP Content-Type
content("text/html; charset=utf-8")

-- Beginning of the HTML page
print[[<!doctype html><html><body style="background: #202020; color: white; font-family: sans-serif; margin: 4em;">]]

-- Create or re-use a KeyValue object, using "counter" as the identifier.
-- Then increase or create the value of a key named "number".
-- Then print out this increased value.
print("This is page view #" .. KeyValue("counter"):inc("number"))

-- End of the HTML page
print[[</body></html>]]
