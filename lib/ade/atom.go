package ade

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"os"
)

type Container struct {
	Name     string
	Size     uint32
	Children []Atom
}

type Atom struct {
	Name []byte
	Type []byte
	Data []byte
	Size uint32
}

func FromBytes(buf []byte) (atoms []Atom) {
	var (
		s uint32 // size
		n []byte // name
		t []byte // type
	)

	for i := 0; ; {
		s = binary.BigEndian.Uint32(buf[i : i+4])
		n = buf[i+5 : i+9]
		t = buf[i+10 : i+14]
		fmt.Printf("got atom %s:%s (size %d)\n", n, t, s)
		i += 14

		atom := Atom{Name: n, Type: t, Size: s}
		atoms = append(atoms, atom)
	}

	return atoms
}

func (c Container) String() string {
	output := bytes.NewBufferString(fmt.Sprintf("%s:CONT\n", c.Name))
	for _, a := range c.Children {
		output.WriteString(a.String())
	}
	return output.String()
}

func (a Atom) String() string {
	return fmt.Sprintf("%s:%s:%s", a.Name, a.Type, a.Data)
}

func FromFile(path string) (atom Atom, err error) {
	var buf []byte

	fstat, err := os.Stat(path)
	if err != nil {
		return
	}

	buf, err = ioutil.ReadFile(path)
	if err != nil {
		return
	}
	var encoded_size int64 = int64(binary.BigEndian.Uint32(buf[0:4]))
	if encoded_size != fstat.Size() {
		fmt.Errorf("Invalid AtomContainer file (encoded size does not match filesize)")
	}

	atom = FromBytes(buf)[0]
	return
}

// func fromFile(path string)
// func fromFile(fh *os.File)
