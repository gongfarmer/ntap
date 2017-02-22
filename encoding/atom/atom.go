// Package atom provides encodings for the ADE AtomContainer format.
// It includes conversions between text and binary formats, as well as an
// encoding-independent struct to provide convenient accessors.
// TODO Encoder and Decoder as in encoding/xml, encoding/json
// see encoding/gob/example_test.go for model of how this works
package atom

import (
	"encoding"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"unicode"
)

// Verify that type Atom satisifies these encoding interfaces at compile time
var _ encoding.BinaryUnmarshaler = &Atom{}
var _ encoding.BinaryMarshaler = &Atom{}
var _ encoding.TextUnmarshaler = &Atom{}
var _ encoding.TextMarshaler = &Atom{}

type Atom struct {
	Name     string
	Value    *codec
	Children []*Atom
	typ      ADEType
	data     []byte
}
type ADEType string
type GoType string

// Return true if string is printable, false otherwise
func isPrintableString(s string) bool {
	for _, r := range s {
		if !unicode.IsPrint(r) {
			return false
		}
	}
	return true
}

// unicode.IsPrint does not work for this, it returns true for large swathes of
// ascii 127-255.
func isPrintableBytes(buf []byte) bool {
	for _, b := range buf {
		if !strings.ContainsRune(printableChars, rune(b)) {
			return false
		}
	}
	return true
}

// Read-only access to ADE data type
func (a *Atom) Type() ADEType {
	return a.typ
}

// Set atom type.
// Get a codec that can write/read this ADE type.
// Allocate proper amount of backing memory for non-String types.
func (a *Atom) SetType(newType ADEType) {
	a.typ = newType
	a.Value = NewCodec(a)
	a.ZeroData()
}

// Set atom data to zero value of its ADE type.
// Allocate the correct amount of backing memory.
func (a *Atom) ZeroData() {
	switch a.typ {
	case UI08, SI08:
		a.data = make([]byte, 1)
	case UI16, SI16:
		a.data = make([]byte, 2)
	case UI01, UI32, SI32, FP32, UF32, SF32, SR32, UR32, FC32, IP32, ENUM:
		a.data = make([]byte, 4)
	case UI64, SI64, FP64, UF64, SF64, UR64, SR64:
		a.data = make([]byte, 8)
	case UUID:
		a.data = make([]byte, 36)
	case IPAD, CSTR, USTR, DATA, CNCT, Cnct:
		a.data = make([]byte, 0)
	case CONT, NULL:
		a.data = make([]byte, 0)
	default:
		panic(fmt.Sprintf("Unknown ADE type: %s", string(a.typ)))
	}
}

// String returns the atom's text description in ADE ContainerText format.
func (a Atom) String() string {
	buf, err := a.MarshalText()
	if err != nil {
		panic(fmt.Errorf("Failed to write Atom '%s:%s' to text: %s", a.Name, a.Type(), err))
	}
	return string(buf)
}

// AddChild makes the Atom pointed to by the argument a child of this Atom.
// Returns false when called on non-container Atoms.
func (c *Atom) AddChild(a *Atom) bool {
	if c.typ != CONT {
		return false
	}
	c.Children = append(c.Children, a)
	return true
}

// NumChildren returns a count of the number of children of this Atom.
// Returns -1 for non-container Atoms.
func (c *Atom) NumChildren() int {
	if c.typ != CONT {
		return -1
	}
	return len(c.Children)
}

// AtomList returns a list of pointers to every Atom in hierarchical order.
func (c *Atom) AtomList() []*Atom {
	return c.getAtomList(new([]*Atom))
}
func (c *Atom) getAtomList(list *([]*Atom)) []*Atom {
	*list = append(*list, c)
	for _, a := range c.Children {
		a.getAtomList(list)
	}
	return *list
}

// FromFile reads a binary AtomContainer from the named file path.
func FromFile(path string) (a Atom, err error) {
	fstat, err := os.Stat(path)
	if err != nil {
		return
	}

	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return
	}
	var encoded_size = int64(binary.BigEndian.Uint32(buf[0:4]))
	if encoded_size != fstat.Size() {
		err = fmt.Errorf(
			"Invalid AtomContainer file, encoded size %d does not match file size %d.",
			encoded_size, fstat.Size())
		return
	}

	err = a.UnmarshalBinary(buf)
	return
}

// Panic if an unexpected error is encountered here.
// Return the same error if it's expected.
func checkError(err error) error {
	if err == nil || err == io.EOF {
		return err
	}
	panic(err)
}

// Sets this atom object to the zero value of an Atom.
// Updates the atom data ptr to point to zero-length slice, releasing any
// previous memory allocated for data.
// Updates the list of children to an empty slice.
func (a *Atom) Zero() {
	a.Name = ""
	a.SetType(NULL)
	a.Children = []*Atom{}
}
