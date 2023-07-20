package vt100

import (
	"fmt"
	"hash/fnv"
	"image/color"
	"os"
	"strings"
	"sync"
)

// Color aliases, for ease of use, not for performance

type AttributeColor struct {
	Data []byte
	Hash uint32
}

var (
	// Non-color attributes
	ResetAll   = NewAttributeColor("Reset all attributes")
	Bright     = NewAttributeColor("Bright")
	Dim        = NewAttributeColor("Dim")
	Underscore = NewAttributeColor("Underscore")
	Blink      = NewAttributeColor("Blink")
	Reverse    = NewAttributeColor("Reverse")
	Hidden     = NewAttributeColor("Hidden")

	None AttributeColor

	// There is also: reset, dim, underscore, reverse and hidden

	// Dark foreground colors (+ light gray)
	Black     = NewAttributeColor("Black")
	Red       = NewAttributeColor("Red")
	Green     = NewAttributeColor("Green")
	Yellow    = NewAttributeColor("Yellow")
	Blue      = NewAttributeColor("Blue")
	Magenta   = NewAttributeColor("Magenta")
	Cyan      = NewAttributeColor("Cyan")
	LightGray = NewAttributeColor("White")

	// Light foreground colors (+ dark gray)
	DarkGray     = NewAttributeColor("90")
	LightRed     = NewAttributeColor("91")
	LightGreen   = NewAttributeColor("92")
	LightYellow  = NewAttributeColor("93")
	LightBlue    = NewAttributeColor("94")
	LightMagenta = NewAttributeColor("95")
	LightCyan    = NewAttributeColor("96")
	White        = NewAttributeColor("97")

	// Aliases
	Pink = LightMagenta
	Gray = DarkGray

	// Dark background colors (+ light gray)
	BackgroundBlack     = NewAttributeColor("40")
	BackgroundRed       = NewAttributeColor("41")
	BackgroundGreen     = NewAttributeColor("42")
	BackgroundYellow    = NewAttributeColor("43")
	BackgroundBlue      = NewAttributeColor("44")
	BackgroundMagenta   = NewAttributeColor("45")
	BackgroundCyan      = NewAttributeColor("46")
	BackgroundLightGray = NewAttributeColor("47")

	// Aliases
	BackgroundWhite = BackgroundLightGray
	BackgroundGray  = BackgroundLightGray

	// Default colors (usually gray)
	Default           = NewAttributeColor("39")
	DefaultBackground = NewAttributeColor("49")
	BackgroundDefault = NewAttributeColor("49")

	// cache
	scache = make(map[string]string)
	smut   = &sync.RWMutex{}

	// Lookup tables

	DarkColorMap = map[string]AttributeColor{
		"black":        Black,
		"Black":        Black,
		"red":          Red,
		"Red":          Red,
		"green":        Green,
		"Green":        Green,
		"yellow":       Yellow,
		"Yellow":       Yellow,
		"blue":         Blue,
		"Blue":         Blue,
		"magenta":      Magenta,
		"Magenta":      Magenta,
		"cyan":         Cyan,
		"Cyan":         Cyan,
		"gray":         DarkGray,
		"Gray":         DarkGray,
		"white":        LightGray,
		"White":        LightGray,
		"lightwhite":   White,
		"LightWhite":   White,
		"darkred":      Red,
		"DarkRed":      Red,
		"darkgreen":    Green,
		"DarkGreen":    Green,
		"darkyellow":   Yellow,
		"DarkYellow":   Yellow,
		"darkblue":     Blue,
		"DarkBlue":     Blue,
		"darkmagenta":  Magenta,
		"DarkMagenta":  Magenta,
		"darkcyan":     Cyan,
		"DarkCyan":     Cyan,
		"darkgray":     DarkGray,
		"DarkGray":     DarkGray,
		"lightred":     LightRed,
		"LightRed":     LightRed,
		"lightgreen":   LightGreen,
		"LightGreen":   LightGreen,
		"lightyellow":  LightYellow,
		"LightYellow":  LightYellow,
		"lightblue":    LightBlue,
		"LightBlue":    LightBlue,
		"lightmagenta": LightMagenta,
		"LightMagenta": LightMagenta,
		"lightcyan":    LightCyan,
		"LightCyan":    LightCyan,
		"lightgray":    LightGray,
		"LightGray":    LightGray,
	}

	LightColorMap = map[string]AttributeColor{
		"black":        Black,
		"Black":        Black,
		"red":          LightRed,
		"Red":          LightRed,
		"green":        LightGreen,
		"Green":        LightGreen,
		"yellow":       LightYellow,
		"Yellow":       LightYellow,
		"blue":         LightBlue,
		"Blue":         LightBlue,
		"magenta":      LightMagenta,
		"Magenta":      LightMagenta,
		"cyan":         LightCyan,
		"Cyan":         LightCyan,
		"gray":         LightGray,
		"Gray":         LightGray,
		"white":        White,
		"White":        White,
		"lightwhite":   White,
		"LightWhite":   White,
		"lightred":     LightRed,
		"LightRed":     LightRed,
		"lightgreen":   LightGreen,
		"LightGreen":   LightGreen,
		"lightyellow":  LightYellow,
		"LightYellow":  LightYellow,
		"lightblue":    LightBlue,
		"LightBlue":    LightBlue,
		"lightmagenta": LightMagenta,
		"LightMagenta": LightMagenta,
		"lightcyan":    LightCyan,
		"LightCyan":    LightCyan,
		"lightgray":    LightGray,
		"LightGray":    LightGray,
		"darkred":      Red,
		"DarkRed":      Red,
		"darkgreen":    Green,
		"DarkGreen":    Green,
		"darkyellow":   Yellow,
		"DarkYellow":   Yellow,
		"darkblue":     Blue,
		"DarkBlue":     Blue,
		"darkmagenta":  Magenta,
		"DarkMagenta":  Magenta,
		"darkcyan":     Cyan,
		"DarkCyan":     Cyan,
		"darkgray":     DarkGray,
		"DarkGray":     DarkGray,
	}

	s2b = map[string]byte{
		"Reset":                0,
		"reset":                0,
		"Reset all attributes": 0,
		"reset all attributes": 0,
		"Bright":               1,
		"bright":               1,
		"Dim":                  2,
		"dim":                  2,
		"Underscore":           4,
		"underscore":           4,
		"Blink":                5,
		"blink":                5,
		"Reverse":              7,
		"reverse":              7,
		"Hidden":               8,
		"hidden":               8,
		"Black":                30,
		"black":                30,
		"Red":                  31,
		"red":                  31,
		"Green":                32,
		"green":                32,
		"Yellow":               33,
		"yellow":               33,
		"Blue":                 34,
		"blue":                 34,
		"Magenta":              35,
		"magenta":              35,
		"Cyan":                 36,
		"cyan":                 36,
		"White":                37,
		"white":                37,
		"0":                    0,
		"1":                    1,
		"2":                    2,
		"3":                    3,
		"4":                    4,
		"5":                    5,
		"6":                    6,
		"7":                    7,
		"8":                    8,
		"9":                    9,
		"10":                   10,
		"11":                   11,
		"12":                   12,
		"13":                   13,
		"14":                   14,
		"15":                   15,
		"16":                   16,
		"17":                   17,
		"18":                   18,
		"19":                   19,
		"20":                   20,
		"21":                   21,
		"22":                   22,
		"23":                   23,
		"24":                   24,
		"25":                   25,
		"26":                   26,
		"27":                   27,
		"28":                   28,
		"29":                   29,
		"30":                   30,
		"31":                   31,
		"32":                   32,
		"33":                   33,
		"34":                   34,
		"35":                   35,
		"36":                   36,
		"37":                   37,
		"38":                   38,
		"39":                   39,
		"40":                   40,
		"41":                   41,
		"42":                   42,
		"43":                   43,
		"44":                   44,
		"45":                   45,
		"46":                   46,
		"47":                   47,
		"48":                   48,
		"49":                   49,
		"50":                   50,
		"51":                   51,
		"52":                   52,
		"53":                   53,
		"54":                   54,
		"55":                   55,
		"56":                   56,
		"57":                   57,
		"58":                   58,
		"59":                   59,
		"60":                   60,
		"61":                   61,
		"62":                   62,
		"63":                   63,
		"64":                   64,
		"65":                   65,
		"66":                   66,
		"67":                   67,
		"68":                   68,
		"69":                   69,
		"70":                   70,
		"71":                   71,
		"72":                   72,
		"73":                   73,
		"74":                   74,
		"75":                   75,
		"76":                   76,
		"77":                   77,
		"78":                   78,
		"79":                   79,
		"80":                   80,
		"81":                   81,
		"82":                   82,
		"83":                   83,
		"84":                   84,
		"85":                   85,
		"86":                   86,
		"87":                   87,
		"88":                   88,
		"89":                   89,
		"90":                   90,
		"91":                   91,
		"92":                   92,
		"93":                   93,
		"94":                   94,
		"95":                   95,
		"96":                   96,
		"97":                   97,
		"98":                   98,
		"99":                   99,
		"100":                  100,
		"101":                  101,
		"102":                  102,
		"103":                  103,
		"104":                  104,
		"105":                  105,
		"106":                  106,
		"107":                  107,
		"108":                  108,
		"109":                  109,
		"110":                  110,
		"111":                  111,
		"112":                  112,
		"113":                  113,
		"114":                  114,
		"115":                  115,
		"116":                  116,
		"117":                  117,
		"118":                  118,
		"119":                  119,
		"120":                  120,
		"121":                  121,
		"122":                  122,
		"123":                  123,
		"124":                  124,
		"125":                  125,
		"126":                  126,
		"127":                  127,
		"128":                  128,
		"129":                  129,
		"130":                  130,
		"131":                  131,
		"132":                  132,
		"133":                  133,
		"134":                  134,
		"135":                  135,
		"136":                  136,
		"137":                  137,
		"138":                  138,
		"139":                  139,
		"140":                  140,
		"141":                  141,
		"142":                  142,
		"143":                  143,
		"144":                  144,
		"145":                  145,
		"146":                  146,
		"147":                  147,
		"148":                  148,
		"149":                  149,
		"150":                  150,
		"151":                  151,
		"152":                  152,
		"153":                  153,
		"154":                  154,
		"155":                  155,
		"156":                  156,
		"157":                  157,
		"158":                  158,
		"159":                  159,
		"160":                  160,
		"161":                  161,
		"162":                  162,
		"163":                  163,
		"164":                  164,
		"165":                  165,
		"166":                  166,
		"167":                  167,
		"168":                  168,
		"169":                  169,
		"170":                  170,
		"171":                  171,
		"172":                  172,
		"173":                  173,
		"174":                  174,
		"175":                  175,
		"176":                  176,
		"177":                  177,
		"178":                  178,
		"179":                  179,
		"180":                  180,
		"181":                  181,
		"182":                  182,
		"183":                  183,
		"184":                  184,
		"185":                  185,
		"186":                  186,
		"187":                  187,
		"188":                  188,
		"189":                  189,
		"190":                  190,
		"191":                  191,
		"192":                  192,
		"193":                  193,
		"194":                  194,
		"195":                  195,
		"196":                  196,
		"197":                  197,
		"198":                  198,
		"199":                  199,
		"200":                  200,
		"201":                  201,
		"202":                  202,
		"203":                  203,
		"204":                  204,
		"205":                  205,
		"206":                  206,
		"207":                  207,
		"208":                  208,
		"209":                  209,
		"210":                  210,
		"211":                  211,
		"212":                  212,
		"213":                  213,
		"214":                  214,
		"215":                  215,
		"216":                  216,
		"217":                  217,
		"218":                  218,
		"219":                  219,
		"220":                  220,
		"221":                  221,
		"222":                  222,
		"223":                  223,
		"224":                  224,
		"225":                  225,
		"226":                  226,
		"227":                  227,
		"228":                  228,
		"229":                  229,
		"230":                  230,
		"231":                  231,
		"232":                  232,
		"233":                  233,
		"234":                  234,
		"235":                  235,
		"236":                  236,
		"237":                  237,
		"238":                  238,
		"239":                  239,
		"240":                  240,
		"241":                  241,
		"242":                  242,
		"243":                  243,
		"244":                  244,
		"245":                  245,
		"246":                  246,
		"247":                  247,
		"248":                  248,
		"249":                  249,
		"250":                  250,
		"251":                  251,
		"252":                  252,
		"253":                  253,
		"254":                  254,
		"255":                  255,
	}

	b2s = map[byte]string{
		0:   "0",
		1:   "01",
		2:   "02",
		3:   "03",
		4:   "04",
		5:   "05",
		6:   "06",
		7:   "07",
		8:   "08",
		9:   "09",
		10:  "10",
		11:  "11",
		12:  "12",
		13:  "13",
		14:  "14",
		15:  "15",
		16:  "16",
		17:  "17",
		18:  "18",
		19:  "19",
		20:  "20",
		21:  "21",
		22:  "22",
		23:  "23",
		24:  "24",
		25:  "25",
		26:  "26",
		27:  "27",
		28:  "28",
		29:  "29",
		30:  "30",
		31:  "31",
		32:  "32",
		33:  "33",
		34:  "34",
		35:  "35",
		36:  "36",
		37:  "37",
		38:  "38",
		39:  "39",
		40:  "40",
		41:  "41",
		42:  "42",
		43:  "43",
		44:  "44",
		45:  "45",
		46:  "46",
		47:  "47",
		48:  "48",
		49:  "49",
		50:  "50",
		51:  "51",
		52:  "52",
		53:  "53",
		54:  "54",
		55:  "55",
		56:  "56",
		57:  "57",
		58:  "58",
		59:  "59",
		60:  "60",
		61:  "61",
		62:  "62",
		63:  "63",
		64:  "64",
		65:  "65",
		66:  "66",
		67:  "67",
		68:  "68",
		69:  "69",
		70:  "70",
		71:  "71",
		72:  "72",
		73:  "73",
		74:  "74",
		75:  "75",
		76:  "76",
		77:  "77",
		78:  "78",
		79:  "79",
		80:  "80",
		81:  "81",
		82:  "82",
		83:  "83",
		84:  "84",
		85:  "85",
		86:  "86",
		87:  "87",
		88:  "88",
		89:  "89",
		90:  "90",
		91:  "91",
		92:  "92",
		93:  "93",
		94:  "94",
		95:  "95",
		96:  "96",
		97:  "97",
		98:  "98",
		99:  "99",
		100: "100",
		101: "101",
		102: "102",
		103: "103",
		104: "104",
		105: "105",
		106: "106",
		107: "107",
		108: "108",
		109: "109",
		110: "110",
		111: "111",
		112: "112",
		113: "113",
		114: "114",
		115: "115",
		116: "116",
		117: "117",
		118: "118",
		119: "119",
		120: "120",
		121: "121",
		122: "122",
		123: "123",
		124: "124",
		125: "125",
		126: "126",
		127: "127",
		128: "128",
		129: "129",
		130: "130",
		131: "131",
		132: "132",
		133: "133",
		134: "134",
		135: "135",
		136: "136",
		137: "137",
		138: "138",
		139: "139",
		140: "140",
		141: "141",
		142: "142",
		143: "143",
		144: "144",
		145: "145",
		146: "146",
		147: "147",
		148: "148",
		149: "149",
		150: "150",
		151: "151",
		152: "152",
		153: "153",
		154: "154",
		155: "155",
		156: "156",
		157: "157",
		158: "158",
		159: "159",
		160: "160",
		161: "161",
		162: "162",
		163: "163",
		164: "164",
		165: "165",
		166: "166",
		167: "167",
		168: "168",
		169: "169",
		170: "170",
		171: "171",
		172: "172",
		173: "173",
		174: "174",
		175: "175",
		176: "176",
		177: "177",
		178: "178",
		179: "179",
		180: "180",
		181: "181",
		182: "182",
		183: "183",
		184: "184",
		185: "185",
		186: "186",
		187: "187",
		188: "188",
		189: "189",
		190: "190",
		191: "191",
		192: "192",
		193: "193",
		194: "194",
		195: "195",
		196: "196",
		197: "197",
		198: "198",
		199: "199",
		200: "200",
		201: "201",
		202: "202",
		203: "203",
		204: "204",
		205: "205",
		206: "206",
		207: "207",
		208: "208",
		209: "209",
		210: "210",
		211: "211",
		212: "212",
		213: "213",
		214: "214",
		215: "215",
		216: "216",
		217: "217",
		218: "218",
		219: "219",
		220: "220",
		221: "221",
		222: "222",
		223: "223",
		224: "224",
		225: "225",
		226: "226",
		227: "227",
		228: "228",
		229: "229",
		230: "230",
		231: "231",
		232: "232",
		233: "233",
		234: "234",
		235: "235",
		236: "236",
		237: "237",
		238: "238",
		239: "239",
		240: "240",
		241: "241",
		242: "242",
		243: "243",
		244: "244",
		245: "245",
		246: "246",
		247: "247",
		248: "248",
		249: "249",
		250: "250",
		251: "251",
		252: "252",
		253: "253",
		254: "254",
		255: "255",
	}
)

