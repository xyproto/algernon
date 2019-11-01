// Package tinysvg supports generating and writing TinySVG 1.2 images
//
// This package has been refactored out from the "github.com/xyproto/onthefly" package.
//
// Some function names are suffixed with "2" if they take structs instead of ints/floats,
// "i" if they take ints and "f" if they take floats. There is no support for multiple dispatch in Go.
//

package tinysvg

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const (
	TRANSPARENT = 0.0
	OPAQUE      = 1.0
)

type (
	Vec2 struct {
		X, Y float64
	}
	Pos    Vec2
	Radius Vec2
	Size   struct {
		W, H float64
	}
	Color struct {
		R int     // red, 0..255
		G int     // green, 0..255
		B int     // blue, 0..255
		A float64 // alpha, 0.0..1.0
		N string  // name (optional, will override the above values)
	}
	Font struct {
		Family string
		Size   int
	}
)

var ErrPair = errors.New("position pairs must be exactly two comma separated numbers")

// Create a new TinySVG document, where the width and height is defined in pixels, using the "px" suffix
func NewTinySVG(w, h int) (*Document, *Tag) {
	page := NewDocument([]byte(""), []byte(`<?xml version="1.0" encoding="UTF-8"?>`))
	svg := page.root.AddNewTag([]byte("svg"))
	svg.AddAttrib("xmlns", []byte("http://www.w3.org/2000/svg"))
	svg.AddAttrib("version", []byte("1.2"))
	svg.AddAttrib("baseProfile", []byte("tiny"))
	svg.AddAttrib("viewBox", []byte(fmt.Sprintf("%d %d %d %d", 0, 0, w, h)))
	svg.AddAttrib("width", []byte(fmt.Sprintf("%dpx", w)))
	svg.AddAttrib("height", []byte(fmt.Sprintf("%dpx", h)))
	return page, svg
}

// NewTinySVG2 creates new TinySVG 1.2 image. Pos and Size defines the viewbox
func NewTinySVG2(p *Pos, s *Size) (*Document, *Tag) {
	// No page title is needed when building an SVG tag tree
	page := NewDocument([]byte(""), []byte(`<?xml version="1.0" encoding="UTF-8"?>`))

	// No longer needed for TinySVG 1.2. See: https://www.w3.org/TR/SVGTiny12/intro.html#defining
	// <!DOCTYPE svg PUBLIC "-//W3C//DTD SVG 1.1 Tiny//EN" "http://www.w3.org/Graphics/SVG/1.1/DTD/svg11-tiny.dtd">

	// Add the root tag
	svg := page.root.AddNewTag([]byte("svg"))
	svg.AddAttrib("xmlns", []byte("http://www.w3.org/2000/svg"))
	svg.AddAttrib("version", []byte("1.2"))
	svg.AddAttrib("baseProfile", []byte("tiny"))
	svg.AddAttrib("viewBox", []byte(fmt.Sprintf("%f %f %f %f", p.X, p.Y, s.W, s.H)))
	return page, svg
}

func f2b(x float64) []byte {
	fs := fmt.Sprintf("%f", x)
	// Drop ".0" if the number ends with that
	if strings.HasSuffix(fs, ".0") {
		return []byte(fs[:len(fs)-2])
	}
	// Drop ".000000" if the number ends with that
	if strings.HasSuffix(fs, ".000000") {
		return []byte(fs[:len(fs)-7])
	}
	return []byte(fs)
}

// Rect a rectangle, given x and y position, width and height.
// No color is being set.
func (svg *Tag) Rect2(p *Pos, s *Size, c *Color) *Tag {
	rect := svg.AddNewTag([]byte("rect"))
	rect.AddAttrib("x", f2b(p.X))
	rect.AddAttrib("y", f2b(p.Y))
	rect.AddAttrib("width", f2b(s.W))
	rect.AddAttrib("height", f2b(s.H))
	rect.Fill2(c)
	return rect
}

