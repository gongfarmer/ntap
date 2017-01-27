package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/gongfarmer/ntap/encoding/atom"
)

// FIXME Support parsing of files containing hex, since hex+binary are
// supported on STDIN they should both be allowed within files.

func printAtom(a atom.Atom) {
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
	rc := 0

	// Read from files
	var files = os.Args[1:]
	for _, path := range files {
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
	if err != nil && err != atom.ErrInvalidInput {
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
