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
	UI01 = iota // unsigned int / bool
	UI08        // unsigned int
	SI08        // signed int
	UI16        // unsigned int
	SI16        // signed int
	UI32        // unsigned int
	SI32        // signed int
	UI64        // unsigned int
	SI64        // signed int
	FP32        // floating point
	FP64        // floating point
	UF32        // unsigned fixed point (integer part / fractional part)
	SF32        // signed fixed point   (integer part / fractional part)
	UF64        // unsigned fixed point (integer part / fractional part)
	SF64        // signed fixed point   (integer part / fractional part)
	UR32        // unsigned fraction
	SR32        // signed fraction
	UR64        // unsigned fraction
	SR64        // unsigned fraction
	FC32        // four char string
	IP32        // ipv4 address
	IPAD        // ipv4 or ipv6 address
	CSTR        // C string
	USTR        // unicode string
	DATA        // Raw data or equivalent
	ENUM        // Enumeration
	UUID        // output: DEC88E51-D85B-4425-8808-2168BC362443, input: same or hex
	NULL        // NULL type, must have empty data section
	CNCT        // binary data printed as hexadecimal value with leading 0x
	CONT
)

var adeTypeMap = map[string]int{
	"UI01": UI01,
	"UI08": UI08,
	"SI08": SI08,
	"UI16": UI16,
	"SI16": SI16,
	"UI32": UI32,
	"SI32": SI32,
	"UI64": UI64,
	"SI64": SI64,
	"FP32": FP32,
	"FP64": FP64,
	"UF32": UF32,
	"SF32": SF32,
	"UF64": UF64,
	"SF64": SF64,
	"UR32": UR32,
	"SR32": SR32,
	"UR64": UR64,
	"SR64": SR64,
	"FC32": FC32,
	"IP32": IP32,
	"IPAD": IPAD,
	"CSTR": CSTR,
	"USTR": USTR,
	"DATA": DATA,
	"ENUM": ENUM,
	"UUID": UUID,
	"NULL": NULL,
	"CNCT": CNCT,
}

/**********************************************************/

// decOp is the signature of a decoding operator for a given type.
type decOp func(buf []byte, value *reflect.Value)

var decOpTable = [...]decOp{
	UI01: decUI01,
	UI08: decUI08,
	UI16: decUI16,
	UI32: decUI32,
	UI64: decUI64,
	SI08: decSI08,
	SI16: decSI16,
	SI32: decSI32,
	SI64: decSI64,
	FP32: decFP32,
	FP64: decFP64,
	UF32: decUF32,
	UF64: decUF64,
	SF32: decSF32,
	SF64: decSF64,
	UR32: decUR32,
	SR32: decSR32,
	UR64: decUR64,
	SR64: decSR64,
	FC32: decFC32,
	IP32: decIP32,
	IPAD: decIPAD,
	CSTR: decCSTR,
	USTR: decUSTR,
	DATA: decDATA,
	CNCT: decDATA,
	ENUM: decENUM,
	UUID: decUUID,
}

/**********************************************************/

