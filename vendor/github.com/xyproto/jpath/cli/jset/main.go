package main

import (
	"flag"
	"fmt"
	"github.com/xyproto/jpath"
	"log"
	"os"
)

func main() {
	flag.Parse()

	if len(flag.Args()) != 3 {
		fmt.Println("Syntax: jset [filename] [JSON path] [value]")
		fmt.Println("Example: jset books.json x[1].author Catniss")
		os.Exit(1)
	}

	filename := flag.Args()[0]
	JSONpath := flag.Args()[1]
	value := flag.Args()[2]

	err := jpath.SetString(filename, JSONpath, value)
	if err != nil {
		log.Fatal(err)
	}
}
