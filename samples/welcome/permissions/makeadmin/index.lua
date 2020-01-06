--- Make "bob" an administrator
SetAdminStatus("bob")
print("bob is now administrator: " .. tostring(IsAdmin("bob")))
