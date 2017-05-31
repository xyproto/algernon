// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//
package scanner

import (
	"bytes"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/wellington/sass/token"
)

var trace bool // = true

func printf(format string, v ...interface{}) {
	if trace {
		if len(v) > 0 {
			if vv, ok := v[0].(string); ok {
				vv = strings.Replace(vv, "\t", "", -1)
				vv = strings.Replace(vv, "\r", "", -1)
				vv = strings.Replace(vv, "\n", "", -1)
				vv = strings.Replace(vv, "  ", " ", -1)
				v[0] = vv
			}
		}
		fmt.Printf(format, v...)
	}
}

func IsSymbol(r rune) bool {
	return strings.ContainsRune("(),;{}#:", r)
}

func isSpace(r rune) bool {
	return unicode.IsSpace(r)
}

func isLetter(ch rune) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_' || ch >= 0x80 && unicode.IsLetter(ch)
}

func isDigit(ch rune) bool {
	return '0' <= ch && ch <= '9' || ch >= 0x80 && unicode.IsDigit(ch)
}

const (
	symbols = `.*-_&>-:$%,+~[]=()`
)

func isAllowedRune(r rune) bool {
	return unicode.IsNumber(r) ||
		unicode.IsLetter(r) ||
		strings.ContainsRune(symbols, r)
}

// An ErrorHandler may be provided to Scanner.Init. If a syntax error is
// encountered and a handler was installed, the handler is called with a
// position and an error message. The position points to the beginning of
// the offending token.
//
type ErrorHandler func(pos token.Position, msg string)

type prefetch struct {
	pos token.Pos
	tok token.Token
	lit string
}

type Scanner struct {
	src    []byte
	ch     rune
	offset int

	// hack use a channel as a FIFO queue, this is probably a terrible idea
	queue chan prefetch

	mode Mode

	// Many things in Sass change on left or right side of colon
	// rhs will track which side of the colon we are in.
	rhs bool
	// Track whether we are inside function params. If so, treat everything
	// as whitespace delimited
	inParams bool

	// inDirective controls special scanning whilst in declaration
	// it resets at '{'
	inDirective bool

	// inQuote is a hack to apply different text rules whilst
	// inside quotes
	inQuote rune

	file       *token.File
	dir        string
	err        ErrorHandler
	ErrorCount int
	rdOffset   int
	lineOffset int
}

// Mode controls scanner behavior
type Mode uint

const (
	ScanComments Mode = 1 << iota // return comments during Scan
)

func (s *Scanner) Init(file *token.File, src []byte, err ErrorHandler, mode Mode) {
	// Explicitly initialize all fields since a scanner may be reused.
	if file.Size() != len(src) {
		panic(fmt.Sprintf("file size (%d) does not match src len (%d)", file.Size(), len(src)))
	}
	s.file = file
	s.dir, _ = filepath.Split(file.Name())
	s.src = src
	s.err = err
	s.mode = mode
	// There should never more than 2 in the queue, but buffer to 10
	// just to be safe
	// Correction, selectors now use the queue and it can grow large
	s.queue = make(chan prefetch, 25)
	s.rhs = false

	s.ch = ' '
	s.offset = 0
	s.rdOffset = 0
	s.lineOffset = 0
	s.ErrorCount = 0

	s.next()
	// if s.ch == bom {
	// 	s.next() // ignore BOM at file beginning
	// }

}

// rewind resets the scanner to a previous position
// DANGER: You should only use this if you know what you are doing.
//
func (s *Scanner) rewind(offs int) {
	s.rdOffset = offs
	s.next()
}

// backup one rune
func (s *Scanner) backup() {
	w := utf8.RuneLen(s.ch)
	s.rdOffset -= w

	// Copy of slice, this is expensive
	r, w := utf8.DecodeLastRune(s.src[:s.rdOffset])
	s.offset = s.rdOffset - w
	s.ch = r
}

