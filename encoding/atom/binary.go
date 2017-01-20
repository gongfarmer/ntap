package atom

// Enable reading and writing of binary format ADE AtomContainers by fulfilling
// these interfaces:
//
// type BinaryMarshaler interface {
// 	MarshalBinary() (data []byte, err error)
// }
//
// type BinaryUnmarshaler interface {
// 	UnmarshalBinary(data []byte) error
// }

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"reflect"
)

// AtomContainer object, paired with its end byte position in the stream
type cont struct {
	atomPtr *Atom
	end     uint32
}

// LIFO stack of Atoms of type container.
type containerStack []cont

// Return a pointer to the last (top) element of the stack, without removing
// the element. Second return value is false if stack is empty.
func (s *containerStack) Peek() (value *cont, ok bool) {
	if len(*s) == 0 {
		value, ok = &(cont{}), false
	} else {
		value, ok = &((*s)[len(*s)-1]), true
	}
	return
}
func (s *containerStack) Empty() bool {
	return len(*s) == 0
}
func (s *containerStack) Push(c cont) {
	(*s) = append((*s), c)
}
func (s *containerStack) Pop() cont {
	d := (*s)[len(*s)-1]
	(*s) = (*s)[:len(*s)-1]
	return d
}

// Pop fully-read containers off the container stack based on the given byte
// position.
// FIXME: handle corrupt container case where cont length is incorrect
func (s *containerStack) PopCompleted(pos uint32) []*Atom {
	var closedConts []*Atom
	// Pop until the given byte offset precedes the top object's end position.
	for p, ok := s.Peek(); ok; p, ok = s.Peek() {
		if pos == p.end {
			closedConts = append(closedConts, s.Pop().atomPtr)
			continue // next CONT might end too
		}
		if pos > p.end {
			panic(fmt.Errorf("%s:CONT wanted to end at byte %d, but read position is now %d", p.atomPtr.Name, p.end, pos))
		}
		break
	}

	return closedConts
}

// atomHeader models the binary encoding values that start every ADE
// AtomContainer. Must have fixed size for initialization by encoding/binary.
type atomHeader struct {
	Size uint32
	Name [4]byte
	Type [4]byte
}

var headerBytes uint32 = uint32(reflect.TypeOf(atomHeader{}).Size())

// UnmarshalBinary implements encoding.BinaryUnmarshaler.
// It can be used to rehydrate an Atom starting from the zero value of Atom.
func (a *Atom) UnmarshalBinary(data []byte) error {
	err := a.UnmarshalFromReader(bytes.NewReader(data))
	return err
}

func (a *Atom) UnmarshalFromReader(r io.Reader) error {
	atoms, err := ReadAtomsFromBinaryStream(r)
	if err != nil {
		panic(fmt.Errorf("Failed to parse binary stream: %s", err))
	}

	switch len(atoms) {
	case 1:
		*a = *atoms[0]
	case 0:
		panic(fmt.Errorf("Binary stream contained no atoms"))
	default:
		panic(fmt.Errorf("Binary stream contained multiple atoms, but Atom.Unmarshal can only handle 1 atom."))
	}
	return err
}

func ReadAtomsFromBinaryStream(r io.Reader) (atoms []*Atom, err error) {
	var (
		bytesRead  uint32
		CONT       = [4]byte{'C', 'O', 'N', 'T'}
		containers containerStack
	)
	for err := error(nil); err != io.EOF; {
		// read binary header for next atom
		h, err := readAtomHeader(r, &bytesRead)
		if checkError(err) == io.EOF {
			break
		}

		// construct complete Atom object (with data if any)
		var a Atom
		if h.Type == CONT {
			a = Atom{Name: string(h.Name[:]), Type: string(h.Type[:])}
		} else {
			data, err := readAtomData(r, h.Size-headerBytes, &bytesRead)
			checkError(err)
			a = Atom{Name: string(h.Name[:]), Type: string(h.Type[:]), Data: data}
		}

		// add atom to parent.Children, or to atoms list if no parent
		if parent, ok := containers.Peek(); ok {
			parent.atomPtr.addChild(&a)
		} else {
			atoms = append(atoms, &a)
		}

		// push container onto stack
		if h.Type == CONT {
			endPos := bytesRead + h.Size - headerBytes
			containers.Push(cont{&a, endPos})
		}

		// pop fully read containers off stack
		containers.PopCompleted(bytesRead)
	}
	return atoms, err // err is never set after initialization
}

// Panic if an unexpected error is encountered here.
// Return the same error if it's expected.
func checkError(err error) error {
	switch err {
	case nil, io.EOF:
		return err
	default:
		panic(fmt.Errorf("unable to read from byte stream: %s", err))
	}
	return nil
}

func readAtomHeader(r io.Reader, bytesRead *uint32) (h atomHeader, err error) {
	h = atomHeader{}
	err = binary.Read(r, binary.BigEndian, &h)
	*bytesRead += headerBytes
	return
}

func readAtomData(r io.Reader, length uint32, bytesRead *uint32) (data []byte, err error) {
	data = make([]byte, length)
	err = binary.Read(r, binary.BigEndian, &data)
	*bytesRead += length
	return
}
