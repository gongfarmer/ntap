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
		fmt.Println("Please provide input filename of a text file.\n")
		os.Exit(1)
	}
	inFile := os.Args[1]

	//if len(os.argv) < 3 {
	//	fmt.Println("Please provide output filename for a binary file.\n")
	//	os.Exit(1)
	//}
	//outFile = os.args[2]

	var a atom.Atom
	var buf []byte
	var err error

	if buf, err = ioutil.ReadFile(inFile); err != nil {
		fmt.Println("ctac: Unable to read input file: ", err)
		os.Exit(1)
	}

	// FIXME strip out comments

	if err = a.UnmarshalText(buf); err != nil {
		fmt.Println("ctac: Unable to convert text to atoms: ", err)
		os.Exit(1)
	}
	fmt.Println(a)

}
