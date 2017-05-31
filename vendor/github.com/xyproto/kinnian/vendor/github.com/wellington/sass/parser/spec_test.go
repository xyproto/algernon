package parser

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wellington/sass/token"
)

func TestSpec_files(t *testing.T) {

	inputs, err := filepath.Glob("../sass-spec/spec/basic/*/input.scss")
	if err != nil {
		t.Fatal(err)
	}

	mode := DeclarationErrors
	mode = Trace | ParseComments
	var name string
	for _, name = range inputs {
		if strings.Contains(name, "25_") && testing.Short() {
			// This is the last test we currently parse properly
			return
		}
		if !strings.Contains(name, "29_") {
			continue
		}
		if strings.Contains(name, "06_") {
			continue
		}
		if strings.Contains(name, "14_") {
			continue
		}
		// These are fucked things in Sass like lists
		if strings.Contains(name, "15_") {
			continue
		}
		// namespaces are wtf
		if strings.Contains(name, "24_") {
			continue
		}
		fmt.Println("Parsing", name)
		_, err := ParseFile(token.NewFileSet(), name, nil, mode)
		if err != nil {
			t.Fatalf("ParseFile(%s): %v", name, err)
		}
		fmt.Println("Parsed", name)
	}
}
