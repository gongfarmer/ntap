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
	Atom
	end uint32
}

// LIFO stack of Atoms of type container.
type containerStack []cont

// Return a pointer to the last (top) element of the stack, without removing
// the element. Second return value is false if stack is empty.
func (s containerStack) Peek() (value *cont, ok bool) {
	if len(s) == 0 {
		value, ok = &(cont{}), false
	} else {
		value, ok = &(s[len(s)-1]), true
	}
	return
}
func (s *containerStack) Push(c cont) { (*s) = append((*s), c) }
func (s *containerStack) Pop() cont {
	d := (*s)[len(*s)-1]
	(*s) = (*s)[:len(*s)-1]
	return d
}

// Pop fully-read containers off the container stack, as they're now closed.
// FIXME: handle corrupt container case where cont length is incorrect
func (s *containerStack) PopCompleted(pos uint32) []Atom {
	var closedConts []Atom
	// Pop until the given byte offset precedes the top object's end position.
	for p, ok := s.Peek(); ok; p, ok = s.Peek() {
		if pos == p.end {
			closedConts = append(closedConts, s.Pop().Atom)
			continue // next CONT might end too
		}
		if pos > p.end {
			panic(fmt.Errorf("%s:CONT wanted to end at byte %d, but read position is now %d", p.Name, p.end, pos))
		}
		break
	}

	return closedConts
}

// atomHeader models the binary encoding values that start every ADE
// AtomContainer.  Must have fixed size.
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
	fmt.Printf("UnmarshalBinary got %v\n", a)
	return err
}

func (a *Atom) UnmarshalFromReader(r io.Reader) error {
	atoms, err := ReadAtomsFromBinaryStream(r)
	if err != nil {
		panic(fmt.Errorf("Failed to parse binary stream: %s", err))
	}

	switch len(atoms) {
	case 1:
		(*a) = atoms[0]
	case 0:
		panic(fmt.Errorf("Binary stream contained no atoms"))
	default:
		panic(fmt.Errorf("Binary stream contained multiple atoms, but Atom.Unmarshal can only handle 1 atom."))
	}
	return err
}

// FIXME watch for cases where EOF is handled before last Atom object is
// created, thus dropping some data
func ReadAtomsFromBinaryStream(r io.Reader) (atoms []Atom, err error) {
	var (
		bytesRead  uint32
		CONT       [4]byte
		containers containerStack
		a          Atom
	)
	copy(CONT[:], "CONT")

	for {
		containers.PopCompleted(bytesRead)

		// Read next atom
		h, err := readAtomHeader(r, &bytesRead)
		// FIXME what if EOF happens here? Not possible in well-formed atomcontainer.
		if h.Type == CONT { // new AtomContainer
			endIndex := bytesRead + h.Size - headerBytes
			a = Atom{Name: string(h.Name[:]), Type: string(h.Type[:])}
			if endIndex != bytesRead { // empty Container ends immediately
				containers.Push(cont{a, endIndex})
			}
		} else {
			data, err := readAtomData(r, h.Size-headerBytes, &bytesRead)
			checkError(err)
			a = Atom{Name: string(h.Name[:]), Type: string(h.Type[:]), Data: data}
		}
		//		fmt.Println("Got atom ", a)

		// Add new atom to children of parent container, if any
		if p, ok := containers.Peek(); ok {
			fmt.Printf("Got child atom: %v\n", a)
			(*p).addChild(a)
			fmt.Printf("Got parent (%v) now has children : %v\n", p.Name, len(p.Children))
		} else {
			fmt.Printf("Got top atom: %v\n", a)
			atoms = append(atoms, a)
		}

		if io.EOF == err {
			break
		}
	}
	fmt.Printf("Finished reading %d bytes from stream\n", bytesRead)
	fmt.Printf("Got atoms: %v\n", atoms[0])

	if err == io.EOF {
		err = nil
	}
	return atoms, err
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
