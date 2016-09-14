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
		fmt.Println("Syntax: jadd [filename] [JSON path] [JSON data]")
		fmt.Println("Example: jadd books.json x '{\"author\": \"Catniss\", \"book\": \"Yeah\"}'")
		os.Exit(1)
	}

	filename := flag.Args()[0]
	JSONpath := flag.Args()[1]
	JSONdata := []byte(flag.Args()[2])

	err := jpath.AddJSON(filename, JSONpath, JSONdata, true)
	if err != nil {
		log.Fatal(err)
	}
}
