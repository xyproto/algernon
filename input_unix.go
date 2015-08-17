// +build !darwin dragonfly freebsd linux nacl netbsd openbsd solaris

package algernon

import (
	"github.com/bobappleyard/readline"
)

func getInput(prompt string) (string, error) {
	return readline.String(prompt)
}

func saveHistory(historyFilename string) error {
	return readline.SaveHistory(historyFilename)
}

func loadHistory(historyFilename string) error {
	return readline.LoadHistory(historyFilename)
}

func addHistory(line string) {
	readline.AddHistory(line)
}
