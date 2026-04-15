package vt

// X11Colors maps every canonical X11 color name (no numbered variants) to a
// TrueColor AttributeColor. The RGB values are taken from the X11 rgb.txt
// database. A handful of CSS/SVG-only names that are absent from X11 but
// commonly expected (aqua, fuchsia, indigo, lime, olive, silver, teal) are
// included for convenience, using their W3C/CSS3 definitions.
//
// Grey aliases (darkgrey, slategrey, …) are included alongside their grey
// spellings.
var X11Colors = map[string]AttributeColor{
	// A
	"aliceblue":    TrueColor(240, 248, 255),
	"antiquewhite": TrueColor(250, 235, 215),
	"aquamarine":   TrueColor(127, 255, 212),
	"azure":        TrueColor(240, 255, 255),
	// B
	"beige":          TrueColor(245, 245, 220),
	"bisque":         TrueColor(255, 228, 196),
	"blanchedalmond": TrueColor(255, 235, 205),
	"blueviolet":     TrueColor(138, 43, 226),
	"brown":          TrueColor(165, 42, 42),
	"burlywood":      TrueColor(222, 184, 135),
	// C
	"cadetblue":      TrueColor(95, 158, 160),
	"chartreuse":     TrueColor(127, 255, 0),
	"chocolate":      TrueColor(210, 105, 30),
	"coral":          TrueColor(255, 127, 80),
	"cornflowerblue": TrueColor(100, 149, 237),
	"cornsilk":       TrueColor(255, 248, 220),
	"crimson":        TrueColor(220, 20, 60),
	// D
	"darkblue":       TrueColor(0, 0, 139),
	"darkcyan":       TrueColor(0, 139, 139),
	"darkgoldenrod":  TrueColor(184, 134, 11),
	"darkgray":       TrueColor(169, 169, 169),
	"darkgrey":       TrueColor(169, 169, 169),
	"darkgreen":      TrueColor(0, 100, 0),
	"darkkhaki":      TrueColor(189, 183, 107),
	"darkmagenta":    TrueColor(139, 0, 139),
	"darkolivegreen": TrueColor(85, 107, 47),
	"darkorange":     TrueColor(255, 140, 0),
	"darkorchid":     TrueColor(153, 50, 204),
	"darkred":        TrueColor(139, 0, 0),
	"darksalmon":     TrueColor(233, 150, 122),
	"darkseagreen":   TrueColor(143, 188, 143),
	"darkslateblue":  TrueColor(72, 61, 139),
	"darkslategray":  TrueColor(47, 79, 79),
	"darkslategrey":  TrueColor(47, 79, 79),
	"darkturquoise":  TrueColor(0, 206, 209),
	"darkviolet":     TrueColor(148, 0, 211),
	"deeppink":       TrueColor(255, 20, 147),
	"deepskyblue":    TrueColor(0, 191, 255),
	"dimgray":        TrueColor(105, 105, 105),
	"dimgrey":        TrueColor(105, 105, 105),
	"dodgerblue":     TrueColor(30, 144, 255),
	// F
	"firebrick":   TrueColor(178, 34, 34),
	"floralwhite": TrueColor(255, 250, 240),
	"forestgreen": TrueColor(34, 139, 34),
	// G
	"gainsboro":   TrueColor(220, 220, 220),
	"ghostwhite":  TrueColor(248, 248, 255),
	"gold":        TrueColor(255, 215, 0),
	"goldenrod":   TrueColor(218, 165, 32),
	"greenyellow": TrueColor(173, 255, 47),
	// H
	"honeydew": TrueColor(240, 255, 240),
	"hotpink":  TrueColor(255, 105, 180),
	// I
	"indianred": TrueColor(205, 92, 92),
	// K
	"khaki": TrueColor(240, 230, 140),
	// L
	"lavender":             TrueColor(230, 230, 250),
	"lavenderblush":        TrueColor(255, 240, 245),
	"lawngreen":            TrueColor(124, 252, 0),
	"lemonchiffon":         TrueColor(255, 250, 205),
	"lightblue":            TrueColor(173, 216, 230),
	"lightcoral":           TrueColor(240, 128, 128),
	"lightcyan":            TrueColor(224, 255, 255),
	"lightgoldenrod":       TrueColor(238, 221, 130),
	"lightgoldenrodyellow": TrueColor(250, 250, 210),
	"lightgray":            TrueColor(211, 211, 211),
	"lightgrey":            TrueColor(211, 211, 211),
	"lightgreen":           TrueColor(144, 238, 144),
	"lightpink":            TrueColor(255, 182, 193),
	"lightsalmon":          TrueColor(255, 160, 122),
	"lightseagreen":        TrueColor(32, 178, 170),
	"lightskyblue":         TrueColor(135, 206, 250),
	"lightslateblue":       TrueColor(132, 112, 255),
	"lightslategray":       TrueColor(119, 136, 153),
	"lightslategrey":       TrueColor(119, 136, 153),
	"lightsteelblue":       TrueColor(176, 196, 222),
	"lightyellow":          TrueColor(255, 255, 224),
	"limegreen":            TrueColor(50, 205, 50),
	"linen":                TrueColor(250, 240, 230),
	// M
	"maroon":            TrueColor(176, 48, 96),
	"mediumaquamarine":  TrueColor(102, 205, 170),
	"mediumblue":        TrueColor(0, 0, 205),
	"mediumorchid":      TrueColor(186, 85, 211),
	"mediumpurple":      TrueColor(147, 112, 219),
	"mediumseagreen":    TrueColor(60, 179, 113),
	"mediumslateblue":   TrueColor(123, 104, 238),
	"mediumspringgreen": TrueColor(0, 250, 154),
	"mediumturquoise":   TrueColor(72, 209, 204),
	"mediumvioletred":   TrueColor(199, 21, 133),
	"midnightblue":      TrueColor(25, 25, 112),
	"mintcream":         TrueColor(245, 255, 250),
	"mistyrose":         TrueColor(255, 228, 225),
	"moccasin":          TrueColor(255, 228, 181),
	// N
	"navajowhite": TrueColor(255, 222, 173),
	"navy":        TrueColor(0, 0, 128),
	"navyblue":    TrueColor(0, 0, 128),
	// O
	"oldlace":   TrueColor(253, 245, 230),
	"olivedrab": TrueColor(107, 142, 35),
	"orange":    TrueColor(255, 165, 0),
	"orangered": TrueColor(255, 69, 0),
	"orchid":    TrueColor(218, 112, 214),
	// P
	"palegoldenrod": TrueColor(238, 232, 170),
	"palegreen":     TrueColor(152, 251, 152),
	"paleturquoise": TrueColor(175, 238, 238),
	"palevioletred": TrueColor(219, 112, 147),
	"papayawhip":    TrueColor(255, 239, 213),
	"peachpuff":     TrueColor(255, 218, 185),
	"peru":          TrueColor(205, 133, 63),
	"pink":          TrueColor(255, 192, 203),
	"plum":          TrueColor(221, 160, 221),
	"powderblue":    TrueColor(176, 224, 230),
	"purple":        TrueColor(160, 32, 240),
	// R
	"rosybrown": TrueColor(188, 143, 143),
	"royalblue": TrueColor(65, 105, 225),
	// S
	"saddlebrown": TrueColor(139, 69, 19),
	"salmon":      TrueColor(250, 128, 114),
	"sandybrown":  TrueColor(244, 164, 96),
	"seagreen":    TrueColor(46, 139, 87),
	"seashell":    TrueColor(255, 245, 238),
	"sienna":      TrueColor(160, 82, 45),
	"skyblue":     TrueColor(135, 206, 235),
	"slateblue":   TrueColor(106, 90, 205),
	"slategray":   TrueColor(112, 128, 144),
	"slategrey":   TrueColor(112, 128, 144),
	"snow":        TrueColor(255, 250, 250),
	"springgreen": TrueColor(0, 255, 127),
	"steelblue":   TrueColor(70, 130, 180),
	// T
	"tan":       TrueColor(210, 180, 140),
	"thistle":   TrueColor(216, 191, 216),
	"tomato":    TrueColor(255, 99, 71),
	"turquoise": TrueColor(64, 224, 208),
	// V
	"violet":    TrueColor(238, 130, 238),
	"violetred": TrueColor(208, 32, 144),
	// W
	"wheat":      TrueColor(245, 222, 179),
	"whitesmoke": TrueColor(245, 245, 245),
	// Y
	"yellowgreen": TrueColor(154, 205, 50),

	// CSS/SVG colors absent from X11 rgb.txt but widely expected
	"aqua":    TrueColor(0, 255, 255),   // CSS aqua = X11 cyan
	"fuchsia": TrueColor(255, 0, 255),   // CSS fuchsia = X11 magenta
	"indigo":  TrueColor(75, 0, 130),    // CSS indigo
	"lime":    TrueColor(0, 255, 0),     // CSS lime = X11 green (0,255,0)
	"olive":   TrueColor(128, 128, 0),   // CSS olive
	"silver":  TrueColor(192, 192, 192), // CSS silver
	"teal":    TrueColor(0, 128, 128),   // CSS teal
}

func init() {
	for name, color := range X11Colors {
		if _, exists := DarkColorMap[name]; !exists {
			DarkColorMap[name] = color
		}
		if _, exists := LightColorMap[name]; !exists {
			LightColorMap[name] = color
		}
	}
}
