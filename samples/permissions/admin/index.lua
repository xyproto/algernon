--- List all usernames, if the user is logged in with administrator rights
print("Super secret information that only logged in administrators must see!\n")
print("List of all users: " .. table.concat(AllUsernames(), ", "))
