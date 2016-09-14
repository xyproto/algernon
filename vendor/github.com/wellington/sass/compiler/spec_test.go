package compiler

import (
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wellington/sass/parser"
)

type file struct {
	input  string // path to Sass input.scss
	expect []byte // path to expected_output.css
}

func findPaths() []file {
	inputs, err := filepath.Glob("../sass-spec/spec/basic/*/input.scss")
	if err != nil {
		log.Fatal(err)
	}

	var input string
	var files []file
	// files := make([]file, len(inputs))
	for _, input = range inputs {

		// detailed commenting
		if strings.Contains(input, "06_") {
			continue
		}
		// skip insane list math
		if strings.Contains(input, "15_") {
			continue
		}
		// Skip for built-in rules
		if strings.Contains(input, "24_") {
			continue
		}

		// extra commas
		if strings.Contains(input, "36_") {
			continue
		}

		exp, err := ioutil.ReadFile(strings.Replace(input,
			"input.scss", "expected_output.css", 1))
		if err != nil {
			log.Println("failed to read", input)
			continue
		}

		files = append(files, file{
			input:  input,
			expect: exp,
		})
		// Indicates the first test that will not pass tests
		if strings.Contains(input, "35_") && testing.Short() {
			break
		}

	}
	return files
}

func TestCompile_spec(t *testing.T) {
	// It will be a long time before these are all supported, so let's just
	// short these for now.

	files := findPaths()
	var f file
	defer func() {
		fmt.Println("exited on: ", f.input)
	}()
	for _, f = range files {

		fmt.Printf(`
=================================
compiling: %s\n
=================================
`, f.input)
		ctx := NewContext()

		ctx.SetMode(parser.Trace)
		out, err := ctx.runString(f.input, nil)
		sout := strings.Replace(out, "`", "", -1)
		if err != nil {
			log.Println("failed to compile", f.input, err)
		}
		if e := string(f.expect); e != sout {
			// t.Fatalf("got:\n%s", out)
			// t.Fatalf("got:\n%q\nwanted:\n%q", out, e)
			t.Fatalf("got:\n%s\nwanted:\n%s", out, e)
		}
		fmt.Printf(`
=================================
compiled: %s\n
=================================
`, f.input)
	}

}
