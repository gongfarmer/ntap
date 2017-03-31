// Package atom provides encodings for ADE AtomContainers.
// It includes a struct type with getters/setters for ADE data types, and
// provides conversions to and from text and binary atom container formats.
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

// Verify that type Atom satisfies these interfaces at compile time
var _ encoding.BinaryUnmarshaler = &Atom{}
var _ encoding.BinaryMarshaler = &Atom{}
var _ encoding.TextUnmarshaler = &Atom{}
var _ encoding.TextMarshaler = &Atom{}

// Atom represents a single ADE atom, which may be a container containing other atoms.
type Atom struct {
	name     []byte
	typ      ADEType
	data     []byte
	children []*Atom
	Value    *Codec
}

// Name returns a copy of the atoms's name.
// If printable, it's 4 printable chars.  Otherwise, it's
// a hex string preceded by 0x.
func (a Atom) Name() (name string) {
	name, _ = FC32ToString(a.name)
	return name
}

// NameAsUint32 returns a copy of the atoms's 4-byte name as a single uint32
// value.
func (a Atom) NameAsUint32() uint32 {
	return binary.BigEndian.Uint32(a.name)
}

// Type returns a copy of the atoms's ADE data type.
func (a Atom) Type() string {
	return string(a.typ)
}

// Children returns a slice of this Atom's child atoms
func (a Atom) Children() []*Atom {
	return a.children
}

// AtomAtPath returns the single Atom descendant at the given path, or nil if none.
// ValueAtPath returns the atom Value object at the given path, or nil if none.
// FIXME: provide a way to get all children of a node without specifying all
// their names -- needed to access data in attribute containers.
// FIXME: provide a way to replace specifier at 1 level with *? multi-level with **?

// Zero sets the atom to the type Atom's zero value.
// It sets the atom data to a zero-length slice, releasing any
// previous memory allocated for data.
// It also empties the list of child atoms.
func (a *Atom) Zero() {
	a.name = []byte{0, 0, 0, 0}
	a.SetType(NULL)
	a.children = []*Atom{}
}

// SetType sets the type of an Atom object, and handles updating the Codec and
// data fields to match.
func (a *Atom) SetType(newType ADEType) {
	a.typ = newType
	a.Value = NewCodec(a)
	a.ZeroData()
}

// ZeroData sets an atom's data to the zero value of its ADE type.
// For fixed-size types, the byte slice capacity remains the same so that a new
// value can be set without needing memory allocation.
// For variable-sized types, data is set to nil and all memory released for
// garbage collection.
func (a *Atom) ZeroData() {
	switch a.typ {
	case UI08, SI08:
		zeroOrAllocateByteSlice(&a.data, 1)
	case UI16, SI16:
		zeroOrAllocateByteSlice(&a.data, 2)
	case UI01, UI32, SI32, FP32, UF32, SF32, SR32, UR32, FC32, IP32, ENUM:
		zeroOrAllocateByteSlice(&a.data, 4)
	case UI64, SI64, FP64, UF64, SF64, UR64, SR64:
		zeroOrAllocateByteSlice(&a.data, 8)
	case UUID:
		zeroOrAllocateByteSlice(&a.data, 36)
	case IPAD, CSTR, USTR, DATA, CNCT, Cnct:
		a.data = nil
	case CONT, NULL:
		a.data = nil
	default:
		panic(fmt.Sprintf("unknown ADE type: %s", string(a.typ)))
	}
}

// String returns the atom's text description in ADE ContainerText format.
// This is a single line, it doesn't include a list of child atoms.
func (a Atom) String() string {
	if a.typ == CONT {
		return fmt.Sprintf("%s:%s:", a.Name(), a.Type())
	} else {
		str, _ := a.Value.StringDelimited()
		return fmt.Sprintf("%s:%s:%s", a.Name(), a.Type(), str)
	}
}

// AddChild makes the Atom pointed to by the argument a child of this Atom.
// Returns false when called on non-container Atoms.
func (a *Atom) AddChild(child *Atom) bool {
	if a.typ != CONT {
		return false
	}
	a.children = append(a.children, child)
	return true
}

// NumChildren returns a count of the number of children of this Atom.
// Returns -1 for non-container Atoms.
func (a *Atom) NumChildren() int {
	if a.typ != CONT {
		return -1
	}
	return len(a.children)
}

// Descendants returns a list of pointers to every Atom in hierarchical order.
// Starts with self.
func (a *Atom) Descendants() []*Atom {
	return a.getDescendants(new([]*Atom))
}
func (a *Atom) getDescendants(list *([]*Atom)) []*Atom {
	*list = append(*list, a)
	for _, child := range a.children {
		child.getDescendants(list)
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
	var encodedSize = int64(binary.BigEndian.Uint32(buf[0:4]))
	if encodedSize != fstat.Size() {
		err = fmt.Errorf(
			"invalid AtomContainer file, encoded size %d does not match file size %d",
			encodedSize, fstat.Size())
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

// zeroOrAllocateByteSlice verifies that the give byte slice has
// the specified capacity, and zeroes it out.
// It avoids memory allocation when possible.
func zeroOrAllocateByteSlice(buf *[]byte, size int) {
	if cap(*buf) == size {
		// zero out the buffer, O(1)
		for i := range *buf {
			(*buf)[i] = 0
		}
	} else {
		// newly allocated mem is already zeroed
		*buf = make([]byte, size)
	}
}
