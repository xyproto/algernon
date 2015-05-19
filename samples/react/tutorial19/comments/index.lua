content("application/javascript")
setheader("Cache-Control", "no-cache")

-- Make the comment List in Redis available for use
comments = List("comments")

if method() == "POST" then
  -- Add the form data to the comment list, as JSON
  comments:add(JSON(formdata()))
else
  -- Combine all the JSON comments to a JSON document
  print("["..table.concat(comments:getall(), ",").."]")
end
