package engine

import (
	"bytes"
	"net/http"
	"strings"

	"github.com/xyproto/algernon/utils"
)

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
	js := `
    <script>
    if (!!window.EventSource) {
	  window.setTimeout(function() {
        var source = new EventSource(window.location.protocol + '//` + fullHost + ac.defaultEventPath + `');
        source.addEventListener('message', function(e) {
          const path = '/' + e.data;
          if (path.indexOf(window.location.pathname) >= 0) {
            location.reload()
          }
        }, false);
	  }, ` + utils.DurationToMS(ac.refreshDuration, multiplier) + `);
	}
    </script>`

	// Reduce the size slightly
	js = strings.TrimSpace(strings.Replace(js, "\n", "", utils.EveryInstance))
	// Remove all whitespace that is more than one space
	for strings.Contains(js, "  ") {
		js = strings.Replace(js, "  ", " ", utils.EveryInstance)
	}
	// Place the script at the end of the body, if there is a body
	if bytes.Contains(htmldata, []byte("</body>")) {
		return bytes.Replace(htmldata, []byte("</body>"), []byte(js+"</body>"), 1)
	} else if bytes.Contains(htmldata, []byte("<head>")) {
		// If not, place the script in the <head>, if there is a head
		return bytes.Replace(htmldata, []byte("<head>"), []byte("<head>"+js), 1)
	} else if bytes.Contains(htmldata, []byte("<html>")) {
		// If not, place the script in the <html> as a new <head>
		return bytes.Replace(htmldata, []byte("<html>"), []byte("<html><head>"+js+"</head>"), 1)
	}
	// In the unlikely event that no place to insert the JavaScript was found
	return htmldata
}
