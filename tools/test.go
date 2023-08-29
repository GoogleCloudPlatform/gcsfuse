package main

import (
	"fmt"
	"log"
	"os"
	"strings"
)

func main() {
	// Check if the key_id.txt file exists
	if _, err := os.Stat("key.txt"); err != nil {
		log.Fatalf("file does not exist")
	}

	// Read the key_id.txt file and get the key ID
	content, err := os.ReadFile("key.txt")
	if err != nil {
		log.Fatalf("Error in reading key file data:%v", err)
	}

	// Split the file contents into words
	words := strings.Fields(string(content))

	// Find the index of the word after two spaces
	wordIndex := 2
	if len(words) < wordIndex {
		fmt.Println("Could not find word after two spaces")
		return
	}

	// Remove the braces from the key ID
	keyID := strings.Trim(words[wordIndex], "[]")

	// Get the first 40 characters of the key ID
	keyID = keyID[:40]
}
