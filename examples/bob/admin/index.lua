--- List all usernames, if the user is logged in with administrator rights
print("super secret information that only logged in administrators must see!\n")
print("list of all users: "..table.concat(AllUsernames(), ", "))
