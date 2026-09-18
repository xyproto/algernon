/*
Package parser implements a parser for markdown text that generates an AST (abstract syntax tree).
*/
package parser

import (
	"strconv"
	"strings"

	"github.com/gomarkdown/markdown/ast"
)

// Parser is a type that holds extensions and the runtime state used by
// Parse. You cannot use it directly; construct it with New.
type Parser struct {

	// ReferenceOverride is an optional function callback that is called every
	// time a reference is resolved. It can be set before starting parsing.
	//
	// In Markdown, the link reference syntax can be made to resolve a link to
	// a reference instead of an inline URL, in one of the following ways:
	//
	//  * [link text][refid]
	//  * [refid][]
	//
	// Usually, the refid is defined at the bottom of the Markdown document. If
	// this override function is provided, the refid is passed to the override
	// function first, before consulting the defined refids at the bottom. If
	// the override function indicates an override did not occur, the refids at
	// the bottom will be used to fill in the link details.
	ReferenceOverride ReferenceOverrideFunc

	// IsSafeURLOverride allows overriding the default URL matcher. URL is
	// safe if the overriding function returns true. Can be used to extend
	// the default list of safe URLs.
	IsSafeURLOverride func(url []byte) bool

	Opts Options

	// after parsing, this is AST root of parsed markdown text
	Doc ast.Node

	extensions Extensions

	refs           map[string]*reference
	refsRecord     map[string]struct{}
	inlineCallback [256]InlineParser
	nesting        int
	maxNesting     int
	InsideLink     bool
	indexCnt       int // incremented after every index

	// Footnotes need to be ordered as well as available to quickly check for
	// presence. If a ref is also a footnote, it's stored both in refs and here
	// in notes. Slice is nil if footnotes not enabled.
	notes []*reference

	tip                  ast.Node // = doc
	oldTip               ast.Node
	lastMatchedContainer ast.Node // = doc
	allClosed            bool

	// Attributes are attached to block level elements.
	attr *ast.Attribute

	// codeSpans remembers where the backtick run under the inline cursor
	// closes, so retrying it from each of its bytes does not rescan the
	// rest of the input each time.
	codeSpans codeSpanCache

	// spaces remembers the space run under the inline cursor, since the
	// line-break callback fires for every space in it.
	spaces runCache

	// angles remembers the last '<' before the inline cursor, which the
	// autolink callback needs for every URL it considers.
	angles lastByteCache

	// nextGt and nextCommentEnd remember where the next '>' and "-->" lie,
	// so a '<' that nothing closes does not scan to the end of the buffer.
	nextGt         nextMatchCache
	nextCommentEnd nextMatchCache

	// pendingRefDef is a reference definition detected at the start of a
	// paragraph line. It is added after the preceding paragraph so source
	// order is preserved.
	pendingRefDef *ast.ReferenceDefinition

	includeStack *incStack

	// collect headings where we auto-generated id so that we can
	// ensure they are unique at the end
	allHeadingsWithAutoID []*ast.Heading

	didParse bool

	// Matching ']' for each '[' in the current Inline() buffer.
	brackets bracketTable
}

func (p *Parser) getRef(refid string) (ref *reference, found bool) {
	if p.ReferenceOverride != nil {
		r, overridden := p.ReferenceOverride(refid)
		if overridden {
			if r == nil {
				return nil, false
			}
			return &reference{
				link:  []byte(r.Link),
				title: []byte(r.Title),
				text:  []byte(r.Text)}, true
		}
	}
	// refs are case insensitive
	ref, found = p.refs[strings.ToLower(refid)]
	return ref, found
}

func (p *Parser) isFootnote(ref *reference) bool {
	_, ok := p.refsRecord[string(ref.link)]
	return ok
}

func (p *Parser) Finalize(block ast.Node) {
	p.tip = block.GetParent()
}

func (p *Parser) addChild(node ast.Node) ast.Node {
	for !canNodeContain(p.tip, node) {
		p.Finalize(p.tip)
	}
	ast.AppendChild(p.tip, node)
	p.tip = node
	return node
}

