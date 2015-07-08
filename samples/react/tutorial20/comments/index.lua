content("application/javascript")
setheader("Cache-Control", "no-cache")

-- Use a Redis list for the comments
comments = List("comments")

if method() == "POST" then
  -- Add the form data to the comment list, as JSON
  comments:add(toJSON(formdata()))
else
  -- Combine all the JSON comments to a JSON document
  print("["..table.concat(comments:getall(), ",").."]")
end
