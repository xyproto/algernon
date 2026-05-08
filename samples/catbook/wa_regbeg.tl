-- Begin a WebAuthn registration ceremony (must be logged in)
content("application/json")
local username: string = Username()
if username == "" or not IsLoggedIn(username) then
  print(json({error = "Not logged in"}))
  return
end
if not WebAuthnBeginRegister(username) then
  print(json({error = "WebAuthn registration failed"}))
end
