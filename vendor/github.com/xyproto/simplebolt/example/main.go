package main

import (
	"log"

	"github.com/xyproto/simplebolt"
)

func main() {
	// New bolt database
	db, err := simplebolt.New("bolt.db")
	if err != nil {
		log.Fatalf("Could not create database! %s", err)
	}
	defer db.Close()

	// Create a list named "greetings"
	list, err := simplebolt.NewList(db, "greetings")
	if err != nil {
		log.Fatalf("Could not create list! %s", err)
	}

	// Add "hello" to the list
	if err := list.Add("hello"); err != nil {
		log.Fatalf("Could not add an item to list! %s", err)
	}

	// Get the last item of the list
	if item, err := list.GetLast(); err != nil {
		log.Fatalf("Could not fetch the last item from the list! %s", err)
	} else {
		log.Println("The value of the stored item is:", item)
	}

	// Remove the list
	if err := list.Remove(); err != nil {
		log.Fatalf("Could not remove the list! %s", err)
	}
}
