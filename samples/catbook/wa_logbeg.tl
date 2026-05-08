-- Begin a WebAuthn login ceremony
local fields: {string:string} = formdata()
local username: string = sanhtml(fields["username"] or "")
if username == "" then
  content("application/json")
  print(json({error = "Missing username"}))
  return
end
if not WebAuthnBeginLogin(username) then
  content("application/json")
  print(json({error = "No passkey registered"}))
end
