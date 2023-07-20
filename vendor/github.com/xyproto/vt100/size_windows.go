package vt100

import (
	"github.com/nathan-fiscaletti/consolesize-go"
)

func TermSize() (uint, uint, error) {
	rows, cols := consolesize.GetConsoleSize()
	return uint(rows), uint(cols), nil
}
