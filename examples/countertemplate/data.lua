-- Functions where the return value will be made available
-- for templates in the same directory, by using the name of the function.

-- Increase the counter and return the new values as a string
function counter()
  -- Use the "counter" KeyValue and increase the "number" key with 1
  return KeyValue("counter"):inc("number")
end
