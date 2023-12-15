# Algernon Tutorial

## "Hello"

### Goal

Display `Hello Algernon!` in your browser.

### Steps

#### Create `hello.lua`

Create a `hello.lua` file that looks like this:

```lua
handle("/", function()
  content("text/plain")
  print("Hello Algernon!")
end)
```

* `handle` is a built-in function that can serve an endpoint with a given function.
* `function() [...] end` is an anonymous Lua function.
* `content` is a built-in function for settting the content type / MIME type, such as `text/plain`, `text/html` or `image/png`.
* `print` is a built-in function for outputting text to the HTTP client (typically a browser) that is visiting this page.

#### Start Algernon:

Run `algernon hello.lua`

#### View the result

Visit `http://localhost:3000` in your favorite browser.

## "Simple"

### Goal

Use a directory structure instead of a single Lua file with handlers.

### Steps

#### Create and enter a directory

```bash
mkdir simple
cd simple
```

#### Create an `index.lua` file

Create an `index.lua` file that looks like this:

```lua
print("the light")
```

#### Serve the current directory

Serve the current directory with Algernon:

```bash
algernon .
```

#### View the result

Visit `http://localhost:3000` in a browser and see `"the light"`.

## "Eyes"

### Goal

Serve an image and then use it within a web page.

### Steps

#### Create and enter a directory

```bash
mkdir eyes
cd eyes
```

#### Create `eye.lua`

Create an `eye.lua` file that looks like this:

```lua
content("image/svg+xml")

print [[
<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100">
  <circle cx="50" cy="50" r="40" fill="white" stroke="black" stroke-width="2" />
  <circle cx="38" cy="50" r="20" fill="black" />
  <circle cx="28" cy="42" r="5" fill="white" />
</svg>
]]
```

This outputs an `SVG` image.

#### Create `index.lua`

Create an `index.lua` file that looks like this:

```lua
content("text/html")

print [[
<!doctype html><html><body>
<img src="eye.lua" width="25em">
<img src="eye.lua" width="25em">
]]
```

This is the main/default handler for this directory, and outputs a simple `HTML` page that displays two images.

The browser does not care if the images end with an unusual file extension such as `.lua`, because the content-type / MIME type regulates this anyways.

#### Serve the current directory

```bash
algernon -e .
```

Note that the `-e` flag is for "development mode", where error messages may appear directly in the browser, and pages are not cached.

#### View the result

* Visit `http://localhost:3000/eye.lua` in a browser and observe the result of serving an SVG image.
* Visit `http://localhost:3000/` in a browser and observe the result of serving the HTML document (which uses `eye.lua`, twice).

#### Examine the network traffic

* In ie. Firefox, press `F12` (or `fn`+`F12` on macOS). Select the `Network` tab and then reload the page.
* Click the `200 | GET | localhost:3000 | eye.lua | ...` row.
* On the right hand side, observe that `Content-Type` is `image/svg+xml` and that the `Server` is `Algernon` and a version number.
* There are also various security-related headers that have been set (that can be turned off with the `--no-headers` flag).

## "Style"

### Goal

Style the `HTML` page from the previous steps.

### Steps

#### Create and enter a directory

```bash
mkdir style
cd style
```

#### Create the `eye.svg` image

Create an `eye.svg` file that looks like this:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100">
  <circle cx="50" cy="50" r="40" fill="white" stroke="black" stroke-width="4" />
  <circle cx="38" cy="50" r="20" fill="black" />
  <circle cx="28" cy="42" r="5" fill="white" />
</svg>
```

#### Create the `mouth.svg` image

Create a `mouth.svg` file that looks like this:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" width="100" height="50" viewBox="0 0 100 50">
  <path d="M 10,25 Q 50,50 90,25 Q 50,0 10,25 Z" fill="white" stroke="black" stroke-width="2"/>
</svg>
```

#### Create the HTML document

Create an `index.html` file that looks like this:

```
<!doctype html>
<html>
  <head>
    <title>Face</title>
    <link rel="stylesheet" href="style.css">
  </head>
  <body>
    <div id="face">
      <img src="eye.svg" id="left_eye">
      <img src="eye.svg" id="right_eye">
      <img src="mouth.svg" id="mouth">
    </div>
  </body>
</html>
```

#### Style the document with CSS

Create a `style.css` file that looks like this:

```
#face {
  position: relative;
  top: 4em;
  left: 4em;
  width: 12.5em;
  height: 12.5em;
  background: lightblue;
  border-radius: 1.25em;
}

#left_eye, #right_eye, #mouth {
  position: absolute;
}

#left_eye, #right_eye {
  animation: rotate 16s linear infinite;
}

#left_eye {
  top: 2em;
  left: 0.75em;
  width: 5.625em;
}

#right_eye {
  top: 2em;
  left: 6.125em;
  width: 5.625em;
}

#mouth {
  top: 7em;
  left: 3.125em;
  width: 6.25em;
}

@keyframes rotate {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
```

#### Serve the page

Serve the page with:

```bash
algernon -a -e --no-headers .
```

* Note that serving with `algernon -a -e` and visiting `http://127.0.0.1:3000` instead of using `--no-headers` and visiting `http://localhost:3000` also works.

#### View the result

Visit `http://localhost:3000` to see an image of an unusual looking eye.

#### Auto-refresh

Algernon comes with an auto-refresh feature that inserts a tiny bit of JavaScript into a page, watches files for changes and also serves file changed events as SSE (server sent events).

Here's one way to try it out:

* While still serving the page, and displaying it in the browser, change the numbers in `style.css`, save the file and observe that the page in the browser instantly changes.
* Also try changing numbers in `eye.svg` and `mouth.svg`, save the file and watch the page instantly being updated.

## Next step

What should the next step in this tutorial be? Please submit an issue with a suggestion. Thanks for reading! :)
