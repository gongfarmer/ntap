package atom

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"unicode"
)

// ADE Data types are defined in 112-0002_r4.0B_StorageGRID_Data_Types.
// The ADE headers for these types are in OSL_Types.h.
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
	USTR ADEType = "USTR" // unicode string, encoded as UTF-32 Big-endian
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
)

/**********************************************************/

type (
	ADEType string
	GoType  string

	codec struct {
		Atom    *Atom
		Decoder decoder
		Encoder encoder
		Writer  io.Writer // writes bytes directly to Atom.data
	}
	decoder struct {
		String        func(buf []byte) (string, error)
		StringEscaped func(buf []byte) (string, error)
		Bool          func(buf []byte) (bool, error)
		Uint          func(buf []byte) (uint64, error)
		Int           func(buf []byte) (int64, error)
		Float         func(buf []byte) (float64, error)
		SliceOfUint   func(buf []byte) ([]uint64, error)
		SliceOfInt    func(buf []byte) ([]int64, error)
		SliceOfByte   func(buf []byte) ([]byte, error)
	}
	encoder struct {
		SetString        func(*Atom, string) error
		SetStringEscaped func(*Atom, string) error
		SetBool          func(*Atom, bool) error
		SetUint          func(*Atom, uint64) error
		SetInt           func(*Atom, int64) error
		SetFloat         func(*Atom, float64) error
		SetSliceOfUint   func(*Atom, []uint64) error
		SetSliceOfInt    func(*Atom, []int64) error
		SetSliceOfByte   func(*Atom, []byte) error
	}

	uuidType struct {
		TimeLow          uint32
		TimeMid          uint16
		TimeHiAndVersion uint16
		ClkSeqHiRes      uint8
		ClkSeqLow        uint8
		Node             [6]byte
	}

	errFunc (func(string, int, int) error)
)

// error construction functions

func errByteCount(t string, bytesWant int, bytesGot int) (e error) {
	return fmt.Errorf("invalid byte count for ADE type %s: want %d, got %d", t, bytesWant, bytesGot)
}
func errStrInvalid(t string, v string) error {
	return fmt.Errorf("invalid string value for ADE type %s: \"%s\"", t, v)
}
func errRange(t string, v interface{}) (e error) {
	switch v := v.(type) {
	case uint, uint8, uint16, uint32, uint64, int, int32, int64:
		e = fmt.Errorf("value exceeds range for type %s: %d", t, v)
	case float32, float64:
		e = fmt.Errorf("value exceeds range for type %s: %f", t, v)
	case []uint64, []int64:
		e = fmt.Errorf("value exceeds range for type %s: %v", t, v)
	case string:
		e = fmt.Errorf("value exceeds range for type %s: %v", t, v)
	default:
		panic(fmt.Errorf("cannot handle type %T", v))
	}
	return
}

// NewCodec returns a codec that performs type conversion for atom data.
// A codec provides encoder/decoder methods for converting data from an atom's
// ADE type into suitable Go types, and vice versa.
func NewCodec(a *Atom) *codec {
	c := codec{
		Atom:    a,
		Decoder: decoderByType[a.Type()],
		Encoder: encoderByType[a.Type()],
		Writer:  bytes.NewBuffer(a.data), // allows writing directly to a.data, not a copy
	}
	return &c
}

// NewEncoder returns a new encoder that provides functions for converting Go
// native types into ADE Atom data.  The returned encoder contains all of the
// default encoding methods, which simply return an error stating that the
// encoding is not supported.
// The caller should implement whichever encoding methods are appropriate for
// the ADE data type's codec.
func NewEncoder(i interface{}) encoder {
	return encoder{
		SetString:        func(a *Atom, v string) (e error) { return noEncoder(a, "string") },
		SetStringEscaped: func(a *Atom, v string) (e error) { return noEncoder(a, "string") },
		SetBool:          func(a *Atom, v bool) (e error) { return noEncoder(a, "bool") },
		SetUint:          func(a *Atom, v uint64) (e error) { return noEncoder(a, "uint64") },
		SetInt:           func(a *Atom, v int64) (e error) { return noEncoder(a, "int64") },
		SetFloat:         func(a *Atom, v float64) (e error) { return noEncoder(a, "float64") },
		SetSliceOfUint:   func(a *Atom, v []uint64) (e error) { return noEncoder(a, "[]uint64") },
		SetSliceOfInt:    func(a *Atom, v []int64) (e error) { return noEncoder(a, "[]int64") },
		SetSliceOfByte:   func(a *Atom, v []byte) (e error) { return noEncoder(a, "[]byte") },
	}
}

// NewEncoder returns a new encoder that provides functions for converting Go
// native types into ADE Atom data.  The returned encoder contains all of the
// default encoding methods, which simply return an error stating that the
// encoding is not supported.
// The caller should implement whichever encoding methods are appropriate for
// the ADE data type's codec.
func NewDecoder(from ADEType) decoder {
	return decoder{
		String:        func([]byte) (v string, e error) { return v, noDecoder(from, "string") },
		StringEscaped: func([]byte) (v string, e error) { return v, noDecoder(from, "string") },
		Bool:          func([]byte) (v bool, e error) { return v, noDecoder(from, "bool") },
		Uint:          func([]byte) (v uint64, e error) { return v, noDecoder(from, "uint64") },
		Int:           func([]byte) (v int64, e error) { return v, noDecoder(from, "int64") },
		Float:         func([]byte) (v float64, e error) { return v, noDecoder(from, "float64") },
		SliceOfUint:   func([]byte) (v []uint64, e error) { return v, noDecoder(from, "[]uint64") },
		SliceOfInt:    func([]byte) (v []int64, e error) { return v, noDecoder(from, "[]int64") },
		SliceOfByte:   func(data []byte) (v []byte, e error) { return data, nil },
	}
}

