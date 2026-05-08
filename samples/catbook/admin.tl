-- Admin backstage: server info and user list
content("application/json")

local username: string = Username()
if username == "" or not IsAdmin(username) then
  print(json({error = "Access denied"}))
  return
end

print(json({
  ok = true,
  server = ServerInfo(),
  ver = version(),
  users = AllUsernames()
}))
