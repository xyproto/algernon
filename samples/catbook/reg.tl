-- Register a new user
content("application/json")

local fields: {string:string} = formdata()
local username: string = sanhtml(fields["username"] or "")
local password: string = fields["password"] or ""
local email: string = sanhtml(fields["email"] or "")

if username == "" or password == "" or email == "" then
  print(json({error = "Missing fields"}))
  return
end

if HasUser(username) then
  print(json({error = "Username is taken"}))
  return
end

AddUser(username, password, email)
MarkConfirmed(username)
if username == "admin" then
  SetAdminStatus(username)
end
print(json({ok = true}))
