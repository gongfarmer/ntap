package atom

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"reflect"
	"strings"
	"unicode"
)

// ADE Data types
// Defined in 112-0002_r4.0B_StorageGRID_Data_Types
// The ADE code type definitions are in OSL_Types.h
const (
	UI01 ADEType = "UI01" // unsigned int / bool
	UI08 ADEType = "UI08" // unsigned int
	UI16 ADEType = "UI16" // unsigned int
	UI32 ADEType = "UI32" // unsigned int
	UI64 ADEType = "UI64" // unsigned int
	SI08 ADEType = "SI08" // signed int
	SI16 ADEType = "SI16" // signed int
	SI32 ADEType = "SI32" // signed int
	SI64 ADEType = "SI64" // signed int
	FP32 ADEType = "FP32" // floating point
	FP64 ADEType = "FP64" // floating point
	UF32 ADEType = "UF32" // unsigned fixed point (integer part / fractional part)
	UF64 ADEType = "UF64" // unsigned fixed point (integer part / fractional part)
	SF32 ADEType = "SF32" // signed fixed point   (integer part / fractional part)
	SF64 ADEType = "SF64" // signed fixed point   (integer part / fractional part)
	UR32 ADEType = "UR32" // unsigned fraction
	UR64 ADEType = "UR64" // unsigned fraction
	SR32 ADEType = "SR32" // signed fraction
	SR64 ADEType = "SR64" // signed fraction
	FC32 ADEType = "FC32" // four char string
	IP32 ADEType = "IP32" // ipv4 address
	IPAD ADEType = "IPAD" // ipv4 or ipv6 address
	CSTR ADEType = "CSTR" // C string
	USTR ADEType = "USTR" // unicode string
	DATA ADEType = "DATA" // Raw data or equivalent
	ENUM ADEType = "ENUM" // Enumeration
	UUID ADEType = "UUID" // UUID
	NULL ADEType = "NULL" // NULL type, must have empty data section
	CNCT ADEType = "CNCT" // binary data printed as hexadecimal value with leading 0x
	Cnct ADEType = "cnct" // alias for CNCT
	CONT ADEType = "CONT" // Atom Container

	String GoType = "String"
	Uint   GoType = "Uint"
	Int    GoType = "Int"
	Bool   GoType = "Bool"
	Bytes  GoType = "Bytes"
	Float  GoType = "Float"

	MaxUint16Plus1 = math.MaxUint16 + 1
	MaxUint32Plus1 = math.MaxUint32 + 1
)

/**********************************************************/

type (
	// decOp is the signature of a decoding operator for a given type.
	// Returns a reflect.Value containing the data within a go type suitable for
	// the data.
	decOp func(buf []byte) reflect.Value

	// decOp is the signature of a function that prints value as a string.
	strOp func(buf []byte) string

	Operators struct {
		Decode decOp
		String strOp
	}
)

var opTable = map[ADEType]Operators{
	UI01: Operators{decUI01, strUI01},
	UI08: Operators{decUI08, strUI08},
	UI16: Operators{decUI16, strUI16},
	UI32: Operators{decUI32, strUI32},
	UI64: Operators{decUI64, strUI64},
	SI08: Operators{decSI08, strSI08},
	SI16: Operators{decSI16, strSI16},
	SI32: Operators{decSI32, strSI32},
	SI64: Operators{decSI64, strSI64},
	FP32: Operators{decFP32, strFP32},
	FP64: Operators{decFP64, strFP64},
	UF32: Operators{decUF32, strUF32},
	UF64: Operators{decUF64, strUF64},
	SF32: Operators{decSF32, strSF32},
	SF64: Operators{decSF64, strSF64},
	UR32: Operators{decUR32, strUR32},
	UR64: Operators{decUR64, strUR64},
	SR32: Operators{decSR32, strSR32},
	SR64: Operators{decSR64, strSR64},
	FC32: Operators{decFC32, strFC32},
	IP32: Operators{decIP32, strIP32},
	IPAD: Operators{decIPAD, strIPAD},
	CSTR: Operators{decCSTR, strCSTR},
	USTR: Operators{decUSTR, strUSTR},
	DATA: Operators{decDATA, strDATA},
	CNCT: Operators{decDATA, strDATA},
	Cnct: Operators{decDATA, strDATA},
	ENUM: Operators{decSI32, strSI32},
	UUID: Operators{decDATA, strUUID},
	NULL: Operators{decNULL, strNULL},
	CONT: Operators{decNULL, strNULL},
}

func noDecoder(from ADEType, to GoType) error {
	return fmt.Errorf("no decoder exists to convert ADE type %s to go type %s.", from, to)
}

