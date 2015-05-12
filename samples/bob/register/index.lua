--- Create a new user with a username, password and email address
AddUser("bob", "hunter1", "bob@zombo.com")
print("User bob was created: " .. tostring(HasUser("bob")))
