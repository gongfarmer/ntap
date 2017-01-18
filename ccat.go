package main

import (
	"fmt"
	"github.com/gongfarmer/ade"
	"log"
	"os"
)

func main() {
	var files = os.Args[1:]
	for _, path := range files {
		atom, err := ade.FromFile(path)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(atom)
	}
}
