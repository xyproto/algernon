-- Finish a WebAuthn login ceremony (username in query string, body is WebAuthn assertion)
local username: string = sanhtml(urldata()["username"] or "")
content("application/json")
if username == "" then
  print(json({error = "Missing username"}))
  return
end
if not WebAuthnFinishLogin(username) then
  print(json({error = "Passkey verification failed"}))
  return
end
Login(username)
print(json({ok = IsLoggedIn(username)}))
