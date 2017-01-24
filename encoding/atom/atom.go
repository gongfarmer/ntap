// Package atom provides encodings for the ADE AtomContainer format.
// It includes conversions between text and binary formats, as well as an
// encoding-independent struct to provide convenient accessors.
// TODO Encoder and Decoder as in encoding/xml, encoding/json
// see encoding/gob/example_test.go for model of how this works
package atom

import (
	"encoding"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"unicode"
)

// Verify that atom meets encoding interfaces at compile time
var _ encoding.BinaryUnmarshaler = &(Atom{})

// var _ encoding.BinaryMarshaler = Atom{}
// var _ encoding.TextUnmarshaler = Atom{}
var _ encoding.TextMarshaler = &(Atom{})

// GOAL: make this concurrency-safe, perhaps immutable

type Atom struct {
	Name     string
	Type     ADEType
	Data     []byte
	Children []*Atom
}
type ADEType string

// Return true if string is printable, false otherwise
func isPrint(s string) bool {
	for _, r := range s {
		if !unicode.IsPrint(r) {
			return false
		}
	}
	return true
}

func (a Atom) String() string {
	buf, err := a.MarshalText()
	if err != nil {
		panic(fmt.Errorf("Failed to write Atom '%s:%s' to text: %s", a.Name, a.Type, err))
	}
	return string(buf)
}

func (a Atom) Value() reflect.Value {
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("while handling atom %s:%s, %s", a.Name, a.Type, r)
			panic(err)
		}
	}()
	if !a.hasValue() {
		panic(fmt.Errorf("Atom type %s has no value", a.Type))
	}
	var ptr reflect.Value = reflect.New(a.goType().Elem())
	opTable[a.Type].Decode(a.Data, &ptr)
	return ptr
}

func (a Atom) ValueString() string {
	return opTable[a.Type].String(a.Data)
}

// Return a go type suitable for holding a value of the given ADE Atom type
func (a Atom) goType() reflect.Type {
	var p reflect.Value
	switch a.Type {
	case UI01:
		p = reflect.ValueOf(new(bool))
	case UI08, UI16, UI32, UI64:
		p = reflect.ValueOf(new(uint64))
	case SI08, SI16, SI32, SI64:
		p = reflect.ValueOf(new(int64))
	case FP32, FP64, UF32, UF64, SF32, SF64:
		p = reflect.ValueOf(new(float64)) // FIXME signed float64 has smaller max value than UF64
	case UR32, UR64:
		p = reflect.ValueOf(new([2]uint64))
	case SR32, SR64:
		p = reflect.ValueOf(new([2]int64))
	case FC32, IP32, IPAD, CSTR, USTR, UUID:
		p = reflect.ValueOf(new(string)) // FIXME signed float64 has smaller max value than UF64
	case DATA, CNCT:
		p = reflect.ValueOf(a.Data) // FIXME signed float64 has smaller max value than UF64
	case ENUM:
		p = reflect.ValueOf(new(int32))
	default:
		panic(fmt.Errorf("Don't know how to handle type %s", a.Type))
	}
	return p.Type()
}

func (c *Atom) addChild(a *Atom) {
	if c.Type == "CONT" {
		c.Children = append(c.Children, a)
	} else {
		panic(fmt.Errorf("Cannot add child to non-CONT atom %s:%s", c.Name, c.Type))
	}
}

func FromFile(path string) (a Atom, err error) {
	var (
		buf []byte
	)

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

	err = a.UnmarshalBinary(buf)
	return
}
