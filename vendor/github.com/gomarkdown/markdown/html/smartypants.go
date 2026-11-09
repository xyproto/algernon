package html

import (
	"bytes"
	"io"

	"github.com/gomarkdown/markdown/parser"
)

// SmartyPants rendering

var (
	isSpace       = parser.IsSpace
	isPunctuation = parser.IsPunctuation
)

// SPRenderer is a struct containing state of a Smartypants renderer.
type SPRenderer struct {
	inSingleQuote bool
	inDoubleQuote bool
	callbacks     [256]smartCallback
	tmp           bytes.Buffer // reused by Process for each callback's output
}

func wordBoundary(c byte) bool { return c == 0 || isSpace(c) || isPunctuation(c) }

// byteAt returns text[i], or 0 when i is outside text.
func byteAt(text []byte, i int) byte {
	if i < 0 || i >= len(text) {
		return 0
	}
	return text[i]
}

func tolower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c - 'A' + 'a'
	}
	return c
}

func isdigit(c byte) bool { return c >= '0' && c <= '9' }

func quoteContext(c byte) int {
	switch {
	case c == 0:
		return 0
	case isSpace(c):
		return 1
	case isPunctuation(c):
		return 2
	default:
		return 3
	}
}

// quoteState maps {edge, space, punctuation, text} context pairs to
// {-1: close, 0: toggle, 1: open}.
var quoteState = [4][4]int8{
	{0, -1, -1, 1},
	{1, 0, 1, 1},
	{-1, -1, 0, 1},
	{-1, -1, -1, -1},
}

func smartQuote(out *bytes.Buffer, previous, next, quote byte, isOpen *bool, addNBSP bool) {
	switch quoteState[quoteContext(previous)][quoteContext(next)] {
	case -1:
		*isOpen = false
	case 0:
		*isOpen = !*isOpen
	case 1:
		*isOpen = true
	}

	// Note that with the limited lookahead, this non-breaking
	// space will also be appended to single double quotes.
	if addNBSP && !*isOpen {
		out.WriteString("&nbsp;")
	}

	out.WriteByte('&')
	if *isOpen {
		out.WriteByte('l')
	} else {
		out.WriteByte('r')
	}
	out.WriteByte(quote)
	out.WriteString("quo;")

	if addNBSP && *isOpen {
		out.WriteString("&nbsp;")
	}

}

func (r *SPRenderer) smartSingleQuote(out *bytes.Buffer, previousChar byte, text []byte) int {
	if len(text) >= 2 {
		t1 := tolower(text[1])

		if t1 == '\'' {
			return r.smartDoubledQuote(out, previousChar, text)
		}

		for _, suffix := range [...]string{"s", "t", "m", "d", "re", "ll", "ve"} {
			end := 1 + len(suffix)
			if len(text) >= end && bytes.EqualFold(text[1:end], []byte(suffix)) &&
				(len(text) == end || wordBoundary(text[end])) {
				out.WriteString("&rsquo;")
				return 0
			}
		}
	}

	return quoteToken(out, previousChar, text, 1, 's', &r.inSingleQuote, false)
}

// smartToken is a run of characters that smartypants writes as an entity.
type smartToken struct{ token, entity string }

var (
	parenTokens  = []smartToken{{"(c)", "&copy;"}, {"(r)", "&reg;"}, {"(tm)", "&trade;"}}
	latexDashes  = []smartToken{{"---", "&mdash;"}, {"--", "&ndash;"}}
	periodTokens = []smartToken{{"...", "&hellip;"}, {". . .", "&hellip;"}}
)

// smartReplace writes the entity of the first token that text starts with,
// ignoring case, and returns how many bytes after text[0] it used. Without
// a match it writes text[0] and returns 0.
func smartReplace(out *bytes.Buffer, text []byte, tokens []smartToken) int {
	for _, t := range tokens {
		if len(text) >= len(t.token) && bytes.EqualFold(text[:len(t.token)], []byte(t.token)) {
			out.WriteString(t.entity)
			return len(t.token) - 1
		}
	}
	out.WriteByte(text[0])
	return 0
}

// tokenCallback returns a callback that writes the entity of the first of
// tokens that the text starts with.
func tokenCallback(tokens []smartToken) smartCallback {
	return func(out *bytes.Buffer, _ byte, text []byte) int {
		return smartReplace(out, text, tokens)
	}
}

func (r *SPRenderer) smartDash(out *bytes.Buffer, previousChar byte, text []byte) int {
	if len(text) >= 2 {
		if text[1] == '-' {
			out.WriteString("&mdash;")
			return 1
		}

		if wordBoundary(previousChar) && wordBoundary(text[1]) {
			out.WriteString("&ndash;")
			return 0
		}
	}

	out.WriteByte(text[0])
	return 0
}

func (r *SPRenderer) smartAmpVariant(out *bytes.Buffer, previousChar byte, text []byte, quote byte, addNBSP bool) int {
	if bytes.HasPrefix(text, []byte("&quot;")) {
		return quoteToken(out, previousChar, text, len("&quot;"), quote, &r.inDoubleQuote, addNBSP)
	}

	if bytes.HasPrefix(text, []byte("&#0;")) {
		return 3
	}

	out.WriteByte('&')
	return 0
}

// smartDoubledQuote writes a doubled quote mark (two apostrophes or two
// backticks) at the start of text as a double quote, and returns 1 to skip
// its second mark.
func (r *SPRenderer) smartDoubledQuote(out *bytes.Buffer, previousChar byte, text []byte) int {
	return quoteToken(out, previousChar, text, 2, 'd', &r.inDoubleQuote, false)
}

