# :large_blue_diamond: SimpleForm

[![Build Status](https://travis-ci.com/xyproto/simpleform.svg?branch=main)](https://travis-ci.com/xyproto/simpleform) [![GoDoc](https://godoc.org/github.com/xyproto/simpleform?status.svg)](http://godoc.org/github.com/xyproto/simpleform) [![Go Report Card](https://goreportcard.com/badge/github.com/xyproto/simpleform)](https://goreportcard.com/report/github.com/xyproto/simpleform)

SimpleForm is a language for constructing HTML forms out of very little text.

Here is an example login form:


```ruby
Login

Welcome dear user!

Username: {{ username }}
Password: {{ password }}

[Login](/login)
```

* The first line is recognized as the title, and is used both for the title (like this: `<title>Login</title>` and as a title in the body, like this: `<h2>Login</h2>`.
* The lines with `{{` and `}}` are recognized as single line input fields, where the label is the word before `:`.
* The `[Login](/login)` is recognized as a button that can submit the contents of the form as a POST request to `/login`.
* The syntax is inspired by `Jinja2` and `Markdown`.

Here's the generated HTML from the login form above:

```html
<!doctype html>
<html lang="en">
  <head>
    <title>Login</title>
  </head>
  <body>
    <h2>Login</h2>
    <p>Welcome dear user!</p>
    <form method="POST">
      <label for="username">Username:</label><input type="text" id="username" name="username"><br><br>
      <label for="password">Password:</label><input type="password" id="password" name="password"><br><br>
      <input type="submit" formaction="/login" value="Login"><br><br>
    </form>
  </body>
</html>
```

Unstyled, it looks like this:

![loginform](img/loginform.png)

When styled with [MVP.CSS](https://andybrewer.github.io/mvp/), this is how it looks:

![loginform_styled](img/loginform_styled.png)

## Features and limitations

* If the input ID starts with `password` or `pwd`, then the input type `"password"` is used.
* Multiple buttons can be provided on a single line.
* All text that is not recognized as either the title or as form elements is combined and returned in a `<p>` tag.
* If `[[` and `]]` are used instead of `{{` and `}}`, then a multi-line text input box is created instead.

# TODO

* Formal spec (or at least a PDF describing the language).
* Radio button support, with options separated by `|`.
* Support for required fields, by using the exclamation mark.
* Spport all available form elements, while keeping complexity low.

## General Info

* Version: 0.2.0
* License: MIT
* Author: Alexander F. Rødseth &lt;xyproto@archlinux.org&gt;
