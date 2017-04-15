package ade

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

	"github.com/gongfarmer/ntap/encoding/ade/codec"
)

func TestUnmarshalBinary(t *testing.T) {
	for _, test := range Tests {
		a := new(Atom)
		if err := a.UnmarshalBinary(test.binBytes); err != nil {
			t.Errorf("UnmarshalBinary(%s): expect no error, got %s", test.Name(), err.Error())
		}
	}
}

// NOTE: marshaled output is not guaranteed to always match its input, as odd
// but valid inputs may become normalized. However, they do match for these
// particular tests.
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

// var BID0_HEX = []byte("0000004442494430434F4E5400000010425645525549333200000001000000144254494D55493634000546592CD6DB2C000000144E45585455493634DDDDF0000C000000")
//
// var BID0_TEXT = []byte(`
// BID0:CONT:
// 	BVER:UI32:1
// 	BTIM:UI64:1484723701865260
// 	NEXT:UI64:15987198135227121664
// END
// `)
//
// type binaryTest struct {
// 	Input     interface{}
// 	WantValue []Atom
// 	WantError error
// }
//
// func runBinaryTests(t *testing.T, tests []binaryTest) {
// 	for _, test := range tests {
// 		atoms, gotErr := ReadAtomsFromHex(strings.NewReader(test.Input.(string)))
// 		switch {
// 		case gotErr == nil && test.WantError == nil:
// 		case gotErr != nil && test.WantError == nil:
// 			t.Errorf("%v(%b): got err %s, want err <nil>", "ReadAtomsFromHex", test.Input, gotErr)
// 		case gotErr == nil && test.WantError != nil:
// 			t.Errorf("%v(%b): got err <nil>, want err %s", "ReadAtomsFromHex", test.Input, test.WantError)
// 		case gotErr.Error() != test.WantError.Error():
// 			t.Errorf("%v(%b): got err %s, want err %s", "ReadAtomsFromHex", test.Input, gotErr, test.WantError)
// 			return
// 		}
//
// 		if fmt.Sprint(atoms) != fmt.Sprint(test.WantValue) {
// 			t.Errorf("%v(%x): got %T \"%[3]v\", want %[4]T \"%[4]v\"", "ReadAtomsFromHex", test.Input, fmt.Sprint(atoms), fmt.Sprint(test.WantValue))
// 		}
// 	}
// }
//
// func TestReadAtomsFromHexErrors(t *testing.T) {
// 	var bid0 Atom
// 	if err := bid0.UnmarshalText(BID0_TEXT); err != nil {
// 		panic("Could not create test atom")
// 	}
// 	tests := []binaryTest{
// 		binaryTest{BID0_HEX, []Atom{bid0}, nil},
// 	}
// 	runBinaryTests(t, tests)
// }
