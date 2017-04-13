package atom_test

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"

	"github.com/gongfarmer/ntap/encoding/atom"
	"github.com/gongfarmer/ntap/encoding/atom/codec"
)

func ExampleAtom_UnmarshalText() {
	var atomText = []byte(`
	TEST:CONT:
	BVER:UI32:6
	END
	`)
	var a atom.Atom
	if err := a.UnmarshalText(atomText); err != nil {
		panic(err)
	}

	fmt.Println(a.Name(), a.Type())
	// Output: TEST CONT
}

func ExampleAtom_MarshalBinary() {
	var atomText = []byte(`
	TEST:CONT:
	BVER:UI32:6
	END
	`)

	var a atom.Atom
	a.UnmarshalText(atomText)

	var buf []byte
	var e error
	if buf, e = a.MarshalBinary(); e != nil {
		panic(e)
	}

	fmt.Printf("Number of bytes: %d.  UINT32 value of first 4 bytes: %d.", len(buf), binary.BigEndian.Uint32(buf[0:4]))
	// Output: Number of bytes: 28.  UINT32 value of first 4 bytes: 28.
}

func ExampleNewAtom() {
	var a *atom.Atom
	a, _ = atom.NewAtom("BVER", codec.UI32)
	a.Value.SetUint(777)

	fmt.Print(a)
	// Output: BVER:UI32:777
}

func ExampleAtom_AddChild() {
	var a *atom.Atom
	a, _ = atom.NewAtom("ROOT", codec.CONT)
	for i := 0; i < 10; i++ {
		c, _ := atom.NewAtom("CHLD", "SI32")
		c.Value.SetInt(int64(i))
		a.AddChild(c)
	}

	text, _ := a.MarshalText()
	fmt.Println(string(text))
	// Output: ROOT:CONT:
	// 	CHLD:SI32:0
	// 	CHLD:SI32:1
	// 	CHLD:SI32:2
	// 	CHLD:SI32:3
	// 	CHLD:SI32:4
	// 	CHLD:SI32:5
	// 	CHLD:SI32:6
	// 	CHLD:SI32:7
	// 	CHLD:SI32:8
	// 	CHLD:SI32:9
	// END
}

func ExampleAtom_Children() {
	var a *atom.Atom
	a, _ = atom.NewAtom("ROOT", codec.CONT)
	for i := 0; i < 10; i++ {
		c, _ := atom.NewAtom("CHLD", "SI32")
		c.Value.SetInt(int64(i))
		a.AddChild(c)
	}

	for _, c := range a.Children() {
		fmt.Println(c)
	}
	// Output: CHLD:SI32:0
	// CHLD:SI32:1
	// CHLD:SI32:2
	// CHLD:SI32:3
	// CHLD:SI32:4
	// CHLD:SI32:5
	// CHLD:SI32:6
	// CHLD:SI32:7
	// CHLD:SI32:8
	// CHLD:SI32:9
}

func ExampleAtom_Descendants() {
	var TEXT = `
ROOT:CONT:
  ONE_:CONT:
		DOGS:UI32:1
		DOGC:CONT:
			CHOW:UI32:3
		END
		DOGS:UI32:2
	END
  TWO_:CONT:
		CATS:UI32:2
	END
  THRE:CONT:
		PIGS:UI32:2
	END
END
`

	var root atom.Atom
	root.UnmarshalText([]byte(TEXT))
	for _, a := range root.Descendants() {
		fmt.Println(a)
	}
	// Output: ROOT:CONT:
	// ONE_:CONT:
	// DOGS:UI32:1
	// DOGC:CONT:
	// CHOW:UI32:3
	// DOGS:UI32:2
	// TWO_:CONT:
	// CATS:UI32:2
	// THRE:CONT:
	// PIGS:UI32:2
}

func ExampleAtom_Name() {
	a, e := atom.NewAtom("HELO", codec.CONT)
	if e != nil {
		panic(e)
	}
	fmt.Println(a.Name())

	a, e = atom.NewAtom("0x0000FFFF", codec.CONT)
	if e != nil {
		panic(e)
	}
	fmt.Println(a.Name())
	// Output: HELO
	// 0x0000FFFF
}

func ExampleAtom_NameAsUint32() {
	a, e := atom.NewAtom("0x0000FFFF", codec.CONT)
	if e != nil {
		panic(e)
	}
	fmt.Printf("type %T, value %[1]d", a.NameAsUint32())
	// Output: type uint32, value 65535
}

func ExampleReadAtomsFromHex() {
	buffer := []byte("0x0000000C534D414C434F4E54")
	atoms, err := atom.ReadAtomsFromHex(bytes.NewReader(buffer))
	if err != nil {
		panic(err)
	}
	text, _ := atoms[0].MarshalText()
	fmt.Println(string(text))

	// Output: SMAL:CONT:
	// END
}

func ExampleReadAtomsFromBinary() {
	// create binary bytes containing a small AtomContainer
	src := []byte("0000000C534D414C434F4E54")
	bin := make([]byte, hex.DecodedLen(len(src)))
	_, err := hex.Decode(bin, src)
	if err != nil {
		log.Fatal(err)
	}

	// read atoms from binary
	atoms, err := atom.ReadAtomsFromBinary(bytes.NewReader(bin))
	if err != nil {
		panic(err)
	}
	text, _ := atoms[0].MarshalText()
	fmt.Println(string(text))

	// Output: SMAL:CONT:
	// END
}