var decoderByType = make(map[ADEType]decoder)
var encoderByType = make(map[ADEType]encoder)

func noEncoder(a *Atom, from interface{}) error {
	return fmt.Errorf("no encoder exists to convert go type '%s' to ADE type '%s'.", from, a.typ)
}
func noDecoder(from ADEType, to GoType) error {
	return fmt.Errorf("no decoder exists to convert ADE type '%s' to go type '%s'.", from, to)
}

// Decoder methods: pass atom data to the decoder for type conversion to go type
func (c codec) String() (string, error)        { return c.Decoder.String(c.Atom.data) }
func (c codec) StringEscaped() (string, error) { return c.Decoder.StringEscaped(c.Atom.data) }
func (c codec) Bool() (bool, error)            { return c.Decoder.Bool(c.Atom.data) }
func (c codec) Uint() (uint64, error)          { return c.Decoder.Uint(c.Atom.data) }
func (c codec) Int() (int64, error)            { return c.Decoder.Int(c.Atom.data) }
func (c codec) Float() (float64, error)        { return c.Decoder.Float(c.Atom.data) }
func (c codec) SliceOfUint() ([]uint64, error) { return c.Decoder.SliceOfUint(c.Atom.data) }
func (c codec) SliceOfInt() ([]int64, error)   { return c.Decoder.SliceOfInt(c.Atom.data) }
func (c codec) SliceOfByte() ([]byte, error)   { return c.Atom.data, nil }

// Encoder methods: convert given data type to []byte and store in ATom
func (c codec) SetString(v string) error        { return c.Encoder.SetString(c.Atom, v) }
func (c codec) SetStringEscaped(v string) error { return c.Encoder.SetStringEscaped(c.Atom, v) }
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
	dec.Bool = UI01ToBool
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
	dec.String = FC32ToStringDelimited
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
	dec.Uint = IP32ToUint64
	decoderByType[IP32] = dec

	dec = NewDecoder(IPAD)
	dec.String = IPADToString
	decoderByType[IPAD] = dec

	// ADE String types
	dec = NewDecoder(CSTR)
	dec.String = CSTRToString
	dec.StringEscaped = CSTRToStringEscaped
	decoderByType[CSTR] = dec

	dec = NewDecoder(USTR)
	dec.String = USTRToString
	dec.StringEscaped = USTRToStringEscaped
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
	if e = checkByteCount(buf, 1, "UI08"); e != nil {
		return
	}
	return uint64(buf[0]), e
}
func UI16ToUint64(buf []byte) (v uint64, e error) {
	if e = checkByteCount(buf, 2, "UI16"); e != nil {
		return
	}
	return uint64(binary.BigEndian.Uint16(buf)), e
}
func UI01ToBool(buf []byte) (v bool, e error) {
	if e = checkByteCount(buf, 4, "UI01"); e != nil {
		return
	}
	ui32 := binary.BigEndian.Uint32(buf)
	if ui32 > 1 {
		e = errRange("bool", ui32)
	}
	return ui32 == 1, e
}
func UI32ToUint32(buf []byte) (v uint32, e error) {
	if e = checkByteCount(buf, 4, "UI32"); e != nil {
		return
	}
	return binary.BigEndian.Uint32(buf), e
}
func UI32ToUint64(buf []byte) (v uint64, e error) {
	if e = checkByteCount(buf, 4, "UI32"); e != nil {
		return
	}
	var ui32 uint32 = binary.BigEndian.Uint32(buf)
	return uint64(ui32), e
}
func UI64ToUint64(buf []byte) (v uint64, e error) {
	if e = checkByteCount(buf, 8, "UI64"); e != nil {
		return
	}
	return binary.BigEndian.Uint64(buf), e
}
func UI08ToString(buf []byte) (v string, e error) {
	if e = checkByteCount(buf, 1, "UI08"); e != nil {
		return
	}
	return fmt.Sprintf("%d", (buf[0])), e
}
func UI16ToString(buf []byte) (v string, e error) {
	if e = checkByteCount(buf, 2, "UI16"); e != nil {
		return
	}
	return fmt.Sprintf("%d", binary.BigEndian.Uint16(buf)), e
}
func UI32ToString(buf []byte) (v string, e error) {
	if e = checkByteCount(buf, 4, "UI32"); e != nil {
		return
	}
	return fmt.Sprintf("%d", binary.BigEndian.Uint32(buf)), e
}
func UI64ToString(buf []byte) (v string, e error) {
	if e = checkByteCount(buf, 8, "UI64"); e != nil {
		return
	}
	return fmt.Sprintf("%d", binary.BigEndian.Uint64(buf)), e
}

// ADE signed int types

