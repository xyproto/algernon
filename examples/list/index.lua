print("creating a new list")
-- Create a list of fruit
local l = NewList("fruit")
-- Add elements
l:add("apple")
l:add("banana")
l:add("kiwi")
-- Print information about the list
print(l)
-- Print the contents of the list, in two different ways
print(tostring(l))
print(table.concat(l:getall(), ", "))
-- Print the last 2 items
print(table.concat(l:getlastn(2), ", "))
-- Print the last item
print(l:getlast())
-- Remove the list
l:remove()
