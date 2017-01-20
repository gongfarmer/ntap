// Package atom provides encodings for the ADE AtomContainer format.
// It includes conversions between text and binary formats, as well as an
// encoding-independent struct to provide convenient accessors.
// TODO Encoder and Decoder as in encoding/xml, encoding/json
// see encoding/gob/example_test.go for model of how this works
package atom

import (
	"bytes"
	"encoding"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"unicode"
)

// Verify that atom meets encoding interfaces at compile time
var _ encoding.BinaryUnmarshaler = &(Atom{})

// var _ encoding.BinaryMarshaler = Atom{} // TODO
// var _ encoding.TextUnmarshaler = Atom{} // TODO
// var _ encoding.TextMarshaler = Atom{} // TODO

// GOAL: make this concurrency-safe, perhaps immutable
type Atom struct {
	Name     string
	Type     string
	Data     []byte
	Children []*Atom
}

// Return true if string is printable, false otherwise
func isPrint(s string) bool {
	for _, r := range s {
		if !unicode.IsPrint(r) {
			return false
		}
	}
	return true
}

func (a Atom) String() string {
	output := buildString(&a, 0)
	return output.String()
}

func (a Atom) Value() (value reflect.Value) {
	decOpTable[adeTypeMap[a.Type]](a.Data, value)
	return
}

// FIXME Rename because its not a string
func buildString(a *Atom, depth int) bytes.Buffer {
	var (
		output        bytes.Buffer
		printableName string
	)
	// print atom name + type
	if isPrint(a.Name) {
		printableName = a.Name
	} else {
		printableName = fmt.Sprintf("0x%+08X", a.Name)
	}
	fmt.Fprintf(&output, "% *s%s:%s:", depth*4, "", printableName, a.Type)
	fmt.Fprintln(&output, a.Value())

	// FIXME print data

	// print children
	if a.Type == "CONT" {
		for _, childPtr := range a.Children {
			buf := buildString(childPtr, depth+1)
			output.WriteString(buf.String())
		}
	}
	return output
}

func (c *Atom) addChild(a *Atom) {
	if c.Type == "CONT" {
		c.Children = append(c.Children, a)
	} else {
		panic(fmt.Errorf("Cannot add child to non-CONT atom %s:%s", c.Name, c.Type))
	}
}

func FromFile(path string) (a Atom, err error) {
	var (
		buf []byte
	)

	fstat, err := os.Stat(path)
	if err != nil {
		return
	}

	buf, err = ioutil.ReadFile(path)
	if err != nil {
		return
	}
	var encoded_size int64 = int64(binary.BigEndian.Uint32(buf[0:4]))
	if encoded_size != fstat.Size() {
		fmt.Errorf("Invalid AtomContainer file (encoded size does not match filesize)")
	}

	err = a.UnmarshalBinary(buf)
	return
}