// RoundedRect2 a rectangle, given x and y position, width and height.
// No color is being set.
func (svg *Tag) RoundedRect2(p *Pos, r *Radius, s *Size, c *Color) *Tag {
	rect := svg.AddNewTag([]byte("rect"))
	rect.AddAttrib("x", f2b(p.X))
	rect.AddAttrib("y", f2b(p.Y))
	rect.AddAttrib("rx", f2b(r.X))
	rect.AddAttrib("ry", f2b(r.Y))
	rect.AddAttrib("width", f2b(s.W))
	rect.AddAttrib("height", f2b(s.H))
	rect.Fill2(c)
	return rect
}

// Text adds text. No color is being set
func (svg *Tag) Text2(p *Pos, f *Font, message string, c *Color) *Tag {
	text := svg.AddNewTag([]byte("text"))
	text.AddAttrib("x", f2b(p.X))
	text.AddAttrib("y", f2b(p.Y))
	text.AddAttrib("font-family", []byte(f.Family))
	text.AddAttrib("font-size", []byte(strconv.Itoa(f.Size)))
	text.Fill2(c)
	text.AddContent([]byte(message))
	return text
}

// Circle adds a circle, given a position, radius and color
func (svg *Tag) Circle2(p *Pos, radius int, c *Color) *Tag {
	circle := svg.AddNewTag([]byte("circle"))
	circle.AddAttrib("cx", f2b(p.X))
	circle.AddAttrib("cy", f2b(p.Y))
	circle.AddAttrib("r", []byte(strconv.Itoa(radius)))
	circle.Fill2(c)
	return circle
}

// Circle adds a circle, given a position, radius and color
func (svg *Tag) Circlef(p *Pos, radius float64, c *Color) *Tag {
	circle := svg.AddNewTag([]byte("circle"))
	circle.AddAttrib("cx", f2b(p.X))
	circle.AddAttrib("cy", f2b(p.Y))
	circle.AddAttrib("r", f2b(radius))
	circle.Fill2(c)
	return circle
}

// Ellipse adds an ellipse with a given position (x,y) and radius (rx, ry).
func (svg *Tag) Ellipse2(p *Pos, r *Radius, c *Color) *Tag {
	ellipse := svg.AddNewTag([]byte("ellipse"))
	ellipse.AddAttrib("cx", f2b(p.X))
	ellipse.AddAttrib("cy", f2b(p.Y))
	ellipse.AddAttrib("rx", f2b(r.X))
	ellipse.AddAttrib("ry", f2b(r.Y))
	ellipse.Fill2(c)
	return ellipse
}

// Line adds a line from (x1, y1) to (x2, y2) with a given stroke width and color
func (svg *Tag) Line2(p1, p2 *Pos, thickness int, c *Color) *Tag {
	line := svg.AddNewTag([]byte("line"))
	line.AddAttrib("x1", f2b(p1.X))
	line.AddAttrib("y1", f2b(p1.Y))
	line.AddAttrib("x2", f2b(p2.X))
	line.AddAttrib("y2", f2b(p2.Y))
	line.Thickness(thickness)
	line.Stroke2(c)
	return line
}

// Triangle adds a colored triangle
func (svg *Tag) Triangle2(p1, p2, p3 *Pos, c *Color) *Tag {
	triangle := svg.AddNewTag([]byte("path"))
	triangle.AddAttrib("d", []byte(fmt.Sprintf("M %f %f L %f %f L %f %f L %f %f", p1.X, p1.Y, p2.X, p2.Y, p3.X, p3.Y, p1.X, p1.Y)))
	triangle.Fill2(c)
	return triangle
}

// Poly2 adds a colored path with 4 points
func (svg *Tag) Poly2(p1, p2, p3, p4 *Pos, c *Color) *Tag {
	poly4 := svg.AddNewTag([]byte("path"))
	poly4.AddAttrib("d", []byte(fmt.Sprintf("M %f %f L %f %f L %f %f L %f %f L %f %f", p1.X, p1.Y, p2.X, p2.Y, p3.X, p3.Y, p4.X, p4.Y, p1.X, p1.Y)))
	poly4.Fill2(c)
	return poly4
}

// Fill selects the fill color that will be used when drawing
func (svg *Tag) Fill2(c *Color) {
	// If no color name is given and the color is transparent, don't set a fill color
	if (c == nil) || (len(c.N) == 0 && c.A == TRANSPARENT) {
		return
	}
	svg.AddAttrib("fill", c.Bytes())
}

