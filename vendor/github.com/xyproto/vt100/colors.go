package vt100

import (
	"fmt"
	"image/color"
	"os"
	"strings"
	"sync"
)

// Color aliases, for ease of use, not for performance

type AttributeColor []byte

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

	scache = make(map[string]string)
	smut   = &sync.RWMutex{}
)

func s2b(s string) byte {
	switch s {
	case "Reset":
		return 0
	case "reset":
		return 0
	case "Reset all attributes":
		return 0
	case "reset all attributes":
		return 0
	case "Bright":
		return 1
	case "bright":
		return 1
	case "Dim":
		return 2
	case "dim":
		return 2
	case "Underscore":
		return 4
	case "underscore":
		return 4
	case "Blink":
		return 5
	case "blink":
		return 5
	case "Reverse":
		return 7
	case "reverse":
		return 7
	case "Hidden":
		return 8
	case "hidden":
		return 8
	case "Black":
		return 30
	case "black":
		return 30
	case "Red":
		return 31
	case "red":
		return 31
	case "Green":
		return 32
	case "green":
		return 32
	case "Yellow":
		return 33
	case "yellow":
		return 33
	case "Blue":
		return 34
	case "blue":
		return 34
	case "Magenta":
		return 35
	case "magenta":
		return 35
	case "Cyan":
		return 36
	case "cyan":
		return 36
	case "White":
		return 37
	case "white":
		return 37
	case "0":
		return 0
	case "1":
		return 1
	case "2":
		return 2
	case "3":
		return 3
	case "4":
		return 4
	case "5":
		return 5
	case "6":
		return 6
	case "7":
		return 7
	case "8":
		return 8
	case "9":
		return 9
	case "10":
		return 10
	case "11":
		return 11
	case "12":
		return 12
	case "13":
		return 13
	case "14":
		return 14
	case "15":
		return 15
	case "16":
		return 16
	case "17":
		return 17
	case "18":
		return 18
	case "19":
		return 19
	case "20":
		return 20
	case "21":
		return 21
	case "22":
		return 22
	case "23":
		return 23
	case "24":
		return 24
	case "25":
		return 25
	case "26":
		return 26
	case "27":
		return 27
	case "28":
		return 28
	case "29":
		return 29
	case "30":
		return 30
	case "31":
		return 31
	case "32":
		return 32
	case "33":
		return 33
	case "34":
		return 34
	case "35":
		return 35
	case "36":
		return 36
	case "37":
		return 37
	case "38":
		return 38
	case "39":
		return 39
	case "40":
		return 40
	case "41":
		return 41
	case "42":
		return 42
	case "43":
		return 43
	case "44":
		return 44
	case "45":
		return 45
	case "46":
		return 46
	case "47":
		return 47
	case "48":
		return 48
	case "49":
		return 49
	case "50":
		return 50
	case "51":
		return 51
	case "52":
		return 52
	case "53":
		return 53
	case "54":
		return 54
	case "55":
		return 55
	case "56":
		return 56
	case "57":
		return 57
	case "58":
		return 58
	case "59":
		return 59
	case "60":
		return 60
	case "61":
		return 61
	case "62":
		return 62
	case "63":
		return 63
	case "64":
		return 64
	case "65":
		return 65
	case "66":
		return 66
	case "67":
		return 67
	case "68":
		return 68
	case "69":
		return 69
	case "70":
		return 70
	case "71":
		return 71
	case "72":
		return 72
	case "73":
		return 73
	case "74":
		return 74
	case "75":
		return 75
	case "76":
		return 76
	case "77":
		return 77
	case "78":
		return 78
	case "79":
		return 79
	case "80":
		return 80
	case "81":
		return 81
	case "82":
		return 82
	case "83":
		return 83
	case "84":
		return 84
	case "85":
		return 85
	case "86":
		return 86
	case "87":
		return 87
	case "88":
		return 88
	case "89":
		return 89
	case "90":
		return 90
	case "91":
		return 91
	case "92":
		return 92
	case "93":
		return 93
	case "94":
		return 94
	case "95":
		return 95
	case "96":
		return 96
	case "97":
		return 97
	case "98":
		return 98
	case "99":
		return 99
	case "100":
		return 100
	case "101":
		return 101
	case "102":
		return 102
	case "103":
		return 103
	case "104":
		return 104
	case "105":
		return 105
	case "106":
		return 106
	case "107":
		return 107
	case "108":
		return 108
	case "109":
		return 109
	case "110":
		return 110
	case "111":
		return 111
	case "112":
		return 112
	case "113":
		return 113
	case "114":
		return 114
	case "115":
		return 115
	case "116":
		return 116
	case "117":
		return 117
	case "118":
		return 118
	case "119":
		return 119
	case "120":
		return 120
	case "121":
		return 121
	case "122":
		return 122
	case "123":
		return 123
	case "124":
		return 124
	case "125":
		return 125
	case "126":
		return 126
	case "127":
		return 127
	case "128":
		return 128
	case "129":
		return 129
	case "130":
		return 130
	case "131":
		return 131
	case "132":
		return 132
	case "133":
		return 133
	case "134":
		return 134
	case "135":
		return 135
	case "136":
		return 136
	case "137":
		return 137
	case "138":
		return 138
	case "139":
		return 139
	case "140":
		return 140
	case "141":
		return 141
	case "142":
		return 142
	case "143":
		return 143
	case "144":
		return 144
	case "145":
		return 145
	case "146":
		return 146
	case "147":
		return 147
	case "148":
		return 148
	case "149":
		return 149
	case "150":
		return 150
	case "151":
		return 151
	case "152":
		return 152
	case "153":
		return 153
	case "154":
		return 154
	case "155":
		return 155
	case "156":
		return 156
	case "157":
		return 157
	case "158":
		return 158
	case "159":
		return 159
	case "160":
		return 160
	case "161":
		return 161
	case "162":
		return 162
	case "163":
		return 163
	case "164":
		return 164
	case "165":
		return 165
	case "166":
		return 166
	case "167":
		return 167
	case "168":
		return 168
	case "169":
		return 169
	case "170":
		return 170
	case "171":
		return 171
	case "172":
		return 172
	case "173":
		return 173
	case "174":
		return 174
	case "175":
		return 175
	case "176":
		return 176
	case "177":
		return 177
	case "178":
		return 178
	case "179":
		return 179
	case "180":
		return 180
	case "181":
		return 181
	case "182":
		return 182
	case "183":
		return 183
	case "184":
		return 184
	case "185":
		return 185
	case "186":
		return 186
	case "187":
		return 187
	case "188":
		return 188
	case "189":
		return 189
	case "190":
		return 190
	case "191":
		return 191
	case "192":
		return 192
	case "193":
		return 193
	case "194":
		return 194
	case "195":
		return 195
	case "196":
		return 196
	case "197":
		return 197
	case "198":
		return 198
	case "199":
		return 199
	case "200":
		return 200
	case "201":
		return 201
	case "202":
		return 202
	case "203":
		return 203
	case "204":
		return 204
	case "205":
		return 205
	case "206":
		return 206
	case "207":
		return 207
	case "208":
		return 208
	case "209":
		return 209
	case "210":
		return 210
	case "211":
		return 211
	case "212":
		return 212
	case "213":
		return 213
	case "214":
		return 214
	case "215":
		return 215
	case "216":
		return 216
	case "217":
		return 217
	case "218":
		return 218
	case "219":
		return 219
	case "220":
		return 220
	case "221":
		return 221
	case "222":
		return 222
	case "223":
		return 223
	case "224":
		return 224
	case "225":
		return 225
	case "226":
		return 226
	case "227":
		return 227
	case "228":
		return 228
	case "229":
		return 229
	case "230":
		return 230
	case "231":
		return 231
	case "232":
		return 232
	case "233":
		return 233
	case "234":
		return 234
	case "235":
		return 235
	case "236":
		return 236
	case "237":
		return 237
	case "238":
		return 238
	case "239":
		return 239
	case "240":
		return 240
	case "241":
		return 241
	case "242":
		return 242
	case "243":
		return 243
	case "244":
		return 244
	case "245":
		return 245
	case "246":
		return 246
	case "247":
		return 247
	case "248":
		return 248
	case "249":
		return 249
	case "250":
		return 250
	case "251":
		return 251
	case "252":
		return 252
	case "253":
		return 253
	case "254":
		return 254
	case "255":
		return 255
	default:
		return 0
	}
}

