package simpleform

import (
	"errors"
	"fmt"
	"html"
	"strings"
)

// VersionString is the current name and version of this package
const VersionString = "SimpleForm 0.2.0"

var (
	// MultilineColumns is the number of columns that multiline input text should have
	MultilineColumns = 80
	// MultilineRows is the number of rows that multiline input text should have
	MultilineRows = 25
)

// stripBothSides can strip ie. {{ and }} from the left and right side of a string
func stripBothSides(s, left, right string) string {
	trimmed := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(s), left))
	return strings.TrimSpace(strings.TrimSuffix(trimmed, right))
}

func startForm(body *strings.Builder) {
	if !strings.Contains((*body).String(), "<form") {
		(*body).WriteString("<form method=\"POST\">")
	}
}

// HTML transforms the contents of a .frm file to a HTML form
//
// Example use:
//
//   Return a full HTML document:
//     HTML(frmContents, true, "en", "/style.css, "/favicon.ico")
//
//   Return just the body of the HTML document, using the default values of the rest:
//     HTML(frmContents, false)
//
//   Return HTML styled by MPV.CSS:
//     HTML(frmContents, true, "en", "https://unpkg.com/mvp.css")
//
// If entireDocument is true, doctype + html + head + body is returned, and not just the body contents.
//
// The optional parameter can be from 0 to 3 strings with:
// - The language of the document, default is "en"
// - The CSS URL of the document, no default, example: "/css/style.css"
// - The favicon URL of the document, no default, example: "/img/favicon.ico"
//
func HTML(frmContents string, entireDocument bool, options ...string) (string, error) {

	var (
		language   string
		CSSURL     string
		faviconURL string
	)

	if len(options) > 0 {
		language = options[0]
	}
	if len(options) > 1 {
		CSSURL = options[1]
	}
	if len(options) > 2 {
		faviconURL = options[2]
	}

	var (
		first     = true
		lines     = strings.Split(frmContents, "\n")
		shtml     strings.Builder
		text      strings.Builder
		body      strings.Builder
		title     strings.Builder
		bodyFront strings.Builder
	)

	shtml.WriteString("<!doctype html><html lang=\"")
	if len(language) == 0 {
		shtml.WriteString("en")
	} else {
		shtml.WriteString(language)
	}
	shtml.WriteString("\"><head>")
	if len(CSSURL) > 0 {
		shtml.WriteString("<link rel=\"stylesheet\" href=\"" + CSSURL + "\">")
	}
	if len(faviconURL) > 0 {
		shtml.WriteString("<link rel=\"icon\" href=\"" + faviconURL + "\">")
	}

	// Build the form, and extract the title and the text
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if strings.Contains(trimmedLine, "{{") { // text field
			startForm(&body)
			if strings.Contains(trimmedLine, ":") {
				fields := strings.SplitN(trimmedLine, ":", 2)
				label := html.EscapeString(fields[0])
				inputField := fields[1]
				fieldName := stripBothSides(inputField, "{{", "}}")
				body.WriteString("<label for=\"" + fieldName + "\">" + label + ":" + "</label>")
				if strings.HasPrefix(fieldName, "password") || strings.HasPrefix(fieldName, "pwd") {
					body.WriteString("<input type=\"password\" id=\"" + fieldName + "\" name=\"" + fieldName + "\">")
				} else {
					body.WriteString("<input type=\"text\" id=\"" + fieldName + "\" name=\"" + fieldName + "\">")
				}
				body.WriteString("<br><br>")
			} else {
				return "", fmt.Errorf("unrecognized input field description: %s", trimmedLine)
			}
		} else if strings.Contains(trimmedLine, "[[") { // multiline text field
			startForm(&body)
			if strings.Contains(trimmedLine, ":") {
				fields := strings.SplitN(trimmedLine, ":", 2)
				label := html.EscapeString(fields[0])
				inputField := fields[1]
				fieldName := stripBothSides(inputField, "[[", "]]")
				body.WriteString("<label for=\"" + fieldName + "\">" + label + ":" + "</label>")
				body.WriteString(fmt.Sprintf("<textarea name=\"%s\" cols=\"%d\" rows=\"%d\"></textarea>", fieldName, MultilineColumns, MultilineRows))
			} else {
				return "", fmt.Errorf("unrecognized multiline input field description: %s", trimmedLine)
			}
			body.WriteString("<br><br>")
		} else if strings.Contains(trimmedLine, "](") { // one or more buttons
			startForm(&body)
			buttons := strings.Split(trimmedLine, ")")
			for _, button := range buttons {
				labelAndLink := strings.TrimSpace(button)
				if len(labelAndLink) == 0 {
					continue
				}
				if !strings.Contains(labelAndLink, "](") {
					return "", fmt.Errorf("unrecognized button description :%s", trimmedLine)
				}
				fields := strings.SplitN(labelAndLink, "](", 2)
				label := html.EscapeString(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(fields[0]), "[")))
				link := strings.TrimSpace(fields[1])
				body.WriteString("<input type=\"submit\" formaction=\"" + link + "\" value=\"" + label + "\">")
			}
			body.WriteString("<br><br>")
		} else if first {
			if len(trimmedLine) == 0 {
				return "", errors.New("empty title")
			}
			shtml.WriteString("<title>" + html.EscapeString(trimmedLine) + "</title>")
			title.WriteString("<h2>" + html.EscapeString(trimmedLine) + "</h2>")
			first = false
		} else {
			text.WriteString(line)
		}
	}
	body.WriteString("</form>")
	shtml.WriteString("</head>")

	// Prepare to add the headline and descriptive text to the top of the body
	bodyFront.WriteString(title.String())
	if len(strings.TrimSpace(text.String())) > 0 {
		bodyFront.WriteString("<p>" + html.EscapeString(text.String()) + "</p>")
	}

	if entireDocument {
		// Assemble and return the complete HTML document
		shtml.WriteString("<body>" + bodyFront.String() + body.String() + "</body></html>")
		return shtml.String(), nil
	}

	// Assemble and return just the body of the HTML document, without the body tags
	return bodyFront.String() + body.String(), nil
}
