package html

import (
	"bytes"
	"html"
	"io"
	"regexp"
)

const (
	htmlTag               = "(?:" + openTag + "|" + closeTag + "|" + htmlComment + "|" + processingInstruction + "|" + declaration + "|" + cdata + ")"
	closeTag              = "</" + tagName + "\\s*[>]"
	openTag               = "<" + tagName + attribute + "*" + "\\s*/?>"
	attribute             = "(?:\\s+" + attributeName + attributeValueSpec + "?)"
	attributeValue        = "(?:" + unquotedValue + "|" + singleQuotedValue + "|" + doubleQuotedValue + ")"
	attributeValueSpec    = "(?:\\s*=\\s*" + attributeValue + ")"
	attributeName         = "[a-zA-Z_:][a-zA-Z0-9:._-]*"
	cdata                 = "<!\\[CDATA\\[[\\s\\S]*?\\]\\]>"
	declaration           = "<![A-Z]+\\s+[^>]*>"
	doubleQuotedValue     = `"[^\"]*"`
	htmlComment           = "<!---->|<!--(?:-?[^>-])(?:-?[^-])*-->"
	processingInstruction = "[<][?].*?[?][>]"
	singleQuotedValue     = "'[^']*'"
	tagName               = "[A-Za-z][A-Za-z0-9-]*"
	unquotedValue         = "[^\"'=<>`\\x00-\\x20]+"
)

var htmlTagRe = regexp.MustCompile("(?i)^" + htmlTag)

// Escaper maps HTML special characters to their escaped forms.
var Escaper = [256][]byte{
	'&': []byte("&amp;"),
	'<': []byte("&lt;"),
	'>': []byte("&gt;"),
	'"': []byte("&quot;"),
}

func escapeAttr(s string) string {
	var buf bytes.Buffer
	EscapeHTML(&buf, []byte(s))
	return buf.String()
}

func isSafeAttrName(name string) bool {
	if name == "" {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		if !('a' <= c && c <= 'z') && !('A' <= c && c <= 'Z') &&
			!('0' <= c && c <= '9') && c != '-' && c != '_' && c != ':' {
			return false
		}
	}
	return true
}

// EscapeHTML writes d with HTML special characters escaped.
func EscapeHTML(w io.Writer, d []byte) {
	start := 0
	for i, char := range d {
		if escaped := Escaper[char]; escaped != nil {
			w.Write(d[start:i])
			w.Write(escaped)
			start = i + 1
		}
	}
	w.Write(d[start:])
}

func EscLink(w io.Writer, text []byte) {
	EscapeHTML(w, []byte(html.UnescapeString(string(text))))
}

// Escape writes text while removing Markdown escape backslashes.
func Escape(w io.Writer, text []byte) {
	escaped := false
	for _, char := range text {
		if char == '\\' {
			escaped = !escaped
			if escaped {
				continue
			}
		}
		w.Write([]byte{char})
	}
}
