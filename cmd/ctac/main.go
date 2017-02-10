package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/gongfarmer/ntap/encoding/atom"
)

const helpText = `
Usage: ctac <filename> <outfilename>
Purpose: This program reads a text file containing ADE ContainerText and writes
a binary file containing the same data in the ADE binary container format.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Please provide input filename.\n")
		fmt.Println("The input should be a text file in ADE ContainerText format.\n")
		os.Exit(1)
	}
	inFile := os.Args[1]

	if len(os.Args) < 3 {
		fmt.Println("Please provide output filename.\n")
		os.Exit(1)
	}
	outFile := os.Args[2]

	var a atom.Atom
	var buf []byte
	var err error

	if buf, err = ioutil.ReadFile(inFile); err != nil {
		fmt.Println("ctac: Unable to read input file: ", err)
		os.Exit(1)
	}

	// FIXME strip out comments

	// Read input file
	if err = a.UnmarshalText(buf); err != nil {
		fmt.Println("ctac: Invalid input container: ", err)
		os.Exit(1)
	}

	// Convert to binary
	buf, err = a.MarshalBinary()
	if err != nil {
		fmt.Println("ctac: Unable to convert container to binary: ", err)
		os.Exit(1)
	}

	// Write binary to file
	err = ioutil.WriteFile(outFile, buf, 0777)
	if err != nil {
		fmt.Println("ctac: Unable to write to file: ", err)
		os.Exit(1)
	}
}
