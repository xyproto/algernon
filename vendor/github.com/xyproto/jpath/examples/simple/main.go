package main

import (
	"fmt"
	"github.com/xyproto/jpath"
	"log"
)

func main() {
	// Some JSON
	data := []byte(`{"a":2, "b":3}`)

	// Create a new *jpath.Node
	js, err := jpath.New(data)
	if err != nil {
		log.Fatal(err)
	}

	// Retrieve the value of "a", as an int
	val := js.Get("a").Int()

	// Output the result
	fmt.Println("a is", val)
}
