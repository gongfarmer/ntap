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
	Name     string
	typ      ADEType
	data     []byte
	Value    *Codec
	Children []*Atom
}

// Type returns a copy of the atoms's ADE data type.
func (a *Atom) Type() ADEType {
	return a.typ
}

// AtomsAtPath returns a slice of all Atom descendants at the given path.
// If no atom is found, it returns an error message that describes which path
// part doesn't exist.
//
// Requirements for Path definition wildcards:
//   * provide a way to select all attribute container data elemnts while
//     omitting the index element. (???)
//   * provide a terse syntax to use form command-line clients to search for
//     an element by name at any position in the tree.  (**/NAME)
// Path definition wildcards to borrow from XPATH:
//   * match any single path element of any type
//   ** match any number of nested path elements
//   *[1] return first child of container elt (there's no 0 elt)
//   *[last()] return first child of container elt named "book"
//   *[position()<3] return first 2 child elts of container elt named "book"
//   *[not(position()] return first 2 child elts of container elt named "book"
//   *[@type=XXXX] match any element of type XXXX
//   *[@name=XXXX] match any element with name XXXX
//   *[@data<35] match any element whose numeric value < 35 (raise error on non-numeric)
//   *[not(@type!=UI32) and @data<35] boolean syntax
//
// TODO:
//   define boolean syntax for operators
//
// FIXME paths should be resolveable using hex or non-hex FC32 representation.
// Currently, the user-provided path is matched only against what is stored as
// the Name field, which is one or the other.
func (a *Atom) AtomsAtPath(path string) (atoms []*Atom, e error) {
	var pathParts = append([]string{a.Name}, strings.Split(path, "/")...)
	return getAtomsAtPath(a, pathParts, 1)
}

// must return no atoms on error
func getAtomsAtPath(a *Atom, pathParts []string, index int) (atoms []*Atom, e error) {
	if a.Type() != CONT {
		e = fmt.Errorf("atom '%s' is not a container", strings.Join(pathParts[:index], "/"))
		return
	}

	// find all child atoms whose name matches the next path part
	var nextAtoms []*Atom
	for _, child := range a.Children {
		if pathParts[index] == "*" || child.Name == pathParts[index] {
			nextAtoms = append(nextAtoms, child)
		}
	}

	// return error if no child atoms matched
	if len(nextAtoms) == 0 {
		pathSoFar := strings.Join(pathParts[:index], "/")
		e = fmt.Errorf("atom '%s' has no child named '%s'", pathSoFar, pathParts[index])
		return
	}

	// if this is the final path part, then return all matched atoms regardless of type
	if index == len(pathParts)-1 { // if last path part
		atoms = append(atoms, nextAtoms...)
		return
	}

	// search child atoms for the rest of the path
	var foundCont bool
	for _, child := range nextAtoms {
		if child.Type() != CONT {
			continue
		}
		foundCont = true
		if moreAtoms, err := getAtomsAtPath(child, pathParts, index+1); err == nil {
			atoms = append(atoms, moreAtoms...)
		} else {
			return atoms, err
		}
	}

	// if none of the matching children were containers, return error
	if !foundCont {
		pathSoFar := strings.Join(pathParts[:index], "/")
		e = fmt.Errorf("atom '%s' has no container child named '%s'", pathSoFar, pathParts[index])
		return
	}
	return
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
	a.Name = ""
	a.SetType(NULL)
	a.Children = []*Atom{}
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
func (a Atom) String() string {
	buf, err := a.MarshalText()
	if err != nil {
		panic(fmt.Errorf("failed to write Atom '%s:%s' to text: %s", a.Name, a.Type(), err))
	}
	return string(buf)
}

// AddChild makes the Atom pointed to by the argument a child of this Atom.
// Returns false when called on non-container Atoms.
func (a *Atom) AddChild(child *Atom) bool {
	if a.typ != CONT {
		return false
	}
	a.Children = append(a.Children, child)
	return true
}

// NumChildren returns a count of the number of children of this Atom.
// Returns -1 for non-container Atoms.
func (a *Atom) NumChildren() int {
	if a.typ != CONT {
		return -1
	}
	return len(a.Children)
}

// AtomList returns a list of pointers to every Atom in hierarchical order.
func (a *Atom) AtomList() []*Atom {
	return a.getAtomList(new([]*Atom))
}
func (a *Atom) getAtomList(list *([]*Atom)) []*Atom {
	*list = append(*list, a)
	for _, child := range a.Children {
		child.getAtomList(list)
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