type decoder struct {
	String      func(buf []byte) (string, error)
	StringRaw   func(buf []byte) (string, error)
	Bool        func(buf []byte) (bool, error)
	Uint        func(buf []byte) (uint64, error)
	Int         func(buf []byte) (int64, error)
	Float       func(buf []byte) (float64, error)
	SliceOfUint func(buf []byte) ([]uint64, error)
	SliceOfInt  func(buf []byte) ([]int64, error)
	SliceOfByte func(buf []byte) ([]byte, error)
}
type encoder struct {
}

// Zero value of ADE type decoder panics on every type conversion.
func NewDecoder(from ADEType) decoder {
	return decoder{
		String:      func([]byte) (v string, e error) { return v, noDecoder(from, "string") },
		StringRaw:   func([]byte) (v string, e error) { return v, noDecoder(from, "string") },
		Bool:        func([]byte) (v bool, e error) { return v, noDecoder(from, "bool") },
		Uint:        func([]byte) (v uint64, e error) { return v, noDecoder(from, "uint64") },
		Int:         func([]byte) (v int64, e error) { return v, noDecoder(from, "int64") },
		Float:       func([]byte) (v float64, e error) { return v, noDecoder(from, "float64") },
		SliceOfUint: func([]byte) (v []uint64, e error) { return v, noDecoder(from, "[]uint64") },
		SliceOfInt:  func([]byte) (v []int64, e error) { return v, noDecoder(from, "[]int64") },
		SliceOfByte: func([]byte) (v []byte, e error) { return v, noDecoder(from, "[]byte") },
	}
}

var decoderByType = make(map[ADEType]decoder)
var encoderByType = make(map[ADEType]encoder)

type codec struct {
	Atom    *Atom
	Decoder decoder
	Encoder encoder
}

// Given an atom, return a codec that provides type conversion for the atom
// data from the atom's ADE type into Go types, and vice versa.
func NewCodec(a *Atom) *codec {
	c := codec{
		Atom:    a,
		Decoder: decoderByType[a.Type],
		Encoder: encoderByType[a.Type],
	}
	return &c
}

// Decoder methods: pass atom data to the decoder for type conversion to go type
func (c codec) String() (string, error) {
	return c.Decoder.String(c.Atom.data)
}
func (c codec) StringRaw() (string, error)     { return c.Decoder.StringRaw(c.Atom.data) }
func (c codec) Bool() (bool, error)            { return c.Decoder.Bool(c.Atom.data) }
func (c codec) Uint() (uint64, error)          { return c.Decoder.Uint(c.Atom.data) }
func (c codec) Int() (int64, error)            { return c.Decoder.Int(c.Atom.data) }
func (c codec) Float() (float64, error)        { return c.Decoder.Float(c.Atom.data) }
func (c codec) SliceOfUint() ([]uint64, error) { return c.Decoder.SliceOfUint(c.Atom.data) }
func (c codec) SliceOfInt() ([]int64, error)   { return c.Decoder.SliceOfInt(c.Atom.data) }
func (c codec) SliceOfByte() ([]byte, error)   { return c.Atom.data, nil }

