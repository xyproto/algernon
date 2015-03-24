package main

import (
	"flag"
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type T struct {
	A string
	B struct {
		C int
		D []int ",flow"
	}
}

func main() {
	flag.Parse()
	filename := "test.yml"
	if len(flag.Args()) >= 1 {
		filename = flag.Args()[0]
	}
	fmt.Println(filename)
	file, err := os.Open(filename)
	defer file.Close()
	if err != nil {
		fmt.Printf("Can't open %s: %s", filename, err)
	}
	data, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Printf("Could not read bytes: %s", err)
	}
	t := T{}
	err = yaml.Unmarshal([]byte(data), &t)
	if err != nil {
		fmt.Printf("Could not unmarshal YAML: %s", err)
	}
	fmt.Println(t)
}
