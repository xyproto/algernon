package parser

import (
	"testing"

	"fmt"
	"github.com/mamaar/risotto/file"
	"github.com/mamaar/risotto/token"
	"github.com/stretchr/testify/assert"
)

func TestLexer(t *testing.T) {
	setup := func(src string) *_parser {
		parser := newParser("", src)
		return parser
	}

	test := func(src string, test ...interface{}) {
		parser := setup(src)
		for len(test) > 0 {
			tkn, literal, idx := parser.scan()
			if len(test) > 0 {
				expTkn := test[0].(token.Token)
				assert.Equal(t, expTkn, tkn,
					fmt.Sprintf("Token %v != %v", expTkn, tkn))
				test = test[1:]
			}
			if len(test) > 0 {
				expLit := test[0].(string)
				assert.Equal(t, expLit, literal,
					fmt.Sprintf("Literal %v != %v", expLit, literal))
				test = test[1:]
			}
			if len(test) > 0 {
				expIdx := file.Idx(test[0].(int))
				assert.Equal(t, expIdx, idx,
					fmt.Sprintf("Idx %v != %v", expIdx, idx))

				test = test[1:]
			}
		}
	}

	test("",
		token.EOF, "", 1,
	)

	test("1",
		token.NUMBER, "1", 1,
		token.EOF, "", 2,
	)

	test(".0",
		token.NUMBER, ".0", 1,
		token.EOF, "", 3,
	)

	test("abc",
		token.IDENTIFIER, "abc", 1,
		token.EOF, "", 4,
	)

	test("abc(1)",
		token.IDENTIFIER, "abc", 1,
		token.LEFT_PARENTHESIS, "", 4,
		token.NUMBER, "1", 5,
		token.RIGHT_PARENTHESIS, "", 6,
		token.EOF, "", 7,
	)

	test(".",
		token.PERIOD, "", 1,
		token.EOF, "", 2,
	)

	test("===.",
		token.STRICT_EQUAL, "", 1,
		token.PERIOD, "", 4,
		token.EOF, "", 5,
	)

	test(">>>=.0",
		token.UNSIGNED_SHIFT_RIGHT_ASSIGN, "", 1,
		token.NUMBER, ".0", 5,
		token.EOF, "", 7,
	)

	test(">>>=0.0.",
		token.UNSIGNED_SHIFT_RIGHT_ASSIGN, "", 1,
		token.NUMBER, "0.0", 5,
		token.PERIOD, "", 8,
		token.EOF, "", 9,
	)

	test("\"abc\"",
		token.STRING, "\"abc\"", 1,
		token.EOF, "", 6,
	)

	test("abc = //",
		token.IDENTIFIER, "abc", 1,
		token.WHITESPACE, " ", 4,
		token.ASSIGN, "", 5,
		token.WHITESPACE, " ", 6,
		token.EOF, "", 9,
	)

	test("abc = 1 / 2",
		token.IDENTIFIER, "abc", 1,
		token.WHITESPACE, " ", 4,
		token.ASSIGN, "", 5,
		token.WHITESPACE, " ", 6,
		token.NUMBER, "1", 7,
		token.WHITESPACE, " ", 8,
		token.SLASH, "", 9,
		token.WHITESPACE, " ", 10,
		token.NUMBER, "2", 11,
		token.EOF, "", 12,
	)

	test("xyzzy = 'Nothing happens.'",
		token.IDENTIFIER, "xyzzy", 1,
		token.WHITESPACE, " ", 6,
		token.ASSIGN, "", 7,
		token.WHITESPACE, " ", 8,
		token.STRING, "'Nothing happens.'", 9,
		token.EOF, "", 27,
	)

	test("abc = !false",
		token.IDENTIFIER, "abc", 1,
		token.WHITESPACE, " ", 4,
		token.ASSIGN, "", 5,
		token.WHITESPACE, " ", 6,
		token.NOT, "", 7,
		token.BOOLEAN, "false", 8,
		token.EOF, "", 13,
	)

	test("abc = !!true",
		token.IDENTIFIER, "abc", 1,
		token.WHITESPACE, " ", 4,
		token.ASSIGN, "", 5,
		token.WHITESPACE, " ", 6,
		token.NOT, "", 7,
		token.NOT, "", 8,
		token.BOOLEAN, "true", 9,
		token.EOF, "", 13,
	)

	test("abc *= 1",
		token.IDENTIFIER, "abc", 1,
		token.WHITESPACE, " ", 4,
		token.MULTIPLY_ASSIGN, "", 5,
		token.WHITESPACE, " ", 7,
		token.NUMBER, "1", 8,
		token.EOF, "", 9,
	)

	test("if 1 else",
		token.IF, "if", 1,
		token.WHITESPACE, " ", 3,
		token.NUMBER, "1", 4,
		token.WHITESPACE, " ", 5,
		token.ELSE, "else", 6,
		token.EOF, "", 10,
	)

	test("null",
		token.NULL, "null", 1,
		token.EOF, "", 5,
	)

	test(`"\u007a\x79\u000a\x78"`,
		token.STRING, "\"\\u007a\\x79\\u000a\\x78\"", 1,
		token.EOF, "", 23,
	)

	test(`"[First line \
Second line \
 Third line\
.     ]"`,
		token.STRING, "\"[First line \\\nSecond line \\\n Third line\\\n.     ]\"", 1,
		token.EOF, "", 51,
	)

	test("/",
		token.SLASH, "", 1,
		token.EOF, "", 2,
	)

	test("var abc = \"abc\uFFFFabc\"",
		token.VAR, "var", 1,
		token.WHITESPACE, " ", 4,
		token.IDENTIFIER, "abc", 5,
		token.WHITESPACE, " ", 8,
		token.ASSIGN, "", 9,
		token.WHITESPACE, " ", 10,
		token.STRING, "\"abc\uFFFFabc\"", 11,
		token.EOF, "", 22,
	)

	test(`'\t' === '\r'`,
		token.STRING, "'\\t'", 1,
		token.WHITESPACE, " ", 5,
		token.STRICT_EQUAL, "", 6,
		token.WHITESPACE, " ", 9,
		token.STRING, "'\\r'", 10,
		token.EOF, "", 14,
	)

	test(`var \u0024 = 1`,
		token.VAR, "var", 1,
		token.WHITESPACE, " ", 4,
		token.IDENTIFIER, "$", 5,
		token.WHITESPACE, " ", 11,
		token.ASSIGN, "", 12,
		token.WHITESPACE, " ", 13,
		token.NUMBER, "1", 14,
		token.EOF, "", 15,
	)

	test("10e10000",
		token.NUMBER, "10e10000", 1,
		token.EOF, "", 9,
	)

	test(`var if var class`,
		token.VAR, "var", 1,
		token.WHITESPACE, " ", 4,
		token.IF, "if", 5,
		token.WHITESPACE, " ", 7,
		token.VAR, "var", 8,
		token.WHITESPACE, " ", 11,
		token.KEYWORD, "class", 12,
		token.EOF, "", 17,
	)

	test(`-0`,
		token.MINUS, "", 1,
		token.NUMBER, "0", 2,
		token.EOF, "", 3,
	)

	test(`.01`,
		token.NUMBER, ".01", 1,
		token.EOF, "", 4,
	)

	test(`.01e+2`,
		token.NUMBER, ".01e+2", 1,
		token.EOF, "", 7,
	)

	test(";",
		token.SEMICOLON, "", 1,
		token.EOF, "", 2,
	)

	test(";;",
		token.SEMICOLON, "", 1,
		token.SEMICOLON, "", 2,
		token.EOF, "", 3,
	)

	test("//",
		token.EOF, "", 3,
	)

	test(";;//",
		token.SEMICOLON, "", 1,
		token.SEMICOLON, "", 2,
		token.EOF, "", 5,
	)

	test("1",
		token.NUMBER, "1", 1,
	)

	test("12 123",
		token.NUMBER, "12", 1,
		token.WHITESPACE, " ", 3,
		token.NUMBER, "123", 4,
	)

	test("1.2 12.3",
		token.NUMBER, "1.2", 1,
		token.WHITESPACE, " ", 4,
		token.NUMBER, "12.3", 5,
	)

	test("/ /=",
		token.SLASH, "", 1,
		token.WHITESPACE, " ", 2,
		token.QUOTIENT_ASSIGN, "", 3,
	)

	test(`"abc"`,
		token.STRING, `"abc"`, 1,
	)

	test(`'abc'`,
		token.STRING, `'abc'`, 1,
	)

	test("++",
		token.INCREMENT, "", 1,
	)

	test(">",
		token.GREATER, "", 1,
	)

	test(">=",
		token.GREATER_OR_EQUAL, "", 1,
	)

	test(">>",
		token.SHIFT_RIGHT, "", 1,
	)

	test(">>=",
		token.SHIFT_RIGHT_ASSIGN, "", 1,
	)

	test(">>>",
		token.UNSIGNED_SHIFT_RIGHT, "", 1,
	)

	test(">>>=",
		token.UNSIGNED_SHIFT_RIGHT_ASSIGN, "", 1,
	)

	test("1 \"abc\"",
		token.NUMBER, "1", 1,
		token.WHITESPACE, " ", 2,
		token.STRING, "\"abc\"", 3,
	)

	test(",",
		token.COMMA, "", 1,
	)

	test("1, \"abc\"",
		token.NUMBER, "1", 1,
		token.COMMA, "", 2,
		token.WHITESPACE, " ", 3,
		token.STRING, "\"abc\"", 4,
	)

	test("new abc(1, 3.14159);",
		token.NEW, "new", 1,
		token.WHITESPACE, " ", 4,
		token.IDENTIFIER, "abc", 5,
		token.LEFT_PARENTHESIS, "", 8,
		token.NUMBER, "1", 9,
		token.COMMA, "", 10,
		token.WHITESPACE, " ", 11,
		token.NUMBER, "3.14159", 12,
		token.RIGHT_PARENTHESIS, "", 19,
		token.SEMICOLON, "", 20,
	)

	test("1 == \"1\"",
		token.NUMBER, "1", 1,
		token.WHITESPACE, " ", 2,
		token.EQUAL, "", 3,
		token.WHITESPACE, " ", 5,
		token.STRING, "\"1\"", 6,
	)

	test("1\n[]\n",
		token.NUMBER, "1", 1,
		token.WHITESPACE, "\n", 2,
		token.LEFT_BRACKET, "", 3,
		token.RIGHT_BRACKET, "", 4,
		token.WHITESPACE, "\n", 5,
	)

	test("1\ufeff[]\ufeff",
		token.NUMBER, "1", 1,
		token.WHITESPACE, "\ufeff", 2,
		token.LEFT_BRACKET, "", 5,
		token.RIGHT_BRACKET, "", 6,
		token.WHITESPACE, "\ufeff", 7,
	)

	// ILLEGAL

	test(`3ea`,
		token.ILLEGAL, "3e", 1,
		token.IDENTIFIER, "a", 3,
		token.EOF, "", 4,
	)

	test(`3in`,
		token.ILLEGAL, "3", 1,
		token.IN, "in", 2,
		token.EOF, "", 4,
	)

	test("\"Hello\nWorld\"",
		token.ILLEGAL, "", 1,
		token.IDENTIFIER, "World", 8,
		token.ILLEGAL, "", 13,
		token.EOF, "", 14,
	)

	test("\u203f = 10",
		token.ILLEGAL, "", 1,
		token.WHITESPACE, " ", 4,
		token.ASSIGN, "", 5,
		token.WHITESPACE, " ", 6,
		token.NUMBER, "10", 7,
		token.EOF, "", 9,
	)

	test(`"\x0G"`,
		token.STRING, "\"\\x0G\"", 1,
		token.EOF, "", 7,
	)

	test("<div />",
		token.LESS, "", 1,
		token.IDENTIFIER, "div", 2,
		token.WHITESPACE, " ", 5,
		token.SLASH, "", 6,
		token.GREATER, "", 7,
		token.EOF, "", 8)

	test("<div param=\"value\"></div>",
		token.LESS, "", 1,
		token.IDENTIFIER, "div", 2,
		token.WHITESPACE, " ", 5,
		token.IDENTIFIER, "param", 6,
		token.ASSIGN, "", 11,
		token.STRING, "\"value\"", 12,
		token.GREATER, "", 19,
		token.LESS, "", 20,
		token.SLASH, "", 21,
		token.IDENTIFIER, "div", 22,
		token.GREATER, "", 25,
		token.EOF, "", 26)

	test(`a b  c
`,
		token.IDENTIFIER, "a", 1,
		token.WHITESPACE, " ", 2,
		token.IDENTIFIER, "b", 3,
		token.WHITESPACE, "  ", 4,
		token.IDENTIFIER, "c", 6,
		token.WHITESPACE, "\n", 7,
		token.EOF, "", 8)

}