// Stroke selects the stroke color that will be used when drawing
func (svg *Tag) Stroke2(c *Color) {
	// If no color name is given and the color is transparent, don't set a stroke color
	if (c == nil) || (len(c.N) == 0 && c.A == TRANSPARENT) {
		return
	}
	svg.AddAttrib("stroke", c.Bytes())
}

// RGBBytes converts r, g and b (integers in the range 0..255)
// to a color string on the form "#nnnnnn".
func RGBBytes(r, g, b int) []byte {
	rs := strconv.FormatInt(int64(r), 16)
	gs := strconv.FormatInt(int64(g), 16)
	bs := strconv.FormatInt(int64(b), 16)
	if len(rs) == 1 {
		rs = "0" + rs
	}
	if len(gs) == 1 {
		gs = "0" + gs
	}
	if len(bs) == 1 {
		bs = "0" + bs
	}
	return []byte("#" + rs + gs + bs)
}

// RGBABytes converts integers r, g and b (the color) and also
// a given alpha (opacity) to a color-string on the form
// "rgba(255, 255, 255, 1.0)".
func RGBABytes(r, g, b int, a float64) []byte {
	return []byte(fmt.Sprintf("rgba(%d, %d, %d, %f)", r, g, b, a))
}

// RGBA creates a new Color with the given red, green and blue values.
// The colors are in the range 0..255
func RGB(r, g, b int) *Color {
	return &Color{r, g, b, OPAQUE, ""}
}

// RGBA creates a new Color with the given red, green, blue and alpha values.
// Alpha is between 0 and 1, the rest are 0..255.
// For the alpha value, 0 is transparent and 1 is opaque.
func RGBA(r, g, b int, a float64) *Color {
	return &Color{r, g, b, a, ""}
}

// ColorByName creates a new Color with a given name, like "blue"
func ColorByName(name string) *Color {
	return &Color{N: name}
}

// NewColor is the same as ColorByName
func NewColor(name string) *Color {
	return ColorByName(name)
}

// String returns the color as an RGB (#1234FF) string
// or as an RGBA (rgba(0, 1, 2 ,3)) string.
func (c *Color) Bytes() []byte {
	// Return an empty string if nil
	if c == nil {
		return make([]byte, 0)
	}
	// Return the name, if specified
	if len(c.N) != 0 {
		return []byte(c.N)
	}
	// Return a regular RGB string if alpha is 1.0
	if c.A == OPAQUE {
		// Generate a rgb string
		return RGBBytes(c.R, c.G, c.B)
	}
	// Generate a rgba string if alpha is < 1.0
	return RGBABytes(c.R, c.G, c.B, c.A)
}

// --- Convenience functions and functions for backward compatibility ---

func NewTinySVGi(x, y, w, h int) (*Document, *Tag) {
	return NewTinySVG2(&Pos{float64(x), float64(y)}, &Size{float64(w), float64(h)})
}

func NewTinySVGf(x, y, w, h float64) (*Document, *Tag) {
	return NewTinySVG2(&Pos{x, y}, &Size{w, h})
}

// AddRect adds a rectangle, given x and y position, width and height.
// No color is being set.
func (svg *Tag) AddRect(x, y, w, h int) *Tag {
	return svg.Rect2(&Pos{float64(x), float64(y)}, &Size{float64(w), float64(h)}, nil)
}

// AddRoundedRect adds a rectangle, given x and y position, radius x, radius y, width and height.
// No color is being set.
func (svg *Tag) AddRoundedRect(x, y, rx, ry, w, h int) *Tag {
	return svg.RoundedRect2(&Pos{float64(x), float64(y)}, &Radius{float64(rx), float64(ry)}, &Size{float64(w), float64(h)}, nil)
}

// AddRectf adds a rectangle, given x and y position, width and height.
// No color is being set.
func (svg *Tag) AddRectf(x, y, w, h float64) *Tag {
	return svg.Rect2(&Pos{x, y}, &Size{w, h}, nil)
}

// AddRoundedRectf adds a rectangle, given x and y position, radius x, radius y, width and height.
// No color is being set.
func (svg *Tag) AddRoundedRectf(x, y, rx, ry, w, h float64) *Tag {
	return svg.RoundedRect2(&Pos{x, y}, &Radius{rx, ry}, &Size{w, h}, nil)
}

