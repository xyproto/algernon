package gat

import (
	"fmt"
	"github.com/koyachi/go-term-ansicolor/ansicolor"
	"strings"
	"time"
)

func PrintCommand(args []string) {
	ClearPrompt()
	fmt.Println(ansicolor.Yellow(strings.Join(args, " ")))
}

func PrintCommandOutput(out []byte) {
	fmt.Print(string(out))
}

func RedGreen(pass bool) {
	if pass {
		fmt.Print(ansicolor.Green("PASS"))
	} else {
		fmt.Print(ansicolor.Red("FAIL"))
	}
}

func ShowDuration(dur time.Duration) {
	fmt.Printf(" (%.2f seconds)\n", dur.Seconds())
}

const CSI = "\x1b["

// remove from the screen anything that's been typed
// from github.com/kierdavis/ansi
func ClearPrompt() {
	fmt.Printf("%s2K", CSI)     // clear line
	fmt.Printf("%s%dG", CSI, 0) // go to column 0
}
