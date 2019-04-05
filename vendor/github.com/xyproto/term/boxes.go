package term

// These are used for drawing "ASCII-art" boxes
const (
	TLCHAR = '╭' // top left
	TRCHAR = '╮' // top right
	BLCHAR = '╰' // bottom left
	BRCHAR = '╯' // bottom right
	VCHAR  = '│' // vertical line, left side
	VCHAR2 = '│' // vertical line, right side
	HCHAR  = '─' // horizontal line
	HCHAR2 = '─' // horizontal bottom line
)

// Rect is a position, width and height
type Rect struct {
	X int
	Y int
	W int
	H int
}

// Box has one outer and one innner rectangle.
// This is useful when having margins that surrounds content.
type Box struct {
	frame *Rect // The rectangle around the box, for placement
	inner *Rect // The rectangle inside the box, for content
}

// Create a new Box / container.
func NewBox() *Box {
	return &Box{&Rect{0, 0, 0, 0}, &Rect{0, 0, 0, 0}}
}

// Place a Box at the center of the given container.
func (b *Box) Center(container *Box) {
	widthleftover := container.inner.W - b.frame.W
	heightleftover := container.inner.H - b.frame.H
	b.frame.X = container.inner.X + widthleftover/2
	b.frame.Y = container.inner.Y + heightleftover/2
}

// Place a Box so that it fills the entire given container.
func (b *Box) Fill(container *Box) {
	b.frame.X = container.inner.X
	b.frame.Y = container.inner.Y
	b.frame.W = container.inner.W
	b.frame.H = container.inner.H
}

// Place a Box inside a given container, with the given margins.
// Margins are given in number of characters.
func (b *Box) FillWithMargins(container *Box, margins int) {
	b.Fill(container)
	b.frame.X += margins
	b.frame.Y += margins
	b.frame.W -= margins * 2
	b.frame.H -= margins * 2
}

// Place a Box inside a given container, using the given percentage wise ratios.
// horizmarginp can for example be 0.1 for a 10% horizontal margin around
// the inner box. vertmarginp works similarly, but for the vertical margins.
func (b *Box) FillWithPercentageMargins(container *Box, horizmarginp float32, vertmarginp float32) {
	horizmargin := int(float32(container.inner.W) * horizmarginp)
	vertmargin := int(float32(container.inner.H) * vertmarginp)
	b.Fill(container)
	b.frame.X += horizmargin
	b.frame.Y += vertmargin
	b.frame.W -= horizmargin * 2
	b.frame.H -= vertmargin * 2
}

// Retrieves the position of the inner rectangle.
func (b *Box) GetContentPos() (int, int) {
	return b.inner.X, b.inner.Y
}

// Set the size of the Box to 1/3 of the size of the inner rectangle
// of the given container.
func (b *Box) SetThirdSize(container *Box) {
	b.frame.W = container.inner.W / 3
	b.frame.H = container.inner.H / 3
}

// Set the position of the Box to 1/3 of the size of the inner rectangle
// of the given container.
func (b *Box) SetThirdPlace(container *Box) {
	b.frame.X = container.inner.X + container.inner.W/3
	b.frame.Y = container.inner.Y + container.inner.H/3
}

// Place a Box so that it either fills the given
// container, or is placed 1/3 from the upper left edge,
// depending on how much space is left.
func (b *Box) SetNicePlacement(container *Box) {
	b.frame.X = container.inner.X
	b.frame.Y = container.inner.Y
	leftoverwidth := container.inner.W - b.frame.W
	leftoverheight := container.inner.H - b.frame.H
	if leftoverwidth > b.frame.W {
		b.frame.X += leftoverwidth / 3
	}
	if leftoverheight > b.frame.H {
		b.frame.Y += leftoverheight / 3
	}
}

// Place a Box within the given container.
func (b *Box) Place(container *Box) {
	b.frame.X = container.inner.X
	b.frame.Y = container.inner.Y
}

// Get the inner rectangle (content size + pos)
func (b *Box) GetInner() *Rect {
	return b.inner
}

// Get the outer frame (box size + pos)
func (b *Box) GetFrame() *Rect {
	return b.frame
}

// Set the inner rectangle (content size + pos)
func (b *Box) SetInner(r *Rect) {
	b.inner = r
}

// Set the outer frame (box size + pos)
func (b *Box) SetFrame(r *Rect) {
	b.frame = r
}
