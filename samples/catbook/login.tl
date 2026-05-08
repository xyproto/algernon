-- Log in an existing user
content("application/json")

local fields: {string:string} = formdata()
local username: string = sanhtml(fields["username"] or "")
local password: string = fields["password"] or ""

if username == "" or password == "" then
  print(json({error = "Missing fields"}))
  return
end

if not HasUser(username) then
  print(json({error = "User not found"}))
  return
end

if not IsConfirmed(username) then
  print(json({error = "User not confirmed"}))
  return
end

if not CorrectPassword(username, password) then
  print(json({error = "Wrong password"}))
  return
end

Login(username)
print(json({ok = IsLoggedIn(username)}))
