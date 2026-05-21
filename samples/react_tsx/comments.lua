content("application/json")

local db = SQLiteFile("comments.db")

if method() == "POST" then
  local d = formdata()
  db:add("comments", {author = d.author, text = d.text})
end

local docs = db:docs("comments")
print(#docs == 0 and "[]" or json(docs))
