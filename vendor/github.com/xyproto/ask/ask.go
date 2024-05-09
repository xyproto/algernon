package ask

import (
	"bufio"
	"fmt"
	"os"
)

// ReadLn will read a line from stdin, until \n.
func ReadLn() string {
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return ""
	}
	return line[:len(line)-1]
}

// Ask a question, wait for textual input followed by a newline.
func Ask(prompt string) string {
	fmt.Print(prompt)
	return ReadLn()
}

// YesNo will ask a yes/no question. Will not wait for a newline.
func YesNo(question string, noIsDefault bool) bool {
	var s string
	alternatives := "Yn"
	if noIsDefault {
		alternatives = "yN"
	}
	fmt.Printf(question + " [" + alternatives + "] ")
	fmt.Scanf("%s", &s)
	if noIsDefault {
		// Anything that isn't yes is "no" (false)
		return s == "Y" || s == "y"
	}
	// Anything that isn't no is "yes" (true)
	return !(s == "N" || s == "n")
}

// YN is a quick version of the YesNo function, where no is the default
func YN(question string) bool {
	return YesNo(question, true)
}
