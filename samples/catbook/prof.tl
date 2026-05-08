-- Get or update a cat profile
content("application/json")

local profiles = HashMap("catbook:profiles")

if methodPOST() then
  local fields: {string:string} = formdata()
  local username: string = Username()
  local breed: string = maxlen(sanhtml(fields["breed"] or ""), 100)
  local bio: string = maxlen(sanhtml(fields["bio"] or ""), 500)
  local emoji: string = maxlen(sanhtml(fields["emoji"] or "🐱"), 16)

  if username == "" or not IsLoggedIn(username) then
    print(json({error = "Not logged in"}))
    return
  end

  local data: string = json({breed = breed, bio = bio, emoji = emoji})
  profiles:set(username, "profile", data)
  print(json({ok = true}))
else
  local who: string = maxlen(sanhtml(urldata()["user"] or ""), 100)
  if who == "" then
    who = Username()
  end
  if who == "" then
    print(json({user = "", breed = "", bio = "", emoji = "🐱"}))
    return
  end
  local raw: string = profiles:get(who, "profile")
  if raw == "" then
    print(json({user = who, breed = "", bio = "", emoji = "🐱"}))
  else
    -- Wrap raw profile data with the username
    print('{"user":' .. json(who) .. ',' .. raw:sub(2))
  end
end
