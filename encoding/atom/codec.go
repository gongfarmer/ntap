package atom

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"reflect"
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
	CONT ADEType = "CONT" // AtomContainer
)
const SHIFT4 = 0x00010000
const SHIFT8 = 0x100000000

/**********************************************************/

// decOp is the signature of a decoding operator for a given type.
type decOp func(buf []byte) (value reflect.Value)

// decOp is the signature of a function that prints value as a string for a given type
type strOp func(buf []byte) string

type Operators struct {
	Decode decOp
	String strOp
}

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
	UF64: Operators{decUI64, strUF64},
	SF32: Operators{decSF32, strSF32},
	SF64: Operators{decSF64, strSF64},
	UR32: Operators{decUR32, strUR32},
	UR64: Operators{decUR64, strUR64},
	SR32: Operators{decSR32, strSR32},
	SR64: Operators{decSR64, strSR64},
	FC32: Operators{decFC32, strFC32},
	IP32: Operators{decIP32, strIP32},
	IPAD: Operators{decUSTR, strUSTR},
	CSTR: Operators{decCSTR, strCSTR},
	USTR: Operators{decUSTR, strUSTR},
	DATA: Operators{decDATA, strDATA},
	CNCT: Operators{decDATA, strDATA},
	ENUM: Operators{decSI32, strSI32},
	UUID: Operators{decDATA, strUUID},
	NULL: Operators{decNULL, strNULL},
	CONT: Operators{decNULL, strNULL},
}