// HashBytes takes a byte slice and returns a uint32 FNV-1a hash sum
func HashBytes(data []byte) uint32 {
	hash := fnv.New32a()
	hash.Write(data)
	return hash.Sum32()
}

func NewAttributeColor(attributes ...string) AttributeColor {
	result := make([]byte, len(attributes))
	for i, s := range attributes {
		result[i] = s2b[s] // if the element is not found in the map, 0 is used
	}
	return AttributeColor{result, HashBytes(result)}
}

func (ac AttributeColor) Head() byte {
	// no error checking
	return ac.Data[0]
}

func (ac AttributeColor) Tail() []byte {
	// no error checking
	return ac.Data[1:]
}

// Modify color attributes so that they become background color attributes instead
func (ac AttributeColor) Background() AttributeColor {
	newA := make([]byte, 0, len(ac.Data))
	foundOne := false
	for _, attr := range ac.Data {
		if (30 <= attr) && (attr <= 39) {
			// convert foreground color to background color attribute
			newA = append(newA, attr+10)
			foundOne = true
		}
		// skip the rest
	}
	// Did not find a background attribute to convert, keep any existing background attributes
	if !foundOne {
		for _, attr := range ac.Data {
			if (40 <= attr) && (attr <= 49) {
				newA = append(newA, attr)
			}
		}
	}
	return AttributeColor{newA, HashBytes(newA)}
}

