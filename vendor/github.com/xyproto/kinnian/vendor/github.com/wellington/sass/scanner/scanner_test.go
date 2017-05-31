// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//
package scanner

import (
	"log"
	"strings"
	"testing"

	"github.com/wellington/sass/token"
)

func init() {
	trace = true
}

type elt struct {
	tok token.Token
	lit string
	// class int
}

var whitespace = "  \t  \n\n\n" // to separate tokens

var elts = []elt{
	{token.COMMENT, "/* a comment */"},
	{token.COMMENT, "// single comment \n"},
	{token.INT, "0"},
	{token.INT, "314"},
	{token.FLOAT, "3.1415"},

	// Operators and delimiters
	// {token.ADD, "+"}, '+' is overloaded for BACKREF
	{token.SUB, "-"},
	{token.MUL, "*"},
	{token.QUO, "/"},
	{token.REM, "%"},

	// {token.AND, "&"},
	{token.XOR, "^"},
	// {token.LAND, "&&"},
	// {token.LOR, "||"},
	{token.EQL, "=="},
	{token.LSS, "<"},
	// {token.GTR, ">"}, // Broken by reporting of >
	// {token.ASSIGN, "="},
	{token.NOT, "!"},

	{token.NEQ, "!="},
	{token.LEQ, "<="},
	{token.GEQ, ">="},

	// Delimiters
	{token.LPAREN, "("},
	// {token.LBRACK, "["},
	/*{token.LBRACE, "{"},
	{token.COMMA, ","},
	// {token.PERIOD, "."},

	{token.RPAREN, ")"},
	{token.RBRACK, "]"},
	{token.RBRACE, "}"},
	{token.SEMICOLON, ";"},
	{token.COLON, ":"},

	// {token.QUOTE, "\""},
	// {token.AT, "@"},
	// {token.NUMBER, "#"},
	{token.VAR, "$poop"},
	{token.QSTRING, `"a 'red'\! and \"blue\" value"`},
	{token.UPX, "10px"},
	{token.IMPORT, "@import"},
	{token.ATROOT, "@at-root"},
	{token.DEBUG, "@debug"},
	{token.ERROR, "@error"},
	{token.VAR, "$color"},
	// {token.COLOR, "#000"},
	// {token.COLOR, "#abcabc"},
	{token.MIXIN, "@mixin"},*/
	// {token.SELECTOR, "foo($a,$b)"},
	// {token.COLOR, "rgb(10,10,10)"},
}

var source = func(tokens []elt) []byte {
	var src []byte
	for _, t := range tokens {
		src = append(src, t.lit...)
		src = append(src, whitespace...)
	}

	return src
}

var fset = token.NewFileSet()

func newlineCount(s string) int {
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			n++
		}
	}
	return n
}

func checkPos(t *testing.T, lit string, p token.Pos, expected token.Position) {
	pos := fset.Position(p)
	if pos.Filename != expected.Filename {
		t.Errorf("bad filename for %q: got %s, expected %s", lit, pos.Filename, expected.Filename)
	}
	if pos.Offset != expected.Offset {
		t.Errorf("bad position for %q: got %d, expected %d", lit, pos.Offset, expected.Offset)
	}
	if pos.Line != expected.Line {
		t.Errorf("bad line for %q: got %d, expected %d", lit, pos.Line, expected.Line)
	}
	if pos.Column != expected.Column {
		t.Errorf("bad column for %q: got %d, expected %d", lit, pos.Column, expected.Column)
	}
}

func TestScan_tokens(t *testing.T) {
	testScan(t, elts)
}

