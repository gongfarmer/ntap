package atom

//
// Test that UnmarshalText successfully reads all text test files.
//
// Test that MarshalText successfully writes atoms to text, and that the
// serialized atoms match the original files.
//

import (
	"crypto/sha1"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"
)

func TestUnmarshalText(t *testing.T) {
	var buf []byte
	var err error

	for _, f := range txtTestFiles() {

		// read file contents
		if buf, err = ioutil.ReadFile(f); err != nil {
			log.Fatalf("Unable to read test file: %s", err.Error())
		}

		// test unmarshal
		a := new(Atom)
		if err := a.UnmarshalText(buf); err != nil {
			t.Errorf("UnmarshalText(%s): expect no error, got %s", f, err.Error())
		}

		// save for testing MarshalText
		TestAtoms = append(TestAtoms, a)
	}
}

// NOTE: marshaled text output is not guaranteed to always match its input, as
// odd but valid inputs may be normalized.
// The test files use the file extension *.txt for files that can round-trip
// from text->binary->text without being altered.
// The extension *.in represents a file that is valid but will be altered
// during marshaling.
func TestMarshalText(t *testing.T) {
	var a Atom
	var got, want []byte
	var err error

	// Assumes testfiles and TestAtoms have matching order
	for _, f := range txtTestFiles() {
		a, TestAtoms = *TestAtoms[0], TestAtoms[1:] // shift first elt from slice

		// Test that MarshalText succeeds
		if got, err = a.MarshalText(); err != nil {
			t.Errorf("MarshalText(%s): expect no error, got %s", f, err.Error())
		}

		// Read original file
		if want, err = ioutil.ReadFile(f); err != nil {
			log.Fatalf("Unable to read test file: %s", err.Error())
		}

		if len(got) != len(want) {
			t.Errorf("MarshalText: Text size differs from original.  Got %d, want %d:  %s", len(got), len(want), filepath.Base(f))
			if len(got) < 500 {
				fmt.Println("got: ", string(got))
				fmt.Println("wnt: ", string(want))
				os.Exit(1)
			}
		}

		fnm := filepath.Join("/tmp/test", filepath.Base(f))
		ioutil.WriteFile(fnm, got, 644)

		// Verify that original matches marshaled text
		gotSum := sha1.Sum(got)
		wantSum := sha1.Sum(want)
		if gotSum != wantSum {
			t.Errorf("MarshalText: Text output differs from original: %s", filepath.Base(f))
		}
	}
}
