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

	"github.com/gongfarmer/ntap/encoding/atom/codec"
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
func (s *containerStack) PopCompleted(pos uint32) (closedConts []*Atom, e error) {
	// Pop until the given byte offset precedes the top object's end position.
	for p, ok := s.Peek(); ok; p, ok = s.Peek() {
		if pos == p.end {
			closedConts = append(closedConts, s.Pop().atomPtr)
			continue // next CONT might end too
		}
		if pos > p.end {
			e = fmt.Errorf("%s:CONT wanted to end at byte %d, but read position is now %d", p.atomPtr.Name, p.end, pos)
		}
		break
	}

	return
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
	return codec.ADEType(h.Type[:]) == codec.CONT
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
		return fmt.Errorf("failed to parse binary stream: %s", err.Error())
	}

	// Set receiver to the sole top-level AtomContainer
	switch len(atoms) {
	case 1:
		a.Zero()
		*a = *atoms[0]
	case 0:
		err = fmt.Errorf("binary stream contained no atoms")
	default:
		err = fmt.Errorf("binary stream contained multiple atoms, but Atom.Unmarshal can only handle 1 atom")
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
	for err == nil {
		// read next atom header
		var h atomHeader
		h, err = readAtomHeader(r, &bytesRead)
		if err == io.EOF {
			err = nil
			break
		}

		// construct Atom object, read data
		var data []byte
		if !h.isContainer() {
			data, err = readAtomData(r, h.Size-headerSize, &bytesRead)
			if err != nil {
				return nil, fmt.Errorf("Input is invalid for conversion to Atom")
			}
		}
		adeType := codec.ADEType(h.Type[:])
		if err != nil {
			break
		}
		var a = Atom{
			name: h.Name[:],
			typ:  adeType,
			data: data,
		}
		a.Value = codec.NewCodec(&a.data, a.typ)

		// add atom to parent.Children, or to atoms list if no parent
		if parent, ok := containers.Peek(); ok {
			parent.atomPtr.AddChild(&a)
		} else {
			atoms = append(atoms, &a)
		}

		// push container onto stack
		if a.typ == codec.CONT {
			endPos := bytesRead + h.Size - headerSize
			containers.Push(cont{&a, endPos})
		}

		// pop fully read containers off stack
		_, err = containers.PopCompleted(bytesRead)
	}
	if err != nil {
		return nil, err
	}
	return
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

func errOddLength(len int, name string) error {
	if name == "" {
		return fmt.Errorf("odd length hex string (length %d)", len)
	}
	return fmt.Errorf(`odd length hex string in atom "%s" (length %d)`, name, len)
}

// ReadAtomsFromHex reads a stream of hex characters that represent the binary
// encodings of a series of Atoms.  It returns a slice of Atom pointers.
// If no atoms are found on input, an empty slice is returned and the error code is nil.
// If invalid input is encountered, a non-nil error is returned.
func ReadAtomsFromHex(r io.Reader) (atoms []*Atom, err error) {
	var buffer []byte

	buffer, err = ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	// Strip newlines, spaces
	var clean = make([]byte, 0, len(buffer))
	for _, b := range buffer {
		if b != '\n' && b != '\r' && b != ' ' {
			clean = append(clean, b)
		}
	}

	// Don't attempt hex conversion without even length at this point
	if 0 != len(clean)%2 {
		return nil, errOddLength(len(clean), string(buffer[4:8]))
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

// BinaryWrite serializes the receiver atom and its children to binary format.
// The result is written as a byte stream into the given Writer argument.
// If the Writer returns an error, processing stops and the error is returned.
func (a *Atom) BinaryWrite(w io.Writer) (err error) {
	// create members for atom header
	var buf []byte
	var name, typ [4]byte
	if err = codec.StringToFC32Bytes(&buf, string(a.typ)); err != nil {
		return
	}
	copy(typ[:], buf)
	copy(name[:], a.name)

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
	for _, child := range a.children {
		err = child.BinaryWrite(w)
		if err != nil {
			return
		}
	}

	return
}

// Len returns the length this atom would have when encoded as binary bytes.
func (a *Atom) Len() (length uint32) {
	length = uint32(headerSize + len(a.data))
	for _, child := range a.children {
		length += child.Len()
	}
	return
}