func (s *Scanner) next() {

	if s.rdOffset < len(s.src) {
		s.offset = s.rdOffset

		if s.ch == '\n' {
			s.lineOffset = s.offset
			s.file.AddLine(s.offset)
		}
		r, w := rune(s.src[s.rdOffset]), 1
		switch {
		case r == 0:
			s.error(s.offset, "illegal character NUL")
		case r >= 0x80:
			// not ASCII
			r, w = utf8.DecodeRune(s.src[s.rdOffset:])
			if r == utf8.RuneError && w == 1 {
				s.error(s.offset, "illegal UTF-8 encoding")
				// } else if r == bom && s.offset > 0 {
				// 	s.error(s.offset, "illegal byte order mark")
			}
		}
		s.rdOffset += w
		s.ch = r
	} else {

		s.offset = len(s.src)
		if s.ch == '\n' {
			s.lineOffset = s.offset
			s.file.AddLine(s.offset)
		}

		s.ch = -1 // eof
	}

}

type Item struct {
	Type  token.Token
	Pos   int
	Value string
}

func (s *Scanner) skipWhitespace() {
	for s.ch == ' ' || s.ch == '\t' || s.ch == '\n' || s.ch == '\r' {
		s.next()
	}
}

// Scan should differentiate between these cases
// selector[,#=.+>] {}
// :foo(ol) {}
// [hey = 'ho'], a > b {}
// c [hoo *= "ha" ] {}
// div,, , span, ,, {}
// a + b, c {}
// d e, f ~ g + h, > i {}
//
// reference parent selector: &
// function: @function h() {}
// return: @return function-exists();
// mixin: @mixin($var) {}
// call or conversion: abs(-5);
// $variable: $substitution
// rule: value;
// with #{} found anywhere in them
// directives: @import
// math 1 + 3 or (1 + 3)
// New strategy, scan until something important is encountered
func (s *Scanner) Scan() (pos token.Pos, tok token.Token, lit string) {
	defer func() {
		if trace {
			fmt.Printf("scan tok: %s lit: %q pos: %d\n", tok, lit, pos)
		}
	}()

	// Check the queue, which may contain tokens that were fetched
	// in a previous scan while determing ambiguious tokens.
	select {
	case pre := <-s.queue:
		pos, tok, lit = pre.pos, pre.tok, pre.lit
		return
	default:
		// If the queue is empty, do nothing
	}

	//pos, tok, lit = s.scan(-1)
	return s.scan()
}

func (s *Scanner) scan() (pos token.Pos, tok token.Token, lit string) {

scanAgain:
	s.skipWhitespace()
	pos = s.file.Pos(s.offset)
	offs := s.offset
	ch := s.ch

	switch {
	case ch == '>':
		offs := s.offset
		s.next()
		// bypass for GEQ, this is going to mess up
		if s.ch == '=' {
			goto bypassSelector
		} else {
			s.rewind(offs)
		}
		fallthrough
	case ch == '&':
		fallthrough
	case ch == '[':
		fallthrough
	case ch == '.':
		fallthrough
	case ch == '~':
		fallthrough
	case isLetter(ch):
		// Scan until encountering {};
		// selector: { termination
		// rule:  IDENT followed by : it must then be followed by ; or }
		// value: same as above but after the colon followed by ; or }
		pos, tok, lit = s.scanDelim(s.offset)
	case '0' <= ch && ch <= '9':
		// This can not be a selector
		tok, lit = s.scanNumber(false)
		utok, ulit := s.scanUnit()
		if utok != token.ILLEGAL {
			tok = utok
			lit = lit + ulit
		}
	}

	if tok != token.ILLEGAL {
		return
	}

	// move forward
	s.next()
bypassSelector:
	switch ch {
	case -1:
		// Text expects EOF to be empty string
		lit = ""
		tok = token.EOF
	case '$':
		lit = s.scanText(s.offset-1, 0, false, isText)
		tok = token.VAR
	case '#':
		// color:    #fff[000]
		// interp:   #{}
		if s.ch == '{' {
			// tok, lit = s.scanInterp(offs)
			tok, lit = token.INTERP, "#{"
			s.next()
		} else {
			tok, lit = s.scanColor()
		}
	case ':':
		if isLetter(s.ch) {
			pos, tok, lit = s.scanDelim(offs)
		} else {
			// s.rhs = true
			tok = token.COLON
		}
	case '-':
		if isLetter(s.ch) {
			pos, tok, lit = s.scanRule(offs)
		} else {
			tok = token.SUB
		}
	case '\'':
		// toggle inQuote mode, ignore all other runes
		tok = token.QSSTRING
		switch s.inQuote {
		case '\'':
			s.inQuote = 0
		case 0:
			s.inQuote = '\''
		}
	case '"':
		// toggle inQuote mode, ignore all other runes
		tok = token.QSTRING
		switch s.inQuote {
		case '"':
			s.inQuote = 0
		case 0:
			s.inQuote = '"'
		}
	case '.':
		if '0' <= s.ch && s.ch <= '9' {
			tok, lit = s.scanNumber(true)
			utok, ulit := s.scanUnit()
			if utok != token.ILLEGAL {
				tok = utok
				lit = lit + ulit
			}
		} else {
			tok = token.PERIOD
		}
	case '/':
		if s.ch == '/' || s.ch == '*' {
			comment := s.scanComment()
			if s.mode&ScanComments == 0 {
				goto scanAgain
			}
			tok = token.COMMENT
			lit = comment
		} else {
			tok = token.QUO
		}
	case '@':
		tok, lit = s.scanDirective()
	case '^':
		tok = token.XOR
	// case '#':
	// 	tok, lit = s.scanColor()
	case '&':
		tok = token.AND
	case '<':
		tok = s.switch2(token.LSS, token.LEQ)
	case '>':
		tok = s.switch2(token.GTR, token.GEQ)
	case '=':
		tok = s.switch2(token.ASSIGN, token.EQL)
	case '!':
		for isLetter(s.ch) {
			s.next()
		}
		// !global !default
		if s.offset-offs > 1 {
			tok = token.STRING
		} else {
			tok = s.switch2(token.NOT, token.NEQ)
		}
	case ',':
		tok = token.COMMA
	case ';':
		// s.rhs = false
		tok = token.SEMICOLON
		lit = ";"
	case '(':
		s.inParams = true
		tok = token.LPAREN
	case ')':
		s.inParams = false
		tok = token.RPAREN
	case '[':
		tok = token.LBRACK
	case ']':
		tok = token.RBRACK
	case '{':
		// reset inParams set by @each
		s.inParams = false
		s.inDirective = false
		tok = token.LBRACE
	case '}':
		tok = token.RBRACE
	case '%':
		tok = token.REM
	case '+':
		tok = token.ADD
	case '*':
		tok = token.MUL
	default:
		pos, tok, lit = s.scanRule(offs)
		if tok == token.STRING && s.ch == ';' {
			s.rewind(offs)
			lit = s.scanText(offs, 0, true, isText)
		}
		fmt.Printf("default... %q\n", lit)
	}
	return
}

