print("creating a new hash map")
-- Create a place to store users
local h = NewHashMap("users")
-- Add active users
h:set("bob", "password", "hunter1")
h:set("bob", "email", "bob@zombo.com")
h:set("alice", "password", "123")
h:set("alice", "email", "alice@zombo.com")
-- Print information about the hash map
print(h)
-- Print the contents of the hash map, in two different ways
print(tostring(h))
print(table.concat(h:getall(), ", "))
-- Remove the password
print("bob's password:", h:get("bob", "password"))
h:delkey("bob", "password")
print("Password was removed:", not h:has("bob", "password"))
print("bob's password:", h:get("bob", "password"))
-- Is bob there?
print("has bob?", h:exists("bob"))
-- Remove bob
h:del("bob")
-- Is bob there?
print("has bob?", h:exists("bob"))
-- Remove the hash itself
print("removed the hash map", h:remove())
print("does alice still exist?", h:exists("alice"))
