package atom

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"strconv"
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
	// FIXME: switch these to interfaces?
	decoder struct {
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
	encoder struct {
		SetString      func(*Atom, string) error
		SetBool        func(*Atom, bool) error
		SetUint        func(*Atom, uint64) error
		SetInt         func(*Atom, int64) error
		SetFloat       func(*Atom, float64) error
		SetSliceOfUint func(*Atom, []uint64) error
		SetSliceOfInt  func(*Atom, []int64) error
		SetSliceOfByte func(*Atom, []byte) error
	}
	codec struct {
		Atom    *Atom
		Decoder decoder
		Encoder encoder
		Writer  io.Writer // writes bytes directly to Atom.data
	}
)

// NewCodec returns a codec that performs type conversion for atom data.
// It provides methods to convert data from an atom's ADE type into suitable Go
// types, and vice versa.
func NewCodec(a *Atom) *codec {
	c := codec{
		Atom:    a,
		Decoder: decoderByType[a.Type()],
		Encoder: encoderByType[a.Type()],
		Writer:  bytes.NewBuffer(a.data), // allows writing directly to a.data, not a copy
	}
	return &c
}

func NewEncoder(i interface{}) encoder {
	return encoder{
		SetString:      func(a *Atom, v string) (e error) { return noEncoder(a, "string") },
		SetBool:        func(a *Atom, v bool) (e error) { return noEncoder(a, "bool") },
		SetUint:        func(a *Atom, v uint64) (e error) { return noEncoder(a, "uint64") },
		SetInt:         func(a *Atom, v int64) (e error) { return noEncoder(a, "int64") },
		SetFloat:       func(a *Atom, v float64) (e error) { return noEncoder(a, "float64") },
		SetSliceOfUint: func(a *Atom, v []uint64) (e error) { return noEncoder(a, "[]uint64") },
		SetSliceOfInt:  func(a *Atom, v []int64) (e error) { return noEncoder(a, "[]int64") },
		SetSliceOfByte: func(a *Atom, v []byte) (e error) { return noEncoder(a, "[]byte") },
	}
}

// NewDecoder returns a decoder which has every type conversion set to panic.
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
	}
}

var decoderByType = make(map[ADEType]decoder)
var encoderByType = make(map[ADEType]encoder)

func noEncoder(a *Atom, goType interface{}) error {
	panic(fmt.Errorf("no encoder exists to convert go type '%s' to ADE type '%s'.", goType, a.typ))
}
func noDecoder(from ADEType, to GoType) error {
	return fmt.Errorf("no decoder exists to convert ADE type '%s' to go type '%s'.", from, to)
}

// Decoder methods: pass atom data to the decoder for type conversion to go type
func (c codec) String() (string, error)        { return c.Decoder.String(c.Atom.data) }
func (c codec) StringRaw() (string, error)     { return c.Decoder.StringRaw(c.Atom.data) }
func (c codec) Bool() (bool, error)            { return c.Decoder.Bool(c.Atom.data) }
func (c codec) Uint() (uint64, error)          { return c.Decoder.Uint(c.Atom.data) }
func (c codec) Int() (int64, error)            { return c.Decoder.Int(c.Atom.data) }
func (c codec) Float() (float64, error)        { return c.Decoder.Float(c.Atom.data) }
func (c codec) SliceOfUint() ([]uint64, error) { return c.Decoder.SliceOfUint(c.Atom.data) }
func (c codec) SliceOfInt() ([]int64, error)   { return c.Decoder.SliceOfInt(c.Atom.data) }
func (c codec) SliceOfByte() ([]byte, error)   { return c.Atom.data, nil }

// Encoder methods: convert given data type to []byte and store in ATom
func (c codec) SetString(v string) error        { return c.Encoder.SetString(c.Atom, v) }
func (c codec) SetBool(v bool) error            { return c.Encoder.SetBool(c.Atom, v) }
func (c codec) SetUint(v uint64) error          { return c.Encoder.SetUint(c.Atom, v) }
func (c codec) SetInt(v int64) error            { return c.Encoder.SetInt(c.Atom, v) }
func (c codec) SetFloat(v float64) error        { return c.Encoder.SetFloat(c.Atom, v) }
func (c codec) SetSliceOfUint(v []uint64) error { return c.Encoder.SetSliceOfUint(c.Atom, v) }
func (c codec) SetSliceOfInt(v []int64) error   { return c.Encoder.SetSliceOfInt(c.Atom, v) }
func (c codec) SetSliceOfByte(v []byte) error   { return c.Encoder.SetSliceOfByte(c.Atom, v) }

