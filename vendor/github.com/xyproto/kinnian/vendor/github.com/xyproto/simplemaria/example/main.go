package main

import (
	"log"

	"github.com/xyproto/simplemaria"
)

func main() {
	// Check if the simplemaria service is up
	if err := simplemaria.TestConnection(); err != nil {
		log.Fatalln("Could not connect to local database. Is the service up and running?")
	}

	// Create a Host, connect to the local db server
	host := simplemaria.New()

	// Connecting to a different host/port
	//host := simplemaria.NewHost("server:3306/db")

	// Connect to a different db host/port, with a username and password
	// host := simplemaria.NewHost("username:password@server/db")

	// Close the connection when the function returns
	defer host.Close()

	// Create a list named "greetings"
	list, err := simplemaria.NewList(host, "greetings")
	if err != nil {
		log.Fatalln("Could not create list!")
	}

	// Add "hello" to the list, check if there are errors
	if list.Add("hello") != nil {
		log.Fatalln("Could not add an item to list!")
	}

	// Get the last item of the list
	if item, err := list.GetLast(); err != nil {
		log.Fatalln("Could not fetch the last item from the list!")
	} else {
		log.Println("The value of the stored item is:", item)
	}

	// Remove the list
	if list.Remove() != nil {
		log.Fatalln("Could not remove the list!")
	}
}