// Initialize decoder table, which makes decoders accessible by ADE type.
// Variable 'd' is used for assignment, because Go disallows assigning directly
// to a struct member of a map value.  Example:
//    decoderByType[UI01] = NewDecoder(UI01)
//    decoderByType[UI01].Bool = UI32ToBool //illegal
func init() {
	// ADE unsigned int types
	d := NewDecoder(UI01)
	d.String = UI32ToString
	d.Bool = UI32ToBool
	d.Uint = UI32ToUint64
	decoderByType[UI01] = d

	d = NewDecoder(UI08)
	d.String = UI08ToString
	d.Uint = UI08ToUint64
	decoderByType[UI08] = d

	d = NewDecoder(UI16)
	d.String = UI16ToString
	d.Uint = UI16ToUint64
	decoderByType[UI16] = d

	d = NewDecoder(UI32)
	d.String = UI32ToString
	d.Uint = UI32ToUint64
	decoderByType[UI32] = d

	d = NewDecoder(UI64)
	d.String = UI64ToString
	d.Uint = UI64ToUint64
	decoderByType[UI64] = d

	// ADE signed int types
	d = NewDecoder(SI08)
	d.String = SI08ToString
	d.Int = SI08ToInt64
	decoderByType[SI08] = d

	d = NewDecoder(SI16)
	d.String = SI16ToString
	d.Int = SI16ToInt64
	decoderByType[SI16] = d

	d = NewDecoder(SI32)
	d.String = SI32ToString
	d.Int = SI32ToInt64
	decoderByType[SI32] = d

	d = NewDecoder(SI64)
	d.String = SI64ToString
	d.Int = SI64ToInt64
	decoderByType[SI64] = d

	// ADE floating point types
	d = NewDecoder(FP32)
	d.String = FP32ToString
	d.Float = FP32ToFloat64
	decoderByType[FP32] = d

	d = NewDecoder(FP64)
	d.String = FP64ToString
	d.Float = FP64ToFloat64
	decoderByType[FP64] = d

	// ADE fixed point types
	d = NewDecoder(UF32)
	d.String = UF32ToString
	d.Float = UF32ToFloat64
	decoderByType[UF32] = d

	d = NewDecoder(UF64)
	d.String = UF64ToString
	d.Float = UF64ToFloat64
	decoderByType[UF64] = d

	d = NewDecoder(SF32)
	d.String = SF32ToString
	d.Float = SF32ToFloat64
	decoderByType[SF32] = d

	d = NewDecoder(SF64)
	d.String = SF64ToString
	d.Float = SF64ToFloat64
	decoderByType[SF64] = d

	// ADE fractional types

	d = NewDecoder(UR32)
	d.String = UR32ToString
	d.SliceOfUint = UR32ToSliceOfUint
	decoderByType[UR32] = d

	d = NewDecoder(UR64)
	d.String = UR64ToString
	d.SliceOfUint = UR64ToSliceOfUint
	decoderByType[UR64] = d

	d = NewDecoder(SR32)
	d.String = SR32ToString
	d.SliceOfInt = SR32ToSliceOfInt
	decoderByType[SR32] = d

	d = NewDecoder(SR64)
	d.String = SR64ToString
	d.SliceOfInt = SR64ToSliceOfInt
	decoderByType[SR64] = d

	// ADE Four char code
	d = NewDecoder(FC32)
	d.String = FC32ToString
	decoderByType[FC32] = d

	// ADE ENUM type
	d = NewDecoder(ENUM)
	d.String = SI32ToString
	d.Int = SI32ToInt64
	decoderByType[ENUM] = d

	// ADE UUID type
	d = NewDecoder(UUID)
	d.String = UUIDToString
	decoderByType[UUID] = d

	// IP Address types
	d = NewDecoder(IP32)
	d.String = IP32ToString
	decoderByType[IP32] = d

	d = NewDecoder(IPAD)
	d.String = IPADToString
	decoderByType[IPAD] = d

	// ADE String types
	d = NewDecoder(CSTR)
	d.StringRaw = CSTRToString
	d.String = CSTRToStringEscaped
	decoderByType[CSTR] = d

	d = NewDecoder(USTR)
	d.StringRaw = USTRToString
	d.String = USTRToStringEscaped
	decoderByType[USTR] = d

	// DATA type, and aliases
	d = NewDecoder(DATA)
	d.String = BytesToHexString
	decoderByType[DATA] = d
	decoderByType[CNCT] = d
	decoderByType[Cnct] = d

	// NULL type
	d = NewDecoder(NULL)
	d.String = func([]byte) (s string, e error) { return }
	decoderByType[NULL] = d

	// ADE container
	d = NewDecoder(CONT)
	d.String = func([]byte) (s string, e error) { return }
	decoderByType[CONT] = d
}

// ADE unsigned int types

func UI08ToUint64(buf []byte) (v uint64, e error) {
	return uint64(buf[0]), e
}
func UI16ToUint64(buf []byte) (v uint64, e error) {
	return uint64(binary.BigEndian.Uint16(buf)), e
}
func UI32ToBool(buf []byte) (v bool, e error) {
	ui32 := binary.BigEndian.Uint32(buf)
	if ui32 != 0 && ui32 != 1 {
		e = fmt.Errorf("range error: value %d overflows type bool", v)
		return
	}
	return ui32 == 1, e
}
func UI32ToUint32(buf []byte) (v uint32, e error) {
	return binary.BigEndian.Uint32(buf), e
}
func UI32ToUint64(buf []byte) (v uint64, e error) {
	var ui32 uint32 = binary.BigEndian.Uint32(buf)
	return uint64(ui32), e
}
func UI64ToUint64(buf []byte) (v uint64, e error) {
	return binary.BigEndian.Uint64(buf), e
}
func UI08ToString(buf []byte) (v string, e error) {
	return string(buf[0]), e
}
func UI16ToString(buf []byte) (v string, e error) {
	return fmt.Sprintf("%d", binary.BigEndian.Uint16(buf)), e
}
func UI32ToString(buf []byte) (v string, e error) {
	return fmt.Sprintf("%d", binary.BigEndian.Uint32(buf)), e
}
func UI64ToString(buf []byte) (v string, e error) {
	return fmt.Sprintf("%d", binary.BigEndian.Uint64(buf)), e
}

