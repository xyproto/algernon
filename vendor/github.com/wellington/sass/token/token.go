package token

// A type for all the types of items in the language being lexed.
// These only parse SASS specific language elements and not CSS.
type Token int

// const ItemEOF = 0
const NotFound = -1

// Special item types.
const (
	ILLEGAL Token = iota
	EOF
	COMMENT
	LINECOMMENT

	literal_beg
	// Identifiers
	IDENT
	VAR // $var
	INT
	FLOAT
	TEXT
	SELECTOR // a { rules... }
	RULE
	STRING    // word
	COLOR     // #000
	INTERP    // #{value}
	VALUE     // value (rhs of rule)
	ATTRIBUTE // [disabled] [type='button']
	PSEUDO    // :first-child :nth-last-child
	AND       // & backreference
	literal_end

	cssnums_beg
	UIN // 1in
	UCM // 2.54cm
	UMM // 25.4mm

	UPC // 1pc

	UPX // 16px
	UPT // 16pt

	DEG  // 1deg
	GRAD // 1grad
	RAD  // 1rad
	TURN // 1turn

	UEM  // 1em
	UREM // 1rem
	UPCT // 10%
	cssnums_end

	operator_beg
	QSTRING  // "word"
	QSSTRING // 'word'
	selector_beg
	// Are these necessary?
	// BACKREF // &
	TIL // ~
	selector_end
	ADD // +
	SUB // -
	MUL // *
	QUO // /
	REM // %

	OR      // |
	XOR     // ^
	SHL     // <<
	SHR     // >>
	AND_NOT // &^

	LAND  // &&
	LOR   // ||
	ARROW // <-
	INC   // ++
	DEC   // --

	EQL    // ==
	LSS    // <
	GTR    // >
	ASSIGN // =
	NOT    // !
	NEST   // div span

	NEQ      // !=
	LEQ      // <=
	GEQ      // >=
	DEFINE   // :=
	ELLIPSIS // ...

	AT     // @
	DOLLAR // $
	QUOTE  // "

	LPAREN // (
	LBRACK // [
	LBRACE // {
	COMMA  // ,
	PERIOD // .

	RPAREN    // )
	RBRACK    // ]
	RBRACE    // }
	SEMICOLON // ;
	COLON     // :
	operator_end

	keyword_beg
	IF      // @if
	ELSE    // @else
	ELSEIF  // @elseif
	EACH    // @each
	INCLUDE // @include
	FOR     // @for
	FUNC    // @function
	MIXIN   // @mixin
	RETURN  // @return
	WHILE   // @while

	// Directives
	IMPORT // @import
	MEDIA  // @media
	EXTEND // @extend
	ATROOT // @at-root
	DEBUG  // @debug
	WARN   // @warn
	ERROR  // @error
	keyword_end

	CMDVAR

	cmd_beg
	SPRITE
	SPRITEF
	SPRITED
	SPRITEH
	SPRITEW
	cmd_end

	include_mixin_beg
	FILE
	BKND
	include_mixin_end
	FIN
)

