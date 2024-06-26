-- Handle and save the uploaded file
function main()
  content("text/html; charset=utf-8")

  print [[<!doctype html><html><head><title>upload</title>
  <style>body { margin: 3em; font-family: courier; }</style></head><body>]]

  -- Receive the file
  u, err = UploadedFile("fileToUpload")
  if err ~= "" then
    print([[<font style="color: red">]] .. err .. [[</font>]])
    error(413) -- Request entity too large
    return
  end

  -- Output various info about the uploaded file
  print("Filename: " .. u:filename() .. [[<br>]])
  print("Size: " .. u:size() .. [[ bytes<br>]])
  print("Content type: " .. u:mimetype() .. [[<br>]])

  local base64encoded = u:base64()

  print([[<br><hr><br>]])

  -- Display the uploaded image
  print([[<img src="data:]] .. u:mimetype() .. [[;base64, ]] .. base64encoded .. [[" /><br><br>]])

  -- Describe the uploaded image using Ollama
  local description = describeImage(base64encoded)
  print(description .. [[<br>]])

  print [[</body></html>]]
end

main()
