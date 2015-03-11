--- Logout the user with the username "bob"
Logout("bob")
print("bob is now logged out: "..tostring(not IsLoggedIn("bob")))