/**********************************************************/
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
func strUF64(buf []byte) string {
	return fmt.Sprintf("%0.8f", decUF64(buf).Float())
}
func strSF32(buf []byte) string {
	return fmt.Sprintf("%0.4f", decSF32(buf).Float())
}
func strSF64(buf []byte) string {
	return fmt.Sprintf("%0.9f", decSF64(buf).Float())
}
func strUR32(buf []byte) string {
	arr := decUR32(buf).Interface().([2]uint16)
	var numerator, denominator uint16 = arr[0], arr[1]
	return fmt.Sprintf("%d/%d", numerator, denominator)
}
func strUR64(buf []byte) string {
	arr := decUR64(buf).Interface().([2]uint32)
	var numerator, denominator uint32 = arr[0], arr[1]
	return fmt.Sprintf("%d/%d", numerator, denominator)
}
func strSR32(buf []byte) string {
	arr := decSR32(buf).Interface().([2]int16)
	var numerator, denominator int16 = arr[0], arr[1]
	return fmt.Sprintf("%d/%d", numerator, denominator)
}
func strSR64(buf []byte) string {
	arr := decSR64(buf).Interface().([2]int32)
	var numerator, denominator int32 = arr[0], arr[1]
	return fmt.Sprintf("%d/%d", numerator, denominator)
}
func strFC32(buf []byte) string {
	if isPrintableBytes(buf) {
		return fmt.Sprintf("'%s'", string(buf))
	} else {
		return fmt.Sprintf("%#08X", buf)
	}
}
func strIP32(buf []byte) string {
	return fmt.Sprintf("%d.%d.%d.%d", buf[0], buf[1], buf[2], buf[3])
}
func strCSTR(buf []byte) string {
	return fmt.Sprintf("\"%q\"", string(buf))
}
func strUSTR(buf []byte) string {
	return fmt.Sprintf("\"%s\"", string(buf))
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
	err := binary.Read(bytes.NewReader(buf), binary.BigEndian, &v)
	if err != io.EOF && err != nil {
		panic(err)
	}
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

func decUI01(buf []byte) (value reflect.Value) {
	var v uint32
	err := binary.Read(bytes.NewReader(buf), binary.BigEndian, &v)
	if err != io.EOF && err != nil {
		panic(err)
	}
	t := false
	if v == 1 {
		t = true
	}
	value = reflect.ValueOf(t)
	return
}
func decUI08(buf []byte) (value reflect.Value) {
	var v uint8
	err := binary.Read(bytes.NewReader(buf), binary.BigEndian, &v)
	if err != io.EOF && err != nil {
		panic(err)
	}
	value = reflect.ValueOf(v)
	return
}
func decUI16(buf []byte) (value reflect.Value) {
	var v uint16
	err := binary.Read(bytes.NewReader(buf), binary.BigEndian, &v)
	if err != io.EOF && err != nil {
		panic(err)
	}
	value = reflect.ValueOf(v)
	return
}
func decUI32(buf []byte) (value reflect.Value) {
	var v uint32
	err := binary.Read(bytes.NewReader(buf), binary.BigEndian, &v)
	if err != io.EOF && err != nil {
		panic(err)
	}
	value = reflect.ValueOf(v)
	return
}
func decUI64(buf []byte) (value reflect.Value) {
	var v uint64
	err := binary.Read(bytes.NewReader(buf), binary.BigEndian, &v)
	if err != io.EOF && err != nil {
		panic(err)
	}
	value = reflect.ValueOf(v)
	return
}
func decSF32(buf []byte) (value reflect.Value) {
	v := float32(decSI32(buf).Interface().(int32)) / SHIFT4
	value = reflect.ValueOf(v)
	return
}
func decSF64(buf []byte) (value reflect.Value) {
	v := float64(decSI64(buf).Interface().(int64)) / SHIFT8
	value = reflect.ValueOf(v)
	return
}
func decSI08(buf []byte) (value reflect.Value) {
	var v int8
	err := binary.Read(bytes.NewReader(buf), binary.BigEndian, &v)
	if err != io.EOF && err != nil {
		panic(err)
	}
	value = reflect.ValueOf(v)
	return
}
func decSI16(buf []byte) (value reflect.Value) {
	var v int16
	err := binary.Read(bytes.NewReader(buf), binary.BigEndian, &v)
	if err != io.EOF && err != nil {
		panic(err)
	}
	value = reflect.ValueOf(v)
	return
}
func decSI32(buf []byte) (value reflect.Value) {
	var v int32
	err := binary.Read(bytes.NewReader(buf), binary.BigEndian, &v)
	if err != io.EOF && err != nil {
		panic(err)
	}
	value = reflect.ValueOf(v)
	return
}
func decSI64(buf []byte) (value reflect.Value) {
	var v int64
	err := binary.Read(bytes.NewReader(buf), binary.BigEndian, &v)
	if err != io.EOF && err != nil {
		panic(err)
	}
	value = reflect.ValueOf(v)
	return
}
func decFP32(buf []byte) (value reflect.Value) {
	var v float32
	err := binary.Read(bytes.NewReader(buf), binary.BigEndian, &v)
	if err != io.EOF && err != nil {
		panic(err)
	}
	value = reflect.ValueOf(v)
	return
}
func decFP64(buf []byte) (value reflect.Value) {
	var v float64
	err := binary.Read(bytes.NewReader(buf), binary.BigEndian, &v)
	if err != io.EOF && err != nil {
		panic(err)
	}
	value = reflect.ValueOf(v)
	return
}
func decUF32(buf []byte) (value reflect.Value) {
	var v float32 = float32(decUI32(buf).Uint()) / SHIFT4
	value = reflect.ValueOf(v)
	return
}
func decUF64(buf []byte) (value reflect.Value) {
	var v float64 = float64(decUI64(buf).Uint()) / SHIFT8
	value = reflect.ValueOf(v)
	return
}

func decUR32(buf []byte) (value reflect.Value) {
	var arr [2]uint16
	err := binary.Read(bytes.NewReader(buf), binary.BigEndian, &arr)
	if err != io.EOF && err != nil {
		panic(err)
	}
	value = reflect.ValueOf(arr)
	return
}
func decUR64(buf []byte) (value reflect.Value) {
	var arr [2]uint32
	err := binary.Read(bytes.NewReader(buf), binary.BigEndian, &arr)
	if err != io.EOF && err != nil {
		panic(err)
	}
	value = reflect.ValueOf(arr)
	return
}

// ADE ccat puts the sign on the denominator
func decSR32(buf []byte) (value reflect.Value) {
	var arr [2]int16
	err := binary.Read(bytes.NewReader(buf), binary.BigEndian, &arr)
	if err != io.EOF && err != nil {
		panic(err)
	}
	value = reflect.ValueOf(arr)
	return
}

func decSR64(buf []byte) (value reflect.Value) {
	var arr [2]int32
	err := binary.Read(bytes.NewReader(buf), binary.BigEndian, &arr)
	if err != io.EOF && err != nil {
		panic(err)
	}
	value = reflect.ValueOf(arr)
	return
}
func decFC32(buf []byte) (value reflect.Value) {
	var s = string(buf[:])
	value = reflect.ValueOf(s)
	return
}

// Return [4]byte, same way IPv4 is represented in Go's net/ library
func decIP32(buf []byte) (value reflect.Value) {
	value = reflect.ValueOf(buf)
	return
}

// FIXME: has invisible NULL char at end
func decIPAD(buf []byte) (value reflect.Value) {
	s := string(buf)
	value = reflect.ValueOf(s)
	return
}

// FIXME: has invisible NULL char at end
func decCSTR(buf []byte) (value reflect.Value) {
	s := string(buf)
	value = reflect.ValueOf(s)
	return
}
func decUSTR(buf []byte) (value reflect.Value) {
	s := string(buf)
	value = reflect.ValueOf(s)
	return
}
func decDATA(buf []byte) (value reflect.Value) {
	value = reflect.ValueOf(buf)
	return
}
func decNULL(buf []byte) (value reflect.Value) {
	value = reflect.ValueOf(nil)
	return
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

// bounds checking is implicit since uint32 cannot hold invalid values for UI32
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
				err = fmt.Errorf("Invalid value for type %s: %d", adeType, v)
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
