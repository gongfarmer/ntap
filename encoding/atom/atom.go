// Package atom provides encodings for ADE AtomContainers.
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

// Log is a log.Logger object where atom debug-level log messages are sent.
// By default it discards log messages, because there's nothing here that
// users of this API need to see.  Set this to something else if you want to
// see log messages.
var Log *log.Logger

func init() {
	Log = log.New(ioutil.Discard, "atom", log.LstdFlags)
}

// Name returns a copy of the atoms's name.
// If printable, it's 4 printable chars.  Otherwise, it's
// a hex string preceded by 0x.
func (a *Atom) Name() (name string) {
	name, _ = codec.FC32ToString(a.name)
	return name
}

// NameAsUint32 returns a copy of the atoms's 4-byte name as a single uint32
// value.
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
	if len(name) != 4 {
		return nil, fmt.Errorf(`atom name must be 4 chars long, got "%s"`, name)
	}
	if len(name) != 4 {
		return nil, fmt.Errorf(`atom type must be 4 chars long, got "%s"`, name)
	}
	a = &Atom{
		name: []byte{name[0], name[1], name[2], name[3]},
	}
	a.SetType(typ)
	return a, e
}

// Zero sets the atom to the type Atom's zero value.
// It sets the atom data to a zero-length slice, releasing any
// previous memory allocated for data.
// It also empties the list of child atoms.
func (a *Atom) Zero() {
	a.name = []byte{0, 0, 0, 0}
	a.SetType(codec.NULL)
	a.children = []*Atom{}
}

// SetType sets the type of an Atom object, and handles updating the Codec and
// data fields to match.
func (a *Atom) SetType(newType codec.ADEType) {
	a.typ = newType
	a.Value = codec.NewCodec(&a.data, a.typ)
	a.Value.ZeroData()
}

// String returns the atom's text description in ADE ContainerText format.
// This is a single line, it doesn't include a list of child atoms.
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
