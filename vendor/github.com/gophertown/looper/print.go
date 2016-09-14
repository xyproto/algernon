package main

import (
	"fmt"

	"github.com/koyachi/go-term-ansicolor/ansicolor"
)

func Header() {
	fmt.Println(ansicolor.Cyan("Looper 0.3.3 is watching your files"))
	fmt.Println("Type " + ansicolor.Magenta("help") + " for help.\n")
}

func DebugEnabled() {
	DebugMessage("Debug mode enabled.\n")
}

func DebugMessage(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Println(ansicolor.IntenseBlack(msg))
}

func DebugError(msg error) {
	fmt.Println(ansicolor.IntenseBlack(msg.Error()))
}

func DisplayHelp() {
	fmt.Println(ansicolor.Magenta("\nInteractions:\n"))
	fmt.Println("  * a, all  Run all tests")
	fmt.Println("  * h, help You found it")
	fmt.Println("  * e, exit Leave Looper")
}

func PrintWatching(folder string) {
	ClearPrompt()
	fmt.Println(ansicolor.Yellow("Watching path"), ansicolor.Yellow(folder))
}

func UnknownCommand(command string) {
	fmt.Println(ansicolor.Red("ERROR:")+" Unknown command", ansicolor.Magenta(command))
}

const CSI = "\x1b["

// remove from the screen anything that's been typed
// from github.com/kierdavis/ansi
func ClearPrompt() {
	fmt.Printf("%s2K", CSI)     // clear line
	fmt.Printf("%s%dG", CSI, 0) // go to column 0
}
