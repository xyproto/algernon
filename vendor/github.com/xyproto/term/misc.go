package term

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Represents a function that takes no arguments and returns two integers
type ReturnsTwoInts func() (int, int)

// Get the first value from the function that returns two ints
func First(f ReturnsTwoInts) int {
	v, _ := f()
	return v
}

// Get the second value from the function that returns two ints
func Second(f ReturnsTwoInts) int {
	_, v := f()
	return v
}

// Read a line from stdin
func ReadLn() string {
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return ""
	}
	return line[:len(line)-1]
}

// Ask a question, wait for textual input followed by a newline
func Ask(prompt string) string {
	fmt.Print(prompt)
	return ReadLn()
}

// Ask a yes/no question, don't wait for newline
func AskYesNo(question string, noIsDefault bool) bool {
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

// Map a function on each element of a slice of strings
func MapS(f func(string) string, sl []string) (result []string) {
	result = make([]string, len(sl))
	for i := range sl {
		result[i] = f(sl[i])
	}
	return result
}

// Filter out all strings where the function does not return true
func FilterS(f func(string) bool, sl []string) (result []string) {
	result = make([]string, 0)
	for i := range sl {
		if f(sl[i]) {
			result = append(result, sl[i])
		}
	}
	return result
}

// Split a string on any newline: \n, \r or \r\n
func Splitlines(s string) []string {
	s = strings.Replace(s, "\r", "\n", -1)
	s = strings.Replace(s, "\r\n", "\n", -1)
	return MapS(trimnewlines, FilterS(nonempty, strings.Split(s, "\n")))
}

// Helper function for checking if a string is empty or not
func nonempty(s string) bool {
	return trimnewlines(s) != ""
}

// Helper function for trimming away newlines:
func trimnewlines(s string) string {
	return strings.Trim(s, "\r\n")
}

// Repeat a string n number of times
func Repeat(text string, n int) string {
	var sb strings.Builder
	for i := 0; i < n; i++ {
		sb.WriteString(text)
	}
	return sb.String()
}

// Repeat a rune n number of times
func RepeatRune(r rune, n int) string {
	var sb strings.Builder
	for i := 0; i < n; i++ {
		sb.WriteRune(r)
	}
	return sb.String()
}
