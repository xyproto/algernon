package ast

// An attribute can be attached to block elements. They are specified as
// {#id .class key="value"} where quotes for values are mandatory, multiple
// key/value pairs are separated by whitespace.
type Attribute struct {
	ID      []byte
	Classes [][]byte
	Attrs   map[string][]byte
}

// ListType contains bitwise or'ed flags for list and list item objects.
type ListType int

// These are the possible flag values for the ListItem renderer.
// Multiple flag values may be ORed together.
// These are mostly of interest if you are writing a new output format.
const (
	ListTypeOrdered ListType = 1 << iota
	ListTypeDefinition
	ListTypeTerm

	ListItemContainsBlock
	ListItemBeginningOfList // Marks the first item emitted for a list.
	ListItemEndOfList
)

// CellAlignFlags holds a type of alignment in a table cell.
type CellAlignFlags int

// These are the possible flag values for the table cell renderer.
// Only a single one of these values will be used; they are not ORed together.
// These are mostly of interest if you are writing a new output format.
const (
	TableAlignmentLeft CellAlignFlags = 1 << iota
	TableAlignmentRight
	TableAlignmentCenter = (TableAlignmentLeft | TableAlignmentRight)
)

func (a CellAlignFlags) String() string {
	switch a {
	case TableAlignmentLeft:
		return "left"
	case TableAlignmentRight:
		return "right"
	case TableAlignmentCenter:
		return "center"
	default:
		return ""
	}
}

// DocumentMatters holds the type of a {front,main,back}matter in the document
type DocumentMatters int

// These are all possible Document divisions.
const (
	DocumentMatterNone DocumentMatters = iota
	DocumentMatterFront
	DocumentMatterMain
	DocumentMatterBack
)

// CitationTypes holds the type of a citation, informative, normative or suppressed
type CitationTypes int

const (
	CitationTypeNone CitationTypes = iota
	CitationTypeSuppressed
	CitationTypeInformative
	CitationTypeNormative
)

// Node defines an ast node
type Node interface {
	AsContainer() *Container
	AsLeaf() *Leaf
	GetParent() Node
	SetParent(newParent Node)
	GetChildren() []Node
	SetChildren(newChildren []Node)
}

// Container is a type of node that can contain children
type Container struct {
	Parent   Node
	Children []Node
	Prev     Node
	Next     Node

	Literal []byte // Text contents of the leaf nodes
	Content []byte // Markdown content of the block nodes

	*Attribute // Block level attribute
}

// return true if can contain children of a given node type
// used by custom nodes to over-ride logic in canNodeContain
type CanContain interface {
	CanContain(Node) bool
}

// AsContainer returns itself as *Container
func (c *Container) AsContainer() *Container { return c }

// AsLeaf returns nil
func (c *Container) AsLeaf() *Leaf { return nil }

// GetParent returns parent node
func (c *Container) GetParent() Node { return c.Parent }

// SetParent sets the parent node
func (c *Container) SetParent(newParent Node) { c.Parent = newParent }

// GetChildren returns children nodes
func (c *Container) GetChildren() []Node { return c.Children }

// SetChildren sets children node
func (c *Container) SetChildren(newChildren []Node) {
	c.Children = newChildren
	linkSiblings(newChildren)
}

// Leaf is a type of node that cannot have children
type Leaf struct {
	Parent Node
	Prev   Node
	Next   Node

	Literal []byte // Text contents of the leaf nodes
	Content []byte // Markdown content of the block nodes

	*Attribute // Block level attribute
}

// AsContainer returns nil
func (l *Leaf) AsContainer() *Container { return nil }

// AsLeaf returns itself as *Leaf
func (l *Leaf) AsLeaf() *Leaf { return l }

// GetParent returns parent node
func (l *Leaf) GetParent() Node { return l.Parent }

// SetParent sets the parent node
func (l *Leaf) SetParent(newParent Node) { l.Parent = newParent }

// GetChildren returns nil because Leaf cannot have children
func (l *Leaf) GetChildren() []Node { return nil }

// SetChildren will panic if trying to set non-empty children
// because Leaf cannot have children
func (l *Leaf) SetChildren(newChildren []Node) {
	if len(newChildren) != 0 {
		panic("leaf node cannot have children")
	}
}

// Document represents markdown document node, a root of ast
type Document struct{ Container }

// DocumentMatter represents markdown node that signals a document
// division: frontmatter, mainmatter or backmatter.
type DocumentMatter struct {
	Container

	Matter DocumentMatters
}

// BlockQuote represents markdown block quote node
type BlockQuote struct{ Container }

// Aside represents an markdown aside node.
type Aside struct{ Container }

// List represents markdown list node
type List struct {
	Container

	ListFlags       ListType
	Tight           bool   // Skip <p>s around list item data if true
	BulletChar      byte   // '*', '+' or '-' in bullet lists
	Delimiter       byte   // '.' or ')' after the number in ordered lists
	Start           int    // for ordered lists this indicates the starting number if > 0
	RefLink         []byte // If not nil, turns this list item into a footnote item and triggers different rendering
	IsFootnotesList bool   // This is a list of footnotes
}

