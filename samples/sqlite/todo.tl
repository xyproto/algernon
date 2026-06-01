-- Type declarations for the SQLiteFile API
local record SQLiteFileHandle
  exec:  function(SQLiteFileHandle, string, {any}|nil): number, boolean
  query: function(SQLiteFileHandle, string, {any}|nil): {{string: any}}
end

-- Algernon globals used in this script
global content:    function(string)
global method:     function(): string
global formdata:   function(): {string: string}
global json:       function(any): string
global SQLiteFile: function(string|nil): SQLiteFileHandle

content("application/json")

local db = SQLiteFile("todo.db")
db:exec([[CREATE TABLE IF NOT EXISTS todos (
  id   INTEGER PRIMARY KEY AUTOINCREMENT,
  text TEXT    NOT NULL,
  done INTEGER NOT NULL DEFAULT 0
)]])

local function all_todo_items(): string
  local rows = db:query("SELECT id, text, done FROM todos ORDER BY id")
  if #rows == 0 then
    return "[]"
  end
  return json(rows)
end

local m = method()

if m == "GET" then
  print(all_todo_items())
elseif m == "POST" then
  local d = formdata()
  if d.action == "add" and d.text ~= nil and #d.text > 0 then
    db:exec("INSERT INTO todos (text) VALUES (?)", {d.text})
  elseif d.action == "delete" and d.id ~= nil then
    db:exec("DELETE FROM todos WHERE id = ?", {tonumber(d.id)})
  elseif d.action == "toggle" and d.id ~= nil then
    db:exec("UPDATE todos SET done = 1 - done WHERE id = ?", {tonumber(d.id)})
  end
  print(all_todo_items())
else
  print("[]")
end