func TestScan_directive(t *testing.T) {
	testScan(t, []elt{
		{token.IF, "@if"},
		{token.ELSE, "@else"},
		{token.ELSEIF, "@else if"},
	})

	testScan(t, []elt{
		{token.MEDIA, "@media"},
		{token.STRING, "print and (foo: 1 2 3), (bar: 3px hux(muz)), not screen"},
		{token.LBRACE, "{"},
	})

	testScan(t, []elt{
		{token.IDENT, "url"},
		{token.LPAREN, "("},
		{token.QSTRING, `"`},
		{token.STRING, "http://fudge/styles.css"},
		{token.QSTRING, `"`},
		{token.RPAREN, ")"},
	})

	testScan(t, []elt{
		{token.IDENT, "url"},
		{token.LPAREN, "("},
		{token.QSTRING, `"`},
		{token.STRING, "http://"},
		{token.INTERP, "#{"},
		{token.VAR, "$x"},
		{token.RBRACE, "}"},
		{token.STRING, "/styles.css"},
		{token.QSTRING, `"`},
		{token.RPAREN, ")"},
	})

	testScan(t, []elt{
		{token.IDENT, "url"},
		{token.LPAREN, "("},
		{token.STRING, `fudge`},
		{token.INTERP, "#{"},
		{token.VAR, "$x"},
		{token.RBRACE, "}"},
		{token.STRING, ".css"},
		{token.RPAREN, ")"},
	})

	testScan(t, []elt{
		{token.IDENT, "url"},
		{token.LPAREN, "("},
		//{token.STRING, "http://fudge#{$x}/styles.css"}, //"http://fonts.googleapis.com/css?family=Karla:400,700,400italic|Anonymous+Pro:400,700,400italic"},
		//"http://fonts.googleapis.com/css?family=Karla:400,700,400italic|Anonymous+Pro:400,700,400italic"},
		{token.STRING, "http://fudge"},
		{token.INTERP, "#{"},
		{token.VAR, "$x"},
		{token.RBRACE, "}"},
		{token.STRING, "/styles.css"},
		{token.RPAREN, ")"},
	})

	testScan(t, []elt{
		{token.EACH, "@each"},
		{token.VAR, "$i"},
		{token.STRING, "in"},
		{token.LPAREN, "("},
		{token.STRING, "a"},
		{token.STRING, "b"},
		{token.RPAREN, ")"},
		{token.LBRACE, "{"},
	})

	testScan(t, []elt{
		{token.EACH, "@each"},
		{token.VAR, "$i"},
		{token.STRING, "in"},
		{token.STRING, "a"},
		{token.LBRACE, "{"},
	})
}

func TestScan_quotes(t *testing.T) {
	oldWs := whitespace
	defer func() {
		whitespace = oldWs
	}()
	whitespace = ""
	testScanMap(t, `'http://blah.com/blah.html'`, []elt{
		{token.QSSTRING, "'"},
		{token.STRING, "http://blah.com/blah.html"},
		{token.QSSTRING, "'"},
	})

	testScanMap(t, `'http://blah.com/blah.html?query=string:stuf&in&here'`, []elt{
		{token.QSSTRING, "'"},
		{token.STRING, "http://blah.com/blah.html?query=string:stuf&in&here"},
		{token.QSSTRING, "'"},
	})

	testScanMap(t, `'http://blah.com/blah.html?#{qs}'`, []elt{
		{token.QSSTRING, "'"},
		{token.STRING, "http://blah.com/blah.html?"},
		{token.INTERP, "#{"},
		{token.STRING, "qs"},
		{token.RBRACE, "}"},
		{token.QSSTRING, "'"},
	})
}