// ADE signed int types

func SI08ToInt64(buf []byte) (v int64, e error) {
	var i int8
	e = binary.Read(bytes.NewReader(buf), binary.BigEndian, &i)
	return int64(i), e
}
func SI16ToInt64(buf []byte) (v int64, e error) {
	var i int16
	e = binary.Read(bytes.NewReader(buf), binary.BigEndian, &i)
	return int64(i), e
}
func SI32ToInt32(buf []byte) (v int32, e error) {
	var i int32
	e = binary.Read(bytes.NewReader(buf), binary.BigEndian, &i)
	return i, e
}
func SI32ToInt64(buf []byte) (v int64, e error) {
	var i int32
	i, e = SI32ToInt32(buf)
	if e == nil {
		v = int64(i)
	}
	return v, e
}
func SI64ToInt64(buf []byte) (v int64, e error) {
	e = binary.Read(bytes.NewReader(buf), binary.BigEndian, &v)
	return v, e
}
func SI08ToString(buf []byte) (v string, e error) {
	var i int64
	i, e = SI08ToInt64(buf)
	if e != nil {
		return v, e
	}
	return fmt.Sprintf("%d", i), e
}
func SI16ToString(buf []byte) (v string, e error) {
	var i int64
	i, e = SI16ToInt64(buf)
	if e != nil {
		return v, e
	}
	return fmt.Sprintf("%d", i), e
}
func SI32ToString(buf []byte) (v string, e error) {
	var i int64
	i, e = SI32ToInt64(buf)
	if e != nil {
		return v, e
	}
	return fmt.Sprintf("%d", i), e
}
func SI64ToString(buf []byte) (v string, e error) {
	var i int64
	i, e = SI64ToInt64(buf)
	if e != nil {
		return v, e
	}
	return fmt.Sprintf("%d", i), e
}

// ADE floating point types

func FP32ToFloat32(buf []byte) (v float32, e error) {
	var i uint32
	i, e = UI32ToUint32(buf)
	v = math.Float32frombits(i)
	return
}
func FP32ToFloat64(buf []byte) (v float64, e error) {
	var i uint32
	i, e = UI32ToUint32(buf)
	if e == nil {
		v = float64(math.Float32frombits(i))
	}
	return
}
func FP64ToFloat64(buf []byte) (v float64, e error) {
	var i uint64
	i, e = UI64ToUint64(buf)
	if e == nil {
		v = math.Float64frombits(i)
	}
	return
}
func FP32ToString(buf []byte) (v string, e error) {
	var f float64
	f, e = FP32ToFloat64(buf)
	if e == nil {
		v = fmt.Sprintf("%0.8E", f)
	}
	return
}
func FP64ToString(buf []byte) (v string, e error) {
	var f float64
	fmt.Printf("FP64ToString  buffer: % X\n", buf)
	f, e = FP64ToFloat64(buf)
	if e == nil {
		v = fmt.Sprintf("%0.17E", f)
	}
	return
}

// ADE fixed point types, unsigned

func UF32ToFloat64(buf []byte) (v float64, e error) {
	var i uint64
	i, e = UI32ToUint64(buf)
	if e != nil {
		return
	}
	v = float64(i) / 65536.0
	return
}
func UF64ToFloat64(buf []byte) (v float64, e error) {
	var i uint64
	i, e = UI64ToUint64(buf)
	if e != nil {
		return
	}
	v = float64(i) / 4294967296.0
	return
}
func UF32ToString(buf []byte) (v string, e error) {
	var f float64
	f, e = UF32ToFloat64(buf)
	if e == nil {
		v = fmt.Sprintf("%0.4f", f)
	}
	return
}

// ade: CXD_String.cc CXD_String_from_UFIX64(...)
// isolate whole and fractional parts, then combine within the string
func UF64ToString(buf []byte) (v string, e error) {
	var i uint64
	i, e = UI64ToUint64(buf)
	if e == nil {
		iFract := i << 32 >> 32
		fFract := float64(iFract) / 4294967296.0 * math.Pow(10, 9)
		v = fmt.Sprintf("%d.%09.0f", i>>32, fFract)
	}
	return
}

// ADE fixed point types, signed