// this won't be around for long
func isValue(ch rune, whitespace bool) bool {
	if ch == '-' || ch == '!' {
		return true
	}
	return isText(ch, whitespace)
}

func isText(ch rune, whitespace bool) bool {
	switch {
	case ch == '\\': // no f'ing idea
		return true
	case
		isLetter(ch), isDigit(ch),
		ch == '.': //, ch == '/':
		return true
	case whitespace && isSpace(ch):
		return true
	}
	return false
}

// Special parsing of tokens while inside params to account for different
// whitespace handling rules.
func (s *Scanner) scanParams() string {
	return ""
}

type typedScanner func(int) (token.Pos, token.Token, string)

var colondelim = []byte(":")

// scanDelim looks through ambiguous text (selectors, rules, functions)
// and returns a properly parsed set. It scans until the first
// ; : { } is found
//
// a#id { // 'a#id'
// { color: blue; } // 'color' ':' 'blue'
func (s *Scanner) scanDelim(offs int) (pos token.Pos, tok token.Token, lit string) {
	printf("Delim: %q\n", string(s.src[s.offset:]))
	defer func() {
		printf("finDelim %s:%q\n", tok, lit)
		printf("rest? %q\n", string(s.src[s.offset:]))
	}()

	pos = s.file.Pos(offs)
	var ch rune
L:
	// Set prescan up to next quote
	if s.inQuote > 0 {
		for s.ch != s.inQuote && s.ch != -1 {
			s.next()
		}
	} else {
		for !strings.ContainsRune(":;(){}", s.ch) && s.ch != -1 {
			// necessary to check for interpolation
			// interpolation is a real performance killer
			ch = s.ch
			s.next()
		}
	}

	// Return to scanning if '{' was interp not rule start
	if ch == '#' && s.ch == '{' {
		// Found interp
		for s.ch != -1 && s.ch != '}' {
			b := s.scanInterpBlock()
			if !b {
				s.error(offs, "failed to parse interpolation block")
			}
		}
		// eat interpolation RBRACE
		if s.ch != '}' {
			s.error(offs, "failed to parse interpolation end")
		}
		s.next()
		goto L
	}

	end := s.offset
	sel := bytes.TrimSpace(s.src[offs:s.offset])
	printf("prescanned: %q\n", string(sel))
	// Now that we have identified the important delimiter, rewind and
	// send to the appropriate targeted scanner for identifying token
	// and lits
	printf("delim chose ")

	var queue []prefetch
	defer func() {
		if e := recover(); e != nil {
			fmt.Printf("stack %q\n", queue)
			panic(e)
		}
	}()
	// fn should always return ILLEGAL when it fails to
	// locate a token
	var fn typedScanner
	if s.inQuote > 0 {
		fn = s.scanQuoted
		goto Q
	}
	switch s.ch {
	case '(':
		printf("ident\n")
		// function name (ident)
		// libSass supports interpolation, ruby does not
		tok = token.IDENT
		fn = s.scanIdent
	case ':':
		lit = string(s.src[offs:s.offset])
		// http detect!
		if lit == "http" {
			s.next()
			if s.ch == '/' {
				s.next()
				if s.ch == '/' {
					tok = token.STRING
					fn = s.scanHTTP
					break
				}
			}
		}
		s.rewind(offs)
		tok = token.STRING
		lit = s.scanText(offs, 0, false, isValue)
		s.skipWhitespace()
		if s.ch == ':' {
			tok = token.RULE
		}
		return
	case ')':
		fallthrough
	case '}':
		// This is only valid when a rule has one prop/value
		fallthrough
	case ';':
		printf("value\n")
		// value
		// s.rewind(offs)
		// lit = s.scanText(offs, 0, true, isText)
		fn = s.scanValue
	case -1:
		// other compilers identify first non-rule text as
		// a selector
		// like an error, not sure
		s.error(s.offset, "encountered EOF prematurely")
		fallthrough
	case '{':
		if s.inParams || s.inDirective {
			printf("value override\n")
			fn = s.scanValue
			break
		}
		printf("selector\n")
		fn = s.selLoop
		queue = []prefetch{{pos, token.SELECTOR, string(sel)}}
	case '\'', '"':
		// TODO: libSass and Sass preserve some whitespace in quotes
		panic("should never be selected")
		// this behavior is not being supported
		// lit = string(bytes.TrimSpace(s.src[offs:s.offset]))
		// tok = token.STRING
		// return
	default:
		log.Fatalf("unsupported delim %s", string(s.ch))
	}

Q:
	// Rewind and parse by correct typeScanner
	s.rewind(offs)
	// call typedScanner until end is hit
	for {
		s.skipWhitespace()
		pos, tok, lit := fn(s.offset)
		if tok != token.ILLEGAL {
			queue = append(queue, prefetch{pos, tok, lit})
			continue
		}
		s.skipWhitespace()

		// Maybe there's an interp, go ahead and try
		pos, tok, lit = s.scanInterp(s.offset)
		if tok != token.ILLEGAL {

			// Turn off quoting mode while sub scanner runs
			quoted := s.inQuote
			s.inQuote = 0
			queue = append(queue, prefetch{pos, tok, lit})
			for tok != token.EOF && tok != token.RBRACE {
				// body of interpolation
				pos, tok, lit = s.scan()
				//pos, tok, lit = s.scanDelim(s.offset)
				queue = append(queue, prefetch{pos, tok, lit})
			}

			if tok != token.RBRACE {
				s.error(s.offset, "could not find interpolation end")
			}
			s.inQuote = quoted
			continue
		}
		if s.offset > end {
			break
		}
		break
	}

	if len(queue) == 0 {
		return
	}
	if len(queue) > 1 {
		for _, pre := range queue[1:] {
			s.pushPre(pre)
		}
	}

	pos, tok, lit = queue[0].pos, queue[0].tok, queue[0].lit
	return
}

