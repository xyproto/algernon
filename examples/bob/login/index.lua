--- Login the user with the username "bob"
Login("bob")
print("bob is now logged in on the server: " .. tostring(IsLoggedIn("bob")))
if Username() ~= "bob" then
  print("Could not store the username in a cookie!")
end