func SF32ToFloat64(buf []byte) (v float64, e error) {
	var i int32
	i, e = SI32ToInt32(buf)
	if e != nil {
		return
	}
	v = float64(i) / MaxUint16Plus1
	return
}
func SF64ToFloat64(buf []byte) (v float64, e error) {
	var i int64
	i, e = SI64ToInt64(buf)
	if e != nil {
		return
	}
	v = float64(i) / MaxUint32Plus1
	return
}
func SF32ToString(buf []byte) (v string, e error) {
	var f float64
	f, e = SF32ToFloat64(buf)
	if e == nil {
		v = fmt.Sprintf("%0.4f", f)
	}
	return
}
func SF64ToString(buf []byte) (v string, e error) {
	var f float64
	f, e = SF64ToFloat64(buf)
	if e == nil {
		v = fmt.Sprintf("%.9f", f)
	}
	return
}

// ADE fractional types, unsigned

func UR32ToSliceOfUint(buf []byte) (v []uint64, e error) {
	var arr [2]uint16
	e = binary.Read(bytes.NewReader(buf), binary.BigEndian, &arr)
	if e == nil {
		v = append(v, uint64(arr[0]), uint64(arr[1]))
	}
	return
}
func UR64ToSliceOfUint(buf []byte) (v []uint64, e error) {
	var arr [2]uint32
	e = binary.Read(bytes.NewReader(buf), binary.BigEndian, &arr)
	if e == nil {
		v = append(v, uint64(arr[0]), uint64(arr[1]))
	}
	return
}
func UR32ToString(buf []byte) (v string, e error) {
	var arr []uint64
	arr, e = UR32ToSliceOfUint(buf)
	if e == nil {
		v = fmt.Sprintf("%d/%d", arr[0], arr[1])
	}
	return
}
func UR64ToString(buf []byte) (v string, e error) {
	var arr []uint64
	arr, e = UR64ToSliceOfUint(buf)
	if e == nil {
		v = fmt.Sprintf("%d/%d", arr[0], arr[1])
	}
	return
}

// ADE fractional types, signed

func SR32ToSliceOfInt(buf []byte) (v []int64, e error) {
	var arr [2]int16
	e = binary.Read(bytes.NewReader(buf), binary.BigEndian, &arr)
	if e == nil {
		v = append(v, int64(arr[0]), int64(arr[1]))
	}
	return
}
func SR64ToSliceOfInt(buf []byte) (v []int64, e error) {
	var arr [2]int32
	e = binary.Read(bytes.NewReader(buf), binary.BigEndian, &arr)
	if e == nil {
		v = append(v, int64(arr[0]), int64(arr[1]))
	}
	return
}
func SR32ToString(buf []byte) (v string, e error) {
	var arr []int64
	arr, e = SR32ToSliceOfInt(buf)
	if e == nil {
		v = fmt.Sprintf("%d/%d", arr[0], arr[1])
	}
	return
}
func SR64ToString(buf []byte) (v string, e error) {
	var arr []int64
	arr, e = SR64ToSliceOfInt(buf)
	if e == nil {
		v = fmt.Sprintf("%d/%d", arr[0], arr[1])
	}
	return
}

// FC32, Four-char code type
// Mantis #27726: ccat/ctac can't parse container names starting with "#" or " ".
// If string is printable but starts with "# \"'", print it as hex.
func FC32ToString(buf []byte) (v string, e error) {
	var badStartChars = "# \"'"
	if isPrintableBytes(buf) && !strings.ContainsRune(badStartChars, rune(buf[0])) {
		v = fmt.Sprintf("'%s'", string(buf))
	} else {
		v = fmt.Sprintf("0x%08X", buf)
	}
	return
}

func UUIDToString(buf []byte) (v string, e error) {
	var uuid struct {
		TimeLow          uint32
		TimeMid          uint16
		TimeHiAndVersion uint16
		ClkSeqHiRes      uint8
		ClkSeqLow        uint8
		Node             [6]byte
	}
	e = binary.Read(bytes.NewReader(buf), binary.BigEndian, &uuid)
	if e == nil {
		v = fmt.Sprintf(
			"%08X-%04X-%04X-%02X%02X-%012X",
			uuid.TimeLow,
			uuid.TimeMid,
			uuid.TimeHiAndVersion,
			uuid.ClkSeqHiRes,
			uuid.ClkSeqLow,
			uuid.Node)
	}
	return
}

// IP Address types

func IP32ToString(buf []byte) (v string, e error) {
	v = fmt.Sprintf("%d.%d.%d.%d", buf[0], buf[1], buf[2], buf[3])
	return
}

func IPADToString(buf []byte) (v string, e error) {
	v = string(buf[0 : len(buf)-1]) // trim null terminator
	v = fmt.Sprintf("\"%s\"", v)
	return
}