func (s *Scanner) scanQuoted(offs int) (pos token.Pos, tok token.Token, lit string) {
	inQ := s.inQuote
	ch := s.ch
	for s.ch != -1 && s.ch != inQ {
		ch = s.ch
		s.next()
		if ch == '#' && s.ch == '{' {
			s.backup()
			break
		}
	}
	pos = s.file.Pos(offs)
	lit = string(bytes.TrimSpace(s.src[offs:s.offset]))
	if len(lit) > 0 {
		tok = token.STRING
	}
	if trace {
		fmt.Printf("scanQuoted tok: %s lit: %s, rest: %q\n", tok, lit, string(s.src[s.offset:]))
	}
	return
}

func (s *Scanner) scanHTTP(offs int) (pos token.Pos, tok token.Token, lit string) {
	var ch rune
	for isText(s.ch, false) || strings.ContainsRune("-_+=.:/|?,", s.ch) {

		ch = s.ch
		s.next()
		if ch == '#' && s.ch == '{' {
			s.backup()
			break
		}
	}
	if s.ch == -1 {
		s.error(offs, "failed to find end of http")
	}
	pos = s.file.Pos(offs)
	lit = string(bytes.TrimSpace(s.src[offs:s.offset]))
	if len(lit) > 0 {
		fmt.Println("HTTP string", lit)
		tok = token.STRING
	}
	fmt.Printf("scanHTTP tok: %s lit: %s, rest: %q\n", tok, lit, string(s.src[s.offset:]))
	return
}

