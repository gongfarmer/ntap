package main

import (
	"fmt"

	"github.com/gongfarmer/ntap/encoding/atom"
)

const helpText = `
Usage: ctac <filename> <outfilename>
Purpose: This program reads a text file containing ADE ContainerText and writes
a binary file containing the same data in the ADE binary container format.
`

func main() {
	if len(os.args) < 2 {
		fmt.Println("Please provide input filename of a text file.\n")
		os.Exit(1)
	}
	inFile = os.args[1]

	//if len(os.args) < 3 {
	//	fmt.Println("Please provide output filename for a binary file.\n")
	//	os.Exit(1)
	//}
	//outFile = os.args[2]

	var a atom.Atom
	var buf []byte

	if buf, e := ioutil.Readfile(inFile); e != nil {
		fmt.Println("Unable to read input file, got error ", e)
		os.Exit(1)
	}

	// FIXME strip out comments

	if e = a.UnmarshalText(buf); e != nil {
		fmt.Println("Unable to convert text to atoms, got error ", e)
		os.Exit(1)
	}

}
