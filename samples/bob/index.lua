-- From the example for https://github.com/xyproto/permissions2
print("Has user bob: " .. tostring(HasUser("bob")))
print("Logged in on server: " .. tostring(IsLoggedIn("bob")))
print("Is confirmed: " .. tostring(IsConfirmed("bob")))
print("Username stored in cookie (or blank): " .. Username())
print("Current user is logged in, has a valid cookie and *user rights*: " .. tostring(UserRights()))
print("Current user is logged in, has a valid cookie and *admin rights*: " .. tostring(AdminRights()))
print("Try: /register, /confirm, /remove, /login, /logout, /data, /makeadmin and /admin")
