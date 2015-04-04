-- TODO: Find a way to restructure this in a better way.
--       For instance, global lua variables could be accessible to both
--       Amber and GCSS files. Or both could be able to execute Lua inline.

-- Set the HTTP Content-Type
content("text/html; charset=utf-8")

-- Function for reading the contents of a file
function readfile(filename)
  local f = io.open(scriptdir(filename), "rb")
  data = f:read("*a")
  f:close()
  return data
end

-- Print the amber template, with a variable set
aprint([[$counter="]] .. KeyValue("visitor"):inc("counter") .. [["]] .. "\n", readfile("template.amber"))