// Initialize decoder table, which makes decoders accessible by ADE type.
// Variable 'd' is used for assignment, because Go disallows assigning directly
// to a struct member of a map value.  Example:
//    decoderByType[UI01] = NewDecoder(UI01)
//    decoderByType[UI01].Bool = UI32ToBool //illegal
func init() {
	// ADE unsigned int types
	dec := NewDecoder(UI01)
	dec.String = UI32ToString
	dec.Bool = UI32ToBool
	dec.Uint = UI32ToUint64
	decoderByType[UI01] = dec

	dec = NewDecoder(UI08)
	dec.String = UI08ToString
	dec.Uint = UI08ToUint64
	decoderByType[UI08] = dec

	dec = NewDecoder(UI16)
	dec.String = UI16ToString
	dec.Uint = UI16ToUint64
	decoderByType[UI16] = dec

	dec = NewDecoder(UI32)
	dec.String = UI32ToString
	dec.Uint = UI32ToUint64
	decoderByType[UI32] = dec

	dec = NewDecoder(UI64)
	dec.String = UI64ToString
	dec.Uint = UI64ToUint64
	decoderByType[UI64] = dec

	// ADE signed int types
	dec = NewDecoder(SI08)
	dec.String = SI08ToString
	dec.Int = SI08ToInt64
	decoderByType[SI08] = dec

	dec = NewDecoder(SI16)
	dec.String = SI16ToString
	dec.Int = SI16ToInt64
	decoderByType[SI16] = dec

	dec = NewDecoder(SI32)
	dec.String = SI32ToString
	dec.Int = SI32ToInt64
	decoderByType[SI32] = dec

	dec = NewDecoder(SI64)
	dec.String = SI64ToString
	dec.Int = SI64ToInt64
	decoderByType[SI64] = dec

	// ADE floating point types
	dec = NewDecoder(FP32)
	dec.String = FP32ToString
	dec.Float = FP32ToFloat64
	decoderByType[FP32] = dec

	dec = NewDecoder(FP64)
	dec.String = FP64ToString
	dec.Float = FP64ToFloat64
	decoderByType[FP64] = dec

	// ADE fixed point types
	dec = NewDecoder(UF32)
	dec.String = UF32ToString
	dec.Float = UF32ToFloat64
	decoderByType[UF32] = dec

	dec = NewDecoder(UF64)
	dec.String = UF64ToString
	dec.Float = UF64ToFloat64
	decoderByType[UF64] = dec

	dec = NewDecoder(SF32)
	dec.String = SF32ToString
	dec.Float = SF32ToFloat64
	decoderByType[SF32] = dec

	dec = NewDecoder(SF64)
	dec.String = SF64ToString
	dec.Float = SF64ToFloat64
	decoderByType[SF64] = dec

	// ADE fractional types

	dec = NewDecoder(UR32)
	dec.String = UR32ToString
	dec.SliceOfUint = UR32ToSliceOfUint
	decoderByType[UR32] = dec

	dec = NewDecoder(UR64)
	dec.String = UR64ToString
	dec.SliceOfUint = UR64ToSliceOfUint
	decoderByType[UR64] = dec

	dec = NewDecoder(SR32)
	dec.String = SR32ToString
	dec.SliceOfInt = SR32ToSliceOfInt
	decoderByType[SR32] = dec

	dec = NewDecoder(SR64)
	dec.String = SR64ToString
	dec.SliceOfInt = SR64ToSliceOfInt
	decoderByType[SR64] = dec

	// ADE Four char code
	dec = NewDecoder(FC32)
	dec.String = FC32ToString
	decoderByType[FC32] = dec

	// ADE ENUM type
	dec = NewDecoder(ENUM)
	dec.String = SI32ToString
	dec.Int = SI32ToInt64
	decoderByType[ENUM] = dec

	// ADE UUID type
	dec = NewDecoder(UUID)
	dec.String = UUIDToString
	decoderByType[UUID] = dec

	// IP Address types
	dec = NewDecoder(IP32)
	dec.String = IP32ToString
	decoderByType[IP32] = dec

	dec = NewDecoder(IPAD)
	dec.String = IPADToString
	decoderByType[IPAD] = dec

	// ADE String types
	dec = NewDecoder(CSTR)
	dec.StringRaw = CSTRToString
	dec.String = CSTRToStringEscaped
	decoderByType[CSTR] = dec

	dec = NewDecoder(USTR)
	dec.StringRaw = USTRToString
	dec.String = USTRToStringEscaped
	decoderByType[USTR] = dec

	// DATA type, and aliases
	dec = NewDecoder(DATA)
	dec.String = BytesToHexString
	decoderByType[DATA] = dec
	decoderByType[CNCT] = dec
	decoderByType[Cnct] = dec

	// NULL type
	dec = NewDecoder(NULL)
	dec.String = func([]byte) (s string, e error) { return }
	decoderByType[NULL] = dec

	// ADE container
	dec = NewDecoder(CONT)
	dec.String = func([]byte) (s string, e error) { return }
	decoderByType[CONT] = dec
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
		e = fmt.Errorf("range error: value %d overflows type bool", ui32)
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
	return fmt.Sprintf("%d", (buf[0])), e
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
		v = fmt.Sprintf("%0.8G", f)
	}
	return
}
func FP64ToString(buf []byte) (v string, e error) {
	var f float64
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
		iFract := i & 0x00000000FFFFFFFF
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

/**********************************************************/

func asPrintableString(buf []byte) string {
	if isPrintableBytes(buf) {
		return string(buf[:])
	} else {
		i, _ := UI32ToUint32(buf)
		return fmt.Sprintf("0x%08X", i)
	}
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

/**********************************************************
Encoder method table for ADE types
**********************************************************/

func init() {
	// ADE unsigned int types
	enc := NewEncoder(UI01)
	enc.SetString = SetUI01FromString
	enc.SetBool = SetUI01FromBool
	enc.SetUint = SetUI01FromUint64
	encoderByType[UI01] = enc

	enc = NewEncoder(UI08)
	enc.SetString = SetUI08FromString
	enc.SetUint = SetUI08FromUint64
	encoderByType[UI08] = enc

	enc = NewEncoder(UI16)
	enc.SetString = SetUI16FromString
	enc.SetUint = SetUI16FromUint64
	encoderByType[UI16] = enc

	enc = NewEncoder(UI32)
	enc.SetString = SetUI32FromString
	enc.SetUint = SetUI32FromUint64
	encoderByType[UI32] = enc

	enc = NewEncoder(UI64)
	enc.SetString = SetUI64FromString
	enc.SetUint = SetUI64FromUint64
	encoderByType[UI64] = enc

	// ADE signed int types
	enc = NewEncoder(SI08)
	enc.SetString = SetSI08FromString
	enc.SetInt = SetSI08FromInt64
	encoderByType[SI08] = enc

	enc = NewEncoder(SI16)
	enc.SetString = SetSI16FromString
	enc.SetInt = SetSI16FromInt64
	encoderByType[SI16] = enc

	enc = NewEncoder(SI32)
	enc.SetString = SetSI32FromString
	enc.SetInt = SetSI32FromInt64
	encoderByType[SI32] = enc

	enc = NewEncoder(SI64)
	enc.SetString = SetSI64FromString
	enc.SetInt = SetSI64FromInt64
	encoderByType[SI64] = enc

	// ADE floating point types
	enc = NewEncoder(FP32)
	enc.SetString = SetFP32FromString
	enc.SetFloat = SetFP32FromFloat64
	encoderByType[FP32] = enc

	enc = NewEncoder(FP64)
	enc.SetString = SetFP64FromString
	enc.SetFloat = SetFP64FromFloat64
	encoderByType[FP64] = enc

	// ADE fixed point types
	enc = NewEncoder(UF32)
	enc.SetString = SetUF32FromString
	enc.SetFloat = SetUF32FromFloat64
	encoderByType[UF32] = enc

	enc = NewEncoder(UF64)
	enc.SetString = SetUF64FromString
	enc.SetFloat = SetUF64FromFloat64
	encoderByType[UF64] = enc

	// --

	enc = NewEncoder(SF32)
	//	enc.SetString = StringToSF32
	//	enc.SetFloat = Float64ToSF32
	encoderByType[SF32] = enc

	enc = NewEncoder(SF64)
	//	enc.SetString = StringToSF64
	//	enc.SetFloat = Float64ToSF64
	encoderByType[SF64] = enc

	// ADE fractional types

	enc = NewEncoder(UR32)
	enc.SetString = SetUR32FromString
	enc.SetSliceOfUint = SetUR32FromSliceOfUint
	encoderByType[UR32] = enc

	enc = NewEncoder(UR64)
	enc.SetString = SetUR64FromString
	enc.SetSliceOfUint = SetUR64FromSliceOfUint
	encoderByType[UR64] = enc

	enc = NewEncoder(SR32)
	enc.SetString = SetSR32FromString
	enc.SetSliceOfInt = SetSR32FromSliceOfInt
	encoderByType[SR32] = enc

	enc = NewEncoder(SR64)
	enc.SetString = SetSR64FromString
	enc.SetSliceOfInt = SetSR64FromSliceOfInt
	encoderByType[SR64] = enc

	// ADE Four char code
	enc = NewEncoder(FC32)
	//	enc.SetString = StringToFC32
	encoderByType[FC32] = enc

	// ADE ENUM type
	enc = NewEncoder(ENUM)
	//	enc.SetString = StringToSI32
	//	enc.SetInt = Int64ToSI32
	encoderByType[ENUM] = enc

	// ADE UUID type
	enc = NewEncoder(UUID)
	//	enc.SetString = StringToUUID
	encoderByType[UUID] = enc

	// IP Address types
	enc = NewEncoder(IP32)
	//	enc.SetString = StringToIP32
	encoderByType[IP32] = enc

	enc = NewEncoder(IPAD)
	//	enc.SetString = StringToIPAD
	encoderByType[IPAD] = enc

	// ADE String types
	enc = NewEncoder(CSTR)
	//	enc.SetStringRaw = StringToCSTR
	//	enc.SetString = StringEscapedToCSTR
	encoderByType[CSTR] = enc

	enc = NewEncoder(USTR)
	//	enc.SetStringRaw = StringToUSTR
	//	enc.SetString = StringEscapedToUSTR
	encoderByType[USTR] = enc

	// DATA type, and aliases
	enc = NewEncoder(DATA)
	//	enc.SetString = HexStringToBytes
	encoderByType[DATA] = enc
	encoderByType[CNCT] = enc
	encoderByType[Cnct] = enc

	// NULL type
	enc = NewEncoder(NULL)
	//	enc.SetString = func([]byte) (s string, e error) { return }
	encoderByType[NULL] = enc

	// ADE container
	enc = NewEncoder(CONT)
	enc.SetString = func(a *Atom, v string) error { return nil }
	encoderByType[CONT] = enc
}

/************************************************************
Encoding functions - set Atom.data bytes from go type
************************************************************/

func SetUI01FromString(a *Atom, v string) (e error) {
	switch v {
	case "true":
		binary.BigEndian.PutUint32(a.data, uint32(1))
	case "false":
		binary.BigEndian.PutUint32(a.data, uint32(0))
	default:
		var i uint64
		i, e = strconv.ParseUint(v, 0, 1)
		if e != nil {
			return
		}
		return SetUI01FromUint64(a, i)
	}
	return
}

func SetUI01FromBool(a *Atom, v bool) (e error) {
	if v {
		binary.BigEndian.PutUint32(a.data, uint32(1))
	} else {
		binary.BigEndian.PutUint32(a.data, uint32(0))
	}
	return
}

func SetUI01FromUint64(a *Atom, v uint64) (e error) {
	if v == 1 {
		binary.BigEndian.PutUint32(a.data, uint32(1))
	} else if v == 0 {
		binary.BigEndian.PutUint32(a.data, uint32(0))
	} else {
		e = fmt.Errorf("value overflows type UINT01: %d", v)
	}
	return
}

// encode of unsigned integer types
func SetUI08FromString(a *Atom, v string) (e error) {
	var i uint64
	i, e = strconv.ParseUint(v, 0, 8)
	if e != nil {
		return
	}
	return SetUI08FromUint64(a, i)
}

func SetUI08FromUint64(a *Atom, v uint64) (e error) {
	if v > math.MaxUint8 {
		return fmt.Errorf("value overflows type UI08: %d", v)
	}
	if len(a.data) != 1 {
		return fmt.Errorf("UI08 atom data buffer size should be 1, not %d", len(a.data))
	}
	a.data[0] = uint8(v)
	return
}

func SetUI16FromString(a *Atom, v string) (e error) {
	var i uint64
	i, e = strconv.ParseUint(v, 0, 16)
	if e == nil {
		binary.BigEndian.PutUint16(a.data, uint16(i))
	}
	return
}

func SetUI16FromUint64(a *Atom, v uint64) (e error) {
	if v > math.MaxUint16 {
		e = fmt.Errorf("value overflows type UINT16: %d", v)
		return
	}
	binary.BigEndian.PutUint16(a.data, uint16(v))
	return
}

func SetUI32FromString(a *Atom, v string) (e error) {
	var i uint64
	i, e = strconv.ParseUint(v, 0, 32)
	if e == nil {
		binary.BigEndian.PutUint32(a.data, uint32(i))
	}
	return
}

func SetUI32FromUint64(a *Atom, v uint64) (e error) {
	if v > math.MaxUint32 {
		e = fmt.Errorf("value overflows type UINT32: %d", v)
		return
	}
	binary.BigEndian.PutUint32(a.data, uint32(v))
	return
}

func SetUI64FromString(a *Atom, v string) (e error) {
	var i uint64
	i, e = strconv.ParseUint(v, 0, 64)
	if e == nil {
		binary.BigEndian.PutUint64(a.data, uint64(i))
	}
	return
}

func SetUI64FromUint64(a *Atom, v uint64) (e error) {
	binary.BigEndian.PutUint64(a.data, uint64(v))
	return
}

// encode of signed integer types

func SetSI08FromString(a *Atom, v string) (e error) {
	var i int64
	i, e = strconv.ParseInt(v, 0, 8)
	if e != nil {
		return
	}
	return SetSI08FromInt64(a, i)
}

func SetSI08FromInt64(a *Atom, v int64) (e error) {
	if v > math.MaxInt8 {
		return fmt.Errorf("value overflows type SI08: %d", v)
	}
	a.data[0] = byte(v)
	return
}

func SetSI16FromString(a *Atom, v string) (e error) {
	var i int64
	i, e = strconv.ParseInt(v, 0, 16)
	if e == nil {
		binary.BigEndian.PutUint16(a.data, uint16(i))
	}
	return
}

func SetSI16FromInt64(a *Atom, v int64) (e error) {
	if v < math.MinInt16 || v > math.MaxInt16 {
		e = fmt.Errorf("value overflows type int16: %d", v)
		return
	}
	binary.BigEndian.PutUint16(a.data, uint16(v))
	return
}

func SetSI32FromString(a *Atom, v string) (e error) {
	var i int64
	i, e = strconv.ParseInt(v, 0, 32)
	if e == nil {
		binary.BigEndian.PutUint32(a.data, uint32(i))
	}
	return
}

func SetSI32FromInt64(a *Atom, v int64) (e error) {
	if v < math.MinInt32 || v > math.MaxInt32 {
		e = fmt.Errorf("value overflows type Int32: %d", v)
		return
	}
	binary.BigEndian.PutUint32(a.data, uint32(v))
	return
}

func SetSI64FromString(a *Atom, v string) (e error) {
	var i int64
	i, e = strconv.ParseInt(v, 0, 64)
	if e == nil {
		binary.BigEndian.PutUint64(a.data, uint64(i))
	}
	return
}

func SetSI64FromInt64(a *Atom, v int64) (e error) {
	binary.BigEndian.PutUint64(a.data, uint64(v))
	return
}

// encode of unsigned fractional types

func SetUR32FromString(a *Atom, v string) (e error) {
	var num, den uint64
	_, err := fmt.Sscanf(v, "%d / %d", &num, &den)
	if err != nil {
		return err
	}
	return SetUR32FromSliceOfUint(a, []uint64{num, den})
}

func SetUR32FromSliceOfUint(a *Atom, v []uint64) (e error) {
	var num, den uint64
	num = v[0]
	den = v[1]
	if num > math.MaxUint16 || den > math.MaxUint16 {
		e = fmt.Errorf("cannot set UR32, fractional part overflows type uint16: %d", v)
		return e
	}

	value := (uint32(num) << 16) + uint32(den)
	binary.BigEndian.PutUint32(a.data, value)
	return
}

func SetUR64FromString(a *Atom, v string) (e error) {
	var num, den uint64
	_, err := fmt.Sscanf(v, "%d / %d", &num, &den)
	if err != nil {
		return err
	}
	return SetUR64FromSliceOfUint(a, []uint64{num, den})
}

func SetUR64FromSliceOfUint(a *Atom, v []uint64) (e error) {
	var num, den uint64
	num = v[0]
	den = v[1]
	if num > math.MaxUint32 || den > math.MaxUint32 {
		e = fmt.Errorf("cannot set UR64, fractional part overflows type uint32: %d", v)
		return e
	}

	value := (num << 32) + den
	binary.BigEndian.PutUint64(a.data, value)
	return
}

// encode of signed fractional types

func SetSR32FromString(a *Atom, v string) (e error) {
	var num, den int64
	_, err := fmt.Sscanf(v, "%d / %d", &num, &den)
	if err != nil {
		return err
	}
	return SetSR32FromSliceOfInt(a, []int64{num, den})
}

func SetSR32FromSliceOfInt(a *Atom, v []int64) (e error) {
	var num, den int64
	num = v[0]
	den = v[1]
	if num > math.MaxInt16 || den > math.MaxInt16 {
		e = fmt.Errorf("cannot set SR32, fractional part overflows type int16: %d", v)
		return e
	}

	value := (int32(num) << 16) + int32(den)
	binary.BigEndian.PutUint32(a.data, uint32(value))
	return
}

func SetSR64FromString(a *Atom, v string) (e error) {
	var num, den int64
	_, err := fmt.Sscanf(v, "%d / %d", &num, &den)
	if err != nil {
		return err
	}
	return SetSR64FromSliceOfInt(a, []int64{num, den})
}

func SetSR64FromSliceOfInt(a *Atom, v []int64) (e error) {
	var num, den int64
	num = v[0]
	den = v[1]
	if num > math.MaxInt32 || den > math.MaxInt32 || num < math.MinInt32 || den < math.MinInt32 {
		e = fmt.Errorf("cannot set SR64, fractional part overflows type int32: %d", v)
		return e
	}

	value := (num << 32) + den
	binary.BigEndian.PutUint64(a.data, uint64(value))
	return
}

// encode of floating point types

func SetFP32FromString(a *Atom, v string) (e error) {
	var f float64
	f, e = strconv.ParseFloat(v, 32)
	return SetFP32FromFloat64(a, f)
}

func SetFP32FromFloat64(a *Atom, v float64) (e error) {
	if v > math.MaxFloat32 {
		e = fmt.Errorf("value overflows type Float32: %f", v)
		return
	}

	var bits uint32 = math.Float32bits(float32(v))
	binary.BigEndian.PutUint32(a.data, bits)
	return
}

func SetFP64FromString(a *Atom, v string) (e error) {
	var f float64
	f, e = strconv.ParseFloat(v, 64)
	return SetFP64FromFloat64(a, f)
}

func SetFP64FromFloat64(a *Atom, v float64) (e error) {
	binary.BigEndian.PutUint64(a.data, uint64(v))
	var bits uint64 = math.Float64bits(v)
	binary.BigEndian.PutUint64(a.data, bits)
	return
}

// encode of fixed point types

func SetUF32FromString(a *Atom, v string) (e error) {
	var f float64
	f, e = strconv.ParseFloat(v, 64)
	return SetUF32FromFloat64(a, f)
}

func SetUF32FromFloat64(a *Atom, v float64) (e error) {
	binary.BigEndian.PutUint32(a.data, uint32(v*65536.0))
	return
}

// split string into whole and fractional parts
func SetUF64FromString(a *Atom, v string) (e error) {
	pieces := strings.Split(v, ".")
	if len(pieces) > 2 {
		return fmt.Errorf("invalid fixed point data:%s", v)
	}

	// whole part to the first 32 bits of a uint64
	var whole uint64
	whole, e = strconv.ParseUint(pieces[0], 10, 64)
	if e != nil {
		return
	}
	whole <<= 32

	// fractional part
	var fract float64
	fract, e = strconv.ParseFloat(pieces[1], 64)
	if e != nil {
		return
	}
	if 0.0 <= fract && fract < 4294967296.0 {
		fract *= (4294967296.0 / math.Pow(10, 9))
	}

	// combine bits into final value
	binary.BigEndian.PutUint64(a.data, whole+uint64(fract))
	return
}

func SetUF64FromFloat64(a *Atom, v float64) (e error) {
	var i = uint64(v * 4294967296.0)
	binary.BigEndian.PutUint64(a.data, i)
	return
}
