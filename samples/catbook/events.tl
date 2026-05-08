-- Create an event or list upcoming events
local events = List("catbook:events")

if methodPOST() then
  local fields: {string:string} = formdata()
  local username: string = Username()
  local title: string = maxlen(sanhtml(fields["title"] or ""), 200)
  local when: string = maxlen(sanhtml(fields["when"] or ""), 100)
  local where: string = maxlen(sanhtml(fields["where"] or ""), 200)

  if username == "" or not IsLoggedIn(username) then
    print(json({error = "Not logged in"}))
    return
  end
  if title == "" then
    print(json({error = "Missing title"}))
    return
  end

  local entry: string = json({
    user = username,
    title = title,
    when = when,
    where = where,
    time = tostring(unixnano())
  })
  events:add(entry)
  print(json({ok = true}))
else
  print(events:jsonlast(50))
end
