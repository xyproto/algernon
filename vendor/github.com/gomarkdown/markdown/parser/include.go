package parser

import (
	"bytes"
	"path"
	"strings"

	"github.com/gomarkdown/markdown/internal/textutil"
)

var includeCaptions = [...]string{captionFigure, captionTable, captionQuote}

// isInclude parses {{path}}[address]. The optional address selects which lines
// to include. It is treated as an opaque string and just given to readInclude.
func isInclude(data []byte) (filename string, address []byte, consumed int) {
	i := textutil.SkipCharN(data, 0, ' ', 3) // start with up to 3 spaces
	if len(data[i:]) < 3 {
		return "", nil, 0
	}
	if data[i] != '{' || data[i+1] != '{' {
		return "", nil, 0
	}
	start := i + 2

	// find the end delimiter
	i = skipUntilChar(data, i, '}')
	if i+1 >= len(data) {
		return "", nil, 0
	}
	end := i
	i++
	if data[i] != '}' {
		return "", nil, 0
	}
	filename = string(data[start:end])

	if textutil.ByteAt(data, i+1, '[') { // potential address specification
		start := i + 2

		end = skipUntilChar(data, start, ']')
		if end >= len(data) {
			return "", nil, 0
		}
		address = data[start:end]
		return filename, address, end + 1
	}

	return filename, address, i + 1
}

func (p *Parser) readInclude(from, file string, address []byte) []byte {
	if p.Opts.ReadIncludeFn != nil {
		return p.Opts.ReadIncludeFn(from, file, address)
	}

	return nil
}

// isCodeInclude parses <{{...}}, which is like isInclude, except that the
// returned bytes are wrapped in a code block.
func isCodeInclude(data []byte) (filename string, address []byte, consumed int) {
	i := textutil.SkipCharN(data, 0, ' ', 3) // start with up to 3 spaces
	if len(data[i:]) < 3 {
		return "", nil, 0
	}
	if data[i] != '<' {
		return "", nil, 0
	}
	start := i

	filename, address, consumed = isInclude(data[i+1:])
	if consumed == 0 {
		return "", nil, 0
	}
	return filename, address, start + consumed + 1
}

// readCodeInclude acts like include except the returned bytes are wrapped in a fenced code block.
func (p *Parser) readCodeInclude(from, file string, address []byte) []byte {
	data := p.readInclude(from, file, address)
	if data == nil {
		return nil
	}
	ext := path.Ext(file)
	buf := &bytes.Buffer{}
	buf.Write([]byte("```"))
	if ext != "" { // starts with a dot
		buf.WriteString(" " + ext[1:] + "\n")
	} else {
		buf.WriteByte('\n')
	}
	buf.Write(data)
	buf.WriteString("```\n")
	return buf.Bytes()
}

func (p *Parser) parseInclude(data []byte) int {
	read := p.readInclude
	file, address, consumed := isInclude(data)
	if consumed == 0 {
		file, address, consumed = isCodeInclude(data)
		read = p.readCodeInclude
	}
	if consumed == 0 {
		return 0
	}

	included := read(p.includeStack.Last(), file, address)
	if consumed < len(data) {
		rest := data[consumed:]
		captionStart := 0
		if rest[0] == '\n' {
			captionStart++
		}
		for _, prefix := range includeCaptions {
			_, _, captionLen := parseCaption(rest[captionStart:], []byte(prefix))
			if captionLen > 0 {
				included = append(included, rest[captionStart:captionStart+captionLen]...)
				consumed += captionStart + captionLen
				break
			}
		}
	}
	p.includeStack.Push(file)
	p.Block(included)
	p.includeStack.Pop()
	return consumed
}

// incStack holds the current stack of chained includes. Each value is the
// directory of the file being parsed.
type incStack struct {
	stack []string
}

func newIncStack() *incStack {
	return &incStack{}
}

// includePathIsAbs reports whether p is absolute. p uses slashes.
// path.IsAbs only accepts a leading slash, so a Windows drive path such as
// C:/dir/file is absolute here as well.
func includePathIsAbs(p string) bool {
	if path.IsAbs(p) {
		return true
	}
	return len(p) >= 3 && p[1] == ':' && p[2] == '/' && isDriveLetter(p[0])
}

func isDriveLetter(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}

// Push adds the directory of new, which is relative to the current one unless
// it's absolute, to the stack. Paths stay slash-separated: include names are
// document paths, and path.Dir does not treat a backslash as a separator.
// A drive path (C:\dir\file) and a UNC path (\\server\share\file) are absolute.
func (i *incStack) Push(new string) {
	new = strings.ReplaceAll(new, `\`, `/`)
	if !includePathIsAbs(new) {
		new = path.Join(i.Last(), new)
	}
	i.stack = append(i.stack, path.Dir(new))
}

// Pop pops the last value.
func (i *incStack) Pop() {
	if len(i.stack) > 0 {
		i.stack = i.stack[:len(i.stack)-1]
	}
}

func (i *incStack) Last() string {
	if len(i.stack) == 0 {
		return ""
	}
	return i.stack[len(i.stack)-1]
}