func (s *Scanner) selLoop(offs int) (pos token.Pos, tok token.Token, lit string) {
	defer func() {
		printf("selLoop ret %s:%q\n", tok, lit)
		printf("selLoop res %q\n", string(s.src[s.offset:]))
	}()
	pos = s.file.Pos(offs)

	switch ch := s.ch; {
	case ch == '{':
		tok = token.ILLEGAL
	case ch == '#' || ch == '.':
		s.next()
		if !isLetter(s.ch) {
			if s.ch != '{' {
				runes := string(ch) + string(s.ch)
				s.error(offs, runes+" selector must start with letter ie. .cla")
			} else {
				// Just love these interpolations
				s.backup()
				// interp bail
				return
			}
		}
		fallthrough
	// Standard selectors ie. #id .cla div
	case isLetter(ch):
		s.next()
		s.skipWhitespace()
		tok = token.STRING
		for isLetter(s.ch) || isDigit(s.ch) ||
			s.ch == '.' || s.ch == '#' {
			ch = s.ch
			s.next()
			if ch == '#' && s.ch == '{' {
				s.backup()
				// found interpolation, bail
				break
			}
			s.skipWhitespace()
			if s.ch == '&' {
				tok = token.AND
				s.next()
			}
		}
		lit = string(bytes.TrimSpace(s.src[offs:s.offset]))
	default:
		s.next()
		switch ch {
		case -1:
			lit = ""
			tok = token.EOF
		case '*': // Universal selector
			tok = token.MUL
		case '.':
			tok = token.PERIOD
		case '~':
			tok = token.TIL
		case '&':
			tok = token.AND
			for IsSymbol(s.ch) || isLetter(s.ch) || isDigit(s.ch) ||
				s.ch == '.' || s.ch == '#' {
				s.next()
			}
			lit = string(bytes.TrimSpace(s.src[offs:s.offset]))
		case '>':
			tok = s.switch2(token.GTR, token.GEQ)
		case '+':
			tok = token.ADD
		case ',':
			tok = token.COMMA
		case '[':
			tok = token.ATTRIBUTE
			runes := []rune{ch, s.ch}
			for s.ch != ']' {
				if s.ch == -1 {
					s.error(offs, "attribute selector not found")
				}
				// TODO check we ever find ']'
				s.next()
				if !unicode.IsSpace(s.ch) {
					runes = append(runes, s.ch)
				}
			}
			s.next()
			//lit = string(s.src[offs:s.offset])
			lit = string(runes)
		case ':':
			tok = token.PSEUDO
			for s.ch != ',' && !unicode.IsSpace(s.ch) {
				s.next()
			}
		case '/':
			s.backup()
			// found a comment, unwind
		default:
			s.error(s.offset, "unsupported character found: "+string(ch))
			tok = token.ILLEGAL
			lit = string(ch)
		}
	}
	return
}

// scanInterpBlock looks forward and matches all recursive interpolations
// it does not provide any useful lit or tokens and is only used
// for prescanning text.
func (s *Scanner) scanInterpBlock() bool {
	offs := s.offset
	if s.ch != '{' {
		return false
	}
	for s.ch != -1 && s.ch != '}' {
		// check for nested interpolation which is a thing!
		if s.ch == '#' {
			s.next()
			if s.ch == '{' {
				s.scanInterpBlock()
			}
		}
		s.next()
		s.skipWhitespace()
	}
	if s.ch != '}' {
		printf("tried %s %q\n", string(s.ch), string(s.src[offs:s.offset]))
		s.error(offs, "failed to locate interpolation end }")
		return false
	}

	return true
}

