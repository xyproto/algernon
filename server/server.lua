--
-- For use with Algernon / Lua
-- 
-- Project page: https://github.com/xyproto/algernon
-- Web page: http://algernon.roboticoverlords.org/
--

-- Set the headers
content("application/javascript")
setheader("Cache-Control", "no-cache")

-- Use a JSON file for the comments
comments = JSONDB("comments.json", {{"author", "text"}})

-- Handle requests
if method() == "POST" then
  -- Open the file, read the contents, add the data, write the contents, close the file
  comments:add(comments:toJSON(formdata()))
else
  -- Open the file, read the contents, close the file, output the contents
  print(comments:getall())
end
