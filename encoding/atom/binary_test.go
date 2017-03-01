package atom

//
// Test that UnmarshalBinary successfully reads all binary test files.
//
// Test that MarshalBinary successfully writes atoms to binary, and that the
// serialized atoms match the original files.
//

import (
	"crypto/sha1"
	"io/ioutil"
	"log"
	"path/filepath"
	"testing"
)

func TestUnmarshalBinary(t *testing.T) {
	var buf []byte
	var err error

	for _, f := range binTestFiles() {

		// read file contents
		if buf, err = ioutil.ReadFile(f); err != nil {
			log.Fatalf("Unable to read test file: %s", err.Error())
		}

		// test unmarshal
		a := new(Atom)
		if err := a.UnmarshalBinary(buf); err != nil {
			t.Errorf("UnmarshalBinary(%s): expect no error, got %s", f, err.Error())
		}

		// save for testing MarshalBinary
		TestAtoms = append(TestAtoms, a)
	}
}

// NOTE: marshaled output is not guaranteed to always match its input, as
// odd but valid inputs may be normalized. However, they do match for these tests.
func TestMarshalBinary(t *testing.T) {
	var a Atom
	var got, want []byte
	var err error

	// Assumes testfiles and TestAtoms have matching order
	for _, f := range binTestFiles() {
		a, TestAtoms = *TestAtoms[0], TestAtoms[1:] // shift first elt off

		// Test that MarshalBinary succeeds
		if got, err = a.MarshalBinary(); err != nil {
			t.Errorf("MarshalBinary(%s): expect no error, got %s", f, err.Error())
		}

		// Read original file
		if want, err = ioutil.ReadFile(f); err != nil {
			log.Fatalf("Unable to read test file: %s", err.Error())
		}

		// Verify that they match in length
		if len(got) != len(want) {
			t.Errorf("MarshalBinary: want %d bytes, got %d bytes for %s", len(want), len(got), filepath.Base(f))
		}

		// Verify that they are binary-identicatl
		gotSum := sha1.Sum(got)
		wantSum := sha1.Sum(want)
		if gotSum != wantSum {
			t.Errorf("MarshalBinary: binary output checksum differs from original: %s", filepath.Base(f))
		}
	}
}
