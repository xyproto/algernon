-- Set the content-type
content("application/json; charset=utf-8")

-- Only strings (or only integer values) are supported, for now
data = {greeting="Hello", location="there"}

-- Output the JSON document
print(JSON(data))
