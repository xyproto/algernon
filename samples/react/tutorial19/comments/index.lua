content("application/javascript")
setheader("Cache-Control", "no-cache")

if method() == "POST" then
  local data = formdata()

  -- To be implemented
  -- See https://github.com/reactjs/react-tutorial/blob/master/server.go

  data={author=data["author"], text=data["text"]}
  log(JSON(data))
else
  print([[
  [
    {"author": "Pete Hunt", "text": "This is one comment"},
    {"author": "Jordan Walke", "text": "This is *another* comment"}
  ]
  ]])
end
