package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/gongfarmer/ntap/encoding/atom"
)

var FlagOutputHex = flag.Bool("X", false, "print atom as hex string")
var FlagOutputXml = flag.Bool("x", false, "print atom as xml")
var FlagOutputDebug = flag.Bool("d", false, "print atoms in verbose debug format")

func main() {
	flag.Parse()

	// Read atom data from files
	var files = os.Args[1:]
	atoms, err := MakeAtoms(files)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	PrintAtoms(atoms)
	os.Exit(0)
}

func printDebug(a *atom.Atom) {
	atoms := a.AtomList()
	for _, a := range atoms {
		fmt.Printf("name: \"%s\"\n", a.Name)
		fmt.Printf("type: \"%s\"\n", a.Type())
		strData, err := a.Value.String()
		if err != nil {
			fmt.Printf(err.Error())
			os.Exit(1)
		}
		bytesData, err := a.Value.SliceOfByte()
		if err != nil {
			fmt.Printf(err.Error())
			os.Exit(1)
		}
		fmt.Printf("data: \"%s\"\n", strData) // value as string
		fmt.Printf("      % x\n", bytesData)  // value as bytes
		fmt.Println("--")
	}
}

func printString(a *atom.Atom) {
	buf, err := a.MarshalText()
	if err != nil {
		fmt.Printf("failed to print AtomContainer: %s\n", err)
		return
	}
	fmt.Print(string(buf))
}

func stdinIsEmpty() bool {
	stat, _ := os.Stdin.Stat()
	return (stat.Mode() & os.ModeCharDevice) != 0
}

// Read file contents and return Atom instances.
// If no files are provided, read STDIN.
func MakeAtoms(files []string) (atoms []*atom.Atom, err error) {
	if len(files) == 0 && stdinIsEmpty() {
		return
	}

	var buffer []byte
	var someAtoms []*atom.Atom

	if len(files) == 0 { // Read STDIN
		buffer, err = ioutil.ReadAll(os.Stdin)
		if err != nil && err != io.EOF {
			return
		}

		// Convert input to atoms
		someAtoms, err = atom.ReadAtomsFromBinary(bytes.NewReader(buffer))
		if err != nil {
			return
		}
		atoms = append(atoms, someAtoms...)
	}

	// Read each file, expecting ADE binary data
	for _, path := range files {
		buffer, err = ioutil.ReadFile(path)
		if err != nil && err != io.EOF {
			return atoms, fmt.Errorf("failed to read file %s: %s\n", path, err)
		}

		// convert to atoms
		someAtoms, err = atom.ReadAtomsFromBinary(bytes.NewReader(buffer))
		if err != nil {
			return atoms, fmt.Errorf("failed to read atoms from %s: %s\n", path, err)
		}

		atoms = append(atoms, someAtoms...)
	}
	return
}

func PrintAtoms(atoms []*atom.Atom) {
	for _, a := range atoms {
		if true == *FlagOutputDebug {
			printDebug(a)
		} else {
			printString(a)
		}
	}
}
