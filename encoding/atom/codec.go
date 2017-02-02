package atom

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
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
	}
	codec struct {
		Atom    *Atom
		Decoder decoder
		Encoder encoder
	}
)

// NewCodec returns a codec that performs type conversion for atom data.
// It provides methods to convert data from an atom's ADE type into suitable Go
// types, and vice versa.
func NewCodec(a *Atom) *codec {
	c := codec{
		Atom:    a,
		Decoder: decoderByType[a.Type],
		Encoder: encoderByType[a.Type],
	}
	return &c
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

func noDecoder(from ADEType, to GoType) error {
	return fmt.Errorf("no decoder exists to convert ADE type %s to go type %s.", from, to)
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

func Round(f float64) float64 {
	return math.Floor(f + .5)
}
func RoundPlaces(f float64, places int) float64 {
	shift := math.Pow(10, float64(places))
	return Round(f*shift) / shift
}
