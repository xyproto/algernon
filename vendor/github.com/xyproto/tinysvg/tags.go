// Package tinysvg has structs and functions for creating and rendering TinySVG images
package tinysvg

// Everything here deals with bytes, not strings
// TODO: Add a Write function that takes an io.Writer so that the image can be written as it is generated.

import (
	"bytes"
	"fmt"
	"io/ioutil"
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

// Document is an XML document, with a title and a root tag
type Document struct {
	title []byte
	root  *Tag
}

// NewDocument creates a new XML/HTML/SVG image, with a root tag.
// If rootTagName contains "<" or ">", it can be used for preceding declarations,
// like <!DOCTYPE html> or <?xml version=\"1.0\"?>.
// Returns a pointer to a Document.
func NewDocument(title, rootTagName []byte) *Document {
	var image Document
	image.title = []byte(title)
	rootTag := NewTag([]byte(rootTagName))
	image.root = rootTag
	return &image
}

// NewTag creates a new tag based on the given name.
// "name" is what will appear right after "<" when rendering as XML/HTML/SVG.
func NewTag(name []byte) *Tag {
	var tag Tag
	tag.name = name
	tag.attrs = make(map[string][]byte)
	tag.nextSibling = nil
	tag.firstChild = nil
	tag.content = make([]byte, 0)
	tag.lastContent = make([]byte, 0)
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
			ret = append(ret, []byte("=\"")...)
			ret = append(ret, value...)
			ret = append(ret, []byte("\" ")...)
		}
	}
	if len(ret) > 0 {
		ret = ret[:len(ret)-1]
	}
	return ret
}

// getSpaces generates a []byte with spaces, based on the given indentation level
func getSpaces(level int) []byte {
	spacing := make([]byte, 0)
	for i := 1; i < level; i++ {
		spacing = append(spacing, []byte("  ")...)
	}
	return spacing
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
	ret = append(ret, []byte("<")...)
	ret = append(ret, tag.name...)
	if len(attrs) > 0 {
		ret = append(ret, []byte(" ")...)
		ret = append(ret, attrs...)
	}
	if (len(tag.content) == 0) && (len(tag.xmlContent) == 0) && (len(tag.lastContent) == 0) {
		ret = append(ret, []byte(" />")...)
	} else {
		if len(tag.xmlContent) > 0 {
			if tag.xmlContent[0] != ' ' {
				ret = append(ret, []byte(">")...)
				ret = append(ret, spacing...)
				ret = append(ret, tag.xmlContent...)
				ret = append(ret, spacing...)
				ret = append(ret, []byte("</")...)
				ret = append(ret, tag.name...)
				ret = append(ret, []byte(">")...)
			} else {
				ret = append(ret, []byte(">")...)
				ret = append(ret, tag.xmlContent...)
				ret = append(ret, spacing...)
				ret = append(ret, []byte("</")...)
				ret = append(ret, tag.name...)
				ret = append(ret, []byte(">")...)
			}
		} else {
			ret = append(ret, []byte(">")...)
			ret = append(ret, tag.content...)
			ret = append(ret, tag.lastContent...)
			ret = append(ret, []byte("</")...)
			ret = append(ret, tag.name...)
			ret = append(ret, []byte(">")...)
		}
	}
	return ret
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

// GetTag searches all tags for the given name
func (image *Document) GetTag(name []byte) (*Tag, error) {
	return image.root.GetTag(name)
}

// GetRoot returns the root tag of the image
func (image *Document) GetRoot() *Tag {
	return image.root
}

// GetTag finds a tag by name and returns an error if not found.
// Returns the first tag that matches.
func (tag *Tag) GetTag(name []byte) (*Tag, error) {
	if bytes.Index(tag.name, name) == 0 {
		return tag, nil
	}
	couldNotFindError := fmt.Errorf("could not find tag: %s", name)
	if tag.CountChildren() == 0 {
		// No children. Not found so far
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

// Bytes (previously getXMLRecursively) renders XML for a tag, recursively.
// The generated XML is returned as a []byte.
func (tag *Tag) Bytes() []byte {
	var content, xmlContent []byte

	if tag.CountChildren() == 0 {
		return tag.getFlatXML()
	}

	child := tag.firstChild
	for child != nil {
		xmlContent = child.Bytes()
		if len(xmlContent) > 0 {
			content = append(content, xmlContent...)
		}
		child = child.nextSibling
	}

	tag.xmlContent = append(tag.xmlContent, tag.content...)
	tag.xmlContent = append(tag.xmlContent, content...)
	tag.xmlContent = append(tag.xmlContent, tag.lastContent...)

	return tag.getFlatXML()
}

// String returns the XML contents as a string
func (tag *Tag) String() string {
	return string(tag.Bytes())
}

// AddContent adds content to the body tag.
// Returns the body tag and nil if successful.
// Returns and an error if no body tag is found, else nil.
func (image *Document) AddContent(content []byte) (*Tag, error) {
	body, err := image.root.GetTag([]byte("body"))
	if err == nil {
		body.AddContent(content)
	}
	return body, err
}

// Bytes renders the image as an XML document
func (image *Document) Bytes() []byte {
	return image.root.Bytes()
}

// String renders the image as an XML document
func (image *Document) String() string {
	return image.root.String()
}

// SaveSVG will save the current image as an SVG file
func (image *Document) SaveSVG(filename string) error {
	return ioutil.WriteFile(filename, image.Bytes(), 0644)
}
