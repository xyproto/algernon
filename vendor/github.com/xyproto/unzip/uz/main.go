package main

import (
	"flag"
	"github.com/xyproto/term"
	"github.com/xyproto/unzip"
)

func main() {
	o := term.NewTextOutput(true, true)
	flag.Parse()
	if len(flag.Args()) == 0 {
		o.ErrExit("Provide a ZIP filename and (optionally) a directory to extract to.")
	}
	zipfile := flag.Args()[0]
	directory := "."
	if len(flag.Args()) > 1 {
		directory = flag.Args()[1]
		o.Println(o.LightGreen("Extracting " + zipfile + " to " + directory + "..."))
	} else {
		o.Println(o.LightGreen("Extracting " + zipfile + "..."))
	}
	if err := unzip.FilterExtract(zipfile, directory, func(filename string) bool {
		o.Println(o.LightBlue("Extracting " + filename))
		return true
	}); err != nil {
		o.ErrExit(err.Error())
	}
	o.Println(o.LightGreen("Done."))
}
