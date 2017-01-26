package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gongfarmer/ntap/encoding/atom"
)

// FIXME add feature to read and parse atomcontainer from binary string?
// would allow CNCT containers to be easily inspected, not to mention weird
// stuff like CMS database entries that encapsulate atoms
func main() {
	var files = os.Args[1:]
	for _, path := range files {

		// Read it
		atom, err := atom.FromFile(path)
		if err != nil {
			log.Fatalf("Failed to read AtomContainer: %s", err)
		}

		// Print it
		buf, err := atom.MarshalText()
		if err != nil {
			log.Fatalf("Failed to print AtomContainer: %s", err)
		}
		fmt.Print(string(buf))
	}
}
