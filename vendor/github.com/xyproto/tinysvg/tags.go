// Package tinysvg has structs and functions for creating and rendering TinySVG images
package tinysvg

// Everything here deals with bytes, not strings
// TODO: Add a Write function that takes an io.Writer so that the image can be written as it is generated.

import (
	"bytes"
	"fmt"
	"io"
)

// Tag represents an XML tag, as part of a larger XML document
type Tag struct {
	name        []byte
	content     []byte
	lastContent []byte
	xmlContent  []byte
	attrs       map[string][]byte
	nextSibling *Tag // siblings
	firstChild  *Tag // first child
}

var (
	gt                = []byte{'>'}
	lt                = []byte{'<'}
	ltSlash           = []byte("</")
	spaceSlashGt      = []byte(" />")
	equalEscapedQuote = []byte("=\"")
	escapedQuoteSpace = []byte("\" ")
	space             = []byte{' '}
)

// NewTag creates a new tag based on the given name.
// "name" is what will appear right after "<" when rendering as XML/HTML/SVG.
func NewTag(name []byte) *Tag {
	var tag Tag
	tag.name = name
	tag.attrs = make(map[string][]byte)
	return &tag
}

// AddNewTag adds a new tag to another tag. This will place it one step lower
// in the hierarchy of tags. You can for example add a body tag to an html tag.
func (tag *Tag) AddNewTag(name []byte) *Tag {
	child := NewTag(name)
	tag.AddChild(child)
	return child
}

// AddTag adds a tag to another tag
func (tag *Tag) AddTag(child *Tag) {
	tag.AddChild(child)
}

// AddAttrib adds an attribute to a tag, for instance "size" and "20"
func (tag *Tag) AddAttrib(attrName string, attrValue []byte) {
	tag.attrs[attrName] = attrValue
}

// AddAttribMap adds attributes based on a given map
func (tag *Tag) AddAttribMap(attrMap map[string][]byte) {
	//attrName string, attrValue []byte) {
	for attrName, attrValue := range attrMap {
		tag.attrs[attrName] = attrValue
	}
}

// AddSingularAttrib adds attribute without a value
func (tag *Tag) AddSingularAttrib(attrName string) {
	tag.attrs[attrName] = nil
}

// GetAttrString returns a []byte that represents all the attribute keys and
// values of a tag. This can be used when generating XML, SVG or HTML.
func (tag *Tag) GetAttrString() []byte {
	ret := make([]byte, 0)
	for key, value := range tag.attrs {
		if value == nil {
			ret = append(ret, key...)
			ret = append(ret, ' ')
		} else {
			ret = append(ret, key...)
			ret = append(ret, equalEscapedQuote...) // =\"
			ret = append(ret, value...)
			ret = append(ret, escapedQuoteSpace...) // \"
		}
	}
	if len(ret) > 0 {
		ret = ret[:len(ret)-1]
	}
	return ret
}

// getFlatXML renders XML.
// This will generate a []byte for a tag, non-recursively.
func (tag *Tag) getFlatXML() []byte {
	// For the root tag
	if (len(tag.name) > 0) && (tag.name[0] == '<') {
		ret := make([]byte, 0, len(tag.name)+len(tag.content)+len(tag.xmlContent)+len(tag.lastContent))
		ret = append(ret, tag.name...)
		ret = append(ret, tag.content...)
		ret = append(ret, tag.xmlContent...)
		ret = append(ret, tag.lastContent...)
		return ret
	}
	// For indenting
	spacing := make([]byte, 0)
	// Generate the XML based on the tag
	attrs := tag.GetAttrString()
	ret := make([]byte, 0)
	ret = append(ret, spacing...)
	ret = append(ret, '<')
	ret = append(ret, tag.name...)
	if len(attrs) > 0 {
		ret = append(ret, ' ')
		ret = append(ret, attrs...)
	}
	if (len(tag.content) == 0) && (len(tag.xmlContent) == 0) && (len(tag.lastContent) == 0) {
		ret = append(ret, []byte(spaceSlashGt)...) //  />
	} else {
		if len(tag.xmlContent) > 0 {
			if tag.xmlContent[0] != ' ' {
				ret = append(ret, '>')
				ret = append(ret, spacing...)
				ret = append(ret, tag.xmlContent...)
				ret = append(ret, spacing...)
				ret = append(ret, ltSlash...) // </
				ret = append(ret, tag.name...)
				ret = append(ret, '>')
			} else {
				ret = append(ret, '>')
				ret = append(ret, tag.xmlContent...)
				ret = append(ret, spacing...)
				ret = append(ret, ltSlash...) // </
				ret = append(ret, tag.name...)
				ret = append(ret, '>')
			}
		} else {
			ret = append(ret, '>')
			ret = append(ret, tag.content...)
			ret = append(ret, tag.lastContent...)
			ret = append(ret, ltSlash...) // </
			ret = append(ret, tag.name...)
			ret = append(ret, '>')
		}
	}
	return ret
}

