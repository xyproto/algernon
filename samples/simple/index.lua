-- Optional, for HTML
content("text/html")

-- One way of converting Markdown to HTML and sending the result to the client
mprint("# Simple")

-- Another way of converting Markdown to HTML and sending the result to the client
print(markdown("## Example"))

-- Regular text
print("This is a simple example.")
