package html

import (
	"io"

	"github.com/gomarkdown/markdown/ast"
)

// Flags control optional behavior of the HTML renderer.
type Flags int

// IDTag is the tag used for tag identification. It defaults to "id".
var IDTag = "id"

const (
	FlagsNone               Flags = 0
	SkipHTML                Flags = 1 << iota // Skip raw HTML blocks.
	SkipImages                                // Skip embedded images.
	SkipLinks                                 // Render link text without links.
	Safelink                                  // Link only to trusted protocols.
	NofollowLinks                             // Add rel="nofollow" to external links.
	NoreferrerLinks                           // Add rel="noreferrer" to external links.
	NoopenerLinks                             // Add rel="noopener" to external links.
	HrefTargetBlank                           // Open external links in a new tab.
	CompletePage                              // Emit a complete HTML document.
	UseXHTML                                  // Emit XHTML-compatible singleton tags.
	FootnoteReturnLinks                       // Link footnotes back to their references.
	FootnoteNoHRTag                           // Omit the rule before footnotes.
	Smartypants                               // Enable smart punctuation.
	SmartypantsFractions                      // Enable smart fractions.
	SmartypantsDashes                         // Enable smart dashes.
	SmartypantsLatexDashes                    // Enable LaTeX-style dashes.
	SmartypantsAngledQuotes                   // Use angled double quotes.
	SmartypantsQuotesNBSP                     // Use French guillemets with nonbreaking spaces.
	TOC                                       // Generate a table of contents.
	LazyLoadImages                            // Add loading="lazy" to images.

	CommonFlags Flags = Smartypants | SmartypantsFractions | SmartypantsDashes | SmartypantsLatexDashes
)

// RenderNodeFunc can replace the default rendering of selected nodes.
type RenderNodeFunc func(io.Writer, ast.Node, bool) (ast.WalkStatus, bool)

// RendererOptions configures HTML rendering.
type RendererOptions struct {
	AbsolutePrefix             string         // Prefix for relative URLs.
	FootnoteAnchorPrefix       string         // Prefix for footnote anchors.
	FootnoteReturnLinkContents string         // HTML inside footnote return links.
	CitationFormatString       string         // fmt string for citation targets.
	HeadingIDPrefix            string         // Prefix for generated heading IDs.
	HeadingIDSuffix            string         // Suffix for generated heading IDs.
	ParagraphTag               string         // Paragraph tag override.
	Title                      string         // Complete-page document title.
	CSS                        string         // Complete-page stylesheet URL.
	Icon                       string         // Complete-page icon URL.
	Head                       []byte         // Extra complete-page head markup.
	Flags                      Flags          // Optional renderer behavior.
	RenderNodeHook             RenderNodeFunc // Optional node rendering override.
	Comments                   [][]byte       // Code comment markers for callouts.
	Generator                  string         // Complete-page generator meta tag.
}

// Renderer renders an AST as HTML. Construct one with NewRenderer.
type Renderer struct {
	Opts              RendererOptions
	closeTag          string
	headingIDs        map[string]int
	lastOutputLen     int
	DisableTags       int               // Strip tags from Out and Outs when positive.
	IsSafeURLOverride func([]byte) bool // Optional safe-URL predicate.
	sr                *SPRenderer
	documentMatter    ast.DocumentMatters
}

// NewRenderer creates an HTML renderer.
func NewRenderer(opts RendererOptions) *Renderer {
	closeTag := ">"
	if opts.Flags&UseXHTML != 0 {
		closeTag = " />"
	}
	if opts.FootnoteReturnLinkContents == "" {
		opts.FootnoteReturnLinkContents = `<sup>[return]</sup>`
	}
	if opts.CitationFormatString == "" {
		opts.CitationFormatString = `<sup>[%s]</sup>`
	}
	if opts.Generator == "" {
		opts.Generator = `  <meta name="GENERATOR" content="github.com/gomarkdown/markdown markdown processor for Go`
	}
	return &Renderer{
		Opts:       opts,
		closeTag:   closeTag,
		headingIDs: make(map[string]int),
		sr:         NewSmartypantsRenderer(opts.Flags),
	}
}
