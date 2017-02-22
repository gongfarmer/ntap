package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/gongfarmer/ntap/encoding/atom"
)

var FlagDebug = flag.Bool("d", false, "print atoms in detailed debug format")

func printAtom(a atom.Atom) {
	if true == *FlagDebug {
		printDebug(a)
	} else {
		printString(a)
	}
}

func printDebug(a atom.Atom) {
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

func printString(a atom.Atom) {
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

func readContainerFromFile(path string) (a atom.Atom, err error) {
	var e error
	if a, e = atom.FromFile(path); e != nil {
		err = fmt.Errorf("Failed to read AtomContainer: %s", e)
	}
	return
}

func readAtomsFromHexStream(r io.Reader) (atoms []*atom.Atom, err error) {
	var buf []byte
	err = binary.Read(r, binary.BigEndian, &buf)

	fmt.Println(buf)
	return
}

func main() {
	flag.Parse()

	rc := 0

	// Read from files
	var files = os.Args[1:]
	for _, path := range files {
		if strings.HasPrefix(path, "-") {
			continue
		}
		a, err := readContainerFromFile(path)
		if err != nil {
			fmt.Printf("failed to read from %s: %s\n", path, err)
			rc = 1
			continue
		}
		printAtom(a)
	}

	if stdinIsEmpty() {
		os.Exit(rc)
	}

	// Read entire STDIN
	var buffer []byte
	buffer, err := ioutil.ReadAll(os.Stdin)
	if err != nil && err != io.EOF {
		fmt.Println(err)
		os.Exit(1)
	}

	// Parse input as binary stream
	atoms, err := atom.ReadAtomsFromBinary(bytes.NewReader(buffer))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Parse input as hex stream
	moreAtoms, err := atom.ReadAtomsFromHex(bytes.NewReader(buffer))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	atoms = append(atoms, moreAtoms...)

	// Print atoms collected from STDIN
	for _, a := range atoms {
		printAtom(*a)
	}

	os.Exit(rc)
}