// syntax of @each easily fools scanDelim
// @each {variable} in {list/map}
func (s *Scanner) scanEach(offs int) {
	// queue the iterator, look for in and parse the list/map
	s.next()
	s.push(s.scan())

	// find 'in'
	s.skipWhitespace()
	inoffs := s.offset
	for isText(s.ch, false) {
		s.next()
	}
	inlit := string(s.src[inoffs:s.offset])
	if inlit != "in" {
		fmt.Println("NOTIN?", inlit)
		s.error(inoffs, "in must be present in @each statement")
	}
	s.push(s.file.Pos(inoffs), token.STRING, inlit)
	s.skipWhitespace()
	// list is surrounded with params, we're done here
	if s.ch == '(' {
		return
	}
	// must tell scanDelim that we're implicit params
	s.inParams = true
	return
}

func (s *Scanner) scanInterp(offs int) (pos token.Pos, tok token.Token, lit string) {
	if s.ch != '#' {
		return
	}

	s.next()
	if s.ch != '{' {
		return
	}
	s.next()
	return s.file.Pos(offs), token.INTERP, "#{"
}

func (s *Scanner) push(pos token.Pos, tok token.Token, lit string) {
	s.queue <- prefetch{pos, tok, lit}
}

func (s *Scanner) pushPre(pre prefetch) {
	// TODO: check for full queue
	s.queue <- pre
}

// scanInterp attempts to build a valid set of tokens from an interpolation
func (s *Scanner) queueInterp(offs int) bool {
	pos, tok, lit := s.scanInterp(offs)
	if tok == token.INTERP {
		// If found, just push into the queue for next Scan
		s.queue <- prefetch{
			pos: pos,
			tok: tok,
			lit: lit,
		}
		return true
	}

	return false
}

// ScanText is responsible for gobbling non-whitespace characters
//
// This should validate variable naming http://stackoverflow.com/a/17194994
// a-zA-Z0-9_-
// Also these if escaped with \ !"#$%&'()*+,./:;<=>?@[]^{|}~
func (s *Scanner) scanText(offs int, end rune, whitespace bool, fn func(rune, bool) bool) string {
	// offs := s.offset - 1 // catch first quote
	var ch rune
	for s.ch == '\\' || fn(s.ch, whitespace) ||
		// #id
		s.ch == '#' ||
		s.ch == end {
		ch = s.ch
		if _, tok, _ := s.scanInterp(offs); tok != token.ILLEGAL {
			break
		}
		s.next()

		// evidently, escaping only happens when unquoting
		if ch == '\\' && false {
			if strings.ContainsRune(`!"#$%&'()*+,./:;<=>?@[]^{|}~`, s.ch) {
				s.next()
			} else {
				s.error(s.offset, "attempted to escape invalid character "+string(s.ch))
			}
		}

		if ch == end {
			break
		}
	}

	// eat the end character
	if end != 0 && ch != end {
		s.error(s.offset, "expected end of "+string(end))
	}

	ss := string(bytes.TrimSpace(s.src[offs:s.offset]))
	return ss
}

func (s *Scanner) scanRGB(pos int) (tok token.Token, lit string) {
	tok = token.COLOR
	offs := pos

	if s.ch != '(' {
		lit = string(s.src[offs:s.offset])
		s.error(offs, "invalid rgb (: "+lit)
	}

	for s.ch != ')' && s.ch != ';' {
		s.next()
	}
	if s.ch == ';' {
		s.error(offs, "invalid rgb: "+string(s.src[offs:s.offset]))
	}
	s.next()

	lit = string(s.src[offs:s.offset])
	return
}

func (s *Scanner) scanColor() (tok token.Token, lit string) {
	offs := s.offset - 1
	for (s.ch >= 'a' && s.ch <= 'f') ||
		(s.ch >= 'A' && s.ch <= 'F') || isDigit(s.ch) {
		s.next()
	}
	lit = string(s.src[offs:s.offset])
	if len(lit) > 1 {
		return token.COLOR, lit
	}
	return token.ILLEGAL, lit
}

