package onthefly

//
// Support for TinySVG 1.2
//
// Some functions are suffixed with "2" to avoid breaking backward compatibility.
//
// TODO: Refactor this package as a new and shiny package in a different namespace.
//

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

var errPair = errors.New("position pairs must be exactly two comma separated numbers")

type Pos struct {
	x float64
	y float64
}

type Size struct {
	w float64
	h float64
}

type Radius struct {
	x float64
	y float64
}

type Color struct {
	r int     // red, 0..255
	g int     // green, 0..255
	b int     // blue, 0..255
	a float64 // alpha, 0.0..1.0
	n string  // name (optional, will override the above values)
}

type Font struct {
	family string
	size   int
}

// NewTinySVG2 creates new TinySVG 1.2 image. Pos and Size defines the viewbox
func NewTinySVG2(p *Pos, s *Size) (*Page, *Tag) {
	// No page title is needed when building an SVG tag tree
	page := NewPage("", `<?xml version="1.0" encoding="UTF-8"?>`)

	// No longer needed for TinySVG 1.2. See: https://www.w3.org/TR/SVGTiny12/intro.html#defining
	// <!DOCTYPE svg PUBLIC "-//W3C//DTD SVG 1.1 Tiny//EN" "http://www.w3.org/Graphics/SVG/1.1/DTD/svg11-tiny.dtd">

	// Add the root tag
	svg := page.root.AddNewTag("svg")
	svg.AddAttrib("xmlns", "http://www.w3.org/2000/svg")
	svg.AddAttrib("version", "1.2")
	svg.AddAttrib("baseProfile", "tiny")
	svg.AddAttrib("viewBox", fmt.Sprintf("%f %f %f %f", p.x, p.y, s.w, s.h))
	return page, svg
}

// Create a new TinySVG document, where the width and height is defined in pixels, using the "px" suffix
func NewTinySVGPixels(w, h int) (*Page, *Tag) {
	page := NewPage("", `<?xml version="1.0" encoding="UTF-8"?>`)
	svg := page.root.AddNewTag("svg")
	svg.AddAttrib("xmlns", "http://www.w3.org/2000/svg")
	svg.AddAttrib("version", "1.2")
	svg.AddAttrib("baseProfile", "tiny")
	svg.AddAttrib("viewBox", fmt.Sprintf("%d %d %d %d", 0, 0, w, h))
	svg.AddAttrib("width", fmt.Sprintf("%dpx", w))
	svg.AddAttrib("height", fmt.Sprintf("%dpx", h))
	return page, svg
}

func f2s(x float64) string {
	fs := fmt.Sprintf("%f", x)
	// Drop ".0" if the number ends with that
	if strings.HasSuffix(fs, ".0") {
		return fs[:len(fs)-2]
	}
	// Drop ".000000" if the number ends with that
	if strings.HasSuffix(fs, ".000000") {
		return fs[:len(fs)-7]
	}
	return fs
}

func i2s(x int) string {
	return strconv.Itoa(x)
}

// Rect a rectangle, given x and y position, width and height.
// No color is being set.
func (svg *Tag) Rect2(p *Pos, s *Size, c *Color) *Tag {
	rect := svg.AddNewTag("rect")
	rect.AddAttrib("x", f2s(p.x))
	rect.AddAttrib("y", f2s(p.y))
	rect.AddAttrib("width", f2s(s.w))
	rect.AddAttrib("height", f2s(s.h))
	rect.Fill2(c)
	return rect
}

// Text adds text. No color is being set
func (svg *Tag) Text2(p *Pos, f *Font, message string, c *Color) *Tag {
	text := svg.AddNewTag("text")
	text.AddAttrib("x", f2s(p.x))
	text.AddAttrib("y", f2s(p.y))
	text.AddAttrib("font-family", f.family)
	text.AddAttrib("font-size", i2s(f.size))
	text.Fill2(c)
	text.AddContent(message)
	return text
}

