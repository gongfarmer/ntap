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
	"errors"
	"fmt"
	"io"
	"io/ioutil"
)

// AtomContainer object, paired with its end byte position in the stream
type cont struct {
	atomPtr *Atom
	end     uint32
}

var ErrInvalidInput = errors.New("Input is invalid for conversion to Atom")

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

const headerSize = 12

func (h atomHeader) isContainer() bool {
	return ADEType(h.Type[:]) == CONT
}

/**********************************************************/
// Unmarshal from binary
/**********************************************************/

// UnmarshalBinary implements encoding.BinaryUnmarshaler.
// It can be used to rehydrate an Atom starting from the zero value of Atom.
// FIXME: rewrite to handle a stream with multiple top-level atoms
func (a *Atom) UnmarshalBinary(data []byte) error {
	return a.UnmarshalFromReader(bytes.NewReader(data))
}

func (a *Atom) UnmarshalFromReader(r io.Reader) error {
	atoms, err := ReadAtomsFromBinary(r)
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
				return atoms, ErrInvalidInput
			}
		}
		adeType := ADEType(h.Type[:])
		var a = Atom{
			Name: asPrintableString(h.Name[:]),
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

	// Strip all newlines
	buffer = bytes.Replace(buffer, []byte("\n"), nil, -1)
	buffer = bytes.Replace(buffer, []byte("\r"), nil, -1)

	// Strip all spaces
	buffer = bytes.Replace(buffer, []byte(" "), nil, -1)

	// Strip leading 0x
	if string(buffer[0:2]) == "0x" {
		buffer = buffer[2:]
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

// MarshalBinary implements encoding.BinaryMarshaler.
// It can be used to rehydrate an Atom starting from the zero value of Atom.
func (a *Atom) MarshalBinary() (data []byte, err error) {
	buf := new(bytes.Buffer)
	err = a.BinaryWrite(buf)
	return buf.Bytes(), err
}

// Encode this atom and its children to binary bytes.
// Write bytes to the given io.Writer.
func (a *Atom) BinaryWrite(w io.Writer) (err error) {
	// write atom header
	var name [4]byte
	var typ [4]byte
	copy(name[:], a.Name)
	copy(typ[:], a.Type())
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
//	+4 bytes (size field)
//	+4 bytes (atom name field)
//	+4 bytes (atom type field)
// == 12 bytes minimum
func (a *Atom) Len() (length uint32) {
	length = uint32(headerSize + len(a.data))
	for _, child := range a.Children {
		length += child.Len()
	}
	return
}
