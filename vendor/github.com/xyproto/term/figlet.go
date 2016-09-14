package term

import (
	"github.com/getwe/figlet4go"
)

// Use figlet for drawing ascii text in a fancyful manner
func Figlet(msg string) (string, error) {
	// Figlet4go also support using figlet fonts (/usr/share/figlet/*.flf).
	// These also supports lowercase characters.
	// Look at slant.flf and big.flf, for instance.
	// TODO: Use big.flf, if available
	ar := figlet4go.NewAsciiRender()
	return ar.Render(msg)
}
