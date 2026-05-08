-- Return the currently logged-in user and profile, if any
content("application/json")

local username: string = Username()
if username ~= "" and IsLoggedIn(username) then
  local profiles = HashMap("catbook:profiles")
  local raw: string = profiles:get(username, "profile")
  if raw ~= "" then
    print('{"username":' .. json(username) .. ',"profile":' .. raw .. '}')
  else
    print(json({username = username}))
  end
else
  print("{}")
end