// String types

func CSTRToString(buf []byte) (v string, e error) {
	v = string(buf[0 : len(buf)-1]) // trim null terminator
	return v, nil
}

func CSTRToStringEscaped(buf []byte) (v string, e error) {
	v, e = CSTRToString(buf)
	if e == nil {
		v = fmt.Sprintf("\"%s\"", adeCstrEscape(v))
	}
	return
}

func USTRToString(buf []byte) (v string, e error) {
	var runes = make([]rune, len(buf)/4)
	e = binary.Read(bytes.NewReader(buf), binary.BigEndian, &runes)
	if e == nil {
		v = string(runes)
	}
	return
}

func USTRToStringEscaped(buf []byte) (v string, e error) {
	v, e = USTRToString(buf)
	if e == nil {
		v = fmt.Sprintf("\"%s\"", adeCstrEscape(v))
	}
	return
}

func BytesToHexString(buf []byte) (v string, e error) {
	v = fmt.Sprintf("0x%X", buf)
	return
}

// Random types
/**********************************************************
   decoder methods.
	 Convert atom.Data byte slices into a reflect.Value that wraps
	 a Settable variable with an appropriate underlying go type.
***********************************************************/
// FIXME: assert buffer size before decoding?
func decUI01(buf []byte) reflect.Value {
	var v uint32
	checkError(binary.Read(bytes.NewReader(buf), binary.BigEndian, &v))
	return reflect.ValueOf(v)
}
func decUI08(buf []byte) reflect.Value {
	var v uint8
	checkError(binary.Read(bytes.NewReader(buf), binary.BigEndian, &v))
	return reflect.ValueOf(v)
}
func decUI16(buf []byte) reflect.Value {
	var v uint16
	checkError(binary.Read(bytes.NewReader(buf), binary.BigEndian, &v))
	return reflect.ValueOf(v)
}
func decUI32(buf []byte) reflect.Value {
	var v uint32
	checkError(binary.Read(bytes.NewReader(buf), binary.BigEndian, &v))
	return reflect.ValueOf(v)
}
func decUI64(buf []byte) reflect.Value {
	var v uint64
	checkError(binary.Read(bytes.NewReader(buf), binary.BigEndian, &v))
	return reflect.ValueOf(v)
}
func decSF32(buf []byte) reflect.Value {
	v := float32(decSI32(buf).Interface().(int32)) / MaxUint16Plus1
	return reflect.ValueOf(v)
}
func decSF64(buf []byte) reflect.Value {
	v := float64(decSI64(buf).Interface().(int64)) / MaxUint32Plus1
	return reflect.ValueOf(v)
}
func decSI08(buf []byte) reflect.Value {
	var v int8
	checkError(binary.Read(bytes.NewReader(buf), binary.BigEndian, &v))
	return reflect.ValueOf(v)
}
func decSI16(buf []byte) reflect.Value {
	var v int16
	checkError(binary.Read(bytes.NewReader(buf), binary.BigEndian, &v))
	return reflect.ValueOf(v)
}
func decSI32(buf []byte) reflect.Value {
	var v int32
	checkError(binary.Read(bytes.NewReader(buf), binary.BigEndian, &v))
	return reflect.ValueOf(v)
}
func decSI64(buf []byte) reflect.Value {
	var v int64
	checkError(binary.Read(bytes.NewReader(buf), binary.BigEndian, &v))
	return reflect.ValueOf(v)
}
func decFP32(buf []byte) reflect.Value {
	var ui32 uint32 = uint32(decUI32(buf).Uint())
	var v float32 = math.Float32frombits(ui32)
	return reflect.ValueOf(v)
}
func decFP64(buf []byte) reflect.Value {
	var ui64 = decUI64(buf).Uint()
	var v float64 = math.Float64frombits(ui64)
	return reflect.ValueOf(v)
}

// Fixed-point values are stored as integer types. OSL_Types.h:
/*
#define                     UFIX32Type      'UF32'
typedef UINT32              UFIX32;
typedef UINT32Ptr           UFIX32Ptr;

#define                     SFIX32Type      'SF32'
typedef SINT32              SFIX32;
typedef SINT32Ptr           SFIX32Ptr;

#define                     UFIX64Type      'UF64'
typedef UINT64              UFIX64;
typedef UINT64Ptr           UFIX64Ptr;

#define                     SFIX64Type      'SF64'
typedef SINT64              SFIX64;
typedef SINT64Ptr           SFIX64Ptr;
*/
// Returns a 64 bit float despite being stored in 32 bits.
// This is intentional, this type can store some values that are outside the
// range of SF32.
func decUF32(buf []byte) reflect.Value {
	var v = float64(decUI32(buf).Uint()) / 65536.0
	return reflect.ValueOf(v)
}