// writeFlatXML renders an XML tag to an io.Writer.
// This will generate a bytes for a tag, non-recursively.
func (tag *Tag) writeFlatXML(w io.Writer) (n int64, err error) {
	nameLen := len(tag.name)

	if nameLen > 0 && tag.name[0] == '<' { // root tag
		parts := [][]byte{tag.name, tag.content, tag.xmlContent, tag.lastContent}
		for _, part := range parts {
			if len(part) > 0 {
				x, err := w.Write(part)
				n += int64(x)
				if err != nil {
					return n, err
				}
			}
		}
		return n, nil
	}

	attrs := tag.GetAttrString()
	attrsLen := len(attrs)
	contentLen := len(tag.content)
	xmlContentLen := len(tag.xmlContent)
	lastContentLen := len(tag.lastContent)
	ltLen := len(lt)
	gtLen := len(gt)
	spaceLen := len(space)
	spaceSlashGtLen := len(spaceSlashGt)
	ltSlashLen := len(ltSlash)

	totalSize := ltLen + nameLen + gtLen

	if attrsLen > 0 {
		totalSize += spaceLen + attrsLen
	}

	isEmpty := contentLen == 0 && xmlContentLen == 0 && lastContentLen == 0

	if isEmpty {
		totalSize += spaceSlashGtLen - gtLen
	} else {
		totalSize += contentLen + xmlContentLen + lastContentLen
		totalSize += ltSlashLen + nameLen + gtLen
	}

	buf := make([]byte, 0, totalSize)

	buf = append(buf, lt...)
	buf = append(buf, tag.name...)

	if attrsLen > 0 {
		buf = append(buf, space...)
		buf = append(buf, attrs...)
	}

	if isEmpty {
		buf = append(buf, spaceSlashGt...)
	} else {
		buf = append(buf, gt...)

		if xmlContentLen > 0 {
			buf = append(buf, tag.xmlContent...)
		} else {
			buf = append(buf, tag.content...)
			buf = append(buf, tag.lastContent...)
		}

		buf = append(buf, ltSlash...)
		buf = append(buf, tag.name...)
		buf = append(buf, gt...)
	}

	x, err := w.Write(buf)
	return int64(x), err
}

// GetChildren returns all children for a given tag.
// Returns a slice of pointers to tags.
func (tag *Tag) GetChildren() []*Tag {
	var children []*Tag
	current := tag.firstChild
	for current != nil {
		children = append(children, current)
		current = current.nextSibling
	}
	return children
}

// AddChild adds a tag as a child to another tag
func (tag *Tag) AddChild(child *Tag) {
	if tag.firstChild == nil {
		tag.firstChild = child
		return
	}
	lastChild := tag.LastChild()
	child.nextSibling = nil
	lastChild.nextSibling = child
}

// AddContent adds text to a tag.
// This is what will appear between two tag markers, for example:
// <tag>content</tag>
// If the tag contains child tags, they will be rendered after this content.
func (tag *Tag) AddContent(content []byte) {
	tag.content = append(tag.content, content...)
}

// AppendContent appends content to the end of the existing content of a tag
func (tag *Tag) AppendContent(content []byte) {
	tag.lastContent = append(tag.lastContent, content...)
}

