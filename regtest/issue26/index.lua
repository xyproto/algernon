local t = {something="hello", somelist={ "A", "B", "C" }, anotherlist={1,2,"z"}}
--for k, v in pairs(t) do
--  print("key: " .. k .. " value: ")
--  pprint(v)
--end
--print("calling serve2 with table t")
serve2("list.po2", t)
