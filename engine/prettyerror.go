package engine

import (
	"bytes"
	"net/http"
	"strconv"
	"strings"
	"text/template"
)

const (
	// Highlight of errors in the code
	preHighlight  = "<font style='color: red !important'>"
	postHighlight = "</font>"

	// HTML template for the error page
	htmlTemplate = `<!doctype html>
<html>
  <head>
    <title>{{.Title}}</title>
    <style>
      body {
        background-color: #f0f0f0;
        color: #0b0b0b;
        font-family: Lato,'Trebuchet MS',Helvetica,sans-serif;
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
        text-align: right;
      }
      #wrap {
        white-space: pre-wrap;
      }
    </style>
  </head>
  <body>
    <div style="font-size: 3em; font-weight: bold;">{{.Title}}</div>
    Contents of {{.Filename}}:
    <div>
      <pre><code>{{.Code}}</code></pre>
    </div>
    Error message:
    <div>
      <pre id="wrap"><code style="color: #A00000;">{{.ErrorMessage}}</code></pre>
    </div>
    <div id="right">{{.VersionString}}</div>
  </body>
</html>`
)

// Given a lowercase string for the language, return an appropriate error page title
func errorPageTitle(lang string) string {
	switch lang {
	case "":
		return "Error"
	case "css":
		return "CSS Error"
	case "gcss":
		return "GCSS Error"
	case "html":
		return "HTML Error"
	case "jsx":
		return "JSX Error"
	default:
		return lang + " Error"
	}
}

// PrettyError serves an informative error page to the user
func (ac *Config) PrettyError(w http.ResponseWriter, req *http.Request, filename string, filebytes []byte, errormessage, lang string) {
	// HTTP status
	w.WriteHeader(http.StatusOK)

	// HTTP content type
	w.Header().Add("Content-Type", "text/html;charset=utf-8")

	var (
		code string
		err  error
	)

	// The line that the error refers to, for the case of Lua
	linenr := -1

	if len(filebytes) > 0 {
		if lang == "lua" {
			fields := strings.SplitN(errormessage, ":", 3)
			if len(fields) > 2 {
				numberfield := fields[1]
				if strings.Contains(numberfield, "(") {
					numberfield = strings.Split(numberfield, "(")[0]
				}
				linenr, err = strconv.Atoi(numberfield)
				linenr--
				if err != nil {
					linenr = -1
				}
			}
		} else if lang == "amber" {
			if strings.Contains(errormessage, "- Line: ") {
				fields := strings.SplitN(errormessage, "- Line: ", 2)
				if strings.Contains(fields[1], ",") {
					numberfields := strings.SplitN(fields[1], ",", 2)
					linenr, err = strconv.Atoi(strings.TrimSpace(numberfields[0]))
					linenr--
					if err != nil {
						linenr = -1
					}
				}
			}
		}

		filebytes = bytes.ReplaceAll(filebytes, []byte("<"), []byte("&lt;"))
		bytelines := bytes.Split(filebytes, []byte("\n"))
		if (linenr >= 0) && (linenr < len(bytelines)) {
			bytelines[linenr] = []byte(preHighlight + string(bytelines[linenr]) + postHighlight)
		}
		code = string(bytes.Join(bytelines, []byte("\n")))
	}

	title := errorPageTitle(lang)

	data := struct {
		Title         string
		Filename      string
		Code          string
		ErrorMessage  string
		VersionString string
	}{
		Title:         title,
		Filename:      filename,
		Code:          code,
		ErrorMessage:  strings.TrimSpace(errormessage),
		VersionString: ac.versionString,
	}

	tmpl, err := template.New("errorPage").Parse(htmlTemplate)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if ac.autoRefresh {
		var htmlbuf bytes.Buffer
		if err := tmpl.Execute(&htmlbuf, data); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		w.Write(ac.InsertAutoRefresh(req, htmlbuf.Bytes()))
		return
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