// Return the VT100 terminal codes for setting this combination of attributes and color attributes
func (ac AttributeColor) String() string {
	id := string(ac.Data)

	smut.RLock()
	if s, has := scache[id]; has {
		smut.RUnlock()
		return s
	}
	smut.RUnlock()

	var sb strings.Builder
	for i, b := range ac.Data {
		if i != 0 {
			sb.WriteRune(';')
		}
		sb.WriteString(b2s[b])
	}
	attributeString := sb.String()

	// Replace '{attr1};...;{attrn}' with the generated attribute string and return
	s := get(specVT100, "Set Attribute Mode", map[string]string{"{attr1};...;{attrn}": attributeString})

	// Store the value in the cache
	if len(s) > 0 {
		smut.Lock()
		scache[id] = s
		smut.Unlock()
	}

	return s
}

// Get the full string needed for outputting colored texti, with the text and stopping the color attribute
func (ac AttributeColor) StartStop(text string) string {
	return ac.String() + text + NoColor()
}

// An alias for StartStop
func (ac AttributeColor) Get(text string) string {
	return ac.String() + text + NoColor()
}

// Get the full string needed for outputting colored text, with the text, but don't reset the attributes at the end of the string
func (ac AttributeColor) Start(text string) string {
	return ac.String() + text
}

