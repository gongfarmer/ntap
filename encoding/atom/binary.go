package atom

// Enable reading and writing of binary format ADE AtomContainers by fulfilling
// these interfaces from stdlib encoding/:
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
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
)

// AtomContainer object, paired with its end byte position in the stream
type cont struct {
	atomPtr *Atom
	end     uint32
}

// LIFO stack of Atoms of type container.
type containerStack []cont

// Peek returns a pointer to the last (top) element of the stack, without
// removing the element. Second return value is false if stack is empty.
func (s *containerStack) Peek() (value *cont, ok bool) {
	if len(*s) == 0 {
		value, ok = nil, false
	} else {
		value, ok = &((*s)[len(*s)-1]), true
	}
	return
}

// Empty returns true if the stack is empty.
func (s *containerStack) Empty() bool {
	return len(*s) == 0
}

// Push puts an element on top of the stack.
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

// An atomHeader models the binary encoding values that start every ADE
// AtomContainer. All struct elements must have fixed size so that
// encoding/binary can use this type to read/write bytes.
type atomHeader struct {
	Size uint32
	Name [4]byte
	Type [4]byte
}

const headerSize = 12

// isContainer returns true if this atomHeader's type is CONT.
func (h atomHeader) isContainer() bool {
	return ADEType(h.Type[:]) == CONT
}

/**********************************************************/
// Unmarshal from binary
/**********************************************************/

// UnmarshalBinary reads an Atom from a byte slice.
//
// It implements the encoding.BinaryMarshaler interface.
func (a *Atom) UnmarshalBinary(data []byte) error {
	return a.UnmarshalFromReader(bytes.NewReader(data))
}

// UnmarshalFromReader reads bytes from a reader. If the byte stream describes
// a valid ADE Binary Container, the container is reconstructed with the Atom
// receiver as the root container.
// Returns an error if the byte stream is not a valid binary AtomContainer, or
// there is more than one container in the stream.
func (a *Atom) UnmarshalFromReader(r io.Reader) error {
	atoms, err := ReadAtomsFromBinary(r)
	if err != nil {
		return fmt.Errorf("Failed to parse binary stream: %s", err.Error())
	}

	// Set receiver to the sole top-level AtomContainer
	switch len(atoms) {
	case 1:
		a.Zero()
		*a = *atoms[0]
	case 0:
		err = fmt.Errorf("Binary stream contained no atoms")
	default:
		err = fmt.Errorf("Binary stream contained multiple atoms, but Atom.Unmarshal can only handle 1 atom.")
	}
	return err
}

// ReadAtomsFromBinary reads bytes from a reader. It expects the byte stream to
// describe 0 or more ADE binary AtomContainers.
// It reconstructs all the AtomContainers found and returns them in an array of Atom objects.
// Returns an error if the byte stream contains invalid binary container data.
func ReadAtomsFromBinary(r io.Reader) (atoms []*Atom, err error) {
	defer func() {
		if p := recover(); p != nil {
			err = p.(error)
		}
	}()

	var (
		bytesRead  uint32
		containers containerStack
	)
	for err := error(nil); err != io.EOF; {
		// read next atom header
		h, err := readAtomHeader(r, &bytesRead)
		if checkError(err) == io.EOF {
			break
		}

		// construct Atom object, read data
		var data []byte
		if !h.isContainer() {
			data, err = readAtomData(r, h.Size-headerSize, &bytesRead)
			if err != nil {
				return atoms, fmt.Errorf("Input is invalid for conversion to Atom")
			}
		}
		adeType := ADEType(h.Type[:])
		name, err := FC32ToString(h.Name[:])
		if err != nil {
			return atoms, err
		}
		var a = Atom{
			Name: name,
			typ:  adeType,
			data: data,
		}
		a.Value = NewCodec(&a)

		// add atom to parent.Children, or to atoms list if no parent
		if parent, ok := containers.Peek(); ok {
			parent.atomPtr.AddChild(&a)
		} else {
			atoms = append(atoms, &a)
		}

		// push container onto stack
		if a.Type() == CONT {
			endPos := bytesRead + h.Size - headerSize
			containers.Push(cont{&a, endPos})
		}

		// pop fully read containers off stack
		containers.PopCompleted(bytesRead)
	}
	return atoms, err // err is never set after initialization
}

func readAtomHeader(r io.Reader, bytesRead *uint32) (h atomHeader, err error) {
	h = atomHeader{}
	err = binary.Read(r, binary.BigEndian, &h)
	*bytesRead += headerSize
	return
}

func readAtomData(r io.Reader, length uint32, bytesRead *uint32) (data []byte, err error) {
	data = make([]byte, length)
	err = binary.Read(r, binary.BigEndian, &data)
	*bytesRead += length
	return
}

func ReadAtomsFromHex(r io.Reader) (atoms []*Atom, err error) {
	var buffer []byte

	buffer, err = ioutil.ReadAll(r)
	if err != nil {
		panic(err)
	}

	// Strip newlines, spaces
	var clean = make([]byte, 0, len(buffer))
	for _, b := range buffer {
		if b != '\n' && b != '\r' && b != ' ' {
			clean = append(clean, b)
		}
	}

	// Strip leading 0x
	if string(clean[0:2]) == "0x" {
		buffer = clean[2:]
	} else {
		buffer = clean
	}

	// Convert pairs of hex values to single bytes
	buffer, err = hex.DecodeString(string(buffer))
	if err != nil && err != io.ErrUnexpectedEOF {
		return
	}

	// Don't attempt hex conversion without even length at this point
	if 0 != len(buffer)%2 {
		err = hex.ErrLength
		return
	}

	// Attempt conversion of the bytes buffer
	atoms, err = ReadAtomsFromBinary(bytes.NewReader(buffer))
	if err == nil {
		return // success!
	}

	// Conversion failed. Reverse endianness and try one more time.
	for i := 0; i < len(buffer); i += 2 {
		buffer[i], buffer[i+1] = buffer[i+1], buffer[i]
	}

	return ReadAtomsFromBinary(bytes.NewReader(buffer))
}

/**********************************************************/
// Marshal to binary
/**********************************************************/

// MarshalBinary serializes an Atom to a byte slice in ADE binary format.
//
// It implements the encoding.BinaryMarshaler interface.
func (a *Atom) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)
	err := a.BinaryWrite(buf)
	return buf.Bytes(), err
}

// Serialize this atom and its children to bytes.
// Write byte stream to the given io.Writer.
func (a *Atom) BinaryWrite(w io.Writer) (err error) {
	// create members for atom header
	var name, typ [4]byte
	var buf []byte
	if err = FC32StringToBytes(a.Name, &buf); err != nil {
		return
	}
	copy(name[:], buf)
	if err = FC32StringToBytes(string(a.typ), &buf); err != nil {
		return
	}
	copy(typ[:], buf)

	// write atom header
	err = binary.Write(w, binary.BigEndian, atomHeader{a.Len(), name, typ})
	if err != nil {
		return
	}

	// write atom data
	_, err = w.Write(a.data)
	if err != nil {
		return
	}

	// write children
	for _, child := range a.Children {
		err = child.BinaryWrite(w)
		if err != nil {
			return
		}
	}

	return
}

// Return length of this atom when encoded as binary bytes.
func (a *Atom) Len() (length uint32) {
	length = uint32(headerSize + len(a.data))
	for _, child := range a.Children {
		length += child.Len()
	}
	return
}