func canNodeContain(n ast.Node, v ast.Node) bool {
	switch n.(type) {
	case *ast.List:
		return isListItem(v)
	case *ast.Document, *ast.BlockQuote, *ast.Aside, *ast.ListItem, *ast.CaptionFigure:
		return !isListItem(v)
	case *ast.Table:
		switch v.(type) {
		case *ast.TableHeader, *ast.TableBody, *ast.TableFooter:
			return true
		default:
			return false
		}
	case *ast.TableHeader, *ast.TableBody, *ast.TableFooter:
		_, ok := v.(*ast.TableRow)
		return ok
	case *ast.TableRow:
		_, ok := v.(*ast.TableCell)
		return ok
	case *ast.Paragraph, *ast.Heading, *ast.MathBlock, *ast.TableCell, *ast.Caption,
		*ast.DocumentMatter, *ast.Emph, *ast.Strong, *ast.Del, *ast.Link, *ast.Image,
		*ast.CrossReference, *ast.Citation, *ast.Index, *ast.Footnotes,
		*ast.HorizontalRule, *ast.Math, *ast.Text, *ast.HTMLBlock, *ast.CodeBlock,
		*ast.Softbreak, *ast.Hardbreak, *ast.NonBlockingSpace, *ast.Code, *ast.HTMLSpan,
		*ast.Callout, *ast.Subscript, *ast.Superscript, *ast.ReferenceDefinition:
		return false
	}
	if o, ok := n.(ast.CanContain); ok {
		return o.CanContain(v)
	}
	// Custom container nodes (not in package ast) default to true.
	return n.AsLeaf() == nil
}

func (p *Parser) closeUnmatchedBlocks() {
	if p.allClosed {
		return
	}
	for p.oldTip != p.lastMatchedContainer {
		parent := p.oldTip.GetParent()
		p.Finalize(p.oldTip)
		p.oldTip = parent
	}
	p.allClosed = true
}

// Reference represents the details of a link.
// See the documentation in Options for more details on use-case.
type Reference struct {
	// Link is usually the URL the reference points to.
	Link string
	// Title is the alternate text describing the link in more detail.
	Title string
	// Text is the optional text to override the ref with if the syntax used was
	// [refid][]
	Text string
}

// Parse generates AST (abstract syntax tree) representing markdown document.
//
// The result is a root of the tree whose underlying type is *ast.Document
//
// You can then convert AST to html using html.Renderer, to some other format
// using a custom renderer or transform the tree.
//
// Parser is not reusable. Create a new Parser for each Parse() call.
func (p *Parser) Parse(input []byte) ast.Node {
	if p.didParse {
		panic("Parser is not reusable. Must create new Parser for each Parse() call.")
	}
	p.didParse = true

	// the code only works with Unix CR newlines so to make life easy for
	// callers normalize newlines
	input = NormalizeNewlines(input)

	p.Block(input)
	// Walk the tree and finish up some of unfinished blocks
	for p.tip != nil {
		p.Finalize(p.tip)
	}
	// Walk the tree again and process inline markdown in each block.
	p.parseInlineContainers(p.Doc, true)

	if p.Opts.Flags&SkipFootnoteList == 0 {
		p.parseRefsToAST()
	}

	// ensure HeadingIDs generated with AutoHeadingIDs are unique
	// this is delayed here (as opposed to done when we create the id)
	// so that we can preserve more original ids when there are conflicts
	taken := map[string]bool{}
	for _, h := range p.allHeadingsWithAutoID {
		id := h.HeadingID
		if id == "" {
			continue
		}
		n := 0
		for taken[id] {
			n++
			id = h.HeadingID + "-" + strconv.Itoa(n)
		}
		h.HeadingID = id
		taken[id] = true
	}

	return p.Doc
}

func (p *Parser) parseRefsToAST() {
	if p.extensions&Footnotes == 0 || len(p.notes) == 0 {
		return
	}
	p.tip = p.Doc
	list := &ast.List{
		IsFootnotesList: true,
		ListFlags:       ast.ListTypeOrdered,
	}
	p.AddBlock(&ast.Footnotes{})
	block := p.AddBlock(list)
	flags := ast.ListItemBeginningOfList
	// Note: this loop is intentionally explicit, not range-form. This is
	// because the body of the loop will append nested footnotes to p.notes and
	// we need to process those late additions. Range form would only walk over
	// the fixed initial set.
	for i := 0; i < len(p.notes); i++ {
		ref := p.notes[i]
		p.addChild(ref.footnote)
		item := ref.footnote.(*ast.ListItem)
		item.ListFlags = flags | ast.ListTypeOrdered
		item.RefLink = ref.link
		if ref.hasBlock {
			flags |= ast.ListItemContainsBlock
			p.Block(ref.title)
		} else {
			p.Inline(item, ref.title)
		}
		flags &^= ast.ListItemBeginningOfList | ast.ListItemContainsBlock
	}
	above := list.Parent
	p.tip = above

	p.parseInlineContainers(block, false)
}

func (p *Parser) parseInlineContainers(root ast.Node, includeTableCells bool) {
	ast.WalkFunc(root, func(node ast.Node, entering bool) ast.WalkStatus {
		switch node.(type) {
		case *ast.Paragraph, *ast.Heading:
		case *ast.TableCell:
			if !includeTableCells {
				return ast.GoToNext
			}
		default:
			return ast.GoToNext
		}
		container := node.AsContainer()
		p.Inline(node, container.Content)
		container.Content = nil
		return ast.GoToNext
	})
}

func isListItem(d ast.Node) bool {
	_, ok := d.(*ast.ListItem)
	return ok
}
