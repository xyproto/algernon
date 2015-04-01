package main

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
)

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
// Takes a ResponseWriter, filename that will be read, error message and
// a string describing which programming language the file is in, ie. "lua".
func prettyError(w http.ResponseWriter, filename, errormessage, lang string) {

	w.WriteHeader(http.StatusInternalServerError)
	//w.WriteHeader(http.StatusOK)

	w.Header().Add("Content-Type", "text/html; encoding=UTF-8")

	// Read the file contents
	var code string
	filebytes, err := ioutil.ReadFile(filename)
	if err != nil {
		code = err.Error()
	} else {
		// Escape the HTML so that the pretty printer is not confused
		code = strings.Replace(string(filebytes), "<", "&lt;", -1)
	}

	// Find an appropriate title
	title := []string{"Preposterous", "Inconceivable", "Unthinkable", "Defies all reason"}[rand.Intn(4)]

	// Inform the user of the error
	fmt.Fprint(w, `<!doctype html>
<html>
  <head>
    <title>Error in `+filename+`</title>
    <script src="//google-code-prettify.googlecode.com/svn/loader/run_prettify.js?skin=sunburst"></script>
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
	  li {
		list-style-type: decimal !important
	  }
	</style>
  </head>
  <body>
    <div style="font-size: 3em; font-weight: bold;">`+title+`</div>
    Contents of `+filename+`:
    <div>
	  <pre class="prettyprint lang-`+lang+` linenums">`+code+`</pre>
	</div>
    Error message:
    <div>
	  <pre class="prettyprint lang-json">`+errormessage+`</pre>
	</div>
	<div id="right">
	`+version_string+`
	</div>
  </body>
</html>`)
}