// quoteToken writes the quote mark that the n-byte token at the start of
// text stands for, and returns n-1, the bytes it used after text[0].
func quoteToken(out *bytes.Buffer, previous byte, text []byte, n int, quote byte, isOpen *bool, addNBSP bool) int {
	smartQuote(out, previous, byteAt(text, n), quote, isOpen, addNBSP)
	return n - 1
}

func (r *SPRenderer) smartBacktick(out *bytes.Buffer, previousChar byte, text []byte) int {
	if len(text) >= 2 && text[1] == '`' {
		return r.smartDoubledQuote(out, previousChar, text)
	}

	out.WriteByte(text[0])
	return 0
}

func (r *SPRenderer) smartNumberGeneric(out *bytes.Buffer, previousChar byte, text []byte) int {
	if wordBoundary(previousChar) && previousChar != '/' && len(text) >= 3 {
		// is it of the form digits/digits(word boundary)?, i.e., \d+/\d+\b
		// note: check for regular slash (/) or fraction slash (⁄, 0x2044, or 0xe2 81 84 in utf-8)
		//       and avoid changing dates like 1/23/2005 into fractions.
		numEnd := 0
		for len(text) > numEnd && isdigit(text[numEnd]) {
			numEnd++
		}
		if numEnd == 0 {
			out.WriteByte(text[0])
			return 0
		}
		denStart := numEnd + 1
		if len(text) > numEnd+3 && text[numEnd] == 0xe2 && text[numEnd+1] == 0x81 && text[numEnd+2] == 0x84 {
			denStart = numEnd + 3
		} else if len(text) < numEnd+2 || text[numEnd] != '/' {
			out.WriteByte(text[0])
			return 0
		}
		denEnd := denStart
		for len(text) > denEnd && isdigit(text[denEnd]) {
			denEnd++
		}
		if denEnd == denStart {
			out.WriteByte(text[0])
			return 0
		}
		if len(text) == denEnd || wordBoundary(text[denEnd]) && text[denEnd] != '/' {
			out.WriteString("<sup>")
			out.Write(text[:numEnd])
			out.WriteString("</sup>&frasl;<sub>")
			out.Write(text[denStart:denEnd])
			out.WriteString("</sub>")
			return denEnd - 1
		}
	}

	out.WriteByte(text[0])
	return 0
}

func (r *SPRenderer) smartNumber(out *bytes.Buffer, previousChar byte, text []byte) int {
	if wordBoundary(previousChar) && previousChar != '/' && len(text) >= 3 {
		for _, fraction := range [...]struct{ token, suffix, entity string }{
			{"1/2", "", "&frac12;"},
			{"1/4", "th", "&frac14;"},
			{"3/4", "ths", "&frac34;"},
		} {
			if string(text[:3]) != fraction.token {
				continue
			}
			boundary := len(text) == 3 || wordBoundary(text[3]) && text[3] != '/'
			suffix := text[3:]
			if boundary || fraction.suffix != "" && len(suffix) >= len(fraction.suffix) && bytes.EqualFold(suffix[:len(fraction.suffix)], []byte(fraction.suffix)) {
				out.WriteString(fraction.entity)
				return 2
			}
		}
	}

	out.WriteByte(text[0])
	return 0
}

func (r *SPRenderer) smartLeftAngle(out *bytes.Buffer, previousChar byte, text []byte) int {
	i := bytes.IndexByte(text, '>')
	if i < 0 {
		return len(text)
	}
	out.Write(text[:i+1])
	return i
}

type smartCallback func(out *bytes.Buffer, previousChar byte, text []byte) int

// NewSmartypantsRenderer constructs a Smartypants renderer object.
func NewSmartypantsRenderer(flags Flags) *SPRenderer {
	var r SPRenderer
	angled := flags&SmartypantsAngledQuotes != 0
	quote := byte('d')
	if angled {
		quote = 'a'
	}
	r.callbacks['"'] = func(out *bytes.Buffer, previous byte, text []byte) int {
		return quoteToken(out, previous, text, 1, quote, &r.inDoubleQuote, false)
	}
	addNBSP := flags&SmartypantsQuotesNBSP != 0
	r.callbacks['&'] = func(out *bytes.Buffer, previous byte, text []byte) int {
		return r.smartAmpVariant(out, previous, text, quote, addNBSP)
	}
	r.callbacks['\''] = r.smartSingleQuote
	r.callbacks['('] = tokenCallback(parenTokens)
	if flags&SmartypantsDashes != 0 {
		if flags&SmartypantsLatexDashes == 0 {
			r.callbacks['-'] = r.smartDash
		} else {
			r.callbacks['-'] = tokenCallback(latexDashes)
		}
	}
	r.callbacks['.'] = tokenCallback(periodTokens)
	if flags&SmartypantsFractions == 0 {
		r.callbacks['1'] = r.smartNumber
		r.callbacks['3'] = r.smartNumber
	} else {
		for ch := '1'; ch <= '9'; ch++ {
			r.callbacks[ch] = r.smartNumberGeneric
		}
	}
	r.callbacks['<'] = r.smartLeftAngle
	r.callbacks['`'] = r.smartBacktick
	return &r
}

// Process is the entry point of the Smartypants renderer.
func (r *SPRenderer) Process(w io.Writer, text []byte) {
	mark := 0
	for i := 0; i < len(text); i++ {
		if action := r.callbacks[text[i]]; action != nil {
			if i > mark {
				w.Write(text[mark:i])
			}
			r.tmp.Reset()
			i += action(&r.tmp, byteAt(text, i-1), text[i:])
			w.Write(r.tmp.Bytes())
			mark = i + 1
		}
	}
	if mark < len(text) {
		w.Write(text[mark:])
	}
}
