// Package atom provides encodings for the ADE AtomContainer format.
// It includes conversions between text and binary formats, as well as an
// encoding-independent struct to provide convenient accessors.
package atom

import (
	"bytes"
	"encoding"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"os"
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
	Data     interface{}
	Children []Atom
}

func (a Atom) String() string {
	var (
		output bytes.Buffer
		depth  int
	)
	switch a.Type {
	case "CONT":
		output.WriteString(fmt.Sprintf("% *s", depth*2, a.Name))
		output.WriteString(fmt.Sprintf(":CONT with %d children\n", len(a.Children)))
		depth++ // FIXME make all this truly recursive
		for _, c := range a.Children {
			fmt.Println("Yup im in here") // FIXME never reached
			output.WriteString(fmt.Sprintf("% *s\n", depth*2, c.String()))
		}
		output.Truncate(output.Len() - 1) // strip newline
	//case "UI32":
	//	output.WriteString(fmt.Sprintf("%s:%s:", a.Name, a.Type))
	default:
		output.WriteString(fmt.Sprintf("% *s:%s:", depth*2, a.Name, a.Type))
	}

	return output.String()
}

func (c *Atom) addChild(a Atom) {
	if c.Type == "CONT" {
		fmt.Println("b4: Now there are this many children: ", len(c.Children))
		c.Children = append(c.Children, a)
		fmt.Println("ar: Now there are this many children: ", len(c.Children))
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
