package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
)

const (
	// Code highlight
	pre_highlight  = "<font style='color: red !important'>"
	post_highlight = "</font>"

	// Highlight theme. See https://highlightjs.org/ for more themes.
	highlight_theme = "github"
)

// Shorthand function for reading a file
var read = ioutil.ReadFile

// Write the contents of a ResponseRecorder to a ResponseWriter
func writeRecorder(w http.ResponseWriter, recorder *httptest.ResponseRecorder) {
	w.WriteHeader(recorder.Code)
	for key, values := range recorder.HeaderMap {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	recorder.Body.WriteTo(w)
}

// Return an informative error page to the user
// Takes a ResponseWriter, title (can be empty), filename, filebytes, errormessage and
// programming/scripting/template language (i.e. "lua". Can be empty).
func prettyError(w http.ResponseWriter, filename string, filebytes []byte, errormessage, lang string) {

	// HTTP status
	//w.WriteHeader(http.StatusInternalServerError)
	w.WriteHeader(http.StatusOK)

	// HTTP content type
	w.Header().Add("Content-Type", "text/html; encoding=UTF-8")

	// The line that the error refers to, for the case of Lua
	linenr := -1

	// If there is code to be displayed
	var code string
	if len(filebytes) > 0 {
		if lang == "lua" {
			// If the first line of the error message has two colons, see if the second field is a number
			fields := strings.SplitN(errormessage, ":", 3)
			if len(fields) > 2 {
				// Extract the line number from the error message, if possible
				numberfield := fields[1]
				if strings.Contains(numberfield, "(") {
					numberfield = strings.Split(numberfield, "(")[0]
				}
				linenr, err := strconv.Atoi(numberfield)
				// Subtract one to make it a slice index instead of human-friendly line number
				linenr--
				// Set linenumber to -1 if the conversion failed
				if err != nil {
					linenr = -1
				}
			}
		}

		// Escape any HTML in the code, so that the pretty printer is not confused
		filebytes = bytes.Replace(filebytes, []byte("<"), []byte("&lt;"), -1)

		// Modify the line that is to be highlighted
		bytelines := bytes.Split(filebytes, []byte("\n"))
		if (linenr >= 0) && (linenr < len(bytelines)) {
			bytelines[linenr] = []byte(pre_highlight + string(bytelines[linenr]) + post_highlight)
		}

		// Build a string from the bytelines slice
		code = string(bytes.Join(bytelines, []byte("\n")))
	}

	// Set an appropriate title
	title := "Error"
	if lang == "gcss" {
		title = "GCSS Error"
	} else if lang != "" {
		title = strings.Title(lang) + " Error"
	}

	// Set the highlight class
	langclass := lang

	// Turn off highlighting for some languages
	switch lang {
	case "", "amber", "gcss":
		langclass = "nohighlight"
	}

	// Highlighting for the error message
	errorclass := "json" // "nohighlight"

	// Inform the user of the error
	fmt.Fprint(w, `<!doctype html>
<html>
  <head>
    <title>`+title+`</title>
	<link href='//fonts.googleapis.com/css?family=Lato:300' rel='stylesheet' type='text/css'>
	<style>
      body {
	    background-color: #f0f0f0;
		color: #0b0b0b;
		font-family: 'Lato', sans-serif;
		font-weight: 300;
		margin: 3.5em;
		font-size: 1.3em;
      }
	  h1 {
	    color: #101010;
	  }
	  div {
	    margin-bottom: 35pt;
	  }
	  #right {
        text-align:right;
	  }
	</style>
    <link rel="stylesheet" href="//cdnjs.cloudflare.com/ajax/libs/highlight.js/8.5/styles/`+highlight_theme+`.min.css">
    <script src="//cdnjs.cloudflare.com/ajax/libs/highlight.js/8.5/highlight.min.js"></script>	
	<script>hljs.initHighlightingOnLoad();</script>
  </head>
  <body>
    <div style="font-size: 3em; font-weight: bold;">`+title+`</div>
    Contents of `+filename+`:
    <div>
	  <pre><code class="`+langclass+`">`+code+`</code></pre>
	</div>
    Error message:
    <div>
	  <pre><code style="color: #A00000;" class="`+errorclass+`">`+strings.TrimSpace(errormessage)+`</code></pre>
	</div>
	<div id="right">
	`+version_string+`
	</div>
  </body>
</html>`)
}