// AddLastContent appends content to the end of the existing content of a tag.
// Deprecated.
func (tag *Tag) AddLastContent(content []byte) {
	tag.AppendContent(content)
}

// CountChildren returns the number of children a tag has
func (tag *Tag) CountChildren() int {
	child := tag.firstChild
	if child == nil {
		return 0
	}
	count := 1
	if child.nextSibling == nil {
		return count
	}
	child = child.nextSibling
	for child != nil {
		count++
		child = child.nextSibling
	}
	return count
}

// CountSiblings returns the number of siblings a tag has
func (tag *Tag) CountSiblings() int {
	sib := tag.nextSibling
	if sib == nil {
		return 0
	}
	count := 1
	if sib.nextSibling == nil {
		return count
	}
	sib = sib.nextSibling
	for sib != nil {
		count++
		sib = sib.nextSibling
	}
	return count
}

// LastChild returns the last child of a tag
func (tag *Tag) LastChild() *Tag {
	child := tag.firstChild
	for child.nextSibling != nil {
		child = child.nextSibling
	}
	return child
}

// GetTag finds a tag by name and returns an error if not found.
// Returns the first tag that matches.
func (tag *Tag) GetTag(name []byte) (*Tag, error) {
	if bytes.Index(tag.name, name) == 0 {
		return tag, nil
	}
	couldNotFindError := fmt.Errorf("could not find tag: %s", name)
	if tag.CountChildren() == 0 {
		// No children. Not found so far.
		return nil, couldNotFindError
	}

	child := tag.firstChild
	for child != nil {
		found, err := child.GetTag(name)
		if err == nil {
			return found, err
		}
		child = child.nextSibling
	}

	return nil, couldNotFindError
}

// ShallowCopy creates a copy of a tag, but uses the same attribute map!
func (tag *Tag) ShallowCopy() *Tag {
	var nt Tag
	nt.name = tag.name
	nt.content = tag.content
	nt.lastContent = tag.lastContent
	nt.xmlContent = tag.xmlContent
	nt.attrs = tag.attrs
	nt.nextSibling = tag.nextSibling
	nt.firstChild = tag.firstChild
	return &nt
}

// Bytes (previously getXMLRecursively) renders XML for a tag, recursively.
// The generated XML is returned as a []byte.
func (tag *Tag) Bytes() []byte {
	if tag.CountChildren() == 0 {
		return tag.getFlatXML()
	}
	var (
		content    []byte
		xmlContent []byte
		child      = tag.firstChild
	)
	for child != nil {
		xmlContent = child.Bytes()
		if len(xmlContent) > 0 {
			content = append(content, xmlContent...)
		}
		child = child.nextSibling
	}
	tagCopy := tag.ShallowCopy()
	tagCopy.xmlContent = append(tagCopy.xmlContent, tag.content...)
	tagCopy.xmlContent = append(tagCopy.xmlContent, content...)
	tagCopy.xmlContent = append(tagCopy.xmlContent, tag.lastContent...)
	return tagCopy.getFlatXML()
}

// WriteTo renders XML for a tag, recursively.
// The generated XML is written to the given io.Writer.
// This also fullfills the io.WriterTo interface.
func (tag *Tag) WriteTo(w io.Writer) (n int64, err error) {
	if tag.CountChildren() == 0 {
		return tag.writeFlatXML(w)
	}
	var (
		content bytes.Buffer
		child   = tag.firstChild
	)
	for child != nil {
		child.WriteTo(&content)
		child = child.nextSibling
	}
	tagCopy := tag.ShallowCopy()
	tagCopy.xmlContent = append(tagCopy.xmlContent, tag.content...)
	tagCopy.xmlContent = append(tagCopy.xmlContent, content.Bytes()...)
	tagCopy.xmlContent = append(tagCopy.xmlContent, tag.lastContent...)
	return tagCopy.writeFlatXML(w)
}

// String returns the XML contents as a string
func (tag *Tag) String() string {
	return string(tag.Bytes())
}
