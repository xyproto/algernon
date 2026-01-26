-- Set the content type to text/html
content("text/html")

text_content1 = [[
Hello, world.

This is a server-side typewriter effect test that uses flush() and print_nonl().

It also needs a longer timeout. For a timeout of 60 seconds, the --timeout=60 flag can be used.

]]

text_content2 = [[
Here are the 8 rules of writing, by Kurt Vonnegut:

1. Use the time of a total stranger in such a way that he or she will not feel the time was wasted.
2. Give the reader at least one character he or she can root for.
3. Every character should want something, even if it is only a glass of water.
4. Every sentence must do one of two things—reveal character or advance the action.
5. Start as close to the end as possible.
6. Be a sadist. No matter how sweet and innocent your leading characters, make awful things happen to them—in order that the reader may see what they are made of.
7. Write to please just one person. If you open a window and make love to the world, so to speak, your story will get pneumonia.
8. Give your readers as much information as possible as soon as possible. To heck with suspense. Readers should have such complete understanding of what is going on, where and why, that they could finish the story themselves, should cockroaches eat the last few pages.
]]


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
typewriter(text_content1, 0.05, 0.5)
connection_closed = typewriter(text_content2, 0.02, 0.2)

if not connection_closed then
  -- Close the HTML container
  print([[
  </div>
  </body>
  </html>
  ]])
end