func b2s(b byte) string {
	switch b {
	case 0:
		return "0"
	case 1:
		return "01"
	case 2:
		return "02"
	case 3:
		return "03"
	case 4:
		return "04"
	case 5:
		return "05"
	case 6:
		return "06"
	case 7:
		return "07"
	case 8:
		return "08"
	case 9:
		return "09"
	case 10:
		return "10"
	case 11:
		return "11"
	case 12:
		return "12"
	case 13:
		return "13"
	case 14:
		return "14"
	case 15:
		return "15"
	case 16:
		return "16"
	case 17:
		return "17"
	case 18:
		return "18"
	case 19:
		return "19"
	case 20:
		return "20"
	case 21:
		return "21"
	case 22:
		return "22"
	case 23:
		return "23"
	case 24:
		return "24"
	case 25:
		return "25"
	case 26:
		return "26"
	case 27:
		return "27"
	case 28:
		return "28"
	case 29:
		return "29"
	case 30:
		return "30"
	case 31:
		return "31"
	case 32:
		return "32"
	case 33:
		return "33"
	case 34:
		return "34"
	case 35:
		return "35"
	case 36:
		return "36"
	case 37:
		return "37"
	case 38:
		return "38"
	case 39:
		return "39"
	case 40:
		return "40"
	case 41:
		return "41"
	case 42:
		return "42"
	case 43:
		return "43"
	case 44:
		return "44"
	case 45:
		return "45"
	case 46:
		return "46"
	case 47:
		return "47"
	case 48:
		return "48"
	case 49:
		return "49"
	case 50:
		return "50"
	case 51:
		return "51"
	case 52:
		return "52"
	case 53:
		return "53"
	case 54:
		return "54"
	case 55:
		return "55"
	case 56:
		return "56"
	case 57:
		return "57"
	case 58:
		return "58"
	case 59:
		return "59"
	case 60:
		return "60"
	case 61:
		return "61"
	case 62:
		return "62"
	case 63:
		return "63"
	case 64:
		return "64"
	case 65:
		return "65"
	case 66:
		return "66"
	case 67:
		return "67"
	case 68:
		return "68"
	case 69:
		return "69"
	case 70:
		return "70"
	case 71:
		return "71"
	case 72:
		return "72"
	case 73:
		return "73"
	case 74:
		return "74"
	case 75:
		return "75"
	case 76:
		return "76"
	case 77:
		return "77"
	case 78:
		return "78"
	case 79:
		return "79"
	case 80:
		return "80"
	case 81:
		return "81"
	case 82:
		return "82"
	case 83:
		return "83"
	case 84:
		return "84"
	case 85:
		return "85"
	case 86:
		return "86"
	case 87:
		return "87"
	case 88:
		return "88"
	case 89:
		return "89"
	case 90:
		return "90"
	case 91:
		return "91"
	case 92:
		return "92"
	case 93:
		return "93"
	case 94:
		return "94"
	case 95:
		return "95"
	case 96:
		return "96"
	case 97:
		return "97"
	case 98:
		return "98"
	case 99:
		return "99"
	case 100:
		return "100"
	case 101:
		return "101"
	case 102:
		return "102"
	case 103:
		return "103"
	case 104:
		return "104"
	case 105:
		return "105"
	case 106:
		return "106"
	case 107:
		return "107"
	case 108:
		return "108"
	case 109:
		return "109"
	case 110:
		return "110"
	case 111:
		return "111"
	case 112:
		return "112"
	case 113:
		return "113"
	case 114:
		return "114"
	case 115:
		return "115"
	case 116:
		return "116"
	case 117:
		return "117"
	case 118:
		return "118"
	case 119:
		return "119"
	case 120:
		return "120"
	case 121:
		return "121"
	case 122:
		return "122"
	case 123:
		return "123"
	case 124:
		return "124"
	case 125:
		return "125"
	case 126:
		return "126"
	case 127:
		return "127"
	case 128:
		return "128"
	case 129:
		return "129"
	case 130:
		return "130"
	case 131:
		return "131"
	case 132:
		return "132"
	case 133:
		return "133"
	case 134:
		return "134"
	case 135:
		return "135"
	case 136:
		return "136"
	case 137:
		return "137"
	case 138:
		return "138"
	case 139:
		return "139"
	case 140:
		return "140"
	case 141:
		return "141"
	case 142:
		return "142"
	case 143:
		return "143"
	case 144:
		return "144"
	case 145:
		return "145"
	case 146:
		return "146"
	case 147:
		return "147"
	case 148:
		return "148"
	case 149:
		return "149"
	case 150:
		return "150"
	case 151:
		return "151"
	case 152:
		return "152"
	case 153:
		return "153"
	case 154:
		return "154"
	case 155:
		return "155"
	case 156:
		return "156"
	case 157:
		return "157"
	case 158:
		return "158"
	case 159:
		return "159"
	case 160:
		return "160"
	case 161:
		return "161"
	case 162:
		return "162"
	case 163:
		return "163"
	case 164:
		return "164"
	case 165:
		return "165"
	case 166:
		return "166"
	case 167:
		return "167"
	case 168:
		return "168"
	case 169:
		return "169"
	case 170:
		return "170"
	case 171:
		return "171"
	case 172:
		return "172"
	case 173:
		return "173"
	case 174:
		return "174"
	case 175:
		return "175"
	case 176:
		return "176"
	case 177:
		return "177"
	case 178:
		return "178"
	case 179:
		return "179"
	case 180:
		return "180"
	case 181:
		return "181"
	case 182:
		return "182"
	case 183:
		return "183"
	case 184:
		return "184"
	case 185:
		return "185"
	case 186:
		return "186"
	case 187:
		return "187"
	case 188:
		return "188"
	case 189:
		return "189"
	case 190:
		return "190"
	case 191:
		return "191"
	case 192:
		return "192"
	case 193:
		return "193"
	case 194:
		return "194"
	case 195:
		return "195"
	case 196:
		return "196"
	case 197:
		return "197"
	case 198:
		return "198"
	case 199:
		return "199"
	case 200:
		return "200"
	case 201:
		return "201"
	case 202:
		return "202"
	case 203:
		return "203"
	case 204:
		return "204"
	case 205:
		return "205"
	case 206:
		return "206"
	case 207:
		return "207"
	case 208:
		return "208"
	case 209:
		return "209"
	case 210:
		return "210"
	case 211:
		return "211"
	case 212:
		return "212"
	case 213:
		return "213"
	case 214:
		return "214"
	case 215:
		return "215"
	case 216:
		return "216"
	case 217:
		return "217"
	case 218:
		return "218"
	case 219:
		return "219"
	case 220:
		return "220"
	case 221:
		return "221"
	case 222:
		return "222"
	case 223:
		return "223"
	case 224:
		return "224"
	case 225:
		return "225"
	case 226:
		return "226"
	case 227:
		return "227"
	case 228:
		return "228"
	case 229:
		return "229"
	case 230:
		return "230"
	case 231:
		return "231"
	case 232:
		return "232"
	case 233:
		return "233"
	case 234:
		return "234"
	case 235:
		return "235"
	case 236:
		return "236"
	case 237:
		return "237"
	case 238:
		return "238"
	case 239:
		return "239"
	case 240:
		return "240"
	case 241:
		return "241"
	case 242:
		return "242"
	case 243:
		return "243"
	case 244:
		return "244"
	case 245:
		return "245"
	case 246:
		return "246"
	case 247:
		return "247"
	case 248:
		return "248"
	case 249:
		return "249"
	case 250:
		return "250"
	case 251:
		return "251"
	case 252:
		return "252"
	case 253:
		return "253"
	case 254:
		return "254"
	case 255:
		return "255"
	default:
		return "0"
	}
}

