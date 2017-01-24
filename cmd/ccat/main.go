package main

import (
	"fmt"
	"github.com/gongfarmer/ntap/encoding/atom"
	"log"
	"os"
)

func main() {
	var files = os.Args[1:]
	for _, path := range files {
		// Read it
		atom, err := atom.FromFile(path)
		if err != nil {
			log.Fatalf("Failed to read AtomContainer: %s", err)
		}

		fmt.Printf("container: %+v\n", atom)

		// Print it
		buf, err := atom.MarshalText()
		if err != nil {
			log.Fatalf("Failed to print AtomContainer: %s", err)
		}
		fmt.Println(string(buf))
	}
}
