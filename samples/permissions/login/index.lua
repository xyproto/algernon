--- Login the user with the username "bob"
if not Login("bob") then
  --- Try to figure out why
  if not HasUser("bob") then
    print("Could not login bob, the user is not registered.")
  else
    print("Could not login bob, can cookies be stored?")
  end 
end
print("bob is now logged in on the server: " .. tostring(IsLoggedIn("bob")))
