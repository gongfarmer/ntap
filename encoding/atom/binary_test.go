package atom

//
// Verify that UnmarshalBinary successfully reads all binary test files.
//
// Verify that MarshalBinary successfully writes atoms to binary, and that the
// serialized atoms match the original binary files.
//

import (
	"crypto/sha1"
	"testing"
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