// Circle adds a circle, given a position, radius and color
func (svg *Tag) Circle2(p *Pos, radius int, c *Color) *Tag {
	circle := svg.AddNewTag("circle")
	circle.AddAttrib("cx", f2s(p.x))
	circle.AddAttrib("cy", f2s(p.y))
	circle.AddAttrib("r", i2s(radius))
	circle.Fill2(c)
	return circle
}

// Circle adds a circle, given a position, radius and color
func (svg *Tag) Circlef(p *Pos, radius float64, c *Color) *Tag {
	circle := svg.AddNewTag("circle")
	circle.AddAttrib("cx", f2s(p.x))
	circle.AddAttrib("cy", f2s(p.y))
	circle.AddAttrib("r", f2s(radius))
	circle.Fill2(c)
	return circle
}

// Ellipse adds an ellipse with a given position (x,y) and radius (rx, ry).
func (svg *Tag) Ellipse2(p *Pos, r *Radius, c *Color) *Tag {
	ellipse := svg.AddNewTag("ellipse")
	ellipse.AddAttrib("cx", f2s(p.x))
	ellipse.AddAttrib("cy", f2s(p.y))
	ellipse.AddAttrib("rx", f2s(r.x))
	ellipse.AddAttrib("ry", f2s(r.y))
	ellipse.Fill2(c)
	return ellipse
}

// Line adds a line from (x1, y1) to (x2, y2) with a given stroke width and color
func (svg *Tag) Line2(p1, p2 *Pos, thickness int, c *Color) *Tag {
	line := svg.AddNewTag("line")
	line.AddAttrib("x1", f2s(p1.x))
	line.AddAttrib("y1", f2s(p1.y))
	line.AddAttrib("x2", f2s(p2.x))
	line.AddAttrib("y2", f2s(p2.y))
	line.Thickness(thickness)
	line.Stroke2(c)
	return line
}

// Triangle adds a colored triangle
func (svg *Tag) Triangle2(p1, p2, p3 *Pos, c *Color) *Tag {
	triangle := svg.AddNewTag("path")
	triangle.AddAttrib("d", fmt.Sprintf("M %f %f L %f %f L %f %f L %f %f", p1.x, p1.y, p2.x, p2.y, p3.x, p3.y, p1.x, p1.y))
	triangle.Fill2(c)
	return triangle
}

// Poly2 adds a colored path with 4 points
func (svg *Tag) Poly2(p1, p2, p3, p4 *Pos, c *Color) *Tag {
	poly4 := svg.AddNewTag("path")
	poly4.AddAttrib("d", fmt.Sprintf("M %f %f L %f %f L %f %f L %f %f L %f %f", p1.x, p1.y, p2.x, p2.y, p3.x, p3.y, p4.x, p4.y, p1.x, p1.y))
	poly4.Fill2(c)
	return poly4
}

// Fill selects the fill color that will be used when drawing
func (svg *Tag) Fill2(c *Color) {
	// If no color name is given and the color is transparent, don't set a fill color
	if (c == nil) || (len(c.n) == 0 && c.a == TRANSPARENT) {
		return
	}
	svg.AddAttrib("fill", c.String())
}

// Stroke selects the stroke color that will be used when drawing
func (svg *Tag) Stroke2(c *Color) {
	// If no color name is given and the color is transparent, don't set a stroke color
	if (c == nil) || (len(c.n) == 0 && c.a == TRANSPARENT) {
		return
	}
	svg.AddAttrib("stroke", c.String())
}

// RGBString converts r, g and b (integers in the range 0..255)
// to a color string on the form "#nnnnnn".
func RGBString(r, g, b int) string {
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
	return "#" + rs + gs + bs
}

// RGBAString converts integers r, g and b (the color) and also
// a given alpha (opacity) to a color-string on the form
// "rgba(255, 255, 255, 1.0)".
func RGBAString(r, g, b int, a float64) string {
	return fmt.Sprintf("rgba(%d, %d, %d, %f)", r, g, b, a)
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
	return &Color{n: name}
}

// NewColor is the same as ColorByName
func NewColor(name string) *Color {
	return ColorByName(name)
}

