package ade_test

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"

	"github.com/gongfarmer/ntap/encoding/ade"
	"github.com/gongfarmer/ntap/encoding/ade/codec"
)

func ExampleAtom() {
	var atomText = []byte(`
		TEST:CONT:
			BVER:UI32:6
		END
	`)

	// Make an Atom from text
	var a ade.Atom
	if err := a.UnmarshalText(atomText); err != nil {
		panic(err)
	}

	// Print child atom as string
	results, _ := a.AtomsAtPath("/TEST/BVER")
	fmt.Println(results[0])

	// Get child atom value, specifying desired go type (see Codec methods for more)
	value, _ := results[0].Value.Uint()
	fmt.Println(value)
	// Output: BVER:UI32:6
	// 6
}

func ExampleAtom_UnmarshalText() {
	var atomText = []byte(`
		TEST:CONT:
			BVER:UI32:6
		END
	`)
	var a ade.Atom
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

	var a ade.Atom
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
	var a *ade.Atom
	a, _ = ade.NewAtom("BVER", codec.UI32, 777)
	fmt.Println(a)

	a, _ = ade.NewAtom("cont", codec.CONT, nil)
	fmt.Println(a)
	// Output: BVER:UI32:777
	// cont:CONT:
}

func ExampleAtom_AddChild() {
	var a *ade.Atom
	a, _ = ade.NewAtom("ROOT", codec.CONT, nil)
	for i := 0; i < 10; i++ {
		c, _ := ade.NewAtom("CHLD", "SI32", i)
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
	var a *ade.Atom
	a, _ = ade.NewAtom("ROOT", codec.CONT, nil)
	for i := 0; i < 10; i++ {
		c, e := ade.NewAtom("CHLD", "SI32", i)
		if e != nil {
			fmt.Println(e)
		}
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

	var root ade.Atom
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
	a, e := ade.NewAtom("HELO", codec.CONT, nil)
	if e != nil {
		panic(e)
	}
	fmt.Println(a.Name())

	a, e = ade.NewAtom("0x0000FFFF", codec.CONT, nil)
	if e != nil {
		panic(e)
	}
	fmt.Println(a.Name())
	// Output: HELO
	// 0x0000FFFF
}

func ExampleAtom_NameAsUint32() {
	a, e := ade.NewAtom("0x0000FFFF", codec.CONT, nil)
	if e != nil {
		panic(e)
	}
	fmt.Printf("type %T, value %[1]d", a.NameAsUint32())
	// Output: type uint32, value 65535
}

func ExampleReadAtomsFromHex() {
	buffer := []byte("0x0000000C534D414C434F4E54")
	atoms, err := ade.ReadAtomsFromHex(bytes.NewReader(buffer))
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
	atoms, err := ade.ReadAtomsFromBinary(bytes.NewReader(bin))
	if err != nil {
		panic(err)
	}
	text, _ := atoms[0].MarshalText()
	fmt.Println(string(text))

	// Output: SMAL:CONT:
	// END
}

func ExampleAtomPath() {
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
	var root ade.Atom
	root.UnmarshalText([]byte(TEXT))

	// get child atoms of root
	results, _ := root.AtomsAtPath("/ROOT/*")
	fmt.Println(results)

	// get the atom at this nested path
	results, _ = root.AtomsAtPath("/ROOT/ONE_/DOGC/CHOW")
	fmt.Println(results)

	// get all atoms at any level whose data is numeric and greater than 1
	results, _ = root.AtomsAtPath("//*[data() > 1]")
	fmt.Println(results)

	// Output: [ONE_:CONT: TWO_:CONT: THRE:CONT:]
	// [CHOW:UI32:3]
	// [CHOW:UI32:3 DOGS:UI32:2 CATS:UI32:2 PIGS:UI32:2]
}

func ExampleAtom_SetValue() {
	a, _ := ade.NewAtom("BVER", codec.UI32, 1)

	// set UINT32 value
	a.SetValue(6)
	fmt.Println(a)

	// set UFRA64 value
	a, _ = ade.NewAtom("FRAC", codec.UR64, nil)
	a.SetValue("12/144") // string is valid for setting any ADE type
	fmt.Println(a)

	// attempt ot set value on a Container
	a, _ = ade.NewAtom("GINF", codec.CONT, nil)
	err := a.SetValue(5) // illegal: can't set value on a Container
	fmt.Println(err)

	// Output: BVER:UI32:6
	// FRAC:UR64:12/144
	// cannot use go type 'int64' for ADE data type 'CONT'
}

func ExampleAtom_Type() {

	a, _ := ade.NewAtom("BVER", codec.UI32, 5)
	fmt.Println("Type of BVER is", a.Type())

	// set UFRA64 value
	a, _ = ade.NewAtom("FRAC", codec.UR64, "12/144")
	fmt.Println("Type of FRAC is", a.Type())

	// attempt ot set value on a Container
	a, _ = ade.NewAtom("GINF", codec.CONT, nil)
	fmt.Println("Type of GINF is", a.Type())

	// Output: Type of BVER is UI32
	// Type of FRAC is UR64
	// Type of GINF is CONT
}

func ExampleAtom_String() {

	a, _ := ade.NewAtom("BVER", codec.UI32, 5)
	fmt.Println(a.String())

	// set UFRA64 value
	a, _ = ade.NewAtom("FRAC", codec.UR64, "12/144")
	fmt.Println(a.String())

	// attempt ot set value on a Container
	a, _ = ade.NewAtom("GINF", codec.CONT, nil)
	fmt.Println(a.String())

	// Output: BVER:UI32:5
	// FRAC:UR64:12/144
	// GINF:CONT:
}

func ExampleAtom_AtomsAtPath() {
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
	var root ade.Atom
	root.UnmarshalText([]byte(TEXT))

	// get all atoms at any level, that are named PIGS
	results, _ := root.AtomsAtPath("//PIGS")
	fmt.Println(results)

	// get all atom children of ROOT/ONE_/ that have type UI32
	results, _ = root.AtomsAtPath("/ROOT/ONE_/*[@type = UI32]")
	fmt.Println(results)

	// get the ONE_ atom, only if it has a child named DOGS with an odd-numbered value]
	results, _ = root.AtomsAtPath("/ROOT/ONE_[DOGS mod 2 = 1]")
	fmt.Println(results)

	// More practical: Get all rows from NENT, except index row
	results, _ = root.AtomsAtPath("/NENT/*/AVAL/*[name() > 1")

	// Output: [PIGS:UI32:2]
	// [DOGS:UI32:1 DOGS:UI32:2]
	// [ONE_:CONT:]
}

func ExampleNewAtomPath() {
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
	var root ade.Atom
	root.UnmarshalText([]byte(TEXT))

	// get all atoms at any level, that are named PIGS
	ap, _ := ade.NewAtomPath("//PIGS")
	results, _ := ap.GetAtoms(&root)
	fmt.Println(results)

	// get all atom children of ROOT/ONE_/ that have type UI32
	ap, _ = ade.NewAtomPath("/ROOT/ONE_/*[@type = UI32]")
	results, _ = ap.GetAtoms(&root)
	fmt.Println(results)

	// get the ONE_ atom, only if it has a child named DOGS with an odd-numbered value]
	ap, _ = ade.NewAtomPath("/ROOT/ONE_[DOGS mod 2 = 1]")
	results, _ = ap.GetAtoms(&root)
	fmt.Println(results)

	// More practical: Get all rows from NENT, except index row
	results, _ = root.AtomsAtPath("/NENT/*/AVAL/*[name() > 1")

	// Output: [PIGS:UI32:2]
	// [DOGS:UI32:1 DOGS:UI32:2]
	// [ONE_:CONT:]
}
