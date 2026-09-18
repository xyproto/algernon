package parser

import (
	"bytes"
	"strconv"

	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/internal/textutil"
)

type linkType int

const (
	linkNormal linkType = iota
	linkImg
	linkDeferredFootnote
	linkInlineFootnote
	linkCitation
)

func isReferenceStyleLink(data []byte, pos int, t linkType) bool {
	if t == linkDeferredFootnote {
		return false
	}
	return pos < len(data)-1 && data[pos] == '[' && data[pos+1] != '^'
}

func referenceID(data []byte, end int, multiline, trimFootnoteMarker bool) []byte {
	if !multiline {
		start := 1
		if trimFootnoteMarker {
			start++
		}
		return data[start:end]
	}
	var id bytes.Buffer
	for i := 1; i < end; i++ {
		if data[i] != '\n' {
			id.WriteByte(data[i])
		} else if data[i-1] != ' ' {
			id.WriteByte(' ')
		}
	}
	return id.Bytes()
}

func link(p *Parser, data []byte, offset int) (int, ast.Node) {
	// no links allowed inside regular links, footnote, and deferred footnotes
	if p.InsideLink && (offset > 0 && data[offset-1] == '[' || len(data)-1 > offset && data[offset+1] == '^') {
		return 0, nil
	}

	var t linkType
	switch {
	// special case: ![^text] == deferred footnote (that follows something with
	// an exclamation point)
	case p.extensions&Footnotes != 0 && len(data)-1 > offset && data[offset+1] == '^':
		t = linkDeferredFootnote
	// ![alt] == image
	case offset >= 0 && data[offset] == '!':
		t = linkImg
		offset++
	// [@citation], [@-citation], [@?citation], [@!citation]
	case p.extensions&Mmark != 0 && len(data)-1 > offset && data[offset+1] == '@':
		t = linkCitation
	// [text] == regular link
	// ^[text] == inline footnote
	// [^refId] == deferred footnote
	case p.extensions&Footnotes != 0:
		if offset >= 0 && data[offset] == '^' {
			t = linkInlineFootnote
			offset++
		} else if len(data)-1 > offset && data[offset+1] == '^' {
			t = linkDeferredFootnote
		}
	default:
		t = linkNormal
	}

	open := offset
	data = data[offset:]

	if t == linkCitation {
		return citation(p, data, 0)
	}

	var (
		i                               int
		noteID                          int
		title, link, linkID, altContent []byte
		textHasNl                       = false
	)

	// Match ']' from the Inline() buffer table so a run of unmatched '[' is
	// O(n) instead of O(n²) (GHSA-85vw-wvf9-r522).
	closeAt, nested, found := p.brackets.lookup(open)
	if !found {
		return 0, nil
	}
	txtE := closeAt - open
	if txtE < 1 || txtE >= len(data) {
		return 0, nil
	}
	textHasNl = bytes.IndexByte(data[1:txtE], '\n') >= 0
	i = txtE + 1
	var footnoteNode ast.Node

	// skip any amount of whitespace or newline
	// (this is much more lax than original markdown syntax)
	i = skipSpace(data, i)

	// inline style link
	switch {
	case i < len(data) && data[i] == '(':
		var ok bool
		i, link, title, ok = parseInlineLink(data, i)
		if !ok {
			return 0, nil
		}

	// reference style link
	case isReferenceStyleLink(data, i, t):
		var id []byte
		altContentConsidered := false

		// look for the id
		i++
		linkB := i
		i = skipUntilChar(data, i, ']')

		if i >= len(data) {
			return 0, nil
		}
		linkE := i

		// find the reference
		if linkB == linkE {
			id = referenceID(data, txtE, textHasNl, false)
			if !textHasNl {
				altContentConsidered = true
			}
		} else {
			id = data[linkB:linkE]
		}

		// find the reference with matching id
		lr, ok := p.getRef(string(id))
		if !ok {
			return 0, nil
		}

		// keep link and title from reference
		linkID = id
		link = lr.link
		title = lr.title
		if altContentConsidered {
			altContent = lr.text
		}
		i++

	// shortcut reference style link or reference or inline footnote
	default:
		id := referenceID(data, txtE, textHasNl, t == linkDeferredFootnote)

		if t == linkInlineFootnote || t == linkDeferredFootnote {
			footnoteNode = &ast.ListItem{}
		}
		if t == linkInlineFootnote {
			// create a new reference
			noteID = len(p.notes) + 1

			var fragment []byte
			if len(id) > 0 {
				length := len(id)
				if length > 16 {
					length = 16
				}
				fragment = make([]byte, length)
				copy(fragment, textutil.Slugify(id))
			} else {
				fragment = append([]byte("footnote-"), []byte(strconv.Itoa(noteID))...)
			}

			ref := &reference{
				noteID:   noteID,
				link:     fragment,
				title:    id,
				footnote: footnoteNode,
			}

			p.notes = append(p.notes, ref)
			p.refsRecord[string(ref.link)] = struct{}{}

			link = ref.link
			title = ref.title
		} else {
			// Nested [...] cannot be a stored reference label (labels end at
			// the first ']'). Skip the map lookup: it cannot match, and doing
			// it for every nested '[' is quadratic (GHSA-85vw-wvf9-r522).
			// Still consult ReferenceOverride, which sees the full inner text.
			if nested && p.ReferenceOverride == nil {
				return 0, nil
			}
			// find the reference with matching id
			lr, ok := p.getRef(string(id))
			if !ok {
				return 0, nil
			}

			if t == linkDeferredFootnote && !p.isFootnote(lr) {
				lr.noteID = len(p.notes) + 1
				lr.footnote = footnoteNode
				p.notes = append(p.notes, lr)
				p.refsRecord[string(lr.link)] = struct{}{}
			}

			// keep link and title from reference
			link = lr.link
			// if inline footnote, title == footnote contents
			title = lr.title
			noteID = lr.noteID
			if len(lr.text) > 0 {
				altContent = lr.text
			}
			if len(linkID) == 0 && noteID == 0 {
				linkID = id
			}
		}

		// rewind the whitespace
		i = txtE + 1
	}

	var uLink []byte
	if (t == linkNormal || t == linkImg) && len(link) > 0 {
		uLink = unescapeBytes(link)
	}

	var inlineAttr *ast.Attribute
	if p.extensions&InlineAttributes != 0 && (t == linkNormal || t == linkImg) {
		if attr, n := parseAttributeList(data[i:], false); n > 0 {
			inlineAttr = attr
			i += n
		}
	}

	// call the relevant rendering function
	switch t {
	case linkNormal:
		link := &ast.Link{
			Destination: uLink,
			Title:       title,
			DeferredID:  linkID,
		}
		applyAttribute(link, inlineAttr)
		if len(altContent) > 0 {
			ast.AppendChild(link, newTextNode(altContent))
		} else {
			// links cannot contain other links, so turn off link parsing
			// temporarily and recurse
			InsideLink := p.InsideLink
			p.InsideLink = true
			p.Inline(link, data[1:txtE])
			p.InsideLink = InsideLink
		}
		return i, link

	case linkImg:
		image := &ast.Image{
			Destination: uLink,
			Title:       title,
		}
		applyAttribute(image, inlineAttr)
		ast.AppendChild(image, newTextNode(data[1:txtE]))
		return i + 1, image

	case linkInlineFootnote, linkDeferredFootnote:
		link := &ast.Link{
			Destination: link,
			Title:       title,
			NoteID:      noteID,
			Footnote:    footnoteNode,
		}
		if t == linkDeferredFootnote {
			link.DeferredID = data[2:txtE]
		}
		if t == linkInlineFootnote {
			i++
		}
		return i, link

	default:
		return 0, nil
	}
}
