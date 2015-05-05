-- Values and functions that are made available to other
-- lua scripts and amber templates in this directory.

title = "JSX example"
message = "jsx"
tag = "h2"

function asciiArtUpper(text)
  return "-=[" .. string.upper(text) .. "]=-"
end
