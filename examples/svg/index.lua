--- Serve a SVG image
content("image/svg+xml")
print([[<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE svg PUBLIC "-//W3C//DTD SVG 1.1 Tiny//EN" "http://www.w3.org/Graphics/SVG/1.1/DTD/svg11-tiny.dtd">
<svg xmlns="http://www.w3.org/2000/svg" version="1.1" baseProfile="tiny" viewBox="0 0 128 64">
  <desc>Hello SVG</desc>
  <circle cx="30" cy="10" r="5" fill="red" />
  <circle cx="110" cy="30" r="2" fill="green" />
  <circle cx="80" cy="40" r="7" fill="blue" />
</svg>]])