// Precision is higher than ADE ccat here.
// It needs rounding to match ccat exactly.
func decUF64(buf []byte) reflect.Value {
	var i uint64
	checkError(binary.Read(bytes.NewReader(buf), binary.BigEndian, &i))
	var v = float64(i) / 4294967296.0
	return reflect.ValueOf(v)
}
func decUR32(buf []byte) reflect.Value {
	var arr [2]uint16
	checkError(binary.Read(bytes.NewReader(buf), binary.BigEndian, &arr))
	return reflect.ValueOf(arr)
}
func decUR64(buf []byte) reflect.Value {
	var arr [2]uint32
	checkError(binary.Read(bytes.NewReader(buf), binary.BigEndian, &arr))
	return reflect.ValueOf(arr)
}

// ADE ccat puts the sign on the denominator
func decSR32(buf []byte) reflect.Value {
	var arr [2]int16
	checkError(binary.Read(bytes.NewReader(buf), binary.BigEndian, &arr))
	return reflect.ValueOf(arr)
}
func decSR64(buf []byte) reflect.Value {
	var arr [2]int32
	checkError(binary.Read(bytes.NewReader(buf), binary.BigEndian, &arr))
	return reflect.ValueOf(arr)
}
func decFC32(buf []byte) reflect.Value {
	return decUI32(buf)
}

// Decode to [4]byte, same way IPv4 is represented in Go's net/ library
func decIP32(buf []byte) reflect.Value {
	return reflect.ValueOf(buf)
}

// Decode to string, because this could be IPv4 or IPv6.
func decIPAD(buf []byte) reflect.Value {
	trimmed := buf[0 : len(buf)-1]
	s := string(trimmed)
	return reflect.ValueOf(s)
}
func decCSTR(buf []byte) reflect.Value {
	s := string(buf)
	return reflect.ValueOf(s)
}
func decUSTR(buf []byte) reflect.Value {
	var runes = make([]rune, len(buf)/4)
	checkError(binary.Read(bytes.NewReader(buf), binary.BigEndian, &runes))
	return reflect.ValueOf(string(runes))
}
func decDATA(buf []byte) reflect.Value {
	return reflect.ValueOf(buf)
}
func decNULL(_ []byte) reflect.Value {
	return reflect.ValueOf(nil)
}

/**********************************************************
 string methods.
 Convert atom.Data byte slices into a string conforming to
 ADE formatting rules in doc 112-0002, "StorageGRID Data Types".
***********************************************************/

func strUI01(buf []byte) string {
	return fmt.Sprint(binary.BigEndian.Uint32(buf))
}
func strUI08(buf []byte) string {
	return fmt.Sprint(buf[0])
}
func strUI16(buf []byte) string {
	return fmt.Sprint(binary.BigEndian.Uint16(buf))
}
func strUI32(buf []byte) string {
	return fmt.Sprint(binary.BigEndian.Uint32(buf))
}
func strUI64(buf []byte) string {
	return fmt.Sprint(binary.BigEndian.Uint64(buf))
}
func strSI08(buf []byte) string {
	return fmt.Sprint(int8(buf[0]))
}
func strSI16(buf []byte) string {
	return fmt.Sprint(decSI16(buf))
}
func strSI32(buf []byte) string {
	return fmt.Sprint(decSI32(buf))
}
func strSI64(buf []byte) string {
	return fmt.Sprint(decSI64(buf))
}
func strFP32(buf []byte) string {
	return fmt.Sprintf("%0.8E", decFP32(buf).Float())
}
func strFP64(buf []byte) string {
	return fmt.Sprintf("%0.17E", decFP64(buf).Float())
}
func strUF32(buf []byte) string {
	return fmt.Sprintf("%0.4f", decUF32(buf).Float())
}

// ade: CXD_String.cc CXD_String_from_UFIX64(...)
// isolate whole and fractional parts, then combine within the string
func strUF64(buf []byte) string {
	iValue := decUI64(buf).Uint()
	var iFract uint64 = iValue << 32 >> 32
	fFract := float64(iFract) / 4294967296.0 * math.Pow(10, 9)
	return fmt.Sprintf("%d.%09.0f", iValue>>32, fFract)
}
func strSF32(buf []byte) string {
	return fmt.Sprintf("%0.4f", decSF32(buf).Float())
}
func strSF64(buf []byte) string {
	return fmt.Sprintf("%.9f", decSF64(buf).Float())
}
func strUR32(buf []byte) string {
	arr := decUR32(buf).Interface().([2]uint16)
	return fmt.Sprintf("%d/%d", arr[0], arr[1])
}
func strUR64(buf []byte) string {
	arr := decUR64(buf).Interface().([2]uint32)
	return fmt.Sprintf("%d/%d", arr[0], arr[1])
}
func strSR32(buf []byte) string {
	arr := decSR32(buf).Interface().([2]int16)
	return fmt.Sprintf("%d/%d", arr[0], arr[1])
}
func strSR64(buf []byte) string {
	arr := decSR64(buf).Interface().([2]int32)
	return fmt.Sprintf("%d/%d", arr[0], arr[1])
}