var Tokens = [...]string{
	ILLEGAL:     "ILLEGAL",
	EOF:         "EOF",
	COMMENT:     "comment",
	LINECOMMENT: "linecomment",

	IDENT:    "IDENT",
	INT:      "INT",
	FLOAT:    "FLOAT",
	VAR:      "VAR",
	RULE:     "rule",
	STRING:   "string",
	QSTRING:  `quote`,
	QSSTRING: `singlequote`,
	COLOR:    "color",
	INTERP:   "INTERPOLATION",
	// Selector tokens
	ATTRIBUTE: "attribute",
	// BACKREF: "&",
	PSEUDO: "pseudo-selector",

	TEXT:     "text",
	SELECTOR: "selector",

	UIN: "in",
	UCM: "cm",
	UMM: "mm",
	UPC: "pc",
	UPX: "px",
	UPT: "pt",

	DEG:  "deg",
	GRAD: "grad",
	RAD:  "rad",
	TURN: "turn",

	UEM:  "em",
	UREM: "rem",
	UPCT: "pct",

	CMDVAR:  "command-variable",
	VALUE:   "value",
	FILE:    "file",
	SPRITE:  "sprite",
	SPRITEF: "sprite-file",
	SPRITED: "sprite-dimensions",
	SPRITEH: "sprite-height",
	SPRITEW: "sprite-width",

	NEST: "nest",
	TIL:  "~",
	ADD:  "+",
	SUB:  "-",
	MUL:  "*",
	QUO:  "/",
	REM:  "%",

	AND: "&",
	//OR: "|",
	XOR: "^",

	AT:     "@",
	EQL:    "==",
	LSS:    "<",
	GTR:    ">",
	ASSIGN: "=",
	NOT:    "!",

	DOLLAR: "$",

	NEQ:    "!=",
	LEQ:    "<=",
	GEQ:    ">=",
	DEFINE: ":=",

	LPAREN: "(",
	LBRACK: "[",
	LBRACE: "{",
	COMMA:  ",",
	PERIOD: ".",

	RPAREN:    ")",
	RBRACK:    "]",
	RBRACE:    "}",
	SEMICOLON: ";",
	COLON:     ":",

	QUOTE: "\"",

	IF:      "@if",
	ELSE:    "@else",
	ELSEIF:  "@else if",
	FOR:     "@for",
	EACH:    "@each",
	INCLUDE: "@include",
	FUNC:    "@function",
	MIXIN:   "@mixin",
	RETURN:  "@return",
	WHILE:   "$while",

	IMPORT: "@import",
	MEDIA:  "@media",
	EXTEND: "@extend",
	ATROOT: "@at-root",
	DEBUG:  "@debug",
	WARN:   "@warn",
	ERROR:  "@error",

	BKND: "background",
	FIN:  "FINISHED",
}

func (tok Token) String() string {
	if tok < 0 {
		return ""
	}
	return Tokens[tok]
}

var directives map[string]Token

func init() {
	directives = make(map[string]Token)
	for i := cmd_beg; i < cmd_end; i++ {
		directives[Tokens[i]] = i
	}
}

// Lookup Token by token string
func Lookup(ident string) Token {
	if tok, is_keyword := directives[ident]; is_keyword {
		return tok
	}
	return NotFound
}

// A set of constants for precedence-based expression parsing.
// Non-operators have lowest precedence, followed by operators
// starting with precedence 1 up to unary operators. The highest
// precedence serves as "catch-all" precedence for selector,
// indexing, and other operator and delimiter tokens.
//
const (
	LowestPrec  = 0 // non-operators
	UnaryPrec   = 6
	HighestPrec = 7
)

func (op Token) SelPrecedence() int {
	switch op {
	case COMMA:
		return 3
	case ADD, GTR, TIL:
		return 4
	case NEST:
		return 5
	}
	return LowestPrec
}

// Precedence returns the operator precedence of the binary
// operator op. If op is not a binary operator, the result
// is LowestPrecedence.
//
func (tok Token) Precedence() int {
	switch tok {
	case LOR:
		return 1
	case LAND:
		return 2
	case EQL, NEQ, LSS, LEQ, GTR, GEQ:
		return 3
	case ADD, SUB, OR, XOR:
		return 4
	case MUL, QUO, REM, SHL, SHR, AND, AND_NOT:
		return 5
	}
	return LowestPrec
}

// Predicates

// IsLiteral returns true for tokens corresponding to identifiers
// and basic type literals; it returns false otherwise.
//
func (tok Token) IsLiteral() bool { return literal_beg < tok && tok < literal_end }

// IsOperator returns true for tokens corresponding to operators and
// delimiters; it returns false otherwise.
//
func (tok Token) IsOperator() bool { return operator_beg < tok && tok < operator_end }

// IsKeyword returns true for tokens corresponding to keywords;
// it returns false otherwise.
//
func (tok Token) IsKeyword() bool { return keyword_beg < tok && tok < keyword_end }

// IsCSSNum returns true if token correponding to cssnums; it returns
// false otherwise
//
func (tok Token) IsCSSNum() bool { return cssnums_beg < tok && tok < cssnums_end }
