-- Set the headers
content("application/javascript")
setheader("Cache-Control", "no-cache")

-- Handle requests
if method() == "POST" then
  -- Add the form data as JSON to the "comments" list in the database (NOTE: unsanitized)
  List("comments"):add(json(formdata()))
else
  -- Retrieve the "comments" list from the database as a JSON list
  print(List("comments"):json())
end
