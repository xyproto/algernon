--
-- Functions in this file are available to call from 
-- the amber.index file in the same directory.
--
-- The functions may take any number of strings and
-- may return one value that will be converted to string.
--

title = "Counter Example"

-- Count from 0 and up. Return the current value.
function counter()
  -- Use the "counter" KeyValue and increase the "number" key with 1
  return KeyValue("counter"):inc("number")
end


