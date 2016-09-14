// Package term offers a simple way to use ncurses and output colored text
package term

// TODO: Consider using https://github.com/fatih/color instead

import (
	"fmt"
	"os"
)

const versionString = "term 0.1"

type TextOutput struct {
	color   bool
	enabled bool
}

func NewTextOutput(color bool, enabled bool) *TextOutput {
	return &TextOutput{color, enabled}
}

// Write an error message in red to stderr if output is enabled
func (o *TextOutput) Err(msg string) {
	if o.enabled {
		fmt.Fprintf(os.Stderr, o.DarkRed(msg)+"\n")
	}
}

// Write an error message to stderr and quit with exit code 1
func (o *TextOutput) ErrExit(msg string) {
	o.Err(msg)
	os.Exit(1)
}

// Write a message to stdout if output is enabled
func (o *TextOutput) Println(msg string) {
	if o.enabled {
		fmt.Println(msg)
	}
}

// Checks if textual output is enabled
func (o *TextOutput) IsEnabled() bool {
	return o.enabled
}

// Changes the color state in the terminal emulator
func (o *TextOutput) colorOn(num1 int, num2 int) string {
	if o.color {
		return fmt.Sprintf("\033[%d;%dm", num1, num2)
	}
	return ""
}

// Changes the color state in the terminal emulator
func (o *TextOutput) colorOff() string {
	if o.color {
		return "\033[0m"
	}
	return ""
}

func (o *TextOutput) DarkRed(s string) string {
	return o.colorOn(0, 31) + s + o.colorOff()
}

func (o *TextOutput) LightGreen(s string) string {
	return o.colorOn(1, 32) + s + o.colorOff()
}

func (o *TextOutput) DarkGreen(s string) string {
	return o.colorOn(0, 32) + s + o.colorOff()
}

func (o *TextOutput) LightYellow(s string) string {
	return o.colorOn(1, 33) + s + o.colorOff()
}

func (o *TextOutput) DarkYellow(s string) string {
	return o.colorOn(0, 33) + s + o.colorOff()
}

func (o *TextOutput) LightBlue(s string) string {
	return o.colorOn(1, 34) + s + o.colorOff()
}

func (o *TextOutput) DarkBlue(s string) string {
	return o.colorOn(0, 34) + s + o.colorOff()
}

func (o *TextOutput) LightCyan(s string) string {
	return o.colorOn(1, 36) + s + o.colorOff()
}

func (o *TextOutput) LightPurple(s string) string {
	return o.colorOn(1, 35) + s + o.colorOff()
}

func (o *TextOutput) DarkPurple(s string) string {
	return o.colorOn(0, 35) + s + o.colorOff()
}

func (o *TextOutput) DarkGray(s string) string {
	return o.colorOn(1, 30) + s + o.colorOff()
}

func (o *TextOutput) White(s string) string {
	return o.colorOn(1, 37) + s + o.colorOff()
}
