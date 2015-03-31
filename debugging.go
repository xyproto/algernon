package main

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strings"
)

func prettyLuaError(w http.ResponseWriter, filename string, luaerr error) {
	var luacode string
	// Output the Lua error message to the browser
	w.Header().Add("Content-Type", "text/html; encoding=UTF-8")
	luabytes, err := ioutil.ReadFile(filename)
	if err != nil {
		luacode = err.Error()
	} else {
		// TODO: Find a better way to escape HTML
		luacode = strings.Replace(string(luabytes), "<", "&lt;", -1)
	}
	// Find an appropriate title
	title := []string{"Fail", "Facepalm", "Disaster"}[rand.Intn(3)]
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
	</style>
  </head>
  <body>
    <div style="font-size: 3em; font-weight: bold;">`+title+`</div>
    Contents of `+filename+`:
    <div>
	  <pre class="prettyprint lang-lua">`+luacode+`</pre>
	</div>
    Error message:
    <div>
	  <pre class="prettyprint">`+luaerr.Error()+`</pre>
	</div>
	`+version_string+`
  </body>
</html>`)
}
