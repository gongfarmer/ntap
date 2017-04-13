// Package atom provides support for ADE AtomContainers.
// It includes a struct type with getters/setters for ADE data types, and
// provides conversions to and from text and binary atom container formats.
package atom

import (
	"encoding"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/gongfarmer/ntap/encoding/atom/codec"
)

// Verify that type Atom satisfies these interfaces at compile time
var _ encoding.BinaryUnmarshaler = &Atom{}
var _ encoding.BinaryMarshaler = &Atom{}
var _ encoding.TextUnmarshaler = &Atom{}
var _ encoding.TextMarshaler = &Atom{}

// Atom represents a single ADE atom, which may be a container containing other atoms.
type Atom struct {
	name     []byte
	typ      codec.ADEType
	data     []byte
	children []*Atom
	Value    *codec.Codec
}

// Log is a log.Logger object where debug-level log messages from atom handling
// operations are sent.
//
// By default it discards log messages.
// To see debug-level logging, redirect logging output by calling Log.SetOutput(w io.Writer), or set this to a different log.Logger object.
var Log *log.Logger

func init() {
	Log = log.New(ioutil.Discard, "atom", log.LstdFlags)
}

// Name returns a printable string version of the atoms's name.
//
// If all 4 bytes are printable ascii, output is a 4 character string.
//
// Otherwise, output is an 8 digit hex string preceded by 0x.
func (a *Atom) Name() (name string) {
	name, _ = codec.FC32ToString(a.name)
	return name
}

// NameAsUint32 returns a copy of the atoms's 4-byte name as a uint32.
func (a *Atom) NameAsUint32() uint32 {
	return binary.BigEndian.Uint32(a.name)
}

// Type returns a copy of the atoms's ADE data type.
func (a *Atom) Type() string {
	return string(a.typ)
}

// Children returns a slice of this Atom's child atoms
func (a *Atom) Children() []*Atom {
	return a.children
}

// NewAtom constructs a new Atom object with the specified name and type.
func NewAtom(name string, typ codec.ADEType) (a *Atom, e error) {
	a = new(Atom)
	e = codec.StringToFC32Bytes(&a.name, name)
	if e != nil {
		return
	}
	a.SetType(typ)
	return
}

// Zero sets the atom to the zero value of type Atom .
// It sets the atom data to a zero-length slice, releasing any
// previous memory allocated for data.
// It also empties the list of child atoms.
func (a *Atom) Zero() {
	a.name = []byte{0, 0, 0, 0}
	a.SetType(codec.NULL)
	a.children = []*Atom{}
}

// SetType sets the type of an Atom object, and updates the Codec and
// data fields to match.
func (a *Atom) SetType(newType codec.ADEType) {
	a.typ = newType
	a.Value = codec.NewCodec(&a.data, a.typ)
	a.Value.ZeroData()
}

// String returns the atom's text description in ADE ContainerText format.
// Output is a single line listing atom name, type and data (if any.)
// It does not include a list of child atoms.
func (a *Atom) String() string {
	if a.typ == codec.CONT {
		return fmt.Sprintf("%s:%s:", a.Name(), a.Type())
	}
	str, _ := a.Value.StringDelimited()
	return fmt.Sprintf("%s:%s:%s", a.Name(), a.Type(), str)
}

// AddChild makes the Atom pointed to by the argument a child of this Atom.
// Returns false when called on non-container Atoms.
func (a *Atom) AddChild(child *Atom) bool {
	if a.typ != codec.CONT {
		return false
	}
	a.children = append(a.children, child)
	return true
}

// NumChildren returns a count of the number of children of this Atom.
// Returns -1 for non-container Atoms.
func (a *Atom) NumChildren() int {
	if a.typ != codec.CONT {
		return -1
	}
	return len(a.children)
}

// Descendants returns a list of pointers to every Atom in hierarchical order.
// (ie. results of in-order tree traversal.)
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
