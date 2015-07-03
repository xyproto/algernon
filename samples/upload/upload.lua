formID = "fileToUpload"

-- Output the beginning of the HTML document. Also set the content-type.
function head()
  content("text/html; charset=utf-8")
  print [[<!doctype html><html><head><title>upload</title>
  <style>body { margin: 3em; font-family: courier; }</style></head><body>]]
end

-- Output the end of the HTML document
function tail()
  print [[</body></html>]]
end

-- Handle and save the uploaded file
function handleUpload()
  u, err = UploadedFile(formID)
  if err ~= "" then
    print([[<font style="color: red">]] .. err .. [[</font>]])
    error(413) -- Request entity too large
    return
  end

  print("Filename: " .. u:filename() .. [[<br>]])
  print("Size: " .. u:size() .. [[<br>]])
  print("Content type: " .. u:mimetype() .. [[<br>]])
  print("Saved: " .. tostring(u:savein("incoming")) .. [[<br>]])
end

function main()
  head()
  handleUpload()
  tail()
end

main()