func TestScan_selectors(t *testing.T) {
	testScan(t, []elt{
		// {token.SELECTOR, "i#grer"}
		{token.STRING, "i#grer"},
		{token.LBRACE, "{"},
	})

	// selectors are so flexible, that they must be tested in isolation
	testScan(t, []elt{
		// {token.SELECTOR, "}
		{token.ADD, "+"},
		{token.STRING, "div"},
		{token.LBRACE, "{"},
	})

	testScan(t, []elt{
		// {token.SELECTOR, ""},
		{token.AND, "foo &.goo"},
		{token.LBRACE, "{"},
	})

	testScan(t, []elt{
		// {token.SELECTOR, "& > boo"},
		{token.AND, "&"},
		{token.GTR, ">"},
		{token.STRING, "boo"},
		{token.LBRACE, "{"},
	})

	testScan(t, []elt{
		// {token.SELECTOR},
		{token.STRING, "a"},
		{token.COMMA, ","},
		{token.STRING, "b"},
		{token.TIL, "~"},
		{token.STRING, "c"},
		{token.ADD, "+"},
		{token.MUL, "*"},
		{token.LBRACE, "{"},
	})

	testScan(t, []elt{
		// {token.SELECTOR, "&.goo"},
		{token.AND, "&"},
		{token.STRING, ".goo"},
		{token.LBRACE, "{"},
		{token.COMMENT, "// blah blah blah \n"},
		{token.COMMENT, "/* hola */"},
		{token.RULE, "-webkit-color"},
		{token.COLON, ":"},
		{token.COLOR, "#fff"},
		{token.SEMICOLON, ";"},
		{token.RBRACE, "}"},
	})

	testScan(t, []elt{
		// {token.SELECTOR, ".color"},
		{token.STRING, ".color"},
		{token.LBRACE, "{"},
		{token.RULE, "color"},
		{token.COLON, ":"},
		{token.VAR, "$blah"},
		{token.RBRACE, "}"},
	})

	testScan(t, []elt{
		// {token.SELECTOR, ".color"},
		{token.STRING, ".color1"},
		{token.LBRACE, "{"},
		{token.RBRACE, "}"},
	})
}

func TestScan_nested(t *testing.T) {
	testScan(t, []elt{
		{token.AND, "&"},
		{token.STRING, ".goo"},
		{token.LBRACE, "{"},
		{token.STRING, "div"},
		{token.LBRACE, "{"},
		{token.RULE, "color"},
		{token.COLON, ":"},
		{token.COLOR, "#fff"},
		{token.SEMICOLON, ";"},
		{token.RBRACE, "}"},
		{token.RBRACE, "}"},
	})

	testScan(t, []elt{
		{token.STRING, "div"},
		{token.LBRACE, "{"},
		{token.AND, "&"},
		{token.LBRACE, "{"},
		{token.RULE, "color"},
		{token.COLON, ":"},
		{token.STRING, "red"},
		{token.SEMICOLON, ";"},
		{token.RBRACE, "}"},
		{token.RBRACE, "}"},
	})
}

func TestScan_media(t *testing.T) {
	testScan(t, []elt{
		{token.MEDIA, "@media"},
		{token.STRING, "print and (foo: 1 2 3)"},
		{token.LBRACE, "{"},
	})
}

func TestScan_duel(t *testing.T) {
	tokens := []byte(`$color;`)

	// error handler
	eh := func(_ token.Position, msg string) {
		t.Errorf("error handler called (msg = %s)", msg)
	}
	var s Scanner
	s.Init(fset.AddFile("", fset.Base(), len(tokens)), tokens, eh, ScanComments)
	_, tok, lit := s.Scan()
	if e := "$color"; e != lit {
		t.Fatalf("got: %s wanted: %s", lit, e)
	}
	_, tok, lit = s.Scan()
	if e := token.SEMICOLON; e != tok {
		t.Fatalf("got: %s wanted: %s", tok, e)
	}
	if e := ";"; e != lit {
		t.Fatalf("got: %s wanted: %s", lit, e)
	}
}

func TestScan_params(t *testing.T) {
	if false {
		testScan(t, []elt{
			{token.LPAREN, "("},
			{token.VAR, "$a"},
			{token.COMMA, ","},
			{token.VAR, "$b"},
			{token.COLON, ":"},
			{token.STRING, "flug"},
			{token.RPAREN, ")"},
		})
	}

	testScan(t, []elt{
		{token.LPAREN, "("},
		{token.VAR, "$y"},
		{token.COLON, ":"},
		{token.STRING, "kwd-y"},
		{token.COMMA, ","},
		{token.VAR, "$x"},
		{token.COLON, ":"},
		{token.STRING, "kwd-x"},
		{token.RPAREN, ")"},
	})

	testScan(t, []elt{
		{token.IDENT, "mix"},
		{token.LPAREN, "("},
		{token.IDENT, "rgba"},
		{token.LPAREN, "("},
		{token.INT, "255"},
		{token.COMMA, ","},
		{token.FLOAT, "0.5"},
		{token.COMMA, ","},
		{token.FLOAT, ".5"},
		{token.RPAREN, ")"},
	})
}