// String returns the color as an RGB (#1234FF) string
// or as an RGBA (rgba(0, 1, 2 ,3)) string.
func (c *Color) String() string {
	// Return an empty string if nil
	if c == nil {
		return ""
	}
	// Return the name, if specified
	if len(c.n) != 0 {
		return c.n
	}
	// Return a regular RGB string if alpha is 1.0
	if c.a == OPAQUE {
		// Generate a rgb string
		return RGBString(c.r, c.g, c.b)
	}
	// Generate a rgba string if alpha is < 1.0
	return RGBAString(c.r, c.g, c.b, c.a)
}

// --- Convenience functions and functions for backward compatibility ---

func NewTinySVG(x, y, w, h int) (*Page, *Tag) {
	return NewTinySVG2(&Pos{float64(x), float64(y)}, &Size{float64(w), float64(h)})
}

func NewTinySVGf(x, y, w, h float64) (*Page, *Tag) {
	return NewTinySVG2(&Pos{x, y}, &Size{w, h})
}

// AddRect adds a rectangle, given x and y position, width and height.
// No color is being set.
func (svg *Tag) AddRect(x, y, w, h int) *Tag {
	return svg.Rect2(&Pos{float64(x), float64(y)}, &Size{float64(w), float64(h)}, nil)
}

// AddRectf adds a rectangle, given x and y position, width and height.
// No color is being set.
func (svg *Tag) AddRectf(x, y, w, h float64) *Tag {
	return svg.Rect2(&Pos{x, y}, &Size{w, h}, nil)
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
	svg.AddAttrib("fill", color)
}

// ColorString converts r, g and b (integers in the range 0..255)
// to a color string on the form "#nnnnnn".
func ColorString(r, g, b int) string {
	return RGB(r, g, b).String()
}

// ColorStringAlpha converts integers r, g and b (the color) and also
// a given alpha (opacity) to a color-string on the form
// "rgba(255, 255, 255, 1.0)".
func ColorStringAlpha(r, g, b int, a float64) string {
	return RGBA(r, g, b, a).String()
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

func (yna YesNoAuto) String() string {
	switch yna {
	case YES:
		return "true"
	case AUTO:
		return "auto"
	default:
		return "false"
	}
}

// Focusable sets the "focusable" attribute to either true, false or auto
// If "auto" is true, it overrides the value of "yes".
func (svg *Tag) Focusable(yes bool, auto bool) {
	svg.AddAttrib("focusable", NewYesNoAuto(yes, auto).String())
}

// Thickness sets the stroke-width attribute
func (svg *Tag) Thickness(thickness int) {
	svg.AddAttrib("stroke-width", i2s(thickness))
}

// Polyline adds a set of connected straight lines, an open shape
func (svg *Tag) Polyline(points []*Pos, c *Color) *Tag {
	polyline := svg.AddNewTag("polyline")
	var buf bytes.Buffer
	lastIndex := len(points) - 1
	for i, p := range points {
		buf.WriteString(f2s(p.x) + "," + f2s(p.y))
		if i != lastIndex {
			buf.WriteString(" ")
		}
	}
	polyline.AddAttrib("points", buf.String())
	polyline.Fill2(c)
	return polyline
}

// Polygon adds a set of connected straight lines, a closed shape
func (svg *Tag) Polygon(points []*Pos, c *Color) *Tag {
	polygon := svg.AddNewTag("polygon")
	var buf bytes.Buffer
	lastIndex := len(points) - 1
	for i, p := range points {
		buf.WriteString(f2s(p.x) + "," + f2s(p.y))
		if i != lastIndex {
			buf.WriteString(" ")
		}
	}
	polygon.AddAttrib("points", buf.String())
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

func PointsFromString(pointString string) ([]*Pos, error) {
	points := make([]*Pos, 0)
	for _, positionPair := range strings.Split(pointString, " ") {
		elements := strings.Split(positionPair, ",")
		if len(elements) != 2 {
			return nil, errPair
		}
		p, err := NewPos(elements[0], elements[1])
		if err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, nil
}