func SI08ToInt64(buf []byte) (v int64, e error) {
	if e = checkByteCount(buf, 1, "SI08"); e != nil {
		return
	}
	var i int8
	e = binary.Read(bytes.NewReader(buf), binary.BigEndian, &i)
	return int64(i), e
}
func SI16ToInt64(buf []byte) (v int64, e error) {
	if e = checkByteCount(buf, 2, "SI16"); e != nil {
		return
	}
	var i int16
	e = binary.Read(bytes.NewReader(buf), binary.BigEndian, &i)
	return int64(i), e
}
func SI32ToInt32(buf []byte) (v int32, e error) {
	if e = checkByteCount(buf, 4, "SI32"); e != nil {
		return
	}
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
	if e = checkByteCount(buf, 8, "SI64"); e != nil {
		return
	}
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
	if e = checkByteCount(buf, 4, "FP32"); e != nil {
		return
	}
	var i uint32
	i, e = UI32ToUint32(buf)
	v = math.Float32frombits(i)
	return
}
func FP32ToFloat64(buf []byte) (v float64, e error) {
	if e = checkByteCount(buf, 4, "FP32"); e != nil {
		return
	}
	var i uint32
	i, e = UI32ToUint32(buf)
	if e == nil {
		v = float64(math.Float32frombits(i))
	}
	return
}
func FP64ToFloat64(buf []byte) (v float64, e error) {
	if e = checkByteCount(buf, 8, "FP64"); e != nil {
		return
	}
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
	if e = checkByteCount(buf, 8, "UF64"); e != nil {
		return
	}
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
	v = float64(i) / (math.MaxUint16 + 1)
	return
}
func SF64ToFloat64(buf []byte) (v float64, e error) {
	var i int64
	i, e = SI64ToInt64(buf)
	if e != nil {
		return
	}
	v = float64(i) / (math.MaxUint32 + 1)
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
	if e = checkByteCount(buf, 4, "UR32"); e != nil {
		return
	}
	var arr [2]uint16
	e = binary.Read(bytes.NewReader(buf), binary.BigEndian, &arr)
	if e == nil {
		v = append(v, uint64(arr[0]), uint64(arr[1]))
	}
	return
}
func UR64ToSliceOfUint(buf []byte) (v []uint64, e error) {
	if e = checkByteCount(buf, 8, "UR64"); e != nil {
		return
	}
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
	if e = checkByteCount(buf, 4, "SR32"); e != nil {
		return
	}
	var arr [2]int16
	e = binary.Read(bytes.NewReader(buf), binary.BigEndian, &arr)
	if e == nil {
		v = append(v, int64(arr[0]), int64(arr[1]))
	}
	return
}
func SR64ToSliceOfInt(buf []byte) (v []int64, e error) {
	if e = checkByteCount(buf, 8, "SR64"); e != nil {
		return
	}
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

// FC32ToString returns a four-char code value as a printable string.
// The string may be either 4 printable characters, or 0x followed by 8 hex
// digits.
//
// This code includes a fix for Mantis #27726: ccat/ctac can't parse container
// names starting with "#" or " ".
func FC32ToString(buf []byte) (v string, e error) {
	if e = checkByteCount(buf, 4, "FC32"); e != nil {
		return
	}
	var badStartChars = `# "'`
	if isPrintableBytes(buf) && !strings.ContainsRune(badStartChars, rune(buf[0])) {
		v = string(buf)
	} else {
		v = fmt.Sprintf("0x%08X", buf)
	}
	return
}

// FC32ToStringDelimited returns a four-char code value as a printable string.
// The string may be either 4 printable characters, or 0x followed by 8 hex
// digits.
//
// If the 4 printable characters version is returned, it will be surrounded by
// single-quote delimiters.
func FC32ToStringDelimited(buf []byte) (v string, e error) {
	v, e = FC32ToString(buf)
	if len(v) == 4 {
		v = fmt.Sprintf("'%s'", v)
	}
	return
}

func UUIDToString(buf []byte) (v string, e error) {
	if e = checkByteCount(buf, 16, "UUID"); e != nil {
		return
	}
	var uuid uuidType
	e = binary.Read(bytes.NewReader(buf), binary.BigEndian, &uuid)
	if e != nil {
		return
	}
	return uuid.String(), e
}

// IP32ToString returns an IP32 value as a string with the IP address
// represented as a dotted quad (eg. 192.168.1.128).
//
// The IP32 type may optionally include multiple 4-byte values, which have
// occasionally (rarely) been used to represent address ranges.
// These are represented as hex, matching the ADE ccat behaviour.
func IP32ToString(buf []byte) (v string, e error) {

	// single address is expressed as dotted quad
	size := len(buf)
	if size == 4 {
		v = fmt.Sprintf("%d.%d.%d.%d", buf[0], buf[1], buf[2], buf[3])
		return
	}

	// need 4 bytes to make a complete address
	if 0 != size%4 || 0 == size {
		e = errByteCount("IP32", 4, size)
		return
	}

	// multiple addresses are expressed as hex
	var addrs = []string{"0x"}
	for i := 0; i < size; i += 4 {
		addrs = append(addrs, fmt.Sprintf("%02X%02X%02X%02X", buf[i], buf[i+1], buf[i+2], buf[i+3]))
	}
	v = strings.Join(addrs, "")

	return
}

// IP32ToUint64 returns an IP32 value as an unsigned integer.
//
// If the IP32 contains a single address, it is returned in the lower 4 bytes
// of a uint64.  Casting that to uint32 retains all 4 octets.
//
// The IP32 type may optionally include a second 4-byte value, which represents
// a range of IPv4 addresses.  In this case, both addresses will be returned in
// the UINT64 value, with one address in the upper 32 bits and one in the lower
// 32 bits.
func IP32ToUint64(buf []byte) (v uint64, e error) {
	switch len(buf) {
	case 4:
		v = uint64(binary.BigEndian.Uint32(buf))
	case 8:
		v = binary.BigEndian.Uint64(buf)
	case 12, 16:
		e = fmt.Errorf("extra-long IP32 value overflows uint64: %x", buf)
	default:
		e = errByteCount("IP32", 4, len(buf))
	}
	return
}

func IPADToString(buf []byte) (v string, e error) {
	v = string(buf[0 : len(buf)-1]) // trim null terminator
	v = fmt.Sprintf("\"%s\"", v)
	return
}

// String types

func CSTRToString(buf []byte) (v string, e error) {
	if len(buf) == 0 {
		return v, fmt.Errorf("Illegal CSTR data lacks null byte terminator")
	}
	v = string(buf[0 : len(buf)-1]) // trim null terminator
	return v, nil
}

func CSTRToStringEscaped(buf []byte) (v string, e error) {
	v, e = CSTRToString(buf)
	if e == nil {
		v = fmt.Sprintf("\"%s\"", adeStringEscape(v))
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
	if e != nil {
		return
	}

	// ADE escaping
	v = fmt.Sprintf("\"%s\"", adeStringEscape(v))

	// FIXME: need escaping on decode / encode to be guaranteed inverse operations

	// Go/unicode escaping
	//v = strconv.Quote(v)
	return
}

func BytesToHexString(buf []byte) (v string, e error) {
	if len(buf) == 0 {
		v = ""
	} else {
		v = fmt.Sprintf("0x%X", buf)
	}
	return
}

/**********************************************************/

// ade: libs/osl/OSL_Types.cc CStr_Escape()
func adeStringEscape(s string) string {
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

	enc = NewEncoder(SF32)
	enc.SetString = SetSF32FromString
	enc.SetFloat = SetSF32FromFloat64
	encoderByType[SF32] = enc

	enc = NewEncoder(SF64)
	enc.SetString = SetSF64FromString
	enc.SetFloat = SetSF64FromFloat64
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
	enc.SetString = SetFC32FromString
	enc.SetUint = SetFC32FromUint64
	encoderByType[FC32] = enc

	// IP Address types
	enc = NewEncoder(IP32)
	enc.SetString = SetIP32FromString
	enc.SetUint = SetIP32FromUint64
	encoderByType[IP32] = enc

	enc = NewEncoder(IPAD)
	enc.SetString = SetIPADFromString
	encoderByType[IPAD] = enc

	// ADE UUID type
	enc = NewEncoder(UUID)
	enc.SetString = SetUUIDFromString
	encoderByType[UUID] = enc

	// ADE String types
	enc = NewEncoder(CSTR)
	enc.SetString = SetCSTRFromString
	enc.SetStringEscaped = SetCSTRFromQuotedEscapedString
	encoderByType[CSTR] = enc

	enc = NewEncoder(USTR)
	enc.SetString = SetUSTRFromString
	enc.SetStringEscaped = SetUSTRFromQuotedEscapedString
	encoderByType[USTR] = enc

	// DATA type, and aliases
	enc = NewEncoder(DATA)
	enc.SetString = SetDATAFromHexString
	encoderByType[DATA] = enc
	encoderByType[CNCT] = enc
	encoderByType[Cnct] = enc

	// ADE ENUM type
	enc = NewEncoder(ENUM)
	enc.SetString = SetSI32FromString
	enc.SetInt = SetSI32FromInt64
	encoderByType[ENUM] = enc

	// NULL type
	enc = NewEncoder(NULL)
	enc.SetString = func(_ *Atom, _ string) (e error) { return }
	encoderByType[NULL] = enc

	// ADE container
	enc = NewEncoder(CONT)
	enc.SetString = func(_ *Atom, _ string) (e error) { return }
	encoderByType[CONT] = enc
}

/************************************************************
Encoding functions - set Atom.data bytes from go type
************************************************************/

func SetUI01FromString(a *Atom, v string) (e error) {
	if len(a.data) != 4 {
		a.data = make([]byte, 4)
	}
	switch v {
	case "false", "0", "+0", "-0":
		binary.BigEndian.PutUint32(a.data, uint32(0))
	case "true", "1", "+1":
		binary.BigEndian.PutUint32(a.data, uint32(1))
	default:
		e = errStrInvalid("UI01", v)
	}
	return
}

func SetUI01FromBool(a *Atom, v bool) (e error) {
	if len(a.data) != 4 {
		a.data = make([]byte, 4)
	}
	if v {
		binary.BigEndian.PutUint32(a.data, uint32(1))
	} else {
		binary.BigEndian.PutUint32(a.data, uint32(0))
	}
	return
}

func SetUI01FromUint64(a *Atom, v uint64) (e error) {
	if len(a.data) != 4 {
		a.data = make([]byte, 4)
	}
	if v == 1 {
		binary.BigEndian.PutUint32(a.data, uint32(1))
	} else if v == 0 {
		binary.BigEndian.PutUint32(a.data, uint32(0))
	} else {
		e = errRange("UI01", v)
	}
	return
}

// encode of unsigned integer types

func SetUI08FromString(a *Atom, v string) (e error) {
	if len(a.data) != 1 {
		a.data = make([]byte, 1)
	}
	var i uint64
	i, e = strconv.ParseUint(v, 0, 8)
	if e != nil {
		return errStrInvalid("UI08", v)
	}
	return SetUI08FromUint64(a, i)
}

func SetUI08FromUint64(a *Atom, v uint64) (e error) {
	if len(a.data) != 1 {
		a.data = make([]byte, 1)
	}
	if v > math.MaxUint8 {
		e = errRange("UI08", v)
		return
	}
	a.data[0] = uint8(v)
	return
}

func SetUI16FromString(a *Atom, v string) (e error) {
	if len(a.data) != 2 {
		a.data = make([]byte, 2)
	}
	var i uint64
	i, e = strconv.ParseUint(v, 0, 16)
	if e != nil {
		return errStrInvalid("UI16", v)
	}
	binary.BigEndian.PutUint16(a.data, uint16(i))
	return
}

func SetUI16FromUint64(a *Atom, v uint64) (e error) {
	if len(a.data) != 2 {
		a.data = make([]byte, 2)
	}
	if v > math.MaxUint16 {
		return errRange("UI16", v)
	}
	binary.BigEndian.PutUint16(a.data, uint16(v))
	return
}

func SetUI32FromString(a *Atom, v string) (e error) {
	if len(a.data) != 4 {
		a.data = make([]byte, 4)
	}
	var i uint64
	i, e = strconv.ParseUint(v, 0, 32)
	if e != nil {
		return errStrInvalid("UI32", v)
	}
	binary.BigEndian.PutUint32(a.data, uint32(i))
	return
}

func SetUI32FromUint64(a *Atom, v uint64) (e error) {
	if len(a.data) != 4 {
		a.data = make([]byte, 4)
	}
	if v > math.MaxUint32 {
		return errRange("UI32", v)
	}
	binary.BigEndian.PutUint32(a.data, uint32(v))
	return
}

func SetUI64FromString(a *Atom, v string) (e error) {
	if len(a.data) != 8 {
		a.data = make([]byte, 8)
	}

	var i uint64
	i, e = strconv.ParseUint(v, 0, 64)
	if e != nil {
		return errStrInvalid("UI64", v)
	}
	binary.BigEndian.PutUint64(a.data, uint64(i))
	return
}

func SetUI64FromUint64(a *Atom, v uint64) (e error) {
	if len(a.data) != 8 {
		a.data = make([]byte, 8)
	}
	binary.BigEndian.PutUint64(a.data, uint64(v))
	return
}

// encode of signed integer types

func SetSI08FromString(a *Atom, v string) (e error) {
	if len(a.data) != 1 {
		a.data = make([]byte, 1)
	}
	var i int64
	i, e = strconv.ParseInt(v, 0, 8)
	if e != nil {
		return errStrInvalid("SI08", v)
	}
	return SetSI08FromInt64(a, i)
}

func SetSI08FromInt64(a *Atom, v int64) (e error) {
	if len(a.data) != 1 {
		a.data = make([]byte, 1)
	}
	if v < math.MinInt8 || v > math.MaxInt8 {
		return errRange("SI08", v)
	}
	a.data[0] = byte(v)
	return
}

func SetSI16FromString(a *Atom, v string) (e error) {
	if len(a.data) != 2 {
		a.data = make([]byte, 2)
	}
	var i int64
	i, e = strconv.ParseInt(v, 0, 16)
	if e != nil {
		return errStrInvalid("SI16", v)
	}
	binary.BigEndian.PutUint16(a.data, uint16(i))
	return
}

func SetSI16FromInt64(a *Atom, v int64) (e error) {
	if len(a.data) != 2 {
		a.data = make([]byte, 2)
	}
	if v < math.MinInt16 || v > math.MaxInt16 {
		return errRange("SI16", v)
	}
	binary.BigEndian.PutUint16(a.data, uint16(v))
	return
}

func SetSI32FromString(a *Atom, v string) (e error) {
	if len(a.data) != 4 {
		a.data = make([]byte, 4)
	}
	var i int64
	i, e = strconv.ParseInt(v, 0, 32)
	if e != nil {
		return errStrInvalid("SI32", v)
	}
	binary.BigEndian.PutUint32(a.data, uint32(i))
	return
}

func SetSI32FromInt64(a *Atom, v int64) (e error) {
	if len(a.data) != 4 {
		a.data = make([]byte, 4)
	}
	if v < math.MinInt32 || v > math.MaxInt32 {
		return errRange("SI32", v)
	}
	binary.BigEndian.PutUint32(a.data, uint32(v))
	return
}

func SetSI64FromString(a *Atom, v string) (e error) {
	if len(a.data) != 8 {
		a.data = make([]byte, 8)
	}
	var i int64
	i, e = strconv.ParseInt(v, 0, 64)
	if e != nil {
		return errStrInvalid("SI64", v)
	}
	binary.BigEndian.PutUint64(a.data, uint64(i))
	return
}

func SetSI64FromInt64(a *Atom, v int64) (e error) {
	if len(a.data) != 8 {
		a.data = make([]byte, 8)
	}
	binary.BigEndian.PutUint64(a.data, uint64(v))
	return
}

// encode of unsigned fractional types

func SetUR32FromString(a *Atom, v string) (e error) {
	if len(a.data) != 4 {
		a.data = make([]byte, 4)
	}

	// The %s is to detect trailing garbage in the line. It should not match
	// anything in the normal case.
	var num, den uint64
	var extra string
	matched, err := fmt.Sscanf(v, "%d/%d%s", &num, &den, &extra)
	if err != io.EOF || matched != 2 {
		return errStrInvalid("UR32", v)
	}
	return SetUR32FromSliceOfUint(a, []uint64{num, den})
}

func SetUR32FromSliceOfUint(a *Atom, v []uint64) (e error) {
	if len(a.data) != 4 {
		a.data = make([]byte, 4)
	}
	var num, den uint64
	num = v[0]
	den = v[1]
	if num > math.MaxUint16 || den > math.MaxUint16 {
		return errRange("UR32", v)
	}

	value := (uint32(num) << 16) + uint32(den)
	binary.BigEndian.PutUint32(a.data, value)
	return
}

func SetUR64FromString(a *Atom, v string) (e error) {
	if len(a.data) != 8 {
		a.data = make([]byte, 8)
	}
	var num, den uint64
	var extra string
	matched, err := fmt.Sscanf(v, "%d/%d%s", &num, &den, &extra)
	if err != io.EOF || matched != 2 {
		return errStrInvalid("UR64", v)
	}
	return SetUR64FromSliceOfUint(a, []uint64{num, den})
}

func SetUR64FromSliceOfUint(a *Atom, v []uint64) (e error) {
	if len(a.data) != 8 {
		a.data = make([]byte, 8)
	}
	var num, den uint64
	num = v[0]
	den = v[1]
	if num > math.MaxUint32 || den > math.MaxUint32 {
		return errRange("UR64", v)
	}

	value := (num << 32) + den
	binary.BigEndian.PutUint64(a.data, value)
	return
}

// encode of signed fractional types

func SetSR32FromString(a *Atom, v string) (e error) {
	if len(a.data) != 4 {
		a.data = make([]byte, 4)
	}
	var num, den int64
	var extra string
	matched, err := fmt.Sscanf(v, "%d/%d%s", &num, &den, &extra)
	if err != io.EOF || matched != 2 {
		return errStrInvalid("SR32", v)
	}
	return SetSR32FromSliceOfInt(a, []int64{num, den})
}

func SetSR32FromSliceOfInt(a *Atom, v []int64) (e error) {
	if len(a.data) != 4 {
		a.data = make([]byte, 4)
	}
	var num, den int64
	num = v[0]
	den = v[1]
	if num > math.MaxInt16 || den > math.MaxInt16 || num < math.MinInt16 || den < math.MinInt16 {
		return errRange("SR32", v)
	}

	value := (int32(num) << 16) + int32(den)
	binary.BigEndian.PutUint32(a.data, uint32(value))
	return
}

func SetSR64FromString(a *Atom, v string) (e error) {
	if len(a.data) != 8 {
		a.data = make([]byte, 8)
	}
	var num, den int64
	var extra string
	matched, err := fmt.Sscanf(v, "%d/%d%s", &num, &den, &extra)
	if err != io.EOF || matched != 2 {
		return errStrInvalid("SR64", v)
	}
	return SetSR64FromSliceOfInt(a, []int64{num, den})
}

func SetSR64FromSliceOfInt(a *Atom, v []int64) (e error) {
	if len(a.data) != 8 {
		a.data = make([]byte, 8)
	}
	var num, den int64
	num = v[0]
	den = v[1]
	if num > math.MaxInt32 || den > math.MaxInt32 || num < math.MinInt32 || den < math.MinInt32 {
		return errRange("SR64", v)
	}

	value := (num << 32) + den
	binary.BigEndian.PutUint64(a.data, uint64(value))
	return
}

// encode of floating point types

func SetFP32FromString(a *Atom, v string) (e error) {
	if len(a.data) != 4 {
		a.data = make([]byte, 4)
	}
	var f float64
	f, e = strconv.ParseFloat(v, 32)
	if e != nil {
		return errStrInvalid("FP32", v)
	}
	return SetFP32FromFloat64(a, f)
}

func SetFP32FromFloat64(a *Atom, v float64) (e error) {
	if len(a.data) != 4 {
		a.data = make([]byte, 4)
	}
	if v > math.MaxFloat32 {
		return errRange("FP32", v)
	}
	var bits uint32 = math.Float32bits(float32(v))
	binary.BigEndian.PutUint32(a.data, bits)
	return
}

func SetFP64FromString(a *Atom, v string) (e error) {
	if len(a.data) != 8 {
		a.data = make([]byte, 8)
	}
	var f float64
	f, e = strconv.ParseFloat(v, 64)
	if e != nil {
		return errStrInvalid("FP64", v)
	}
	return SetFP64FromFloat64(a, f)
}

func SetFP64FromFloat64(a *Atom, v float64) (e error) {
	if len(a.data) != 8 {
		a.data = make([]byte, 8)
	}
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

func SetUF64FromString(a *Atom, v string) (e error) {
	// split string into whole and fractional parts
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

	binary.BigEndian.PutUint64(a.data, whole+uint64(fract))
	return
}

func SetUF64FromFloat64(a *Atom, v float64) (e error) {
	var i = uint64(v * 4294967296.0)
	binary.BigEndian.PutUint64(a.data, i)
	return
}

func SetSF32FromString(a *Atom, v string) (e error) {
	var f float64
	f, e = strconv.ParseFloat(v, 64)
	return SetSF32FromFloat64(a, f)
}

func SetSF32FromFloat64(a *Atom, v float64) (e error) {
	if -32768.0 > v || v >= 32768.0 {
		return errRange("SF32", v)
	}
	binary.BigEndian.PutUint32(a.data, uint32(v*65536.0))
	return
}

func SetSF64FromString(a *Atom, v string) (e error) {
	// split string into whole and fractional parts
	pieces := strings.Split(v, ".")
	if len(pieces) > 2 {
		return fmt.Errorf("invalid fixed point data:%s", v)
	}

	// whole part to the first 32 bits of a uint64
	var whole int64
	whole, e = strconv.ParseInt(pieces[0], 10, 64)
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

	// invert the bits in the fractional value, if negative number
	iFract := int64(fract)
	if whole < 0 {
		iFract = -1 * iFract
	}

	binary.BigEndian.PutUint64(a.data, uint64(whole+iFract))
	return
}

func SetSF64FromFloat64(a *Atom, v float64) (e error) {
	binary.BigEndian.PutUint64(a.data, uint64(v*4294967296.0))
	return
}

func SetFC32FromString(a *Atom, v string) (e error) {
	if len(a.data) != 4 {
		a.data = make([]byte, 4)
	}
	var buf = make([]byte, 0, 4)
	var extra string

	// nonprintable chars are allowed if the name is hex-encoded
	// this is because hex-encoded FCHR32 values may be generated
	switch len(v) {
	case 10: // 8 hex digits plus leading 0x
		if !strings.HasPrefix(v, "0x") {
			return fmt.Errorf("FC32 value is too long: (%s)", v)
		}
		matched, e := fmt.Sscanf(v, "0x%x%s", &buf, &extra)
		if e != io.EOF || matched != 1 {
			return errStrInvalid("FC32", v)
		}
	case 8: // 8 hex digits
		matched, e := fmt.Sscanf(v, "%x%s", &buf, &extra)
		if e != io.EOF || matched != 1 {
			return errStrInvalid("FC32", v)
		}
	case 6: // 4 printable chars, single quote delimited
		if !isPrintableString(v) {
			return fmt.Errorf("FC32 value is not printable: 0x%x", v)
		}
		if v[0] != '\'' || v[5] != '\'' {
			return fmt.Errorf("FC32 value is too long: (%s)", v)
		}
		buf = []byte(v)[1:5]
	case 4:
		buf = []byte(v)
	default:
		return errStrInvalid("FC32", v)
	}

	if len(buf) != 4 {
		return errStrInvalid("FC32", v)
	}
	copied := copy(a.data, buf)
	if copied != 4 {
		return fmt.Errorf("expected 4 chars copied for FC32 value(%s), got %d: e", v, copied)
	}
	return nil
}

func SetFC32FromUint64(a *Atom, v uint64) (e error) {
	if len(a.data) != 4 {
		a.data = make([]byte, 4)
	}
	if v > math.MaxUint32 {
		return errRange("FC32", v)
	}
	binary.BigEndian.PutUint32(a.data, uint32(v))
	return
}

// IP32 is usually a simple 4 bytes = 4 octets type, but it also has a
// rarely used multi-width form used to define a range.
// The double-width form seems to be expressed solely in hex.
func SetIP32FromString(a *Atom, v string) (e error) {
	// Set data to zero value in case of error
	if len(a.data) != 4 {
		a.data = make([]byte, 4)
	}

	// handle multi-address form separately
	if strings.HasPrefix(v, "0x") {
		return SetIP32FromHexString(a, v)
	}
	// Only a single IPv4 address is allowed from here on.

	// Extract 4 octets from string as decimal numbers
	var oct1, oct2, oct3, oct4 uint8
	var extra string
	matched, err := fmt.Sscanf(v, "%d.%d.%d.%d%s", &oct1, &oct2, &oct3, &oct4, &extra)
	if err != io.EOF || matched != 4 {
		return errStrInvalid("IP32", v)
	}
	copied := copy(a.data, []byte{oct1, oct2, oct3, oct4})
	if copied != 4 {
		e = fmt.Errorf("expected 4 bytes copied for IP32 value(%s), got %d: e", v, copied)
	}
	return
}

// Restrictions:
// string must start with "0x"
// following that must be only hex digits, in any number of sets of 8
func SetIP32FromHexString(a *Atom, v string) (e error) {
	if !strings.HasPrefix(v, "0x") {
		return errStrInvalid("IP32", v)
	}

	// allocate enough space
	size := len(v[2:])
	if 0 != size%8 || size == 0 {
		return errStrInvalid("IP32", v)
	}
	if len(a.data) != size {
		a.data = make([]byte, size/2)
	}

	// scan each chunk of 8 hex digits, and store as 4 byte address
	for i := 2; i < len(v); i += 8 {
		addr, err := strconv.ParseUint(v[i:i+8], 16, 32)
		if err != nil {
			a.data = make([]byte, 4) // zero before returning error
			return errStrInvalid("IP32", v)
		}
		iByte := (i - 2) / 2 // number of bytes seen so far
		binary.BigEndian.PutUint32(a.data[iByte:], uint32(addr))
	}
	return
}

func SetIP32FromUint64(a *Atom, v uint64) (e error) {
	if v > math.MaxUint32 {
		// store as 2 IPv4 addresses in 8 bytes
		if len(a.data) != 8 {
			a.data = make([]byte, 8)
		}
		binary.BigEndian.PutUint64(a.data, v)
	} else {
		// store as a single IPv4 address in 4 bytes
		if len(a.data) != 4 {
			a.data = make([]byte, 4)
		}
		binary.BigEndian.PutUint32(a.data, uint32(v))
	}
	return
}

func SetIPADFromString(a *Atom, v string) (e error) {
	size := len(v)
	buf := make([]byte, size)
	if len(v) < 3 && v != "::" {
		return errStrInvalid("IPAD", v)
	}
	copy(buf[:], v)

	// check for optional delimiters
	if buf[0] == '"' && buf[size-1] == '"' {
		buf = buf[1 : size-1] // ignore the delimiters from here on
	}

	// verify valid chars for IPv6
	chars := "0123456789abcdefABCDEF:."
	for _, r := range buf {
		if !strings.ContainsRune(chars, rune(r)) {
			return errStrInvalid("IPAD", v)
		}
	}

	buf = append(buf, '\x00') // add null terminator like a CSTR
	a.data = buf
	return
}

// No NULL terminator is used for this type
// Double-quote delimiters are optional on the input string
func SetUUIDFromString(a *Atom, v string) (e error) {
	if len(a.data) != 36 {
		a.data = make([]byte, 36)
	}

	// Read the UUID string into a UUID object, discarding delimiters
	var uuid uuidType
	size := len(v)
	if size == 38 && v[0] == '"' && v[size-1] == '"' {
		e = uuid.SetFromString(v[1 : size-1])
	} else {
		e = uuid.SetFromString(v)
	}
	if e != nil {
		return errStrInvalid("UUID", v)
	}

	// write raw bytes to Atom.data
	a.data = uuid.Bytes()
	return
}

// Uses NULL terminator
func SetCSTRFromQuotedEscapedString(a *Atom, v string) (e error) {

	// verify delimiters (required for strconv.Unquote() )
	size := len(v)
	if size < 2 || v[0] != '"' || v[size-1] != '"' {
		return fmt.Errorf("CSTR input string must be double-quoted: (%s)", v)
	}

	// unescape the string
	var s string
	s, e = strconv.Unquote(v)
	if e != nil {
		return errStrInvalid("CSTR", v)
	}

	// allocate space
	size = len(s)
	buf := make([]byte, size+1)
	copy(buf[:], s)

	buf[size] = '\x00'
	a.data = buf
	return
}

// Uses NULL terminator
func SetCSTRFromString(a *Atom, v string) (e error) {
	size := len(v)
	buf := make([]byte, size+1)
	copy(buf[:], v)
	buf[size] = '\x00'
	a.data = buf
	return
}

// SetUSTRFromQuotedEscapedString sets the atom data to the byte representation
// of the input string, which must conform to the following rules:
//     1) must start and end with double quotes. These are stripped before encoding.
//     2) special chars must be escaped with a backslash, including '"\
//     3) carriage return (\n) and line feed (\r) must be expressed as 2 literal chars: \n and \r respectively
//     4)  nonprintable chars are expressed with 4 literal characters \xXX, where the XX part is the hex value.
//
func SetUSTRFromQuotedEscapedString(a *Atom, v string) (e error) {

	// verify delimiters (required for strconv.Unquote() )
	if len(v) < 2 || (v[0] != '"' || v[len(v)-1] != '"') {
		return fmt.Errorf("USTR input string must be double-quoted: (%s)", v)
	}

	// unescape the string
	var s string
	s, e = strconv.Unquote(v)
	if e != nil {
		return errStrInvalid("USTR", v)
	}

	return SetUSTRFromString(a, s)
}

// SetUSTRFromString sets the atom data to the byte representation of the
// input string.
//
// The string is encoded as UTF32 big-endian (ie. 4 bytes for each rune, no
// variable-length encoding allowed.)
//
// No NULL terminator is used for this type, unlike CSTR.
func SetUSTRFromString(a *Atom, v string) (e error) {
	// write each rune as 4 bytes.
	buf := bytes.NewBuffer(make([]byte, 0, 4*len(v)))
	for _, r := range v { // iterate by rune, not byte
		// cast rune to uint32 to prevent implicit UTF-8 encoding
		e := binary.Write(buf, binary.BigEndian, uint32(r))
		if e != nil {
			return e
		}
	}
	a.data = buf.Bytes()
	return
}

func SetDATAFromHexString(a *Atom, v string) (e error) {

	// empty input string results in empty data section
	if len(v) == 0 {
		a.data = []byte{}
		return
	}

	// non-empty input must be strictly hex
	if !strings.HasPrefix(v, "0x") {
		return fmt.Errorf("hexadecimal string should start with 0x, got \"%s\"", v)
	}
	buffer, e := hex.DecodeString(v[2:])
	if e != nil {
		return
	}
	a.data = buffer
	return
}

func checkByteCount(buf []byte, bytesExpected int, strType string) (e error) {
	if len(buf) != bytesExpected {
		e = errByteCount(strType, bytesExpected, len(buf))
	}
	return
}

// UUID methods

// SetFromString initializes a UUID from a string.
// The string should be a properly formatted UUID string, including dashes, but
// without delimiters at the start and end.
func (p *uuidType) SetFromString(s string) (e error) {
	if !ValidUUIDString(s) {
		return errStrInvalid("UUID", s)
	}
	var sizes = []int{32, 16, 16, 16, 48} // index corresponds to UUID field
	var values = make([]uint64, 0, 5)     // index corresponds to UUID field
	for i, octet := range strings.Split(s, "-") {
		value, err := strconv.ParseUint(octet, 16, sizes[i])
		if err != nil {
			return errStrInvalid("UUID", s)
		}
		values = append(values, value)
	}
	return p.SetFromUints(values)
}

func (p *uuidType) SetFromUints(values []uint64) (e error) {
	if len(values) != 5 {
		return fmt.Errorf("invalid integer values for type UUID: %v", values)
	}
	p.TimeLow = uint32(values[0])
	p.TimeMid = uint16(values[1])
	p.TimeHiAndVersion = uint16(values[2])
	p.ClkSeqHiRes = uint8(values[3] >> 8)
	p.ClkSeqLow = uint8(values[3] & 0x00000000000000FF)

	var buf = make([]byte, 8)
	binary.BigEndian.PutUint64(buf, values[4])
	copy(p.Node[:], buf[2:])
	return
}

// Bytes returns the UUID data as a slice of bytes.
func (u uuidType) Bytes() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, u)
	return buf.Bytes()
}

func (u uuidType) String() string {
	return fmt.Sprintf(
		"%08X-%04X-%04X-%02X%02X-%012X",
		u.TimeLow,
		u.TimeMid,
		u.TimeHiAndVersion,
		u.ClkSeqHiRes,
		u.ClkSeqLow,
		u.Node)
}

// ValidUUIDString returns true if a string contains a properly formatted UUID.
// Example: 64881431-B6DC-478E-B7EE-ED306619C797
func ValidUUIDString(s string) bool {
	// verify valid chars
	for _, c := range s {
		if !strings.ContainsRune("0123456789abcdefABCDEF-", rune(c)) {
			return false
		}
	}
	// Verify format
	groups := strings.Split(string(s), "-")
	return len(groups) == 5 &&
		len(groups[0]) == 8 &&
		len(groups[1]) == 4 &&
		len(groups[2]) == 4 &&
		len(groups[3]) == 4 &&
		len(groups[4]) == 12
}