func TestScan_value(t *testing.T) {
	testScan(t, []elt{
		{token.VAR, "$x"},
		{token.STRING, "local"},
		{token.STRING, "x"},
		{token.STRING, "changed"},
		{token.STRING, "by"},
		{token.STRING, "foo"},
		{token.STRING, "!global"},
		{token.RBRACE, "}"},
	})

}

func TestScan_attr_sel_now(t *testing.T) {
	testScan(t, []elt{
		//{token.SELECTOR},
		{token.ATTRIBUTE, "[hey  =  'ho']"}, //"[hey  =  'ho'], a > b"
		{token.COMMENT, "/* end of hux */"},
		{token.LBRACE, "{"},
	})
}

func TestScan_string(t *testing.T) {

	testScan(t, []elt{
		{token.QSTRING, `"`},
		{token.STRING, "hello"},
		{token.INTERP, "#{"},
		{token.INT, "2"},
		{token.ADD, "+"},
		{token.INT, "3"},
		{token.RBRACE, "}"},
		{token.STRING, "blah"},
		{token.QSTRING, `"`},
		{token.SEMICOLON, `;`},
	})
}

func TestScan_math(t *testing.T) {
	testScan(t, []elt{
		{token.STRING, "d"},
		{token.QUO, "/"},
		{token.COLOR, "#eee"},
		{token.SEMICOLON, ";"},
	})
}

func TestScan_if(t *testing.T) {
	testScan(t, []elt{
		{token.IF, "@if"},
		{token.IDENT, "type-of"},
		{token.LPAREN, "("},
		{token.RPAREN, ")"},
		{token.EQL, "=="},
		{token.STRING, "number"},
		{token.LBRACE, "{"},
	})
}

func TestScan_interp(t *testing.T) {
	if false {
		testScanMap(t, "f#{$x}r {",
			[]elt{
				// {token.SELECTOR, "f#{$x}r {"},
				{token.STRING, "f"},
				{token.INTERP, "#{"},
				{token.VAR, "$x"},
				{token.RBRACE, "}"},
				{token.STRING, "r"},
				{token.LBRACE, "{"},
			})
	}

	testScan(t, []elt{
		// {token.SELECTOR, ""},
		{token.STRING, "f"},
		{token.INTERP, "#{"},
		{token.VAR, "$x"},
		{token.RBRACE, "}"},
		{token.STRING, "r"},
		{token.LBRACE, "{"},
	})

	testScan(t, []elt{
		{token.STRING, "hello"},
		{token.INTERP, "#{"}, // Sorry this is bizarre
		{token.STRING, "world"},
		{token.RBRACE, "}"},
		{token.SEMICOLON, ";"},
	})
	testScan(t, []elt{
		{token.INT, "123"},
		{token.INT, "1"},
		{token.INTERP, "#{"},
		{token.INT, "23"},
		{token.RBRACE, "}"},
	})
}

func TestScan_func(t *testing.T) {
	// testScan(t, []elt{
	// 	{token.IDENT, "inspect($value)"},
	// })
	// return
	testScan(t, []elt{
		{token.IDENT, "type-of"},
		{token.LPAREN, "("},
		{token.VAR, "$number"},
		{token.RPAREN, ")"},
	})
}

func TestScan_unit(t *testing.T) {
	testScan(t, []elt{
		{token.UPX, "5px"},
		{token.UEM, "5.1em"},
		{token.UCM, "5.1cm"},
		{token.UPCT, "3%"},
		{token.SEMICOLON, ";"},
	})

	testScan(t, []elt{
		{token.LPAREN, "("},
		{token.UPCT, "35%"},
		{token.QUO, "/"},
		{token.INT, "7"},
		{token.RPAREN, ")"},
	})
}

func testScan(t *testing.T, tokens []elt) {

	testScanMap(t, source(tokens), tokens)
}

