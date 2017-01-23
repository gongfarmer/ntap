// Package atom provides encodings for the ADE AtomContainer format.
// It includes conversions between text and binary formats, as well as an
// encoding-independent struct to provide convenient accessors.
// TODO Encoder and Decoder as in encoding/xml, encoding/json
// see encoding/gob/example_test.go for model of how this works
package atom

import (
	"bytes"
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

// var _ encoding.BinaryMarshaler = Atom{} // TODO
// var _ encoding.TextUnmarshaler = Atom{} // TODO
// var _ encoding.TextMarshaler = Atom{} // TODO

// GOAL: make this concurrency-safe, perhaps immutable
type Atom struct {
	Name     string
	Type     string
	Data     []byte
	Children []*Atom
}

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
	output := buildString(&a, 0)
	return output.String()
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
	decoderFunc := decOpTable[adeTypeMap[a.Type]]
	decoderFunc(a.Data, &ptr)
	return ptr
}

// Return a go type suitable for holding a value of the given ADE Atom type
func (a Atom) goType() reflect.Type {
	var p reflect.Value
	switch adeTypeMap[a.Type] {
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

// FIXME Rename because its not a string
func buildString(a *Atom, depth int) bytes.Buffer {
	var (
		output        bytes.Buffer
		printableName string
	)
	// print atom name + type
	if isPrint(a.Name) {
		printableName = a.Name
	} else {
		printableName = fmt.Sprintf("0x%+08X", a.Name)
	}
	fmt.Fprintf(&output, "% *s%s:%s:", depth*4, "", printableName, a.Type)
	if a.hasValue() {
		fmt.Fprintln(&output, a.Value())
	} else {
		fmt.Fprintln(&output)
	}

	// print children
	if a.Type == "CONT" {
		for _, childPtr := range a.Children {
			buf := buildString(childPtr, depth+1)
			output.WriteString(buf.String())
		}
	}
	return output
}

func (a *Atom) hasValue() bool {
	if a.Type == "CONT" || a.Type == "NULL" {
		return false
	}
	return true
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

/*
// from ADE, example of how to construct a new AtomContainer.:
static void FINF_RegisterService(ForwarderInformationInfoPtr theModuleInfo)
{
	CXD_AtomContainerPtr		msgAC = NULL;
	UINT32				rc = 0;

	CXD_Atom_CreateContainer(&msgAC);
	CXD_Atom_SetType(msgAC, CONTAINER_IS_PARENT, EVENT_SERVICE_REGISTER);

	CXD_AtomPath_SetUI32(msgAC, EVENT_SERVICE_REGISTER_VERSION, 1);
	CXD_AtomPath_SetFC32(msgAC, EVENT_SERVICE_REGISTER_SERVICEID,
			FORWARDER_SERVICE_ID);
	CXD_AtomPath_SetUI32(msgAC, EVENT_SERVICE_REGISTER_SERVICEVERSION
			, FORWARDER_SERVICE_VERSION);
	CXD_AtomPath_SetFC32(msgAC, EVENT_SERVICE_REGISTER_SERVICESCOPE,
			FORWARDER_SERVICE_SCOPE);
	CXD_AtomPath_SetCSTR(msgAC, EVENT_SERVICE_REGISTER_SERVICENAME,
			FORWARDER_SERVICE_NAME);
	CXD_AtomPath_SetFC32(msgAC, EVENT_SERVICE_REGISTER_SERVICESTATE,
			SERVICESTATE_ENABLED);
	CXD_AtomPath_SetUI32(msgAC, EVENT_SERVICE_REGISTER_SERVICEPROCESSID,
			ADE_Process_GetPID());

	rc = ADE_Message_PostContainer(theModuleInfo->ServicePID,
			ADE_Process_GetNID(), &msgAC);
	REQUIRES(rc == noErr);

	theModuleInfo->ServiceRegistered = true;

}
*/
/*
Example of bundle attribute structure
GODS:CONT:
    BVER:UI32:1
    BTIM:UI64:1
    GOPT:CONT:
#        "Option"
        AVER:UI32:2
        ATIM:UI64:1
        AVTP:FC32:'CSTR'
        APER:FC32:'READ'
        AVAL:CONT:
            0x00000000:UI32:1
        END
    END
    GOVL:CONT:
#        "Value"
        AVER:UI32:2
        ATIM:UI64:1
        AVTP:FC32:'CSTR'
        APER:FC32:'READ'
        AVAL:CONT:
            0x00000000:UI32:1
        END
    END
END
*/
