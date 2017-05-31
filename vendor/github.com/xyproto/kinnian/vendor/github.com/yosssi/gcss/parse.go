package gcss

import "strings"

// Special characters
const (
	cr               = "\r"
	lf               = "\n"
	crlf             = "\r\n"
	space            = " "
	colon            = ":"
	comma            = ","
	openBrace        = "{"
	closeBrace       = "}"
	semicolon        = ";"
	ampersand        = "&"
	atMark           = "@"
	dollarMark       = "$"
	openParenthesis  = "("
	closeParenthesis = ")"
	slash            = "/"
	doubleSlash      = slash + slash
)

// parse parses the string, generates the elements
// and returns the two channels: the first one returns
// the generated elements and the last one returns
// an error when it occurs.
func parse(lines []string) (<-chan element, <-chan error) {
	elemc := make(chan element, len(lines))
	errc := make(chan error)

	go func() {
		i := 0
		l := len(lines)

		for i < l {
			// Fetch a line.
			ln := newLine(i+1, lines[i])
			i++

			// Ignore the empty line.
			if ln.isEmpty() {
				continue
			}

			if ln.isTopIndent() {
				elem, err := newElement(ln, nil)

				if err != nil {
					errc <- err
					return
				}

				if err := appendChildren(elem, lines, &i, l); err != nil {
					errc <- err
					return
				}

				elemc <- elem
			}
		}

		close(elemc)
	}()

	return elemc, errc
}

// appendChildren parses the lines and appends the child elements
// to the parent element.
func appendChildren(parent element, lines []string, i *int, l int) error {
	for *i < l {
		// Fetch a line.
		ln := newLine(*i+1, lines[*i])

		// Ignore the empty line.
		if ln.isEmpty() {
			*i++
			return nil
		}

		ok, err := ln.childOf(parent)

		if err != nil {
			return err
		}

		if !ok {
			return nil
		}

		child, err := newElement(ln, parent)

		if err != nil {
			return err
		}

		parent.AppendChild(child)

		*i++

		if err := appendChildren(child, lines, i, l); err != nil {
			return err
		}
	}

	return nil
}

// formatLF replaces the line feed codes with LF and
// returns the result string.
func formatLF(s string) string {
	return strings.Replace(strings.Replace(s, crlf, lf, -1), cr, lf, -1)
}
