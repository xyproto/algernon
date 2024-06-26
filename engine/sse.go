package engine

import (
	"bytes"
	"net/http"
	"strings"
	"text/template"

	"github.com/xyproto/algernon/utils"
)

const autoloadTemplate = `
if (!!window.EventSource) {
  window.setTimeout(function() {
    var source = new EventSource(window.location.protocol + '//{{.FullHost}}{{.EventPath}}');
    source.addEventListener('message', function(e) {
      const path = '/' + e.data;
      if (path.indexOf(window.location.pathname) >= 0) {
        location.reload()
      }
    }, false);
  }, {{.RefreshTimeout}});
}
`

type templateData struct {
	FullHost       string
	EventPath      string
	RefreshTimeout string
}

// InsertAutoRefresh inserts JavaScript code to the page that makes the page
// refresh itself when the source files changes.
// The JavaScript depends on the event server being available.
// If JavaScript can not be inserted, return the original data.
// Assumes that the given htmldata is actually HTML
// (looks for body/head/html tags when inserting a script tag)
func (ac *Config) InsertAutoRefresh(req *http.Request, htmldata []byte) []byte {
	fullHost := ac.eventAddr
	// If the host+port starts with ":", assume it's only the port number
	if strings.HasPrefix(fullHost, ":") {
		// Add the hostname in front
		if ac.serverHost != "" {
			fullHost = ac.serverHost + ac.eventAddr
		} else {
			fullHost = utils.GetDomain(req) + ac.eventAddr
		}
	}
	// Wait 70% of an event duration before starting to listen for events
	multiplier := 0.7
	refreshTimeout := utils.DurationToMS(ac.refreshDuration, multiplier)

	tmplData := templateData{
		FullHost:       fullHost,
		EventPath:      ac.defaultEventPath,
		RefreshTimeout: refreshTimeout,
	}

	// Parse the template
	t, err := template.New("auto-refresh").Parse(autoloadTemplate)
	if err != nil {
		return htmldata
	}

	var script bytes.Buffer
	err = t.Execute(&script, tmplData)
	if err != nil {
		return htmldata
	}

	return InsertScriptTag(htmldata, script.Bytes())
}

// InsertScriptTag takes HTML and JS and tries to insert the JS in a good spot, then returns modified HTML.
// If the JS does not start with "<script>" and ends with "</script>" then those tags are added.
func InsertScriptTag(htmldata, js []byte) []byte {

	// Reduce the size slightly
	js = bytes.TrimSpace(bytes.ReplaceAll(js, []byte{'\n'}, []byte{}))

	// Replace whitespace with single whitespace, repeatedly
	for bytes.Contains(js, []byte{' ', ' '}) {
		js = bytes.ReplaceAll(js, []byte{' ', ' '}, []byte{' '})
	}

	// Add tags, if needed (only check for <script and not <script> in case it is there and async)
	if !bytes.HasPrefix(js, []byte("<script")) {
		js = append([]byte("<script>"), js...)
	}
	if !bytes.HasSuffix(js, []byte("</script>")) {
		js = append(js, []byte("</script>")...)
	}

	// If there is no htmldata, then just return the script tag with JS
	if len(htmldata) == 0 {
		return js
	}

	// Place the script at the end of the body, if there is a body
	switch {
	case bytes.Contains(htmldata, []byte("</body>")):
		return bytes.Replace(htmldata, []byte("</body>"), append(js, []byte("</body>")...), 1)
	case bytes.Contains(htmldata, []byte("<head>")):
		// If not, place the script in the <head>, if there is a head
		return bytes.Replace(htmldata, []byte("<head>"), append([]byte("<head>"), js...), 1)
	case bytes.Contains(htmldata, []byte("<html>")):
		// If not, place the script in the <html> as a new <head>
		return bytes.Replace(htmldata, []byte("<html>"), append(append([]byte("<html><head>"), js...), []byte("</head>")...), 1)
	}

	// In the unlikely event that no place to insert the JavaScript was found, just add the script tag to the end
	return append(htmldata, js...)
}