// Get the text and the terminal codes for resetting the attributes
func (ac AttributeColor) Stop(text string) string {
	return text + NoColor()
}

var maybeNoColor *string

// Return a string for resetting the attributes
func Stop() string {
	if maybeNoColor != nil {
		return *maybeNoColor
	}
	s := NoColor()
	maybeNoColor = &s
	return s
}

// Use this color to output the given text. Will reset the attributes at the end of the string. Outputs a newline.
func (ac AttributeColor) Output(text string) {
	fmt.Println(ac.Get(text))
}

// Same as output, but outputs to stderr instead of stdout
func (ac AttributeColor) Error(text string) {
	fmt.Fprintln(os.Stderr, ac.Get(text))
}

func (ac AttributeColor) Combine(other AttributeColor) AttributeColor {
	for _, a1 := range ac.Data {
		a2has := false
		for _, a2 := range other.Data {
			if a1 == a2 {
				a2has = true
				break
			}
		}
		if !a2has {
			other.Data = append(other.Data, a1)
		}
	}
	other.Hash = HashBytes(other.Data)
	return other
}

// Return a new AttributeColor that has "Bright" added to the list of attributes
func (ac AttributeColor) Bright() AttributeColor {
	data := append(ac.Data, Bright.Head())
	return AttributeColor{data, HashBytes(data)}
}

