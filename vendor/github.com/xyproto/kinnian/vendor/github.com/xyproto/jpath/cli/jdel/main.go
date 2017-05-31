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

	if len(flag.Args()) != 2 {
		fmt.Println("Syntax: jdel [filename] [JSON path]")
		fmt.Println("The last part of the JSON path is the key to be removed from a map.")
		fmt.Println()
		fmt.Println("Example: jdel abc.json b")
		os.Exit(1)
	}

	filename := flag.Args()[0]
	JSONpath := flag.Args()[1]

	err := jpath.DelKey(filename, JSONpath)
	if err != nil {
		log.Fatal(err)
	}
}
