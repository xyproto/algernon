-- From the example for https://github.com/xyproto/permissions2

print("Has user bob: " .. tostring(HasUser("bob")))
print("Logged in on server: " .. tostring(IsLoggedIn("bob")))
print("Is confirmed: " .. tostring(IsConfirmed("bob")))
print("Username stored in cookie (or blank): " .. Username())
print("First character of cookie secret: " .. string.sub(CookieSecret(), 1, 1))
print("Current user is logged in, has a valid cookie and *user rights*: " .. tostring(UserRights()))
print("Current user is logged in, has a valid cookie and *admin rights*: " .. tostring(AdminRights()))
print("Try: /register, /confirm, /remove, /login, /logout, /clear, /data, /makeadmin and /admin")

if urlpath() == "/permissions/" then
  -- If running with the welcome.sh script:
  AddAdminPrefix("/permissions/admin")
  AddUserPrefix("/permissions/data")
elseif urlpath() ~= "/" then
  print()
  print[[NOTE: The current URL path is not "/"! For the default URL permissions to work, Algernon must either be run from this directory, or the URL prefixes must be configured correctly. Try running `welcome.sh` from the project root.]]
end
