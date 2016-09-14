package main

import (
	"fmt"
	"github.com/koyachi/go-term-ansicolor/ansicolor"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"
)

var (
	reAdd    = compiledRegexp(`^\+`)
	reDel    = compiledRegexp(`^\-`)
	reHeader = compiledRegexp(`^(@@|diff)`)
)

func compiledRegexp(reStr string) *regexp.Regexp {
	re, err := regexp.Compile(reStr)
	if err != nil {
		log.Fatal(err)
	}
	return re
}

func readFile(inputFile string) {
	content, err := ioutil.ReadFile(inputFile)
	if err != nil {
		log.Fatal(err)
		return
	}
	parseLines(string(content))
}

func parseLines(content string) {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		result := ""
		switch {
		case reAdd.Match([]byte(line)):
			result = ansicolor.Green(line)
		case reDel.Match([]byte(line)):
			result = ansicolor.Red(line)
		case reHeader.Match([]byte(line)):
			result = ansicolor.Blue(line)
		default:
			result = line
		}
		fmt.Println(result)
	}
}

func main() {
	if len(os.Args) == 2 {
		inputFile := os.Args[1]
		readFile(inputFile)
	} else {
		content, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			log.Fatal(err)
			return
		}
		parseLines(string(content))
	}
}
