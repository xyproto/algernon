package parser

import "github.com/gomarkdown/markdown/ast"

// Extensions is a bitmask of enabled parser extensions.
type Extensions int

// Bit flags representing markdown parsing extensions.
// Use | (or) to specify multiple extensions.
const (
	NoExtensions           Extensions = 0
	NoIntraEmphasis        Extensions = 1 << iota // Ignore emphasis markers inside words
	Tables                                        // Parse tables
	FencedCode                                    // Parse fenced code blocks
	Autolink                                      // Detect embedded URLs that are not explicitly marked
	Strikethrough                                 // Strikethrough text using ~~test~~
	LaxHTMLBlocks                                 // Loosen up HTML block parsing rules
	SpaceHeadings                                 // Be strict about prefix heading rules
	HardLineBreak                                 // Translate newlines into line breaks
	NonBlockingSpace                              // Translate backslash-space into a non-breaking space
	TabSizeEight                                  // Expand tabs to eight spaces instead of four
	Footnotes                                     // Pandoc-style footnotes
	NoEmptyLineBeforeBlock                        // No need to insert an empty line to start a (code, quote, ordered list, unordered list) block
	HeadingIDs                                    // Specify heading IDs with {#id}
	Titleblock                                    // Titleblock ala pandoc
	AutoHeadingIDs                                // Create the heading ID from the text
	BackslashLineBreak                            // Translate trailing backslashes into line breaks
	DefinitionLists                               // Parse definition lists
	MathJax                                       // Parse MathJax
	OrderedListStart                              // Keep track of the first number used when starting an ordered list.
	Attributes                                    // Block Attributes
	SuperSubscript                                // Super- and subscript support: 2^10^, H~2~O.
	EmptyLinesBreakList                           // 2 empty lines break out of list
	Includes                                      // Support including other files.
	Mmark                                         // Support Mmark syntax, see https://mmark.miek.nl/post/syntax/
	InlineAttributes                              // Parse {: key="value"} after links and images
	MarkdownInHTML                                // Parse markdown inside HTML table cells and similar tags

	CommonExtensions Extensions = NoIntraEmphasis | Tables | FencedCode |
		Autolink | Strikethrough | SpaceHeadings | HeadingIDs |
		BackslashLineBreak | DefinitionLists | MathJax
)

const (
	tabSizeDefault = 4
	tabSizeDouble  = 8
)

// InlineParser parses inline data beginning at offset.
type InlineParser func(p *Parser, data []byte, offset int) (int, ast.Node)

// ReferenceOverrideFunc resolves a reference, or declines by returning
// overridden false so the parser can use its document-defined references.
type ReferenceOverrideFunc func(reference string) (ref *Reference, overridden bool)

// New creates a markdown parser with CommonExtensions.
func New() *Parser {
	return NewWithExtensions(CommonExtensions)
}

// NewWithExtensions creates a markdown parser with the given extensions.
func NewWithExtensions(extension Extensions) *Parser {
	doc := &ast.Document{}
	p := Parser{
		refs:                 make(map[string]*reference),
		refsRecord:           make(map[string]struct{}),
		maxNesting:           64,
		Doc:                  doc,
		extensions:           extension,
		tip:                  doc,
		oldTip:               doc,
		lastMatchedContainer: doc,
		allClosed:            true,
		includeStack:         newIncStack(),
	}

	p.registerInline(" ", maybeLineBreak)
	p.registerInline("*_", emphasis)
	if extension&Strikethrough != 0 {
		p.registerInline("~", emphasis)
	}
	p.registerInline("`", codeSpan)
	p.registerInline("\n", lineBreak)
	p.registerInline("[", link)
	p.registerInline("<", leftAngle)
	p.registerInline("\\", escape)
	p.registerInline("&", entity)
	p.registerInline("!", maybeImage)
	if extension&Mmark != 0 {
		p.registerInline("(", maybeShortRefOrIndex)
	}
	p.registerInline("^", maybeInlineFootnoteOrSuper)
	if extension&Autolink != 0 {
		p.registerInline("hmfHMF", maybeAutoLink)
	}
	if extension&MathJax != 0 {
		p.registerInline("$", math)
	}

	return &p
}

func (p *Parser) registerInline(chars string, fn InlineParser) {
	for i := 0; i < len(chars); i++ {
		p.inlineCallback[chars[i]] = fn
	}
}

// RegisterInline installs fn for a trigger byte and returns the previous parser.
func (p *Parser) RegisterInline(trigger byte, fn InlineParser) InlineParser {
	previous := p.inlineCallback[trigger]
	p.inlineCallback[trigger] = fn
	return previous
}
