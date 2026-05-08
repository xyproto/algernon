-- Post a chat message or list recent messages
local messages = List("catbook:messages")

if methodPOST() then
  local fields: {string:string} = formdata()
  local username: string = Username()
  local text: string = maxlen(sanhtml(fields["message"] or ""), 500)

  if username == "" or not IsLoggedIn(username) then
    print(json({error = "Not logged in"}))
    return
  end
  if text == "" then
    print(json({error = "Empty message"}))
    return
  end

  local entry: string = json({user = username, text = text, time = tostring(unixnano())})
  messages:add(entry)
  print(json({ok = true}))
else
  print(messages:jsonlast(50))
end