// AddText adds text. No color is being set
func (svg *Tag) AddText(x, y, fontSize int, fontFamily, text string) *Tag {
	return svg.Text2(&Pos{float64(x), float64(y)}, &Font{fontFamily, fontSize}, text, nil)
}

// Box adds a rectangle, given x and y position, width, height and color
func (svg *Tag) Box(x, y, w, h int, color string) *Tag {
	return svg.Rect2(&Pos{float64(x), float64(y)}, &Size{float64(w), float64(h)}, ColorByName(color))
}

// AddCircle adds a circle Add a circle, given a position (x, y) and a radius.
// No color is being set.
func (svg *Tag) AddCircle(x, y, radius int) *Tag {
	return svg.Circle2(&Pos{float64(x), float64(y)}, radius, nil)
}

// AddCirclef adds a circle Add a circle, given a position (x, y) and a radius.
// No color is being set.
func (svg *Tag) AddCirclef(x, y, radius float64) *Tag {
	return svg.Circlef(&Pos{x, y}, radius, nil)
}

// AddEllipse adds an ellipse with a given position (x,y) and radius (rx, ry).
// No color is being set.
func (svg *Tag) AddEllipse(x, y, rx, ry int) *Tag {
	return svg.Ellipse2(&Pos{float64(x), float64(y)}, &Radius{float64(rx), float64(ry)}, nil)
}

// AddEllipsef adds an ellipse with a given position (x,y) and radius (rx, ry).
// No color is being set.
func (svg *Tag) AddEllipsef(x, y, rx, ry float64) *Tag {
	return svg.Ellipse2(&Pos{x, y}, &Radius{rx, ry}, nil)
}

// Line adds a line from (x1, y1) to (x2, y2) with a given stroke width and color
func (svg *Tag) Line(x1, y1, x2, y2, thickness int, color string) *Tag {
	return svg.Line2(&Pos{float64(x1), float64(y1)}, &Pos{float64(x2), float64(y2)}, thickness, ColorByName(color))
}

// Triangle adds a colored triangle
func (svg *Tag) Triangle(x1, y1, x2, y2, x3, y3 int, color string) *Tag {
	return svg.Triangle2(&Pos{float64(x1), float64(y1)}, &Pos{float64(x2), float64(y2)}, &Pos{float64(x3), float64(y3)}, ColorByName(color))
}

// Poly4 adds a colored path with 4 points
func (svg *Tag) Poly4(x1, y1, x2, y2, x3, y3, x4, y4 int, color string) *Tag {
	return svg.Poly2(&Pos{float64(x1), float64(y1)}, &Pos{float64(x2), float64(y2)}, &Pos{float64(x3), float64(y3)}, &Pos{float64(x4), float64(y4)}, ColorByName(color))
}

// Circle adds a circle, given x and y position, radius and color
func (svg *Tag) Circle(x, y, radius int, color string) *Tag {
	return svg.Circle2(&Pos{float64(x), float64(y)}, radius, ColorByName(color))
}

// Ellipse adds an ellipse, given x and y position, radiuses and color
func (svg *Tag) Ellipse(x, y, xr, yr int, color string) *Tag {
	return svg.Ellipse2(&Pos{float64(x), float64(y)}, &Radius{float64(xr), float64(yr)}, ColorByName(color))
}

// Fill selects the fill color that will be used when drawing
func (svg *Tag) Fill(color string) {
	svg.AddAttrib("fill", []byte(color))
}

// ColorBytes converts r, g and b (integers in the range 0..255)
// to a color string on the form "#nnnnnn".
func ColorBytes(r, g, b int) []byte {
	return RGB(r, g, b).Bytes()
}

// ColorBytesAlpha converts integers r, g and b (the color) and also
// a given alpha (opacity) to a color-string on the form
// "rgba(255, 255, 255, 1.0)".
func ColorBytesAlpha(r, g, b int, a float64) []byte {
	return RGBA(r, g, b, a).Bytes()
}

