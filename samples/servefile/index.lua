--- Handling the request for this directory

-- Serve the right page
if method() == "GET" then
  serve("GET.amber")
elseif method() == "POST" then
  serve("POST.amber")
end

-- Also possible
-- serve(method() .. ".amber")
