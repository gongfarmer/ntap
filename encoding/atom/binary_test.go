package atom

//
// Verify that UnmarshalBinary successfully reads all binary test files.
//
// Verify that MarshalBinary successfully writes atoms to binary, and that the
// serialized atoms match the original binary files.
//

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"math/rand"
	"testing"

	"github.com/gongfarmer/ntap/encoding/atom/codec"
)

func TestUnmarshalBinary(t *testing.T) {
	for _, test := range Tests {
		a := new(Atom)
		if err := a.UnmarshalBinary(test.binBytes); err != nil {
			t.Errorf("UnmarshalBinary(%s): expect no error, got %s", test.Name(), err.Error())
		}
	}
}

// NOTE: marshaled output is not guaranteed to always match its input, as
// odd but valid inputs may be normalized. However, they do match for these tests.
func TestMarshalBinary(t *testing.T) {
	var got []byte
	var err error

	for _, test := range Tests {

		// Test that MarshalBinary succeeds
		if got, err = test.atom.MarshalBinary(); err != nil {
			t.Errorf("MarshalBinary(%s): expect no error, got %s", test.Name(), err.Error())
		}

		// Verify that resulting bytes match original length
		if len(got) != len(test.binBytes) {
			t.Errorf("MarshalBinary: want %d bytes, got %d bytes for %s", len(test.binBytes), len(got), test.Name())
		}

		// Verify that resulting bytes are binary-identical
		gotSum := sha1.Sum(got)
		wantSum := sha1.Sum(test.binBytes)
		if gotSum != wantSum {
			t.Errorf("MarshalBinary: binary output differs from original: %s", test.Name())
		}
	}
}

// Randomly select 10 test atoms.  Convert their binary forms to hex using the
// function under test.
// Compare the result with the canonical test atom.
func TestReadAtomsFromHex(t *testing.T) {
	tests := make(map[string]*Atom)
	for _, i := range rand.Perm(len(Tests)) {
		test := Tests[i]
		hexString, err := codec.BytesToHexString(test.binBytes)
		if err != nil {
			panic(fmt.Errorf("failed to convert bytes to hex string: %v", err))
		}
		tests[hexString] = test.atom
	}

	for hex, a := range tests {
		atoms, err := ReadAtomsFromHex(bytes.NewBuffer([]byte(hex)))
		if err != nil {
			t.Errorf("TestReadAtomsFromHex(%s): returned error %v", a.Name(), err)
		}
		if len(atoms) == 0 {
			t.Errorf("TestReadAtomsFromHex(%s): failed to get atom results from hex", a.Name())
		}
		gotText, err := atoms[0].MarshalText()
		if err != nil {
			t.Errorf("TestReadAtomsFromHex(%s): failed to get usable atom value from ReadAtomsFromHex()", a.Name())
		}
		wantText, err := a.MarshalText()
		if err != nil {
			t.Errorf("TestReadAtomsFromHex(%s): could not run test, unable to generate expected result text", a.Name())
		}
		if string(gotText) != string(wantText) {
			t.Errorf("TestReadAtomsFromHex(%s): Result mismatch for atom %s", a.Name())
		}
	}
}