func testScanMap(t *testing.T, v interface{}, tokens []elt) {

	var src []byte
	switch vv := v.(type) {
	case []byte:
		src = vv
	case string:
		src = []byte(vv)
	default:
		log.Fatal("unsupported")
	}

	whitespaceLinecount := newlineCount(whitespace)
	// error handler
	eh := func(_ token.Position, msg string) {
		t.Errorf("error handler called (msg = %s)", msg)
	}

	var s Scanner
	s.Init(fset.AddFile("", fset.Base(), len(src)), src, eh, ScanComments)

	epos := token.Position{
		Filename: "",
		Offset:   0,
		Line:     1,
		Column:   1,
	}

	index := 0
	for {
		pos, tok, lit := s.Scan()

		if tok == token.EOF {
			if len(whitespace) > 0 {
				epos.Line = newlineCount(string(src))
				epos.Column = 2
			}
		}
		// Selectors are magical and they appear when you least expect them
		switch tok {
		case token.SELECTOR:
			continue
		}

		// Some tokens don't respond with literals
		plit := lit
		if len(plit) == 0 {
			plit = tok.String()
		}
		checkPos(t, plit, pos, epos)

		// check token
		e := elt{token.EOF, ""}
		if index < len(tokens) {
			e = tokens[index]
			index++
		}
		if tok != e.tok {
			t.Errorf("bad token for %q: got %s, expected %s", lit, tok, e.tok)
		}
		// check literal
		elit := ""
		switch e.tok {
		case token.COMMENT:
			// no CRs in comments
			elit = string(stripCR([]byte(e.lit)))
			//-style comment literal doesn't contain newline
			if elit[1] == '/' {
				elit = elit[0 : len(elit)-1]
			}
		case token.ATTRIBUTE:
			// FIXME: the parser should eliminate whitespace
			// from selector attributes
			elit = strings.Replace(e.lit, " ", "", -1)
		case token.IDENT:
			elit = e.lit
		case token.SEMICOLON:
			elit = ";"
		default:
			if e.tok.IsLiteral() {
				// no CRs in raw string literals
				elit = e.lit
				if elit[0] == '`' {
					elit = string(stripCR([]byte(elit)))
				}
			} else if e.tok.IsKeyword() || e.tok.IsCSSNum() {
				elit = e.lit
			}
		}

		if lit != elit {
			t.Errorf("bad literal for %q: got %q, expected %q",
				lit, lit, elit)
		}

		if tok == token.EOF {
			break
		}

		// update position
		epos.Offset += len(e.lit) + len(whitespace)
		epos.Line += newlineCount(e.lit) + whitespaceLinecount

		if len(whitespace) == 0 {
			epos.Column += len(e.lit)
		}
	}
}

func TestScan_decl(t *testing.T) {
	testScan(t, []elt{
		{token.IF, "@if"},
		{token.IDENT, "type-of"},
		{token.LPAREN, "("},
		{token.IDENT, "nth"},
		{token.LPAREN, "("},
		{token.VAR, "$x"},
		{token.COMMA, ","},
		{token.INT, "3"},
		{token.RPAREN, ")"},
		{token.RPAREN, ")"},
		{token.EQL, "=="},
		{token.STRING, "number"},
		{token.LBRACE, "{"},
	})

	testScan(t, []elt{
		{token.ELSEIF, "@else if"},
		{token.STRING, "asdf"},
		{token.EQL, "=="},
		{token.STRING, "string"},
		{token.LBRACE, "{"},
		{token.RBRACE, "}"},
		{token.ELSE, "@else"},
		{token.LBRACE, "{"},
	})

	testScan(t, []elt{
		{token.IF, "@if"},
		{token.VAR, "$x"},
		{token.LBRACE, "{"},
		{token.VAR, "$x"},
		{token.COLON, ":"},
		{token.STRING, "false"},
		{token.STRING, "!global"},
		{token.SEMICOLON, ";"},
		{token.RETURN, "@return"},
		{token.STRING, "foo"},
		{token.SEMICOLON, ";"},
	})

	// @function foobar() {
	// 	@if $x {
	// 		$x: false !global;
	// 		@return foo;
	// 	}
	// }

}
