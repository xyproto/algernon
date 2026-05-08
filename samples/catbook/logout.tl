-- Log out the current user
content("application/json")

local username: string = Username()
if username ~= "" then
  Logout(username)
end
print(json({ok = not IsLoggedIn(username)}))
