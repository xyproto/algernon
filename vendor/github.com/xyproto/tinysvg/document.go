package tinysvg

import (
	"io"
	"os"
)

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
	image.title = title
	rootTag := NewTag(rootTagName)
	image.root = rootTag
	return &image
}

// GetTag searches all tags for the given name
func (image *Document) GetTag(name []byte) (*Tag, error) {
	return image.root.GetTag(name)
}

// GetRoot returns the root tag of the image
func (image *Document) GetRoot() *Tag {
	return image.root
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
	return os.WriteFile(filename, image.Bytes(), 0644)
}

// WriteTo will write the current image to the given io.Writer.
// Returns bytes written and possibly an error.
// This also fullfills the io.WriterTo interface.
func (image *Document) WriteTo(w io.Writer) (int64, error) {
	return image.root.WriteTo(w)
}