func NewAttributeColor(attributes ...string) AttributeColor {
	result := make([]byte, len(attributes))
	for i, s := range attributes {
		result[i] = s2b(s) // if the element is not found in the map, 0 is used
	}
	return AttributeColor(result)
}

func (ac AttributeColor) Head() byte {
	// no error checking
	return ac[0]
}

func (ac AttributeColor) Tail() []byte {
	// no error checking
	return ac[1:]
}

// Modify color attributes so that they become background color attributes instead
func (ac AttributeColor) Background() AttributeColor {
	newA := make(AttributeColor, 0, len(ac))
	foundOne := false
	for _, attr := range ac {
		if (30 <= attr) && (attr <= 39) {
			// convert foreground color to background color attribute
			newA = append(newA, attr+10)
			foundOne = true
		}
		// skip the rest
	}
	// Did not find a background attribute to convert, keep any existing background attributes
	if !foundOne {
		for _, attr := range ac {
			if (40 <= attr) && (attr <= 49) {
				newA = append(newA, attr)
			}
		}
	}
	return newA
}

// Return the VT100 terminal codes for setting this combination of attributes and color attributes
func (ac AttributeColor) String() string {
	id := string(ac)

	smut.RLock()
	if s, has := scache[id]; has {
		smut.RUnlock()
		return s
	}
	smut.RUnlock()

	var sb strings.Builder
	for i, b := range ac {
		if i != 0 {
			sb.WriteRune(';')
		}
		sb.WriteString(b2s(b))
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
	for _, a1 := range ac {
		a2has := false
		for _, a2 := range other {
			if a1 == a2 {
				a2has = true
				break
			}
		}
		if !a2has {
			other = append(other, a1)
		}
	}
	return AttributeColor(other)
}

// Return a new AttributeColor that has "Bright" added to the list of attributes
func (ac AttributeColor) Bright() AttributeColor {
	return AttributeColor(append(ac, Bright.Head()))
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
	il := make([]int, len(ac))
	for index, b := range ac {
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
	l1 := len(*ac)
	l2 := len(other)
	if l1 != l2 {
		return false
	}
	// l1 == l2 at this point
	for i := 0; i < l1; i++ {
		if (*ac)[i] != other[i] {
			return false
		}
	}
	return true
}
