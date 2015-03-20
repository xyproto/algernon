--- Remove bob
RemoveUser("bob")
print("User bob was removed: " .. tostring(not HasUser("bob")))
