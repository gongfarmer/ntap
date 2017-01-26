package main

import (
	"fmt"
	"os"

	"github.com/gongfarmer/ntap/encoding/atom"
)

// FIXME add feature to read and parse atomcontainer from binary string?
// would allow CNCT containers to be easily inspected, not to mention weird
// stuff like CMS database entries that encapsulate atoms

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

func main() {
	rc := 0

	// Read from files
	var files = os.Args[1:]
	for _, path := range files {
		a, err := readContainerFromFile(path)
		if err != nil {
			fmt.Printf("Failed to read from %s: %s\n", path, err)
			rc = 1
			continue
		}
		printAtom(a)
	}

	if stdinIsEmpty() {
		os.Exit(rc)
	}

	// Read from STDIN
	// FIXME should handle multiple atoms, not just one

	atoms, err := atom.ReadAtomsFromBinaryStream(os.Stdin)
	if err != nil {
		fmt.Printf("Unable to read binary from STDIN: %s", err)
		rc = 1
	}
	for _, a := range atoms {
		printAtom(*a)
	}

	os.Exit(rc)
}
