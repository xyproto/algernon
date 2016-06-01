content("text/html")

-- output the given code within a prettyprint class
function prettyprint(code)
  print([[<div style="width: 80%;"><pre class="prettyprint">]] .. code .. [[</pre></div>]])
end

-- format the title as relatively large letters
function title(text)
  mprint("# " .. text)
end

-- format the title as big letters
function bigtitle(text)
  print([[<div style="font-size: 3em; font-weight: bold;">]] .. text .. [[</div>]])
end

--  horizontal space. There are better ways, but this works too.
function some_space()
  print[[<br><br>]]
end

-- the beginning of the HTML document
function head()
  print[[<!doctype html>
<html>
  <head>
    <title>prettify</title>
    <script src="//cdn.rawgit.com/google/code-prettify/master/loader/run_prettify.js?skin=sunburst"></script>
  </head>
  <body style="background: gray; color: black; font-family: sans-serif; margin: 5em;">]]
end

-- the end of the HTML document
function tail()
  print[[</body></html>]]
end

-- see if the file exists
function file_exists(file)
  local f = io.open(file, "rb")
  if f then f:close() end
  return f ~= nil
end


--- The page contents ---

head()
bigtitle("Sample Code")
some_space()

local gofile = scriptdir("main.go")
if file_exists(gofile) then
  title("Go")
  local f = io.open(gofile, "rb")
  prettyprint(f:read("*a"))
  f:close()
  some_space()
end

local luafile = scriptdir("main.lua")
if file_exists(luafile) then
  title("Lua")
  local f = io.open(luafile, "rb")
  prettyprint(f:read("*a"))
  f:close()
  some_space()
end

tail()