func decUI01(buf []byte, value *reflect.Value) {
	var v uint32
	err := binary.Read(bytes.NewReader(buf), binary.BigEndian, &v)
	if err != io.EOF && err != nil {
		panic(err)
	}
	t := false
	if v == 1 {
		t = true
	}
	*value = reflect.ValueOf(t)
}
func decUI08(buf []byte, value *reflect.Value) {
	var v uint8
	err := binary.Read(bytes.NewReader(buf), binary.BigEndian, &v)
	if err != io.EOF && err != nil {
		panic(err)
	}
	*value = reflect.ValueOf(v)
}
func decUI16(buf []byte, value *reflect.Value) {
	var v uint16
	err := binary.Read(bytes.NewReader(buf), binary.BigEndian, &v)
	if err != io.EOF && err != nil {
		panic(err)
	}
	*value = reflect.ValueOf(v)
}
func decUI32(buf []byte, value *reflect.Value) {
	var v uint32
	err := binary.Read(bytes.NewReader(buf), binary.BigEndian, &v)
	if err != io.EOF && err != nil {
		panic(err)
	}
	*value = reflect.ValueOf(v)
}
func decUI64(buf []byte, value *reflect.Value) {
	var v uint64
	err := binary.Read(bytes.NewReader(buf), binary.BigEndian, &v)
	if err != io.EOF && err != nil {
		panic(err)
	}
	*value = reflect.ValueOf(v)
}
func decSI08(buf []byte, value *reflect.Value) {
	var v int8
	err := binary.Read(bytes.NewReader(buf), binary.BigEndian, &v)
	if err != io.EOF && err != nil {
		panic(err)
	}
	*value = reflect.ValueOf(v)
}
func decSI16(buf []byte, value *reflect.Value) {
	var v int16
	err := binary.Read(bytes.NewReader(buf), binary.BigEndian, &v)
	if err != io.EOF && err != nil {
		panic(err)
	}
	*value = reflect.ValueOf(v)
}
func decSI32(buf []byte, value *reflect.Value) {
	var v int32
	err := binary.Read(bytes.NewReader(buf), binary.BigEndian, &v)
	if err != io.EOF && err != nil {
		panic(err)
	}
	*value = reflect.ValueOf(v)
}
func decSI64(buf []byte, value *reflect.Value) {
	var v int64
	err := binary.Read(bytes.NewReader(buf), binary.BigEndian, &v)
	if err != io.EOF && err != nil {
		panic(err)
	}
	*value = reflect.ValueOf(v)
}
func decFP32(buf []byte, value *reflect.Value) {
	var v int32
	err := binary.Read(bytes.NewReader(buf), binary.BigEndian, &v)
	if err != io.EOF && err != nil {
		panic(err)
	}
	*value = reflect.ValueOf(v)
}
func decFP64(buf []byte, value *reflect.Value) {
	var v int64
	err := binary.Read(bytes.NewReader(buf), binary.BigEndian, &v)
	if err != io.EOF && err != nil {
		panic(err)
	}
	*value = reflect.ValueOf(v)
}
func decUF32(buf []byte, value *reflect.Value) {
	var integer, fractional uint16
	err := binary.Read(bytes.NewReader(buf), binary.BigEndian, &integer)
	if err != io.EOF && err != nil {
		panic(err)
	}
	err = binary.Read(bytes.NewReader(buf), binary.BigEndian, &fractional)
	if err != io.EOF && err != nil {
		panic(err)
	}
	var v float64 = float64(integer)
	v += (1.0 / float64(fractional))
	*value = reflect.ValueOf(v)
}
func decUF64(buf []byte, value *reflect.Value) {
	var integer, fractional uint32
	err := binary.Read(bytes.NewReader(buf), binary.BigEndian, &integer)
	if err != io.EOF && err != nil {
		panic(err)
	}
	err = binary.Read(bytes.NewReader(buf), binary.BigEndian, &fractional)
	if err != io.EOF && err != nil {
		panic(err)
	}
	var v float64 = float64(integer)
	v += (1.0 / float64(fractional))
	*value = reflect.ValueOf(v)
}
func decSF32(buf []byte, value *reflect.Value) {
	var integer, fractional int16
	err := binary.Read(bytes.NewReader(buf), binary.BigEndian, &integer)
	if err != io.EOF && err != nil {
		panic(err)
	}
	err = binary.Read(bytes.NewReader(buf), binary.BigEndian, &fractional)
	if err != io.EOF && err != nil {
		panic(err)
	}
	var v float64 = float64(integer)
	v += (1.0 / float64(fractional))
	*value = reflect.ValueOf(v)
}
func decSF64(buf []byte, value *reflect.Value) {
	var integer, fractional int32
	err := binary.Read(bytes.NewReader(buf), binary.BigEndian, &integer)
	if err != io.EOF && err != nil {
		panic(err)
	}
	err = binary.Read(bytes.NewReader(buf), binary.BigEndian, &fractional)
	if err != io.EOF && err != nil {
		panic(err)
	}
	var v float64 = float64(integer)
	v += (1.0 / float64(fractional))
	*value = reflect.ValueOf(v)
}
func decUR32(buf []byte, value *reflect.Value) {
	var numerator, denominator uint16
	err := binary.Read(bytes.NewReader(buf), binary.BigEndian, &numerator)
	if err != io.EOF && err != nil {
		panic(err)
	}
	err = binary.Read(bytes.NewReader(buf), binary.BigEndian, &denominator)
	if err != io.EOF && err != nil {
		panic(err)
	}
	var v = [2]uint16{numerator, denominator}
	*value = reflect.ValueOf(v)
}
func decUR64(buf []byte, value *reflect.Value) {
	var numerator, denominator uint32
	err := binary.Read(bytes.NewReader(buf), binary.BigEndian, &numerator)
	if err != io.EOF && err != nil {
		panic(err)
	}
	err = binary.Read(bytes.NewReader(buf), binary.BigEndian, &denominator)
	if err != io.EOF && err != nil {
		panic(err)
	}
	var v = [2]uint32{numerator, denominator}
	*value = reflect.ValueOf(v)
}
func decSR32(buf []byte, value *reflect.Value) {
	var numerator, denominator int16
	err := binary.Read(bytes.NewReader(buf), binary.BigEndian, &numerator)
	if err != io.EOF && err != nil {
		panic(err)
	}
	err = binary.Read(bytes.NewReader(buf), binary.BigEndian, &denominator)
	if err != io.EOF && err != nil {
		panic(err)
	}
	var v = [2]int16{numerator, denominator}
	*value = reflect.ValueOf(v)
}
func decSR64(buf []byte, value *reflect.Value) {
	var numerator, denominator int32
	err := binary.Read(bytes.NewReader(buf), binary.BigEndian, &numerator)
	if err != io.EOF && err != nil {
		panic(err)
	}
	err = binary.Read(bytes.NewReader(buf), binary.BigEndian, &denominator)
	if err != io.EOF && err != nil {
		panic(err)
	}
	var v = [2]int32{numerator, denominator}
	*value = reflect.ValueOf(v)
}
func decFC32(buf []byte, value *reflect.Value) {
	var v [4]byte
	err := binary.Read(bytes.NewReader(buf), binary.BigEndian, &v)
	if err != io.EOF && err != nil {
		panic(err)
	}
	var s = string(v[:])
	*value = reflect.ValueOf(s)
}
func decIP32(buf []byte, value *reflect.Value) {
	var v [4]byte
	err := binary.Read(bytes.NewReader(buf), binary.BigEndian, &v)
	if err != io.EOF && err != nil {
		panic(err)
	}
	s := fmt.Sprintf("%d.%d.%d.%d", v[0], v[1], v[2], v[3])
	*value = reflect.ValueOf(s)
}
func decIPAD(buf []byte, value *reflect.Value) {
	var v []byte
	err := binary.Read(bytes.NewReader(buf), binary.BigEndian, &v)
	if err != io.EOF && err != nil {
		panic(err)
	}
	s := string(v)
	*value = reflect.ValueOf(s)
}
func decENUM(buf []byte, value *reflect.Value) {
	var v int32
	err := binary.Read(bytes.NewReader(buf), binary.BigEndian, &v)
	if err != io.EOF && err != nil {
		panic(err)
	}
	*value = reflect.ValueOf(v)
}
func decCSTR(buf []byte, value *reflect.Value) {
	s := string(buf)
	*value = reflect.ValueOf(s)
}
func decUSTR(buf []byte, value *reflect.Value) {
	s := string(buf)
	*value = reflect.ValueOf(s)
}
func decDATA(buf []byte, value *reflect.Value) {
	*value = reflect.ValueOf(buf)
}

// UUID - 128 bit
// variant must be RFC4122/DCE (10b==2d)
// high 2 bits of octet 8 are variant as per RFC
// version must be one of the five defined in the RFC (1d-5d)
// high 4 bits of octet 6 are version as per RFC
// UUID_NULL_STRING "00000000-0000-0000-0000-000000000000"
func decUUID(buf []byte, value *reflect.Value) {
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
	var s = fmt.Sprintf("%08X-%04X-%04X-%02X%02X-%012X", v.TimeLow, v.TimeMid, v.TimeHiAndVersion, v.ClkSeqHiRes, v.ClkSeqLow, v.Node)
	*value = reflect.ValueOf(s)
}

/**********************************************************/

// Called on a container, create atom at the given path if not exist, and set to given value
func (a Atom) SetUI32(path string, value uint32) (err error) {
	return
}

// bounds checking is implicit since uint32 cannot hold invalid values for UI32
func (a Atom) SetValue(adeType int, value interface{}) (err error) {
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
