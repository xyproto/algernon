print("creating a new set")
-- Create an overview over which users are active
local s = NewSet("active")
-- Add active users
s:add("bob")
s:add("alice")
s:add("kenny")
-- Print information about the set
print(s)
-- Print the contents of the set, in two different ways
print(tostring(s))
print(table.concat(s:getall(), ", "))
-- Remove two users
s:del("bob")
s:del("alice")
print("has kenny?", s:has("kenny"))
print("has bob?", s:has("bob"))
-- Remove the last user
s:del("kenny")
-- Remove the set itself
print("removed the set", s:remove())