// ScanDirective matches Sass directives http://sass-lang.com/documentation/file.SASS_REFERENCE.html#directives
func (s *Scanner) scanDirective() (tok token.Token, lit string) {
	offs := s.offset - 1
	for isLetter(s.ch) || s.ch == '-' {
		s.next()
	}
	lit = string(s.src[offs:s.offset])
	switch lit {
	case "@if":
		tok = token.IF
		s.inDirective = true
	case "@else":
		s.skipWhitespace()
		if s.ch != 'i' {
			tok = token.ELSE
		}
		s.next()
		if s.ch == 'f' {
			s.next()
			lit = string(s.src[offs:s.offset])
			tok = token.ELSEIF
			s.inDirective = true
			break
		}
		// else if check failed
		s.backup()
	case "@for":
		tok = token.FOR
	case "@each":
		tok = token.EACH
		s.scanEach(s.offset)
	case "@include":
		tok = token.INCLUDE
	case "@function":
		tok = token.FUNC
	case "@mixin":
		tok = token.MIXIN
	case "@return":
		tok = token.RETURN
	case "@import":
		tok = token.IMPORT
	case "@media":
		tok = token.MEDIA
		s.skipWhitespace()
		// media queries have a lot of runes, eat until the first {
		offs := s.offset
		for s.ch != '{' {
			s.next()
		}
		lit := s.src[offs:s.offset]
		s.queue <- prefetch{
			pos: s.file.Pos(offs),
			tok: token.STRING,
			lit: string(bytes.TrimSpace(lit)),
		}
	case "@extend":
		tok = token.EXTEND
	case "@at-root":
		tok = token.ATROOT
	case "@debug":
		tok = token.DEBUG
	case "@warn":
		tok = token.WARN
	case "@error":
		tok = token.ERROR
	}

	return
}

func (s *Scanner) scanRule(offs int) (pos token.Pos, tok token.Token, lit string) {
	var interp bool
ruleAgain:
	pos = s.file.Pos(offs)
	for {
		if ok := s.queueInterp(s.offset); ok {
			// If we got here, there could be text in the buffer
			// If so, push interpolate into the queue and return this
			// interpolate prefix

			if s.offset-offs > 1 {
				tok = token.STRING
				lit = string(bytes.TrimSpace(s.src[offs : s.offset-2]))
			}
			break
		}
		if strings.ContainsRune(" \n:();{},$", s.ch) {
			break
		}
		s.next()
	}
	if len(lit) == 0 {
		lit = string(bytes.TrimSpace(s.src[offs:s.offset]))
	}
	s.skipWhitespace()

	switch s.ch {
	case ':':
		tok = token.RULE
	case '(':
		// mixin or func ident
		// IDENT()
		tok = token.IDENT
	case ')', ',':
		tok = token.STRING
	case ';':
		tok = token.STRING
	default:
		tok = token.STRING
		if interp {
			if tok != token.RBRACE {
				// It's like groundhog day, but it's interpolation every day
				goto ruleAgain
			}
			s.queue <- prefetch{
				pos: pos,
				lit: "}",
				tok: token.RBRACE,
			}
			return
		}
		// Not sure, this requires more specifics
		fmt.Printf("                fallback because %q: %s\n", string(s.ch), lit)
		// tok = token.IDENT

	}
	return
}

// scanValue inspects rhs of ':' for every rule blocks
func (s *Scanner) scanValue(offs int) (pos token.Pos, tok token.Token, lit string) {
	pos = s.file.Pos(offs)
	// Only look for text here, numbers and symbols will be
	// caught by Scan()
	if isDigit(s.ch) {
		tok = token.INT
	}
	var maybeFloat bool
	for s.ch == '$' || isValue(s.ch, false) || isDigit(s.ch) {
		if maybeFloat && isDigit(s.ch) {
			tok = token.FLOAT
		} else if maybeFloat {
			maybeFloat = false
		}
		if s.ch == '.' {
			maybeFloat = true
		}
		s.next()
	}

	// lit = s.scanText(offs, 0, true, isText)
	if s.offset > offs {
		if tok == token.ILLEGAL {
			tok = token.STRING
			if s.src[offs] == '$' {
				tok = token.VAR
			}
		}
		lit = string(s.src[offs:s.offset])
	}
	return
}

func (s *Scanner) scanIdent(offs int) (pos token.Pos, tok token.Token, lit string) {
	pos = s.file.Pos(offs)
	for isLetter(s.ch) || isDigit(s.ch) || s.ch == '-' {
		s.next()
	}
	lit = string(s.src[offs:s.offset])
	if len(lit) > 0 {
		tok = token.IDENT
	}
	return
}