// Output a string at x, y with the given colors
func Write(x, y int, text string, fg, bg AttributeColor) {
	SetXY(uint(x), uint(y))
	fmt.Print(fg.Combine(bg).Get(text))
}

// Output a rune at x, y with the given colors
func WriteRune(x, y int, r rune, fg, bg AttributeColor) {
	SetXY(uint(x), uint(y))
	fmt.Print(fg.Combine(bg).Get(string(r)))
}

func (ac AttributeColor) Ints() []int {
	il := make([]int, len(ac.Data))
	for index, b := range ac.Data {
		il[index] = int(b)
	}
	return il
}

// This is not part of the VT100 spec, but an easteregg for displaying 24-bit
// "true color" on some terminals. Example use:
// fmt.Println(vt100.TrueColor(color.RGBA{0xa0, 0xe0, 0xff, 0xff}, "TrueColor"))
func TrueColor(fg color.Color, text string) string {
	c := color.NRGBAModel.Convert(fg).(color.NRGBA)
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm%s\x1b[0m", c.R, c.G, c.B, text)
}

// Equal checks if two colors have the same attributes, in the same order.
// The values that are being compared must have at least 1 byte in them.
func (ac *AttributeColor) Equal(other AttributeColor) bool {
	return (*ac).Hash == other.Hash
}
