-- Finish a WebAuthn registration ceremony (must be logged in)
content("application/json")
local username: string = Username()
if username == "" or not IsLoggedIn(username) then
  print(json({error = "Not logged in"}))
  return
end
print(json({ok = WebAuthnFinishRegister(username)}))