func (s *Scanner) scanUnit() (token.Token, string) {
	offs := s.offset
	for isLetter(s.ch) || s.ch == '%' {
		s.next()
	}

	lit := string(s.src[offs:s.offset])
	tok := token.ILLEGAL
	switch lit {
	case "in":
		tok = token.UIN
	case "cm":
		tok = token.UCM
	case "mm":
		tok = token.UMM

	case "pc":
		tok = token.UPC

	case "px":
		tok = token.UPX
	case "pt":
		tok = token.UPT

	case "deg":
		tok = token.DEG
	case "grad":
		tok = token.GRAD
	case "rad":
		tok = token.RAD
	case "turn":
		tok = token.TURN

	case "em":
		tok = token.UEM
	case "rem":
		tok = token.UREM
	case "%":
		tok = token.UPCT
	default:
		lit = ""
	}

	return tok, lit
}

func (s *Scanner) scanNumber(seenDecimalPoint bool) (token.Token, string) {
	// digitVal(s.ch) < 10
	offs := s.offset
	tok := token.INT

	if seenDecimalPoint {
		offs--
		tok = token.FLOAT
		s.scanMantissa(10)
	}
	if s.ch == '0' {
		// int or float
		offs := s.offset
		s.next()
		if s.ch == 'x' || s.ch == 'X' {
			// hexadecimal int
			s.next()
			s.scanMantissa(16)
			if s.offset-offs <= 2 {
				// only scanned "0x" or "0X"
				s.error(offs, "illegal hexadecimal number")
			}
		} else {
			// octal int or float
			seenDecimalDigit := false
			s.scanMantissa(8)
			if s.ch == '8' || s.ch == '9' {
				// illegal octal int or float
				seenDecimalDigit = true
				s.scanMantissa(10)
			}
			if s.ch == '.' || s.ch == 'e' || s.ch == 'E' || s.ch == 'i' {
				goto fraction
			}
			// octal int
			if seenDecimalDigit {
				s.error(offs, "illegal octal number")
			}
		}
		goto exit
	}

	// decimal int or float
	s.scanMantissa(10)

fraction:
	if s.ch == '.' {
		tok = token.FLOAT
		s.next()
		s.scanMantissa(10)
	}

exit:
	return tok, string(s.src[offs:s.offset])

}

func (s *Scanner) scanMantissa(base int) {
	for {
		if digitVal(s.ch) >= base {
			return
		}
		if s.queueInterp(s.offset) {
			return
		}
		s.next()
	}
}

func digitVal(ch rune) int {
	switch {
	case '0' <= ch && ch <= '9':
		return int(ch - '0')
	case 'a' <= ch && ch <= 'f':
		return int(ch - 'a' + 10)
	case 'A' <= ch && ch <= 'F':
		return int(ch - 'A' + 10)
	}
	return 16 // larger than any legal digit val
}

func (s *Scanner) scanComment() string {
	// initial '/' already consumed; s.ch == '/' || s.ch == '*'
	offs := s.offset - 1 // position of initial '/'
	hasCR := false

	if s.ch == '/' {
		//-style comment
		s.next()
		for s.ch != '\n' && s.ch >= 0 {
			if s.ch == '\r' {
				hasCR = true
			}
			s.next()
		}
		goto exit
	}

	/*-style comment */
	s.next()
	for s.ch >= 0 {
		ch := s.ch
		if ch == '\r' {
			hasCR = true
		}
		s.next()
		if ch == '*' && s.ch == '/' {
			s.next()
			goto exit
		}
	}
	s.error(offs, "comment not terminated")

exit:
	lit := s.src[offs:s.offset]
	if hasCR {
		lit = stripCR(lit)
	}

	return string(lit)
}

func (s *Scanner) error(offs int, msg string) {
	if s.err != nil {
		s.err(s.file.Position(s.file.Pos(offs)), msg)
	}
	s.ErrorCount++
}

func stripCR(b []byte) []byte {
	c := make([]byte, len(b))
	i := 0
	for _, ch := range b {
		if ch != '\r' {
			c[i] = ch
			i++
		}
	}
	return c[:i]
}

func (s *Scanner) switch2(tok0, tok1 token.Token) token.Token {
	if s.ch == '=' {
		s.next()
		return tok1
	}
	return tok0
}
