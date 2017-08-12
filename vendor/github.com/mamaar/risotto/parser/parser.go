package parser

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"

	"github.com/mamaar/risotto/ast"
	"github.com/mamaar/risotto/file"
	"github.com/mamaar/risotto/token"
)

// A Mode value is a set of flags (or 0). They control optional parser functionality.
type Mode uint

const (
	IgnoreRegExpErrors Mode = 1 << iota // Ignore RegExp compatibility errors (allow backtracking)
)

type _parser struct {
	str      string
	length   int
	base     int

	chr       rune // The current character
	chrOffset int  // The offset of current character
	offset    int  // The offset after current character (may be greater than 1)

	idx     file.Idx    // The index of token
	token   token.Token // The token
	literal string      // The literal of the token, if any

	scope             *_scope
	insertSemicolon   bool // If we see a newline, then insert an implicit semicolon
	implicitSemicolon bool // An implicit semicolon exists

	errors ErrorList

	recover struct {
		// Scratch when trying to seek to the next statement, etc.
		idx   file.Idx
		count int
	}

	mode Mode

	file *file.File
}

func _newParser(filename, src string, base int) *_parser {
	return &_parser{
		str:    src,
		offset: -1,
		length: len(src),
		base:   base,
		file:   file.NewFile(filename, src, base),
	}
}

func newParser(filename, src string) *_parser {
	return _newParser(filename, src, 1)
}

func ReadSource(filename string, src interface{}) ([]byte, error) {
	if src != nil {
		switch src := src.(type) {
		case string:
			return []byte(src), nil
		case []byte:
			return src, nil
		case *bytes.Buffer:
			if src != nil {
				return src.Bytes(), nil
			}
		case io.Reader:
			var bfr bytes.Buffer
			if _, err := io.Copy(&bfr, src); err != nil {
				return nil, err
			}
			return bfr.Bytes(), nil
		}
		return nil, errors.New("invalid source")
	}
	return ioutil.ReadFile(filename)
}

// ParseFile parses the source code of a single JavaScript/ECMAScript source file and returns
// the corresponding ast.Program node.
//
// If fileSet == nil, ParseFile parses source without a FileSet.
// If fileSet != nil, ParseFile first adds filename and src to fileSet.
//
// The filename argument is optional and is used for labelling errors, etc.
//
// src may be a string, a byte slice, a bytes.Buffer, or an io.Reader, but it MUST always be in UTF-8.
//
//      // Parse some JavaScript, yielding a *ast.Program and/or an ErrorList
//      program, err := parser.ParseFile(nil, "", `if (abc > 1) {}`, 0)
//
func ParseFile(fileSet *file.FileSet, filename string, src interface{}, mode Mode) (*ast.Program, error) {
	str, err := ReadSource(filename, src)
	if err != nil {
		return nil, err
	}
	{
		str := string(str)

		base := 1
		if fileSet != nil {
			base = fileSet.AddFile(filename, str)
		}

		parser := _newParser(filename, str, base)
		parser.mode = mode
		return parser.parse()
	}
}

func (self *_parser) slice(idx0, idx1 file.Idx) string {
	from := int(idx0) - self.base
	to := int(idx1) - self.base
	if from >= 0 && to <= len(self.str) {
		return self.str[from:to]
	}

	return ""
}

func (self *_parser) parse() (*ast.Program, error) {
	self.next()
	program := self.parseProgram()
	return program, self.errors.Err()
}

// rawNext moves pointer to the next token
func (self *_parser) rawNext() {
	self.token, self.literal, self.idx = self.scan()
}

// next moves pointer to next non-whitespace token
func (self *_parser) next() {
	for {
		self.token, self.literal, self.idx = self.scan()
		if self.token != token.WHITESPACE {
			break
		}
	}
}

func (self *_parser) optionalSemicolon() {
}

func (self *_parser) semicolon() {
	if self.token != token.RIGHT_PARENTHESIS && self.token != token.RIGHT_BRACE {
		if self.implicitSemicolon {
			self.implicitSemicolon = false
			return
		}

		self.expect(token.SEMICOLON)
	}
}

func (self *_parser) idxOf(offset int) file.Idx {
	return file.Idx(self.base + offset)
}

func (self *_parser) expect(value token.Token) file.Idx {
	idx := self.idx
	if self.token != value {
		self.errorUnexpectedToken(self.token)
	}
	self.next()
	return idx
}

func lineCount(str string) (int, int) {
	line, last := 0, -1
	pair := false
	for index, chr := range str {
		switch chr {
		case '\r':
			line += 1
			last = index
			pair = true
			continue
		case '\n':
			if !pair {
				line += 1
			}
			last = index
		case '\u2028', '\u2029':
			line += 1
			last = index + 2
		}
		pair = false
	}
	return line, last
}

func (self *_parser) position(idx file.Idx) file.Position {
	position := file.Position{}
	offset := int(idx) - self.base
	str := self.str[:offset]
	position.Filename = self.file.Name()
	line, last := lineCount(str)
	position.Line = 1 + line
	if last >= 0 {
		position.Column = offset - last
	} else {
		position.Column = 1 + len(str)
	}

	return position
}
