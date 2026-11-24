package parser

import "github.com/gomarkdown/markdown/internal/textutil"

// isEscape reports whether byte i follows an odd-length backslash run.
func isEscape(data []byte, i int) bool {
	return (i-textutil.BackChar(data, i, '\\'))%2 != 0
}
