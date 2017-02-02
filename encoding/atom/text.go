package atom

import (
	"bufio"
	"bytes"
	"fmt"
)

// Enable reading and writing of text format ADE AtomContainers by fulfilling
// these interfaces from stdlib encoding/:

// TextMarshaler is the interface implemented by an object that can marshal
// itself into a textual form.
//
// MarshalText encodes the receiver into UTF-8-encoded text and returns the result.
//
//     type TextMarshaler interface {
//         MarshalText() (text []byte, err error)
//     }

// TextUnmarshaler is the interface implemented by an object that can unmarshal
// a textual representation of itself.
//
// UnmarshalText must be able to decode the form generated by MarshalText.
// UnmarshalText must copy the text if it wishes to retain the text after
// returning.
//
//     type TextUnmarshaler interface {
//     	   UnmarshalText(text []byte) error
//     }

// Write Atom object to a byte slice in ADE ContainerText format.
func (a *Atom) MarshalText() (text []byte, err error) {
	buf := atomToTextBuffer(a, 0)
	return buf.Bytes(), err
}

func atomToTextBuffer(a *Atom, depth int) bytes.Buffer {
	var output bytes.Buffer
	// print atom name,type,data
	fmt.Printf("% *s%s:%s\n", depth*4, "", a.Name, a.Type)
	fmt.Fprintf(&output, "% *s%s:%s:", depth*4, "", a.Name, a.Type)
	s, err := a.Value.String()
	if err != nil {
		panic(fmt.Errorf("conversion of atom to text failed: %s", err))
	}
	fmt.Fprintln(&output, s)

	// print children
	if a.Type == CONT {
		for _, childPtr := range a.Children {
			buf := atomToTextBuffer(childPtr, depth+1)
			output.Write(buf.Bytes())
		}
		fmt.Fprintf(&output, "% *sEND\n", depth*4, "")
	}

	return output
}

// UnmarshalText gets called on a zero-value Atom reciever, and populates it
// based on the contents of the argument string, which contains an ADE
// ContainerText reprentation with a single top-level CONT atom.
// "#" comments are not allowed within this text string.
func (a *Atom) UnMarshalText(input []byte) error {
	var err error
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line = scanner.Text()
	}
	err := Scanner.Err()
	return err
}
