package burnfont

import (
	"errors"
	"fmt"
	"image/color"
)

// Available is a slice with all available runes, for this package
var Available = []rune{'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z', 'Æ', 'Ø', 'Å', 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z', 'æ', 'ø', 'å', '.', ';', ',', '\'', '"', '*', '+', '!', '?', '-', '=', '_', '/', '1', '2', '3', '4', '5', '6', '7', '8', '9', '0', '(', ')', '→', '{', '}', '[', ']', '<', '>', '&', '|', '\\'}

// Drawable is an interface for anything that has a Set function, for drawing
type Drawable interface {
	Set(x, y int, c color.Color)
}

// Draw will draw an image of the selected rune at (x,y).
// If the rune is not available, an error will be returned.
// r,g,b is the main color of the rune.
func Draw(d Drawable, l rune, x, y int, r, g, b byte) error {

	fontLine := func(s string, x, y int) {
		for _, l := range s {
			if l == '*' {
				d.Set(x, y, color.NRGBA{r, g, b, 255})
			} else if l == '-' {
				d.Set(x, y, color.NRGBA{r, g, b, 64})
			}
			x++
		}
		return
	}

	switch l {
	case 'a':
		fontLine("***-", x+1, y+1)
		fontLine("--**", x+1, y+2)
		fontLine("-****", x, y+3)
		fontLine("**-**-", x, y+4)
		fontLine("-**-**", x, y+5)
	case 'A':
		fontLine("-**-", x+1, y)
		fontLine("-****-", x, y+1)
		fontLine("**--**", x, y+2)
		fontLine("******", x, y+3)
		fontLine("**--**", x, y+4)
		fontLine("**  **", x, y+5)
	case 'b':
		fontLine("***", x, y)
		fontLine("-**-", x, y+1)
		fontLine("****-", x+1, y+2)
		fontLine("** **", x+1, y+3)
		fontLine("-**-**", x, y+4)
		fontLine("**-**-", x, y+5)
	case 'B':
		fontLine("*****-", x, y)
		fontLine("-**-**", x, y+1)
		fontLine("****-", x+1, y+2)
		fontLine("** **", x+1, y+3)
		fontLine("-** **", x, y+4)
		fontLine("*****-", x, y+5)
	case 'c':
		fontLine("-****-", x, y+1)
		fontLine("**--**", x, y+2)
		fontLine("** ---", x, y+3)
		fontLine("**--**", x, y+4)
		fontLine("-****-", x, y+5)
	case 'C':
		fontLine("-****-", x, y)
		fontLine("**- -*", x, y+1)
		fontLine("**", x, y+2)
		fontLine("**", x, y+3)
		fontLine("**- -*", x, y+4)
		fontLine("-****-", x, y+5)
	case 'd':
		fontLine("***", x+3, y)
		fontLine("-**", x+3, y+1)
		fontLine("-*****", x, y+2)
		fontLine("**--**", x, y+3)
		fontLine("**  **", x, y+4)
		fontLine("-***-*", x, y+5)
	case 'D':
		fontLine("****-", x, y)
		fontLine("**-**-", x, y+1)
		fontLine("** -**", x, y+2)
		fontLine("** -**", x, y+3)
		fontLine("**-**-", x, y+4)
		fontLine("****-", x, y+5)
	case 'e':
		fontLine("-****-", x, y+1)
		fontLine("** -**", x, y+2)
		fontLine("******", x, y+3)
		fontLine("**-", x, y+4)
		fontLine("-****-", x, y+5)
	case 'E':
		fontLine("-*****", x, y)
		fontLine("**---*", x, y+1)
		fontLine("****-", x, y+2)
		fontLine("**--", x, y+3)
		fontLine("**- **", x, y+4)
		fontLine("*****-", x, y+5)
	case 'f':
		fontLine("-***-", x+1, y)
		fontLine("-**--*", x, y+1)
		fontLine("****", x, y+2)
		fontLine("-**-", x, y+3)
		fontLine("**", x+1, y+4)
		fontLine("****", x, y+5)
	case 'F':
		fontLine("*****-", x, y)
		fontLine("**--**", x, y+1)
		fontLine("**-", x, y+2)
		fontLine("****", x, y+3)
		fontLine("**-", x, y+4)
		fontLine("*-", x, y+5)
	case 'g':
		fontLine("-**-**", x, y+1)
		fontLine("** **-", x, y+2)
		fontLine("-****", x, y+3)
		fontLine("-**", x+2, y+4)
		fontLine("****-", x, y+5)
	case 'G':
		fontLine("-****-", x, y)
		fontLine("**--**", x, y+1)
		fontLine("**", x, y+2)
		fontLine("** ***", x, y+3)
		fontLine("**--**", x, y+4)
		fontLine("-***-*", x, y+5)
	case 'h':
		fontLine("***", x, y)
		fontLine("-**-", x, y+1)
		fontLine("****-", x+1, y+2)
		fontLine("**-**", x+1, y+3)
		fontLine("-** **", x, y+4)
		fontLine("*** **", x, y+5)
	case 'H':
		fontLine("**  **", x, y)
		fontLine("**--**", x, y+1)
		fontLine("******", x, y+2)
		fontLine("**--**", x, y+3)
		fontLine("**  **", x, y+4)
		fontLine("**  **", x, y+5)
	case 'i':
		fontLine("**", x+2, y)
		fontLine("--", x+2, y+1)
		fontLine("***", x+1, y+2)
		fontLine("-**", x+1, y+3)
		fontLine("-**-", x+1, y+4)
		fontLine("****", x+1, y+5)
	case 'I':
		fontLine("****", x+1, y)
		fontLine("-**-", x+1, y+1)
		fontLine("**", x+2, y+2)
		fontLine("**", x+2, y+3)
		fontLine("-**-", x+1, y+4)
		fontLine("****", x+1, y+5)
	case 'j':
		fontLine("**", x+3, y)
		fontLine("--", x+3, y+1)
		fontLine("***", x+2, y+2)
		fontLine("-**", x+2, y+3)
		fontLine("**-**", x, y+4)
		fontLine("-***-", x, y+5)
	case 'J':
		fontLine("****", x+2, y)
		fontLine("-**-", x+2, y+1)
		fontLine("**", x+3, y+2)
		fontLine("** **", x, y+3)
		fontLine("**-**", x, y+4)
		fontLine("-***-", x, y+5)
	case 'k':
		fontLine("***", x, y)
		fontLine("-**", x, y+1)
		fontLine("**-**", x+1, y+2)
		fontLine("****-", x+1, y+3)
		fontLine("-**-**", x, y+4)
		fontLine("*** **", x, y+5)
	case 'K':
		fontLine("***-**", x, y)
		fontLine("-****-", x, y+1)
		fontLine("***-", x+1, y+2)
		fontLine("****-", x+1, y+3)
		fontLine("-**-**", x, y+4)
		fontLine("*** **", x, y+5)
	case 'l':
		fontLine("***", x+1, y)
		fontLine("-**", x+1, y+1)
		fontLine("**", x+2, y+2)
		fontLine("**", x+2, y+3)
		fontLine("-**-", x+1, y+4)
		fontLine("****", x+1, y+5)
	case 'L':
		fontLine("****", x, y)
		fontLine("-**-", x, y+1)
		fontLine("**", x+1, y+2)
		fontLine("**", x+1, y+3)
		fontLine("-**--*", x, y+4)
		fontLine("******", x, y+5)
	case 'm':
		fontLine("**-**-", x, y+1)
		fontLine("-*****", x, y+2)
		fontLine("*-*-*", x+1, y+3)
		fontLine("*-*-*", x+1, y+4)
		fontLine("*-*-*", x+1, y+5)
	case 'M':
		fontLine("**-**", x+1, y)
		fontLine("**-**", x+1, y+1)
		fontLine("*-*-*", x+1, y+2)
		fontLine("*- -*", x+1, y+3)
		fontLine("*- -*", x+1, y+4)
		fontLine("*   *", x+1, y+5)
	case 'n':
		fontLine("**-**-", x, y+1)
		fontLine("-*****", x, y+2)
		fontLine("**-**", x+1, y+3)
		fontLine("** **", x+1, y+4)
		fontLine("** **", x+1, y+5)
	case 'N':
		fontLine("**  **", x, y)
		fontLine("**- **", x, y+1)
		fontLine("***-**", x, y+2)
		fontLine("**-***", x, y+3)
		fontLine("** -**", x, y+4)
		fontLine("**  **", x, y+5)
	case 'o':
		fontLine("-****-", x, y+1)
		fontLine("**--**", x, y+2)
		fontLine("**  **", x, y+3)
		fontLine("**--**", x, y+4)
		fontLine("-****-", x, y+5)
	case 'O':
		fontLine("-****-", x, y)
		fontLine("**--**", x, y+1)
		fontLine("**  **", x, y+2)
		fontLine("**  **", x, y+3)
		fontLine("**--**", x, y+4)
		fontLine("-****-", x, y+5)
	case 'p':
		fontLine("**-**-", x, y+1)
		fontLine("-**--*", x, y+2)
		fontLine("****", x+1, y+3)
		fontLine("-**-", x, y+4)
		fontLine("****", x, y+5)
	case 'P':
		fontLine("*****-", x, y)
		fontLine("-**-**", x, y+1)
		fontLine("****-", x+1, y+2)
		fontLine("**-", x+1, y+3)
		fontLine("-**-", x, y+4)
		fontLine("****", x, y+5)
	case 'q':
		fontLine("-**-**", x, y+1)
		fontLine("*--**-", x, y+2)
		fontLine("-****", x, y+3)
		fontLine("-**-", x+2, y+4)
		fontLine("****", x+2, y+5)
	case 'Q':
		fontLine("-****-", x, y)
		fontLine("**--**", x, y+1)
		fontLine("** -**", x, y+2)
		fontLine("**-***", x, y+3)
		fontLine("-****-", x, y+4)
		fontLine("-**", x+3, y+5)
	case 'r':
		fontLine("**-**-", x, y+1)
		fontLine("-*****", x, y+2)
		fontLine("**--*", x+1, y+3)
		fontLine("-**-", x, y+4)
		fontLine("****", x, y+5)
	case 'R':
		fontLine("****-", x, y)
		fontLine("-*--*-", x, y+1)
		fontLine("*--*-", x+1, y+2)
		fontLine("***-", x+1, y+3)
		fontLine("-*-**-", x, y+4)
		fontLine("**--**", x, y+5)
	case 's':
		fontLine("-*****", x, y+1)
		fontLine("**-", x, y+2)
		fontLine("-****-", x, y+3)
		fontLine("-**", x+3, y+4)
		fontLine("*****-", x, y+5)
	case 'S':
		fontLine("-****-", x, y)
		fontLine("**--**", x, y+1)
		fontLine("-***-", x, y+2)
		fontLine("-**-", x+2, y+3)
		fontLine("**--**", x, y+4)
		fontLine("-****-", x, y+5)
	case 't':
		fontLine("-*", x+1, y)
		fontLine("-**-", x, y+1)
		fontLine("****", x, y+2)
		fontLine("-**-", x, y+3)
		fontLine("**-*", x+1, y+4)
		fontLine("-**-", x+1, y+5)
	case 'T':
		fontLine("******", x, y)
		fontLine("*-**-*", x, y+1)
		fontLine("**", x+2, y+2)
		fontLine("**", x+2, y+3)
		fontLine("-**-", x+1, y+4)
		fontLine("****", x+1, y+5)
	case 'u':
		fontLine("** **", x, y+1)
		fontLine("** **", x, y+2)
		fontLine("** **", x, y+3)
		fontLine("**-**-", x, y+4)
		fontLine("-**-**", x, y+5)
	case 'U':
		fontLine("**  **", x, y)
		fontLine("**  **", x, y+1)
		fontLine("**  **", x, y+2)
		fontLine("**  **", x, y+3)
		fontLine("**--**", x, y+4)
		fontLine("-****-", x, y+5)
	case 'v':
		fontLine("**  **", x, y+1)
		fontLine("**  **", x, y+2)
		fontLine("**--**", x, y+3)
		fontLine("-****-", x, y+4)
		fontLine("-**-", x+1, y+5)
	case 'V':
		fontLine("**  **", x, y)
		fontLine("**  **", x, y+1)
		fontLine("**  **", x, y+2)
		fontLine("**--**", x, y+3)
		fontLine("-****-", x, y+4)
		fontLine("-**-", x+1, y+5)
	case 'w':
		fontLine("*   *", x+1, y+1)
		fontLine("*- -*", x+1, y+2)
		fontLine("*-*-*", x+1, y+3)
		fontLine("*-*-*", x+1, y+4)
		fontLine("-*-*-", x+1, y+5)
	case 'W':
		fontLine("*   *", x+1, y)
		fontLine("*- -*", x+1, y+1)
		fontLine("*- -*", x+1, y+2)
		fontLine("*-*-*", x+1, y+3)
		fontLine("**-**", x+1, y+4)
		fontLine("-*-*-", x+1, y+5)
	case 'x':
		fontLine("**--**", x, y+1)
		fontLine("-****-", x, y+2)
		fontLine("-**-", x+1, y+3)
		fontLine("-****-", x, y+4)
		fontLine("**--**", x, y+5)
	case 'X':
		fontLine("**--**", x, y)
		fontLine("-****-", x, y+1)
		fontLine("-**-", x+1, y+2)
		fontLine("-**-", x+1, y+3)
		fontLine("-****-", x, y+4)
		fontLine("**--**", x, y+5)
	case 'y':
		fontLine("**  **", x, y+1)
		fontLine("**--**", x, y+2)
		fontLine("-****-", x, y+3)
		fontLine("-*", x+4, y+4)
		fontLine("*****-", x, y+5)
	case 'Y':
		fontLine("**  **", x, y)
		fontLine("**--**", x, y+1)
		fontLine("-****-", x, y+2)
		fontLine("-**-", x+1, y+3)
		fontLine("-**-", x+1, y+4)
		fontLine("****", x+1, y+5)
	case 'z':
		fontLine("******", x, y+1)
		fontLine("*--**-", x, y+2)
		fontLine("-**-", x+1, y+3)
		fontLine("-**--*", x, y+4)
		fontLine("******", x, y+5)
	case 'Z':
		fontLine("******", x, y)
		fontLine("*- -**", x, y+1)
		fontLine("-**-", x+2, y+2)
		fontLine("-**-", x+1, y+3)
		fontLine("-**--*", x, y+4)
		fontLine("******", x, y+5)
	case 'æ':
		fontLine("****-", x, y+1)
		fontLine("-*-*", x+1, y+2)
		fontLine("-***-", x, y+3)
		fontLine("*-*-", x, y+4)
		fontLine("-****", x, y+5)
	case 'Æ':
		fontLine("-*****", x, y)
		fontLine("**-**-", x, y+1)
		fontLine("******", x, y+2)
		fontLine("**-**-", x, y+3)
		fontLine("** **-", x, y+4)
		fontLine("** ***", x, y+5)
	case 'ø':
		fontLine("-***-*", x, y+1)
		fontLine("*--**-", x, y+2)
		fontLine("*-**-*", x, y+3)
		fontLine("-**--*", x, y+4)
		fontLine("*-***-", x, y+5)
	case 'Ø':
		fontLine("-***-*", x, y)
		fontLine("*--**-", x, y+1)
		fontLine("*-**-*", x, y+2)
		fontLine("*-**-*", x, y+3)
		fontLine("-**--*", x, y+4)
		fontLine("*-***-", x, y+5)
	case 'å':
		fontLine("-**-", x+1, y)
		fontLine("***-", x+1, y+1)
		fontLine("--**", x+1, y+2)
		fontLine("-****", x, y+3)
		fontLine("**-**-", x, y+4)
		fontLine("-**-**", x, y+5)
	case 'Å':
		fontLine("**", x+2, y)
		fontLine("--", x+2, y+1)
		fontLine("-****-", x, y+2)
		fontLine("**--**", x, y+3)
		fontLine("******", x, y+4)
		fontLine("**--**", x, y+5)
	case '.':
		fontLine("**-", x+1, y+4)
		fontLine("-**", x, y+5)
	case ':':
		fontLine("**-", x+1, y)
		fontLine("-**", x, y+1)
		fontLine("**-", x+1, y+4)
		fontLine("-**", x, y+5)
	case ';':
		fontLine("**-", x+1, y)
		fontLine("-**", x, y+1)
		fontLine("-*", x+1, y+4)
		fontLine("*-", x+1, y+5)
	case ',':
		fontLine("-*", x+1, y+4)
		fontLine("*-", x+1, y+5)
	case '\'':
		fontLine("-*", x+1, y)
		fontLine("*-", x+1, y+1)
	case '"':
		fontLine("-* -*", x, y)
		fontLine("*- *-", x, y+1)
	case '→':
		fontLine("-*-", x+2, y)
		fontLine("-*-", x+3, y+1)
		fontLine("******", x, y+2)
		fontLine("-*-", x+3, y+3)
		fontLine("-*-", x+2, y+4)
	case '*':
		fontLine("* *", x+2, y+1)
		fontLine(" *", x+2, y+2)
		fontLine("* *", x+2, y+3)
	case '+':
		fontLine("  *", x, y)
		fontLine("  *", x, y+1)
		fontLine("*****", x, y+2)
		fontLine("  *", x, y+3)
		fontLine("  *", x, y+4)
	case '!':
		fontLine("**", x+1, y)
		fontLine("-**-", x, y+1)
		fontLine("-**-", x, y+2)
		fontLine("**", x+1, y+3)
		fontLine("--", x+1, y+4)
		fontLine("**", x+1, y+5)
	case '?':
		fontLine("-****-", x, y)
		fontLine("**--**", x, y+1)
		fontLine("-**-", x+2, y+2)
		fontLine("**-", x+2, y+3)
		fontLine("--", x+2, y+4)
		fontLine("**", x+2, y+5)
	case '-':
		fontLine("-****-", x, y+3)
	case '=':
		fontLine("-****-", x, y+2)
		fontLine("-****-", x, y+4)
	case '_':
		fontLine("-****-", x, y+5)
	case '/':
		fontLine("*", x+4, y+1)
		fontLine("*-", x+3, y+2)
		fontLine("*-", x+2, y+3)
		fontLine("*-", x+1, y+4)
		fontLine("*-", x, y+5)
	case '1':
		fontLine("-**", x+1, y)
		fontLine("***", x+1, y+1)
		fontLine("-**", x+1, y+2)
		fontLine("**", x+2, y+3)
		fontLine("**-", x+2, y+4)
		fontLine("****", x+1, y+5)
	case '2':
		fontLine("-****-", x, y)
		fontLine("*- -**", x, y+1)
		fontLine("-**-", x+2, y+2)
		fontLine("-**-", x+1, y+3)
		fontLine("-**--*", x, y+4)
		fontLine("******", x, y+5)
	case '3':
		fontLine("-****-", x, y)
		fontLine("**--**", x, y+1)
		fontLine("**-", x+3, y+2)
		fontLine("-**", x+3, y+3)
		fontLine("**--**", x, y+4)
		fontLine("-****-", x, y+5)
	case '4':
		fontLine("-***", x+1, y)
		fontLine("-*-**", x, y+1)
		fontLine("*--**-", x, y+2)
		fontLine("******", x, y+3)
		fontLine("-**-", x+2, y+4)
		fontLine("****", x+2, y+5)
	case '5':
		fontLine("******", x, y)
		fontLine("**-", x, y+1)
		fontLine("*****-", x, y+2)
		fontLine("-**", x+3, y+3)
		fontLine("**--**", x, y+4)
		fontLine("-****-", x, y+5)
	case '6':
		fontLine("-***", x+1, y)
		fontLine("-**-", x, y+1)
		fontLine("**-", x, y+2)
		fontLine("*****-", x, y+3)
		fontLine("**--**", x, y+4)
		fontLine("-****-", x, y+5)
	case '7':
		fontLine("******", x, y)
		fontLine("*- -**", x, y+1)
		fontLine("-**", x+3, y+2)
		fontLine("-**-", x+2, y+3)
		fontLine("**-", x+2, y+4)
		fontLine("**", x+2, y+5)
	case '8':
		fontLine("-****-", x, y)
		fontLine("**--**", x, y+1)
		fontLine("-****-", x, y+2)
		fontLine("**--**", x, y+3)
		fontLine("**--**", x, y+4)
		fontLine("-****-", x, y+5)
	case '9':
		fontLine("-****-", x, y)
		fontLine("**--**", x, y+1)
		fontLine("-*****", x, y+2)
		fontLine("-**", x+3, y+3)
		fontLine("-**-", x+2, y+4)
		fontLine("-***-", x, y+5)
	case '0':
		fontLine("-****-", x, y)
		fontLine("**--**", x, y+1)
		fontLine("**--**", x, y+2)
		fontLine("**--**", x, y+3)
		fontLine("**--**", x, y+4)
		fontLine("-****-", x, y+5)
	case '(':
		fontLine("-***", x+2, y)
		fontLine("**-", x+2, y+1)
		fontLine("**-", x+1, y+2)
		fontLine("**-", x+1, y+3)
		fontLine("**-", x+2, y+4)
		fontLine("-***", x+2, y+5)
	case ')':
		fontLine("***-", x, y)
		fontLine("-**", x+1, y+1)
		fontLine("-**", x+2, y+2)
		fontLine("-**", x+2, y+3)
		fontLine("-**", x+1, y+4)
		fontLine("***-", x, y+5)
	case '{':
		fontLine("-**", x+2, y)
		fontLine("*", x+2, y+1)
		fontLine("*", x+2, y+2)
		fontLine("*", x+1, y+3)
		fontLine("*", x+2, y+4)
		fontLine("*", x+2, y+5)
		fontLine("-**", x+2, y+6)
	case '}':
		fontLine("**-", x+1, y)
		fontLine("*", x+3, y+1)
		fontLine("*", x+3, y+2)
		fontLine("*", x+4, y+3)
		fontLine("*", x+3, y+4)
		fontLine("*", x+3, y+5)
		fontLine("**-", x+1, y+6)
	case '[':
		fontLine("****", x+1, y)
		fontLine("**", x+1, y+1)
		fontLine("**", x+1, y+2)
		fontLine("**", x+1, y+3)
		fontLine("**", x+1, y+4)
		fontLine("****", x+1, y+5)
	case ']':
		fontLine("****", x+1, y)
		fontLine("  **", x+1, y+1)
		fontLine("  **", x+1, y+2)
		fontLine("  **", x+1, y+3)
		fontLine("  **", x+1, y+4)
		fontLine("****", x+1, y+5)
	case '<':
		fontLine("  *-", x+2, y+1)
		fontLine(" *-", x+2, y+2)
		fontLine("*-", x+2, y+3)
		fontLine(" *-", x+2, y+4)
		fontLine("  *-", x+2, y+5)
	case '>':
		fontLine("-*", x+2, y+1)
		fontLine("-*", x+3, y+2)
		fontLine(" -*", x+3, y+3)
		fontLine("-*", x+3, y+4)
		fontLine("-*", x+2, y+5)
	case '&':
		fontLine(" **", x, y)
		fontLine("** *", x, y+1)
		fontLine(" **  *", x, y+2)
		fontLine(" **-*", x, y+3)
		fontLine("*  **", x, y+4)
		fontLine(" ** **", x, y+5)
	case '|':
		fontLine("*", x+3, y)
		fontLine("*", x+3, y+1)
		fontLine("*", x+3, y+2)
		fontLine("*", x+3, y+3)
		fontLine("*", x+3, y+4)
		fontLine("*", x+3, y+5)
	case '\\':
		fontLine("**", x+1, y+1)
		fontLine("-**", x+1, y+2)
		fontLine("-**", x+2, y+3)
		fontLine("-**", x+3, y+4)
		fontLine("-**", x+4, y+5)

	case 0:
		return errors.New("the rune was 0. Did you pass a coordinate instead of a rune?")
	default:
		return fmt.Errorf("rune %s is not available", string(l))
	}
	return nil
}

// DrawString draws the given string using this font
func DrawString(d Drawable, x, y int, s string, c color.NRGBA) int {
	for _, r := range s {
		Draw(d, r, x, y, c.R, c.G, c.B)
		x += 8
	}
	return x
}
