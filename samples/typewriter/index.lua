-- Set the content type to text/html
content("text/html")

-- Print the HTML header and some inline CSS
print([[
<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <title>Typewriter Effect</title>
  <style>
    body {
      background: #222;
      color: #f8f8f2;
      font-family: "Courier New", Courier, monospace;
      font-weight: bold;
      margin: 0;
      padding: 20px;
    }
    .typewriter {
      background: #444;
      border: 1px solid #666;
      padding: 20px;
      margin: 20px auto;
      width: 80%;
      box-shadow: 0 0 10px rgba(0,0,0,0.5);
      font-size: 18px;
      line-height: 1.5;
    }
  </style>
</head>
<body>
<div class="typewriter">
]])

-- Parameters:
--   text: The string to display.
--   charDelay: Delay (in seconds) between each character.
--   sentenceDelay: Extra delay after punctuation (., !, ?).
function typewriter(text, charDelay, sentenceDelay)
  for i = 1, #text do
    local c = text:sub(i, i)
    print_nonl(c)        -- Print character without a newline.
    could_flush = flush()         -- Flush the output so the character appears immediately.
    if not could_flush then
      return true
    end
    sleep(charDelay)
    if c == '.' or c == '!' or c == '?' then
      sleep(sentenceDelay)
    end
    if c == '\n' then
      print("<br>")
    end
  end
  return false
end

-- Example usage:
connection_closed = typewriter("Hello, world.\n\nThis is a server-side typewriter effect test that uses flush() and print_nonl().\n\nMay the 4th be with you!\n", 0.05, 0.5)

if not connection_closed then
  -- Close the HTML container
  print([[
  </div>
  </body>
  </html>
  ]])
end