// Pixel creates a rectangle that is 1 wide with the given color.
// Note that the size of the "pixel" depends on how large the viewBox is.
func (svg *Tag) Pixel(x, y, r, g, b int) *Tag {
	return svg.Rect2(&Pos{float64(x), float64(y)}, &Size{1.0, 1.0}, RGB(r, g, b))
}

// AlphaDot creates a small circle that can be transparent.
// Takes a position (x, y) and a color (r, g, b, a).
func (svg *Tag) AlphaDot(x, y, r, g, b int, a float32) *Tag {
	return svg.Circle2(&Pos{float64(x), float64(y)}, 1, RGBA(r, g, b, float64(a)))
}

// Dot adds a small colored circle.
// Takes a position (x, y) and a color (r, g, b).
func (svg *Tag) Dot(x, y, r, g, b int) *Tag {
	return svg.Circle2(&Pos{float64(x), float64(y)}, 1, RGB(r, g, b))
}

// Text adds text, with a color
func (svg *Tag) Text(x, y, fontSize int, fontFamily, text, color string) *Tag {
	return svg.Text2(&Pos{float64(x), float64(y)}, &Font{fontFamily, fontSize}, text, ColorByName(color))
}

// ---------------------------------

const (
	YES  = 0
	NO   = 1
	AUTO = 2
)

type YesNoAuto int

// Create a new Yes/No/Auto struct. If auto is true, it overrides the yes value.
func NewYesNoAuto(yes bool, auto bool) YesNoAuto {
	if auto {
		return AUTO
	}
	if yes {
		return YES
	}
	return NO
}

func (yna YesNoAuto) Bytes() []byte {
	switch yna {
	case YES:
		return []byte("true")
	case AUTO:
		return []byte("auto")
	default:
		return []byte("false")
	}
}

// Focusable sets the "focusable" attribute to either true, false or auto
// If "auto" is true, it overrides the value of "yes".
func (svg *Tag) Focusable(yes bool, auto bool) {
	svg.AddAttrib("focusable", NewYesNoAuto(yes, auto).Bytes())
}

// Thickness sets the stroke-width attribute
func (svg *Tag) Thickness(thickness int) {
	svg.AddAttrib("stroke-width", []byte(strconv.Itoa(thickness)))
}

// Polyline adds a set of connected straight lines, an open shape
func (svg *Tag) Polyline(points []*Pos, c *Color) *Tag {
	polyline := svg.AddNewTag([]byte("polyline"))
	var buf bytes.Buffer
	lastIndex := len(points) - 1
	for i, p := range points {
		buf.Write(f2b(p.X))
		buf.WriteByte(',')
		buf.Write(f2b(p.Y))
		if i != lastIndex {
			buf.WriteByte(' ')
		}
	}
	polyline.AddAttrib("points", buf.Bytes())
	return polyline
}

// Polygon adds a set of connected straight lines, a closed shape
func (svg *Tag) Polygon(points []*Pos, c *Color) *Tag {
	polygon := svg.AddNewTag([]byte("polygon"))
	var buf bytes.Buffer
	lastIndex := len(points) - 1
	for i, p := range points {
		buf.Write(f2b(p.X))
		buf.WriteByte(',')
		buf.Write(f2b(p.Y))
		if i != lastIndex {
			buf.WriteByte(' ')
		}
	}
	polygon.AddAttrib("points", buf.Bytes())
	polygon.Fill2(c)
	return polygon
}

func NewPos(xString, yString string) (*Pos, error) {
	x, err := strconv.ParseFloat(xString, 64)
	if err != nil {
		return nil, err
	}
	y, err := strconv.ParseFloat(yString, 64)
	if err != nil {
		return nil, err
	}
	return &Pos{x, y}, nil
}

func NewPosf(x, y float64) *Pos {
	return &Pos{x, y}
}

func PointsFromString(pointString string) ([]*Pos, error) {
	points := make([]*Pos, 0)
	for _, positionPair := range strings.Split(pointString, " ") {
		elements := strings.Split(positionPair, ",")
		if len(elements) != 2 {
			return nil, ErrPair
		}
		p, err := NewPos(elements[0], elements[1])
		if err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, nil
}

// Describe can be used for adding a description to the SVG header
func (svg *Tag) Describe(description string) {
	desc := svg.AddNewTag([]byte("desc"))
	desc.AddContent([]byte(description))
}