// ListItem represents markdown list item node
type ListItem struct {
	Container

	ListFlags       ListType
	Tight           bool   // Skip <p>s around list item data if true
	BulletChar      byte   // '*', '+' or '-' in bullet lists
	Delimiter       byte   // '.' or ')' after the number in ordered lists
	RefLink         []byte // If not nil, turns this list item into a footnote item and triggers different rendering
	IsFootnotesList bool   // This is a list of footnotes
}

// Paragraph represents markdown paragraph node
type Paragraph struct{ Container }

// Math represents markdown MathAjax inline node
type Math struct{ Leaf }

// MathBlock represents markdown MathAjax block node
type MathBlock struct{ Container }

// Heading represents markdown heading node
type Heading struct {
	Container

	Level        int    // This holds the heading level number
	HeadingID    string // This might hold heading ID, if present
	IsTitleblock bool   // Specifies whether it's a title block
	IsSpecial    bool   // We are a special heading (starts with .#)
}

// HorizontalRule represents markdown horizontal rule node
type HorizontalRule struct{ Leaf }

// Emph represents markdown emphasis node
type Emph struct{ Container }

// Strong represents markdown strong node
type Strong struct{ Container }

// Del represents markdown del node
type Del struct{ Container }

// Link represents markdown link node
type Link struct {
	Container

	Destination          []byte   // Destination is what goes into a href
	Title                []byte   // Title is the tooltip thing that goes in a title attribute
	NoteID               int      // NoteID contains a serial number of a footnote, zero if it's not a footnote
	Footnote             Node     // If it's a footnote, this is a direct link to the footnote Node. Otherwise nil.
	DeferredID           []byte   // If a deferred link this holds the original ID.
	AdditionalAttributes []string // Defines additional attributes to use during rendering.
}

// CrossReference is a reference node.
type CrossReference struct {
	Container

	Destination []byte // Destination is where the reference points to
	Suffix      []byte // Potential citation suffix, i.e. (#myid, text)
}

// Citation is a citation node.
type Citation struct {
	Leaf

	Destination [][]byte        // Destination is where the citation points to. Multiple ones are allowed.
	Type        []CitationTypes // 1:1 mapping of destination and citation type
	Suffix      [][]byte        // Potential citation suffix, i.e. [@!RFC1035, p. 144]
}

// Image represents markdown image node
type Image struct {
	Container

	Destination []byte // Destination is what goes into a href
	Title       []byte // Title is the tooltip thing that goes in a title attribute
}

// Text represents markdown text node
type Text struct{ Leaf }

// HTMLBlock represents markdown html node
type HTMLBlock struct{ Leaf }

// CodeBlock represents markdown code block node
type CodeBlock struct {
	Leaf

	IsFenced    bool   // Specifies whether it's a fenced code block or an indented one
	Info        []byte // This holds the info string
	FenceChar   byte
	FenceLength int
	FenceOffset int
}

// Softbreak represents markdown softbreak node
// The built-in parser does not currently emit Softbreak nodes.
type Softbreak struct{ Leaf }

// Hardbreak represents markdown hard break node
type Hardbreak struct{ Leaf }

// NonBlockingSpace represents a markdown non-breaking space node
type NonBlockingSpace struct{ Leaf }

// Code represents markdown code node
type Code struct{ Leaf }

// HTMLSpan represents markdown html span node
type HTMLSpan struct{ Leaf }

// Table represents markdown table node
type Table struct{ Container }

// TableCell represents markdown table cell node
type TableCell struct {
	Container

	IsHeader bool           // This tells if it's under the header row
	Align    CellAlignFlags // This holds the value for align attribute
	ColSpan  int            // How many columns to span
}

// TableHeader represents markdown table head node
type TableHeader struct{ Container }

// TableBody represents markdown table body node
type TableBody struct{ Container }

// TableRow represents markdown table row node
type TableRow struct{ Container }

// TableFooter represents markdown table foot node
type TableFooter struct{ Container }

// Caption represents a figure, code or quote caption
type Caption struct{ Container }

// CaptionFigure is a node (blockquote or codeblock) that has a caption
type CaptionFigure struct {
	Container

	HeadingID string // This might hold heading ID, if present
}

// Callout is a node that can exist both in text (where it is an actual node) and in a code block.
type Callout struct {
	Leaf

	ID []byte // number of this callout
}

// Index is a node that contains an Index item and an optional, subitem.
type Index struct {
	Leaf

	Primary bool
	Item    []byte
	Subitem []byte
	ID      string // ID of the index
}

// Subscript is a subscript node
type Subscript struct{ Leaf }

// Superscript is a superscript node.
type Superscript struct{ Leaf }

// Footnotes is a node that contains all footnotes
type Footnotes struct{ Container }

// ReferenceDefinition is a [label]: destination "title" definition.
// Links still resolve Destination at parse time; this node is additive so
// round-trippers can recover the original reference syntax.
type ReferenceDefinition struct {
	Leaf

	Label       []byte
	Destination []byte
	Title       []byte
}