// Mantis #27726: ccat/ctac can't parse container names starting with "#" or " ".
// If string is printable but starts with "# \"'", print it as hex.
func strFC32(buf []byte) string {
	var badStartChars = "# \"'"
	if isPrintableBytes(buf) && !strings.ContainsRune(badStartChars, rune(buf[0])) {
		return fmt.Sprintf("'%s'", string(buf))
	} else {
		return fmt.Sprintf("0x%08X", buf)
	}
}
func strIP32(buf []byte) string {
	return fmt.Sprintf("%d.%d.%d.%d", buf[0], buf[1], buf[2], buf[3])
}
func strIPAD(buf []byte) string {
	return fmt.Sprintf("\"%s\"", decIPAD(buf).String())
}
func strCSTR(buf []byte) string {
	trimmed := buf[0 : len(buf)-1]
	return fmt.Sprintf("%q", string(trimmed))
}
func strUSTR(buf []byte) string {
	return fmt.Sprintf("\"%s\"", adeCstrEscape(decUSTR(buf).String()))
}
func strDATA(buf []byte) string {
	return fmt.Sprintf("0x%X", buf)
}

// UUID - 128 bit
// variant must be RFC4122/DCE (10b==2d)
// high 2 bits of octet 8 are variant as per RFC
// version must be one of the five defined in the RFC (1d-5d)
// high 4 bits of octet 6 are version as per RFC
// UUID_NULL_STRING "00000000-0000-0000-0000-000000000000"
func strUUID(buf []byte) string {
	var v struct {
		TimeLow          uint32
		TimeMid          uint16
		TimeHiAndVersion uint16
		ClkSeqHiRes      uint8
		ClkSeqLow        uint8
		Node             [6]byte
	}
	checkError(binary.Read(bytes.NewReader(buf), binary.BigEndian, &v))
	return fmt.Sprintf(
		"%08X-%04X-%04X-%02X%02X-%012X",
		v.TimeLow,
		v.TimeMid,
		v.TimeHiAndVersion,
		v.ClkSeqHiRes, v.ClkSeqLow,
		v.Node)
}

func strNULL(_ []byte) string {
	return ""
}

/**********************************************************/

func asPrintableString(buf []byte) string {
	if isPrintableBytes(buf) {
		return string(buf[:])
	} else {
		i := decUI32(buf).Uint()
		return fmt.Sprintf("0x%08X", i)
	}
}

// Called on a container, create atom at the given path if not exist, and set to given value
// FIXME is it useful to write this?
func (a Atom) SetUI32(path string, value uint32) (err error) {
	return
}

// FIXME: replace this massive func with this:
//  v := // (somehow convert user-given interface{} to Value)
//  atom.value.Set(v) // this has all the sanity/bounds-checking built in!
func (a Atom) SetValue(adeType ADEType, value interface{}) (err error) {
	return
}

// ade: libs/osl/OSL_Types.cc CStr_Escape()
func adeCstrEscape(s string) string {
	output := make([]rune, 0, len(s))
	for _, r := range s {
		charsToEscape := "\\\"\x7f"
		if r == '\n' {
			output = append(output, '\\', 'n')
			continue
		} else if r == '\r' {
			output = append(output, '\\', 'r')
			continue
		} else if r == '\\' {
			output = append(output, '\\', '\\')
			continue
		} else if r == '"' {
			output = append(output, '\\', '"')
			continue
		} else if strings.ContainsRune(charsToEscape, r) || r <= rune(0x1f) {
			output = append(output, []rune(fmt.Sprintf("\\x%02X", r))...)
			continue
		} else if !unicode.IsPrint(r) {
			output = append(output, []rune(fmt.Sprintf("%q", r))...)
			continue
		}
		output = append(output, r)
	}
	return string(output)
}

func Round(f float64) float64 {
	return math.Floor(f + .5)
}
func RoundPlaces(f float64, places int) float64 {
	shift := math.Pow(10, float64(places))
	return Round(f*shift) / shift
}
