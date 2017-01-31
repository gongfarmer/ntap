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
// The ADE code C-type mappings are in OSL_Types.h
const (
	UI01 ADEType = "UI01" // unsigned int / bool
	UI08 ADEType = "UI08" // unsigned int
	SI08 ADEType = "SI08" // signed int
	UI16 ADEType = "UI16" // unsigned int
	SI16 ADEType = "SI16" // signed int
	UI32 ADEType = "UI32" // unsigned int
	SI32 ADEType = "SI32" // signed int
	UI64 ADEType = "UI64" // unsigned int
	SI64 ADEType = "SI64" // signed int
	FP32 ADEType = "FP32" // floating point
	FP64 ADEType = "FP64" // floating point
	UF32 ADEType = "UF32" // unsigned fixed point (integer part / fractional part)
	SF32 ADEType = "SF32" // signed fixed point   (integer part / fractional part)
	UF64 ADEType = "UF64" // unsigned fixed point (integer part / fractional part)
	SF64 ADEType = "SF64" // signed fixed point   (integer part / fractional part)
	UR32 ADEType = "UR32" // unsigned fraction
	SR32 ADEType = "SR32" // signed fraction
	UR64 ADEType = "UR64" // unsigned fraction
	SR64 ADEType = "SR64" // unsigned fraction
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
	IPAD: Operators{decUSTR, strCSTR},
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
	var s = string(buf[:])
	return reflect.ValueOf(s)
}

// Store as [4]byte, same way IPv4 is represented in Go's net/ library
func decIP32(buf []byte) reflect.Value {
	return reflect.ValueOf(buf)
}
func decIPAD(buf []byte) reflect.Value {
	s := string(buf)
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
	fmt.Printf("% 8x  %d/%d\n", buf, arr[0], arr[1])
	return fmt.Sprintf("%d/%d", arr[0], arr[1])
}
func strSR64(buf []byte) string {
	arr := decSR64(buf).Interface().([2]int32)
	return fmt.Sprintf("%d/%d", arr[0], arr[1])
}
func strFC32(buf []byte) string {
	if isPrintableBytes(buf) {
		return fmt.Sprintf("'%s'", string(buf))
	} else {
		return fmt.Sprintf("0x%08X", buf)
	}
}
func strIP32(buf []byte) string {
	return fmt.Sprintf("%d.%d.%d.%d", buf[0], buf[1], buf[2], buf[3])
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
	v := reflect.ValueOf(value)
	switch adeType {
	case CONT, NULL:
		err = fmt.Errorf("ADE type %s cannot take a value", adeType)
	case UI01:
		switch v.Kind() {
		case reflect.Bool:
			if v.Bool() == true {
				binary.BigEndian.PutUint32(a.Data, uint32(1))
			} else {
				binary.BigEndian.PutUint32(a.Data, uint32(0))
			}
		case
			reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if v.Uint() == 0 || v.Uint() == 1 {
				v := uint32(v.Uint())
				binary.BigEndian.PutUint32(a.Data, v)
			} else {
				err = fmt.Errorf("Invalid value for type %s: %s", adeType, v.Kind())
			}
		default:
			err = fmt.Errorf("Invalid go type (%s) given for conversion to ADE type %d", v.Type(), adeType)
		}
	case UI08:
		switch v.Kind() {
		case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			fallthrough
		case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			v := v.Uint()
			if v >= 0 && v <= 0xFF {
				a.Data[0] = byte(v)
			} else {
				err = fmt.Errorf("Invalid value for type %s: %d", adeType, v)
			}
		default:
			err = fmt.Errorf("Invalid go type (%s) given for conversion to ADE type %d", v.Type(), adeType)
		}
	case UI16:
		switch v.Kind() {
		case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			fallthrough
		case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			v := v.Uint()
			if v >= 0 && v <= 0xFFFF {
				binary.BigEndian.PutUint16(a.Data, uint16(v))
			} else {
				err = fmt.Errorf("Invalid value for type %s: %d", adeType, v)
			}
		default:
			err = fmt.Errorf("Invalid go type (%s) given for conversion to ADE type %d", v.Type(), adeType)
		}
	case UI32:
		switch v.Kind() {
		case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			fallthrough
		case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			v := v.Uint() // FIXME: what if it was an SINT originally?
			if v >= 0 && v <= 0xFFFFFFFF {
				binary.BigEndian.PutUint32(a.Data, uint32(v))
			} else {
				err = fmt.Errorf("Invalid value for type %s: %d", adeType, v)
			}
		default:
			err = fmt.Errorf("Invalid go type (%s) given for conversion to ADE type %d", v.Type(), adeType)
		}
	case UI64:
		switch v.Kind() {
		case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			fallthrough
		case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			v := v.Uint() // FIXME: what if it was an SINT originally?
			if v >= 0 {
				binary.BigEndian.PutUint64(a.Data, uint64(v))
			} else {
				err = fmt.Errorf("Invalid value for type %s: %d", adeType, v)
			}
		default:
			err = fmt.Errorf("Invalid go type (%s) given for conversion to ADE type %d", v.Type(), adeType)
		}
	default:
		err = fmt.Errorf("unknown ADE type %d", adeType)
	}
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
