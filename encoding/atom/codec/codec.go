package codec

// This file provides methods for interpreting and conversion of ADE Atom data types.
//
// ADE Data types are defined in 112-0002_r4.0B_StorageGRID_Data_Types.
//
// The ADE headers for these types are in OSL_Types.h.
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
	"unicode/utf8"
)

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

	goString goType = "String"
	goUint   goType = "Uint"
	goInt    goType = "Int"
	goBool   goType = "Bool"
	goBytes  goType = "Bytes"
	goFloat  goType = "Float"
)

var PrintableChars = strPrintableChars()

// returns string of all printable chars < ascii 127, excludes whitespace
func strPrintableChars() string {
	var b = make([]byte, 0, 0x7f-0x21) // ascii char values
	for c := byte(0x21); c < 0x7f; c++ {
		b = append(b, c)
	}
	return string(b)
}

//**********************************************************
// Codec / Encoder / Decoder data structure definitions

type (
	// ADEType uniquely identifies ADE Atom types as 4 character string enums.
	ADEType string

	// goType identifies some basic go data types
	goType string

	// A Codec (coder/decoder) object provides access to the data value of an atom.
	// It contains a full set of getter and setter methods which accept or return
	// the atom data as various Go native types.
	// Go types that don't make sense for a given ADE type (eg. SliceOfInt for a
	// string type) simply return an error when called.
	Codec struct {
		typ     ADEType
		dataPtr *[]byte
		Decoder decoder
		Encoder encoder
	}
	decoder struct {
		String          func(buf []byte) (string, error)
		StringDelimited func(buf []byte) (string, error)
		Bool            func(buf []byte) (bool, error)
		Uint            func(buf []byte) (uint64, error)
		Int             func(buf []byte) (int64, error)
		Float           func(buf []byte) (float64, error)
		SliceOfUint     func(buf []byte) ([]uint64, error)
		SliceOfInt      func(buf []byte) ([]int64, error)
		SliceOfByte     func(buf []byte) ([]byte, error)
	}
	encoder struct {
		SetString          func(*[]byte, string) error
		SetStringDelimited func(*[]byte, string) error
		SetBool            func(*[]byte, bool) error
		SetUint            func(*[]byte, uint64) error
		SetInt             func(*[]byte, int64) error
		SetFloat           func(*[]byte, float64) error
		SetSliceOfUint     func(*[]byte, []uint64) error
		SetSliceOfInt      func(*[]byte, []int64) error
		SetSliceOfByte     func(*[]byte, []byte) error
	}

	uuidType struct {
		TimeLow          uint32
		TimeMid          uint16
		TimeHiAndVersion uint16
		ClkSeqHiRes      uint8
		ClkSeqLow        uint8
		Node             [6]byte
	}
)

//**********************************************************
// error construction functions
// These exist so that the unit tests don't have to hardcode the err message they expect

func errNoEncoder(to ADEType, from string) error {
	return fmt.Errorf("no encoder exists to convert go type '%s' to ADE type '%s'", from, to)
}
func errNoDecoder(from ADEType, to goType) error {
	return fmt.Errorf("no decoder exists to convert ADE type '%s' to go type '%s'", from, to)
}
func errByteCount(t string, bytesWant int, bytesGot int) (e error) {
	return fmt.Errorf("invalid byte count for ADE type %s: want %d, got %d", t, bytesWant, bytesGot)
}
func errStrInvalid(t string, v string) error {
	return fmt.Errorf("invalid string value for ADE type %s: \"%s\"", t, strconv.Quote(v))
}
func errRange(t string, v interface{}) (e error) {
	switch v := v.(type) {
	case uint, uint8, uint16, uint32, uint64, int, int32, int64:
		e = fmt.Errorf("value exceeds range of type %s: %d", t, v)
	case float32, float64:
		e = fmt.Errorf("value exceeds range of type %s: %0.9f", t, v)
	case []uint64, []int64:
		e = fmt.Errorf("value exceeds range of type %s: %v", t, v)
	case string:
		e = fmt.Errorf("value exceeds range of type %s: %v", t, v)
	default:
		e = fmt.Errorf("range error cannot handle type %T", v)
	}
	return
}
func errInvalidEscape(t, v, note string) (e error) {
	if note == "" {
		e = fmt.Errorf("invalid escape sequence in %s value: %s", t, v)
	} else {
		e = fmt.Errorf("invalid escape sequence in %s value \"%s\": %s", t, v, note)
	}
	return
}
func errUnescaped(typ string, r rune) error {
	return fmt.Errorf("character %s must be escaped in %s value", strconv.QuoteRune(r), typ)
}
func errZeroDenominator(typ string, v string) (e error) {
	if v == "" {
		e = fmt.Errorf("fractional type %s forbids zero in denominator", typ)
	} else {
		e = fmt.Errorf("fractional type %s forbids zero in denominator, got \"%s\"", typ, v)
	}
	return
}

//**********************************************************

// NewCodec returns a Codec that performs type conversion for atom data.
// A Codec provides encoder/decoder methods for converting data from an atom's
// ADE type into suitable Go types, and vice versa.
func NewCodec(dataPtr *[]byte, atomType ADEType) *Codec {
	c := Codec{
		dataPtr: dataPtr,
		typ:     atomType,
		Decoder: decoderByType[atomType],
		Encoder: encoderByType[atomType],
	}
	return &c
}

// newEncoder returns a new encoder that provides functions for converting Go
// native types into ADE Atom data.  The returned encoder contains all of the
// default encoding methods, which simply return an error stating that the
// encoding is not supported.
// The caller should implement whichever encoding methods are appropriate for
// the ADE data type's Codec.
func newEncoder(from ADEType) encoder {
	return encoder{
		SetString:          func(_ *[]byte, v string) (e error) { return errNoEncoder(from, "string") },
		SetStringDelimited: func(_ *[]byte, v string) (e error) { return errNoEncoder(from, "string(delimited)") },
		SetBool:            func(_ *[]byte, v bool) (e error) { return errNoEncoder(from, "bool") },
		SetUint:            func(_ *[]byte, v uint64) (e error) { return errNoEncoder(from, "uint64") },
		SetInt:             func(_ *[]byte, v int64) (e error) { return errNoEncoder(from, "int64") },
		SetFloat:           func(_ *[]byte, v float64) (e error) { return errNoEncoder(from, "float64") },
		SetSliceOfUint:     func(_ *[]byte, v []uint64) (e error) { return errNoEncoder(from, "[]uint64") },
		SetSliceOfInt:      func(_ *[]byte, v []int64) (e error) { return errNoEncoder(from, "[]int64") },
		SetSliceOfByte:     func(_ *[]byte, v []byte) (e error) { return errNoEncoder(from, "[]byte") },
	}
}

// newDecoder returns a new decoder that provides functions for converting ADE
// Atom data into Go native types .  The returned decoder contains all of the
// default decdoer methods, which simply return an error stating that the
// decoding to the requested type is not supported.
// The caller should implement whichever decoding methods are appropriate for
// the ADE data type.
func newDecoder(from ADEType) decoder {
	return decoder{
		String:          func([]byte) (v string, e error) { return v, errNoDecoder(from, "string") },
		StringDelimited: func([]byte) (v string, e error) { return v, errNoDecoder(from, "string(delimited)") },
		Bool:            func([]byte) (v bool, e error) { return v, errNoDecoder(from, "bool") },
		Uint:            func([]byte) (v uint64, e error) { return v, errNoDecoder(from, "uint64") },
		Int:             func([]byte) (v int64, e error) { return v, errNoDecoder(from, "int64") },
		Float:           func([]byte) (v float64, e error) { return v, errNoDecoder(from, "float64") },
		SliceOfUint:     func([]byte) (v []uint64, e error) { return v, errNoDecoder(from, "[]uint64") },
		SliceOfInt:      func([]byte) (v []int64, e error) { return v, errNoDecoder(from, "[]int64") },
		SliceOfByte:     func(data []byte) (v []byte, e error) { return data, nil },
	}
}

var decoderByType = make(map[ADEType]decoder)
var encoderByType = make(map[ADEType]encoder)

/**********************************************************/
// Decoder methods: pass atom data to the decoder for type conversion to go type

// String returns Atom data as a string.  All Atom types must support this.
func (c Codec) String() (string, error) { return c.Decoder.String(*c.dataPtr) }

// StringDelimited returns Atom data as a string.  If this atom data type is
// delimited, then surround it with its delimiter characters.  Otherwise, just
// return the same result as String.
// All Atom types must support this.
func (c Codec) StringDelimited() (string, error) { return c.Decoder.StringDelimited(*c.dataPtr) }

// Bool returns Atom data as a bool, for Atom types where this makes sense.
func (c Codec) Bool() (bool, error) { return c.Decoder.Bool(*c.dataPtr) }

// Uint returns Atom data as a uint64, for unsigned integer Atom types.  64/32/16/8 bit
// types are all returned as a uint64 value, which can be cast to their native
// integer types without overflow.
func (c Codec) Uint() (uint64, error) { return c.Decoder.Uint(*c.dataPtr) }

// Int returns Atom data as an int64, for integer Atom types.  64/32/16/8 bit
// types are all returned as a uint64 value, which can be cast to their native
// integer types without overflow.
func (c Codec) Int() (int64, error) { return c.Decoder.Int(*c.dataPtr) }

// Float returns Atom data as an float64, for floating point Atom types.
func (c Codec) Float() (float64, error) { return c.Decoder.Float(*c.dataPtr) }

// SliceOfUint returns Atom data as a slice of uint64, for types which can be represented this way.
// This includes fractional types, where the numerator and denominator are
// represented as a 2-element slice.
func (c Codec) SliceOfUint() ([]uint64, error) { return c.Decoder.SliceOfUint(*c.dataPtr) }

// SliceOfInt returns Atom data as a slice of int64, for types which can be represented this way.
// This includes fractional types, where the numerator and denominator are
// represented as a 2-element slice.
func (c Codec) SliceOfInt() ([]int64, error) { return c.Decoder.SliceOfInt(*c.dataPtr) }

// SliceOfByte returns the big-endian byte representation of the Atom data.
// This is how the Atom data is represented within a Binary atom container
// file.  All Atom types must support this.
func (c Codec) SliceOfByte() ([]byte, error) { return *c.dataPtr, nil }

/**********************************************************/
// Encoder methods: convert given data type to []byte and store in Atom

// SetString sets the Atom data to the value represented by the given string.
// Must be implemented by all Atom types.
func (c Codec) SetString(v string) error {
	return c.Encoder.SetString(c.dataPtr, v)
}

// SetStringDelimited sets the Atom data to the value represented by the given string, after stripping off the surrounding delimiters.
// Delimiters must be the correct type according to the ADE escaping rules: double-quote for CSTR and USTR, single quote for FC32, double quote for IPAD.
// For other types, this method has the same result as SetString.
// Must be implemented by all Atom types.
func (c Codec) SetStringDelimited(v string) error {
	return c.Encoder.SetStringDelimited(c.dataPtr, v)
}

// SetBool sets the Atom data to the value of the given bool, for Atom data types that can be represented by a bool.
func (c Codec) SetBool(v bool) error { return c.Encoder.SetBool(c.dataPtr, v) }

// SetUint sets the Atom data to the value of the given unsigned integer, for Atom data types that can be represented by an unsigned integer.
func (c Codec) SetUint(v uint64) error { return c.Encoder.SetUint(c.dataPtr, v) }

// SetInt sets the Atom data to the value of the given integer, for Atom data types that can be represented by an integer.
func (c Codec) SetInt(v int64) error { return c.Encoder.SetInt(c.dataPtr, v) }

// SetFloat sets the Atom data to the value of the given float, for Atom data types that can be represented by an float.
func (c Codec) SetFloat(v float64) error { return c.Encoder.SetFloat(c.dataPtr, v) }

// SetSliceOfUint sets the Atom data to the value of the given slice of unsigned integers, for Atom data types that can be represented by a slice of unsigned integers.
func (c Codec) SetSliceOfUint(v []uint64) error { return c.Encoder.SetSliceOfUint(c.dataPtr, v) }

// SetSliceOfInt sets the Atom data to the value of the given slice of integers, for Atom data types that can be represented by a slice of integers.
func (c Codec) SetSliceOfInt(v []int64) error { return c.Encoder.SetSliceOfInt(c.dataPtr, v) }

// SetSliceOfByte sets the Atom data to the value of the given slice of bytes.
func (c Codec) SetSliceOfByte(v []byte) error { return c.Encoder.SetSliceOfByte(c.dataPtr, v) }

/**********************************************************/

// Initialize decoder table, which makes decoders accessible by ADE type.
// Variable 'd' is used for assignment, because Go disallows assigning directly
// to a struct member of a map value.  Example:
//    decoderByType[UI01] = newDecoder(UI01)
//    decoderByType[UI01].Bool = UI32ToBool //illegal
func init() {
	// ADE unsigned int types
	dec := newDecoder(UI01)
	dec.String = UI32ToString
	dec.StringDelimited = dec.String
	dec.Bool = UI01ToBool
	dec.Uint = UI32ToUint64
	decoderByType[UI01] = dec

	dec = newDecoder(UI08)
	dec.String = UI08ToString
	dec.StringDelimited = dec.String
	dec.Uint = UI08ToUint64
	decoderByType[UI08] = dec

	dec = newDecoder(UI16)
	dec.String = UI16ToString
	dec.StringDelimited = dec.String
	dec.Uint = UI16ToUint64
	decoderByType[UI16] = dec

	dec = newDecoder(UI32)
	dec.String = UI32ToString
	dec.StringDelimited = dec.String
	dec.Uint = UI32ToUint64
	decoderByType[UI32] = dec

	dec = newDecoder(UI64)
	dec.String = UI64ToString
	dec.StringDelimited = dec.String
	dec.Uint = UI64ToUint64
	dec.Int = UI64ToInt64
	decoderByType[UI64] = dec

	// ADE signed int types
	dec = newDecoder(SI08)
	dec.String = SI08ToString
	dec.StringDelimited = dec.String
	dec.Int = SI08ToInt64
	decoderByType[SI08] = dec

	dec = newDecoder(SI16)
	dec.String = SI16ToString
	dec.StringDelimited = dec.String
	dec.Int = SI16ToInt64
	decoderByType[SI16] = dec

	dec = newDecoder(SI32)
	dec.String = SI32ToString
	dec.StringDelimited = dec.String
	dec.Int = SI32ToInt64
	decoderByType[SI32] = dec

	dec = newDecoder(SI64)
	dec.String = SI64ToString
	dec.StringDelimited = dec.String
	dec.Int = SI64ToInt64
	decoderByType[SI64] = dec

	// ADE floating point types
	dec = newDecoder(FP32)
	dec.String = FP32ToString
	dec.StringDelimited = dec.String
	dec.Float = FP32ToFloat64
	decoderByType[FP32] = dec

	dec = newDecoder(FP64)
	dec.String = FP64ToString
	dec.StringDelimited = dec.String
	dec.Float = FP64ToFloat64
	decoderByType[FP64] = dec

	// ADE fixed point types
	dec = newDecoder(UF32)
	dec.String = UF32ToString
	dec.StringDelimited = dec.String
	dec.Float = UF32ToFloat64
	decoderByType[UF32] = dec

	dec = newDecoder(UF64)
	dec.String = UF64ToString
	dec.StringDelimited = dec.String
	dec.Float = UF64ToFloat64
	decoderByType[UF64] = dec

	dec = newDecoder(SF32)
	dec.String = SF32ToString
	dec.StringDelimited = dec.String
	dec.Float = SF32ToFloat64
	decoderByType[SF32] = dec

	dec = newDecoder(SF64)
	dec.String = SF64ToString
	dec.StringDelimited = dec.String
	dec.Float = SF64ToFloat64
	decoderByType[SF64] = dec

	// ADE fractional types

	dec = newDecoder(UR32)
	dec.String = UR32ToString
	dec.StringDelimited = dec.String
	dec.SliceOfUint = UR32ToSliceOfUint
	decoderByType[UR32] = dec

	dec = newDecoder(UR64)
	dec.String = UR64ToString
	dec.StringDelimited = dec.String
	dec.SliceOfUint = UR64ToSliceOfUint
	decoderByType[UR64] = dec

	dec = newDecoder(SR32)
	dec.String = SR32ToString
	dec.StringDelimited = dec.String
	dec.SliceOfInt = SR32ToSliceOfInt
	decoderByType[SR32] = dec

	dec = newDecoder(SR64)
	dec.String = SR64ToString
	dec.StringDelimited = dec.String
	dec.SliceOfInt = SR64ToSliceOfInt
	decoderByType[SR64] = dec

	// ADE Four char code
	dec = newDecoder(FC32)
	dec.String = FC32ToStringDelimited
	dec.StringDelimited = dec.String
	decoderByType[FC32] = dec

	// ADE ENUM type
	dec = newDecoder(ENUM)
	dec.String = SI32ToString
	dec.StringDelimited = dec.String
	dec.Int = SI32ToInt64
	decoderByType[ENUM] = dec

	// ADE UUID type
	dec = newDecoder(UUID)
	dec.String = UUIDToString
	dec.StringDelimited = dec.String
	decoderByType[UUID] = dec

	// IP Address types
	dec = newDecoder(IP32)
	dec.String = IP32ToString
	dec.StringDelimited = dec.String
	dec.Uint = IP32ToUint64
	decoderByType[IP32] = dec

	dec = newDecoder(IPAD)
	dec.String = IPADToString
	dec.StringDelimited = dec.String
	decoderByType[IPAD] = dec

	// ADE String types
	dec = newDecoder(CSTR)
	dec.String = CSTRToString
	dec.StringDelimited = CSTRToStringDelimited
	decoderByType[CSTR] = dec

	dec = newDecoder(USTR)
	dec.String = USTRToString
	dec.StringDelimited = USTRToStringDelimited
	decoderByType[USTR] = dec

	// DATA type, and aliases
	dec = newDecoder(DATA)
	dec.String = BytesToHexString
	dec.StringDelimited = dec.String
	decoderByType[DATA] = dec
	decoderByType[CNCT] = dec
	decoderByType[Cnct] = dec

	// NULL type
	dec = newDecoder(NULL)
	dec.String = func([]byte) (s string, e error) { return }
	dec.StringDelimited = dec.String
	decoderByType[NULL] = dec

	// ADE container
	dec = newDecoder(CONT)
	dec.String = func([]byte) (s string, e error) { return }
	dec.StringDelimited = dec.String
	decoderByType[CONT] = dec
}

// ADE unsigned int types

// UI08ToUint64 accepts ADE UI08 data bytes, and expresses the value as a uint64.
func UI08ToUint64(buf []byte) (v uint64, e error) {
	if e = checkByteCount(buf, 1, "UI08"); e != nil {
		return
	}
	return uint64(buf[0]), e
}

// UI16ToUint16 accepts ADE UI16 data bytes, and expresses the value as a uint16.
func UI16ToUint16(buf []byte) (v uint16, e error) {
	if e = checkByteCount(buf, 2, "UI16"); e != nil {
		return
	}
	return binary.BigEndian.Uint16(buf), e
}

// UI16ToUint64 accepts ADE UI16 data bytes, and expresses the value as a uint64.
func UI16ToUint64(buf []byte) (v uint64, e error) {
	if e = checkByteCount(buf, 2, "UI16"); e != nil {
		return
	}
	return uint64(binary.BigEndian.Uint16(buf)), e
}

// UI01ToBool accepts UI01 data bytes, and expresses the value as a bool.
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

// UI32ToUint32 accepts UI32 data bytes, and expresses the value as a uint32.
func UI32ToUint32(buf []byte) (v uint32, e error) {
	if e = checkByteCount(buf, 4, "UI32"); e != nil {
		return
	}
	return binary.BigEndian.Uint32(buf), e
}

// UI32ToUint64 accepts ADE UI32 data bytes, and expresses the value as a uint64.
func UI32ToUint64(buf []byte) (v uint64, e error) {
	if e = checkByteCount(buf, 4, "UI32"); e != nil {
		return
	}
	var ui32 = binary.BigEndian.Uint32(buf)
	return uint64(ui32), e
}

// UI64ToUint64 accepts ADE UI64 data bytes, and expresses the value as a uint64.
func UI64ToUint64(buf []byte) (v uint64, e error) {
	if e = checkByteCount(buf, 8, "UI64"); e != nil {
		return
	}
	return binary.BigEndian.Uint64(buf), e
}

// UI64ToInt64 accepts ADE UI64 data bytes, and expresses the value as an int64.
func UI64ToInt64(buf []byte) (v int64, e error) {
	if e = checkByteCount(buf, 8, "UI64"); e != nil {
		return
	}
	var ui = binary.BigEndian.Uint64(buf)
	if ui > math.MaxInt64 {
		return 0, errRange("int64", ui)
	}
	return int64(ui), e
}

// UI08ToString accepts ADE UI08 data bytes, and expresses the value as a string.
func UI08ToString(buf []byte) (v string, e error) {
	if e = checkByteCount(buf, 1, "UI08"); e != nil {
		return
	}
	return fmt.Sprintf("%d", (buf[0])), e
}

// UI16ToString accepts ADE UI16 data bytes, and expresses the value as a string.
func UI16ToString(buf []byte) (v string, e error) {
	if e = checkByteCount(buf, 2, "UI16"); e != nil {
		return
	}
	return fmt.Sprintf("%d", binary.BigEndian.Uint16(buf)), e
}

// UI32ToString accepts ADE UI32 data bytes, and expresses the value as a string.
func UI32ToString(buf []byte) (v string, e error) {
	if e = checkByteCount(buf, 4, "UI32"); e != nil {
		return
	}
	return fmt.Sprintf("%d", binary.BigEndian.Uint32(buf)), e
}

// UI64ToString accepts ADE UI64 data bytes, and expresses the value as a string.
func UI64ToString(buf []byte) (v string, e error) {
	if e = checkByteCount(buf, 8, "UI64"); e != nil {
		return
	}
	return fmt.Sprintf("%d", binary.BigEndian.Uint64(buf)), e
}

// ADE signed int types

// SI08ToInt64 accepts ADE SI08 data bytes, and expresses the value as an int64.
func SI08ToInt64(buf []byte) (v int64, e error) {
	if e = checkByteCount(buf, 1, "SI08"); e != nil {
		return
	}
	var i int8
	e = binary.Read(bytes.NewReader(buf), binary.BigEndian, &i)
	return int64(i), e
}

// SI16ToInt16 accepts ADE SI08 data bytes, and expresses the value as an int16.
func SI16ToInt16(buf []byte) (v int16, e error) {
	if e = checkByteCount(buf, 2, "SI16"); e != nil {
		return
	}
	e = binary.Read(bytes.NewReader(buf), binary.BigEndian, &v)
	return
}

// SI16ToInt64 accepts ADE SI16 data bytes, and expresses the value as an int64.
func SI16ToInt64(buf []byte) (v int64, e error) {
	if e = checkByteCount(buf, 2, "SI16"); e != nil {
		return
	}
	var i int16
	e = binary.Read(bytes.NewReader(buf), binary.BigEndian, &i)
	return int64(i), e
}

// SI32ToInt32 accepts ADE SI32 data bytes, and expresses the value as an int32.
func SI32ToInt32(buf []byte) (v int32, e error) {
	if e = checkByteCount(buf, 4, "SI32"); e != nil {
		return
	}
	e = binary.Read(bytes.NewReader(buf), binary.BigEndian, &v)
	return
}

// SI32ToInt64 accepts ADE SI32 data bytes, and expresses the value as an int64.
func SI32ToInt64(buf []byte) (v int64, e error) {
	var i int32
	i, e = SI32ToInt32(buf)
	if e == nil {
		v = int64(i)
	}
	return v, e
}

// SI64ToInt64 accepts ADE SI64 data bytes, and expresses the value as an int64.
func SI64ToInt64(buf []byte) (v int64, e error) {
	if e = checkByteCount(buf, 8, "SI64"); e != nil {
		return
	}
	e = binary.Read(bytes.NewReader(buf), binary.BigEndian, &v)
	return v, e
}

// SI08ToString accepts ADE SI08 data bytes, and expresses the value as a string.
func SI08ToString(buf []byte) (v string, e error) {
	var i int64
	i, e = SI08ToInt64(buf)
	if e != nil {
		return v, e
	}
	return fmt.Sprintf("%d", i), e
}

// SI16ToString accepts ADE SI16 data bytes, and expresses the value as a string.
func SI16ToString(buf []byte) (v string, e error) {
	var i int64
	i, e = SI16ToInt64(buf)
	if e != nil {
		return v, e
	}
	return fmt.Sprintf("%d", i), e
}

// SI32ToString accepts ADE SI32 data bytes, and expresses the value as a string.
func SI32ToString(buf []byte) (v string, e error) {
	var i int64
	i, e = SI32ToInt64(buf)
	if e != nil {
		return v, e
	}
	return fmt.Sprintf("%d", i), e
}

// SI64ToString accepts ADE SI64 data bytes, and expresses the value as a string.
func SI64ToString(buf []byte) (v string, e error) {
	var i int64
	i, e = SI64ToInt64(buf)
	if e != nil {
		return v, e
	}
	return fmt.Sprintf("%d", i), e
}

// ADE floating point types

// FP32ToFloat32 accepts ADE FP32 data bytes, and expresses the value as a float32.
func FP32ToFloat32(buf []byte) (v float32, e error) {
	if e = checkByteCount(buf, 4, "FP32"); e != nil {
		return
	}
	var i uint32
	i, e = UI32ToUint32(buf)
	v = math.Float32frombits(i)
	return
}

// FP32ToFloat64 accepts ADE FP32 data bytes, and expresses the value as a float64.
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

// FP64ToFloat64 accepts ADE FP64 data bytes, and expresses the value as a float64.
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

// FP32ToString accepts ADE FP32 data bytes, and expresses the value as a string.
func FP32ToString(buf []byte) (v string, e error) {
	var f float64
	f, e = FP32ToFloat64(buf)
	if e == nil {
		v = fmt.Sprintf("%0.8E", f)
	}
	return
}

// FP64ToString accepts ADE FP64 data bytes, and expresses the value as a string.
func FP64ToString(buf []byte) (v string, e error) {
	var f float64
	f, e = FP64ToFloat64(buf)
	if e == nil {
		v = fmt.Sprintf("%0.17E", f)
	}
	return
}

// ADE fixed point types, unsigned

// UF32ToFloat64 accepts ADE UF32 data bytes, and expresses the value as a float64.
func UF32ToFloat64(buf []byte) (v float64, e error) {
	if e = checkByteCount(buf, 4, "UF32"); e != nil {
		return
	}
	var i uint64
	i, e = UI32ToUint64(buf)
	if e != nil {
		return
	}
	v = float64(i) / (1 + math.MaxUint16)

	return
}

// UF64ToFloat64 accepts ADE UF64 data bytes, and expresses the value as a float64.
func UF64ToFloat64(buf []byte) (v float64, e error) {
	if e = checkByteCount(buf, 8, "UF64"); e != nil {
		return
	}
	var i uint64
	i, e = UI64ToUint64(buf)
	if e != nil {
		return
	}
	v = float64(i) / (1 + math.MaxUint32) // + 0.0000000002
	return
}

// UF32ToString accepts ADE UF32 data bytes, and expresses the value as a string.
func UF32ToString(buf []byte) (v string, e error) {
	var f float64
	f, e = UF32ToFloat64(buf)
	if e == nil {
		v = fmt.Sprintf("%0.4f", f)
	}
	return
}

// UF64ToString accepts ADE UF64 data bytes, and expresses the value as a string.
func UF64ToString(buf []byte) (v string, e error) {
	if e = checkByteCount(buf, 8, "UF64"); e != nil {
		return
	}
	var i uint64
	i, e = UI64ToUint64(buf)
	if e == nil {
		iFract := i & 0x00000000FFFFFFFF
		fFract := float64(iFract) / (1 + math.MaxUint32) * math.Pow(10, 9)
		v = fmt.Sprintf("%d.%09.0f", i>>32, fFract)
	}
	return
}

// ADE fixed point types, signed

// SF32ToFloat64 accepts ADE SF32 data bytes, and expresses the value as a float64.
func SF32ToFloat64(buf []byte) (v float64, e error) {
	if e = checkByteCount(buf, 4, "SF32"); e != nil {
		return
	}
	var i int32
	i, e = SI32ToInt32(buf)
	if e != nil {
		return
	}
	v = float64(i) / float64(math.MaxUint16+1)
	return
}

// SF64ToFloat64 accepts ADE SF64 data bytes, and expresses the value as a float64.
func SF64ToFloat64(buf []byte) (v float64, e error) {
	if e = checkByteCount(buf, 8, "SF64"); e != nil {
		return
	}
	var i int64
	i, e = SI64ToInt64(buf)
	if e != nil {
		return
	}
	v = float64(i) / (math.MaxUint32 + 1)
	return
}

// SF32ToString accepts ADE SF32 data bytes, and expresses the value as a string.
func SF32ToString(buf []byte) (v string, e error) {
	if e = checkByteCount(buf, 4, "SF32"); e != nil {
		return
	}
	var f float64
	f, e = SF32ToFloat64(buf)
	f = Round(f, 5)
	if e == nil {
		v = fmt.Sprintf("%0.4f", f)
	}
	return
}

// SF64ToString accepts ADE SF64 data bytes, and expresses the value as a string.
func SF64ToString(buf []byte) (v string, e error) {
	if e = checkByteCount(buf, 8, "SF64"); e != nil {
		return
	}

	// convert to int64 to manipulate sign
	var i int64
	i, e = SI64ToInt64(buf)
	isNegative := i < 0
	if isNegative {
		i *= -1
	}

	// convert sign-converted bytes to string
	var byts = make([]byte, 8)
	binary.BigEndian.PutUint64(byts, uint64(i))
	v, e = UF64ToString(byts)
	if e != nil {
		return
	}
	if isNegative {
		v = strings.Join([]string{"-", v}, "")
	}
	return
}

// ADE fractional types, unsigned

// UR32ToSliceOfUint accepts ADE UR32 data bytes, and expresses the value as a slice of uint64.
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

// UR64ToSliceOfUint accepts ADE UR64 data bytes, and expresses the value as a slice of uint64.
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

// UR32ToString accepts ADE UR32 data bytes, and expresses the value as a string.
func UR32ToString(buf []byte) (v string, e error) {
	var arr []uint64
	arr, e = UR32ToSliceOfUint(buf)
	if e == nil {
		v = fmt.Sprintf("%d/%d", arr[0], arr[1])
	}
	return
}

// UR64ToString accepts ADE UR64 data bytes, and expresses the value as a string.
func UR64ToString(buf []byte) (v string, e error) {
	var arr []uint64
	arr, e = UR64ToSliceOfUint(buf)
	if e == nil {
		v = fmt.Sprintf("%d/%d", arr[0], arr[1])
	}
	return
}

// ADE fractional types, signed

// SR32ToSliceOfInt accepts ADE SR32 data bytes, and expresses the value as a slice of int64.
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

// SR64ToSliceOfInt accepts ADE SR64 data bytes, and expresses the value as a slice of int64.
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

// SR32ToString accepts ADE SR32 data bytes, and expresses the value as a string.
func SR32ToString(buf []byte) (v string, e error) {
	var arr []int64
	arr, e = SR32ToSliceOfInt(buf)
	if e == nil {
		v = fmt.Sprintf("%d/%d", arr[0], arr[1])
	}
	return
}

// SR64ToString accepts ADE SR64 data bytes, and expresses the value as a string.
func SR64ToString(buf []byte) (v string, e error) {
	var arr []int64
	arr, e = SR64ToSliceOfInt(buf)
	if e == nil {
		v = fmt.Sprintf("%d/%d", arr[0], arr[1])
	}
	return
}

// unicode.IsPrint does not work for this, it returns true for large swathes of
// ascii 127-255.
func isPrintableBytes(buf []byte) bool {
	for _, b := range buf {
		if !strings.ContainsRune(PrintableChars, rune(b)) {
			return false
		}
	}
	return true
}

// Return true if string is printable, false otherwise
func IsPrintableString(s string) bool {
	for _, r := range s {
		if !unicode.IsPrint(r) {
			return false
		}
	}
	return true
}

// FC32ToString accepts ADE FC32 data bytes, and expresses the value as a string of printable characters.
// The string may be either 4 printable characters, or 0x followed by 8 hex digits.
//
// This code avoids Mantis #27726: ccat/ctac can't parse container names
// starting with "#" or " ".
func FC32ToString(buf []byte) (v string, e error) {
	if e = checkByteCount(buf, 4, "FC32"); e != nil {
		return
	}
	if isPrintableBytes(buf) && !bytes.ContainsAny(buf, `"' `) && buf[0] != '#' {
		v = string(buf)
	} else {
		v = fmt.Sprintf("0x%08X", buf)
	}
	return
}

// FC32ToStringDelimited accepts ADE FC32 data bytes, and expresses the value as a delimited string.
// The string may be either 4 printable characters, or 0x followed by 8 hex digits.
//
// If the 4 printable characters version is returned, it will be surrounded by
// single-quote delimiters.
func FC32ToStringDelimited(buf []byte) (v string, e error) {
	v, e = FC32ToString(buf)
	if e != nil {
		return
	}
	if len(v) == 4 {
		v = fmt.Sprintf("'%s'", v)
	}
	return
}

// UUIDToString accepts ADE UUID data bytes, and expresses the value as a string.
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
// IP32ToString accepts ADE IP32 data bytes, and expresses the value as a string.
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

// IPADToString accepts ADE IPAD data bytes, and expresses the value as a string.
func IPADToString(buf []byte) (v string, e error) {
	v = string(buf[0 : len(buf)-1]) // trim null terminator
	v = fmt.Sprintf("\"%s\"", v)
	return
}

// String types

// CSTRToString accepts ADE CSTR data bytes, and expresses the value as a string.
func CSTRToString(buf []byte) (v string, e error) {
	if bytes.IndexByte(buf, '\x00') != len(buf)-1 || len(buf) == 0 {
		pos := bytes.IndexByte(buf, '\x00')
		if pos == -1 {
			e = fmt.Errorf("CSTR data lacks null byte terminator")
		} else {
			e = fmt.Errorf("CSTR data contains illegal embedded null byte")
		}
		return
	}
	v = CSTRBytesToEscapedString(buf[:len(buf)-1]) // discard null terminator
	return v, nil
}

// CSTRToStringDelimited accepts ADE CSTR data bytes, and expresses the value as a string.
func CSTRToStringDelimited(buf []byte) (v string, e error) {
	if v, e = CSTRToString(buf); e != nil {
		return
	}
	return fmt.Sprintf(`"%s"`, v), e
}

// USTRToString accepts ADE USTR data bytes, and expresses the value as a string.
// These values are stored as UTF32 Big Endian: each char is a uint32 that
// represents the integer value of the codepoint.
// Example: Unlike in UTF-8, 0xFF ==  0x000000FF == `ÿ`.
// These values are not stored as UTF-8 with extra padding, it's actual UTF32,
// which uses different byte values than UTF-8.  Review the unicode tables for a
// refresher if necessary.
func USTRToString(buf []byte) (v string, e error) {
	var output bytes.Buffer
	var codepoint rune
	for i := 0; i < len(buf); i += 4 {
		codepoint = rune(binary.BigEndian.Uint32(buf[i : i+4]))
		switch codepoint { // Apply ADE string escaping rules
		case '\n':
			output.WriteString(`\n`)
		case '\r':
			output.WriteString(`\r`)
		case '\\':
			output.WriteString(`\\`)
		case '"':
			output.WriteString(`\"`)
		default:
			if unicode.IsControl(codepoint) {
				output.WriteString(fmt.Sprintf("\\x%02X", codepoint))
			} else {
				output.WriteRune(codepoint)
			}
		}
	}
	return output.String(), nil
}

// USTRToStringDelimited accepts ADE USTR data bytes, and expresses the value as a string surrounded by double-quote delimiters.
func USTRToStringDelimited(buf []byte) (v string, e error) {
	v, e = USTRToString(buf)
	if e != nil {
		return
	}
	return fmt.Sprintf("\"%s\"", v), e
}

// BytesToHexString accepts bytes, and expresses the value as a hexadecimal string starting with "0x".
func BytesToHexString(buf []byte) (v string, e error) {
	if len(buf) == 0 {
		v = ""
	} else {
		v = fmt.Sprintf("0x%X", buf)
	}
	return
}

/**********************************************************/

// CSTRBytesToEscapedString accepts ADE CSTR data bytes, reads the CSTR data,
// and applies ADE escaping rules before returning the result as a string.
//
// This method always returns valid UTF-8, because invalid UTF-8 is detected
// and escaped.
//
// Dev note: Escaping must be performed on raw byte slice, not on bytes casted
// to string. This is because casting a byte slice containing high ascii
// (128-255) to string will convert invalid codepoint representations (eg. 0xFF
// for U+00FF) to the Unicode replacement character.
//
// Corresponds to ade: libs/osl/OSL_Types.cc CStr_Escape()
func CSTRBytesToEscapedString(input []byte) string {
	output := make([]rune, 0, len(input))
	for i := 0; i < len(input); i++ {
		b := input[i]
		if b == '\n' {
			output = append(output, '\\', 'n')
		} else if b == '\r' {
			output = append(output, '\\', 'r')
		} else if b == '\\' {
			output = append(output, '\\', '\\')
		} else if b == '"' {
			output = append(output, '\\', '"')
		} else if b <= 0x1f || b == 0x7f {
			output = append(output, []rune(fmt.Sprintf(`\x%02X`, b))...)
		} else if r, width := utf8.DecodeRune(input[i:]); r == utf8.RuneError {
			// invalid unicode sequence, consumed 1 byte only
			output = append(output, []rune(fmt.Sprintf(`\x%02X`, b))...)
		} else {
			output = append(output, r) // valid unicode sequence, consumed 1-4 bytes
			i += width - 1             // -1 because will ++ before next loop iter
		}
	}
	return string(output)
}

// CSTRBytesFromEscapedString accepts a string that has had ADE escaping rules
// applied, strips the escape characters, and expresses the value as ADE CSTR
// data bytes.
func CSTRBytesFromEscapedString(input string) (output []byte, e error) {
	buf := bytes.NewBuffer(make([]byte, 0, len(input)+1))

	var isEscaped, isHexEncode bool
	var hexRunes = make([]rune, 0, 2)
	var hexBytes []byte
	for _, r := range input {
		if isHexEncode {
			hexRunes = append(hexRunes, r)
			if len(hexRunes) < 2 {
				continue
			}
			if hexBytes, e = hex.DecodeString(string(hexRunes)); e != nil {
				e = errInvalidEscape("CSTR", fmt.Sprintf("\\x%s", string(hexRunes)), e.Error())
				return
			}
			if len(hexBytes) == 2 {
				r = rune(binary.BigEndian.Uint16(hexBytes))
			} else {
				r = rune(hexBytes[0])
			}
			hexRunes = hexRunes[:0] // clear buffer without altering capacity
			isHexEncode = false

			if r == 0 {
				buf.WriteString(`\x00`) // can't encode null terminator within CSTR
				continue
			}

		} else if isEscaped {
			switch r {
			case 'n':
				r = '\n'
			case 'r':
				r = '\r'
			case '\\', '"':
			case 'x':
				isEscaped = false
				isHexEncode = true
				continue
			default:
				e = errInvalidEscape("CSTR", fmt.Sprintf("\\%c", r), "")
				return
			}
			isEscaped = false
		} else if r == '\\' {
			isEscaped = true
			continue
		} else if adeMustEscapeRune(r) {
			e = errUnescaped("CSTR", r)
			return
		} else if r == rune(0) {
			e = errStrInvalid("CSTR", input)
			return
		}
		_, e = buf.WriteRune(r)
		if e != nil {
			return
		}
	}
	if isHexEncode {
		strInput := fmt.Sprint("\\x", string(hexRunes)) // drop [] delimiters
		e = errInvalidEscape("CSTR", strInput, "EOF during hex encoded character")
		return
	} else if isEscaped {
		e = errInvalidEscape("CSTR", "\\", "EOF during escaped character")
		return
	}
	e = buf.WriteByte('\x00') // add null terminator
	return buf.Bytes(), e
}

// adeMustEscapeRune returns true if the given rune needs to be escaped
// according to ADE string escaping rules.
func adeMustEscapeRune(r rune) bool {
	if r == '\n' || r == '\r' || r == '"' || r == '\\' {
		return true
	}
	if r < 0x20 || r == 0x7f {
		return true
	}
	return false
}

/**********************************************************
Encoder method table for ADE types
**********************************************************/

func init() {
	// ADE unsigned int types
	enc := newEncoder(UI01)
	enc.SetString = StringToUI01Bytes
	enc.SetBool = BoolToUI01Bytes
	enc.SetUint = Uint64ToUI01Bytes
	encoderByType[UI01] = enc

	enc = newEncoder(UI08)
	enc.SetString = StringToUI08Bytes
	enc.SetUint = Uint64ToUI08Bytes
	encoderByType[UI08] = enc

	enc = newEncoder(UI16)
	enc.SetString = StringToUI16Bytes
	enc.SetUint = Uint64ToUI16Bytes
	encoderByType[UI16] = enc

	enc = newEncoder(UI32)
	enc.SetString = StringToUI32Bytes
	enc.SetUint = Uint64ToUI32Bytes
	encoderByType[UI32] = enc

	enc = newEncoder(UI64)
	enc.SetString = StringToUI64Bytes
	enc.SetUint = Uint64ToUI64Bytes
	encoderByType[UI64] = enc

	// ADE signed int types
	enc = newEncoder(SI08)
	enc.SetString = StringToSI08Bytes
	enc.SetInt = Int64ToSI08Bytes
	encoderByType[SI08] = enc

	enc = newEncoder(SI16)
	enc.SetString = StringToSI16Bytes
	enc.SetInt = Int64ToSI16Bytes
	encoderByType[SI16] = enc

	enc = newEncoder(SI32)
	enc.SetString = StringToSI32Bytes
	enc.SetInt = Int64ToSI32Bytes
	encoderByType[SI32] = enc

	enc = newEncoder(SI64)
	enc.SetString = StringToSI64Bytes
	enc.SetInt = Int64ToSI64Bytes
	encoderByType[SI64] = enc

	// ADE floating point types
	enc = newEncoder(FP32)
	enc.SetString = StringToFP32Bytes
	enc.SetFloat = Float64ToFP32Bytes
	encoderByType[FP32] = enc

	enc = newEncoder(FP64)
	enc.SetString = StringToFP64Bytes
	enc.SetFloat = Float64ToFP64Bytes
	encoderByType[FP64] = enc

	// ADE fixed point types
	enc = newEncoder(UF32)
	enc.SetString = StringToUF32Bytes
	enc.SetFloat = Float64ToUF32Bytes
	encoderByType[UF32] = enc

	enc = newEncoder(UF64)
	enc.SetString = StringToUF64Bytes
	enc.SetFloat = Float64ToUF64Bytes
	encoderByType[UF64] = enc

	enc = newEncoder(SF32)
	enc.SetString = StringToSF32Bytes
	enc.SetFloat = Float64ToSF32Bytes
	encoderByType[SF32] = enc

	enc = newEncoder(SF64)
	enc.SetString = StringToSF64Bytes
	enc.SetFloat = Float64ToSF64Bytes
	encoderByType[SF64] = enc

	// ADE fractional types

	enc = newEncoder(UR32)
	enc.SetString = StringToUR32Bytes
	enc.SetSliceOfUint = SliceOfUint64ToUR32Bytes
	encoderByType[UR32] = enc

	enc = newEncoder(UR64)
	enc.SetString = StringToUR64Bytes
	enc.SetSliceOfUint = SliceOfUint64ToUR64Bytes
	encoderByType[UR64] = enc

	enc = newEncoder(SR32)
	enc.SetString = StringToSR32Bytes
	enc.SetSliceOfInt = SliceOfInt64ToSR32Bytes
	encoderByType[SR32] = enc

	enc = newEncoder(SR64)
	enc.SetString = StringToSR64Bytes
	enc.SetSliceOfInt = SliceOfInt64ToSR64Bytes
	encoderByType[SR64] = enc

	// ADE Four char code
	enc = newEncoder(FC32)
	enc.SetString = StringToFC32Bytes
	enc.SetUint = Uint64ToFC32Bytes
	encoderByType[FC32] = enc

	// IP Address types
	enc = newEncoder(IP32)
	enc.SetString = StringToIP32Bytes
	enc.SetUint = Uint64ToIP32Bytes
	encoderByType[IP32] = enc

	enc = newEncoder(IPAD)
	enc.SetString = StringToIPADBytes
	encoderByType[IPAD] = enc

	// ADE UUID type
	enc = newEncoder(UUID)
	enc.SetString = StringToUUIDBytes
	encoderByType[UUID] = enc

	// ADE String types
	enc = newEncoder(CSTR)
	enc.SetString = StringToCSTRBytes
	enc.SetStringDelimited = DelimitedStringToCSTRBytes
	encoderByType[CSTR] = enc

	enc = newEncoder(USTR)
	enc.SetString = StringToUSTRBytes
	enc.SetStringDelimited = DelimitedStringToUSTRBytes
	encoderByType[USTR] = enc

	// DATA type, and aliases
	enc = newEncoder(DATA)
	enc.SetString = HexStringToDATABytes
	enc.SetStringDelimited = HexStringToDATABytes
	encoderByType[DATA] = enc
	encoderByType[CNCT] = enc
	encoderByType[Cnct] = enc

	// ADE ENUM type
	enc = newEncoder(ENUM)
	enc.SetString = StringToSI32Bytes
	enc.SetInt = Int64ToSI32Bytes
	encoderByType[ENUM] = enc

	// NULL type
	enc = newEncoder(NULL)
	enc.SetString = func(_ *[]byte, _ string) (e error) { return }
	encoderByType[NULL] = enc

	// ADE container
	enc = newEncoder(CONT)
	enc.SetString = func(_ *[]byte, _ string) (e error) { return }
	encoderByType[CONT] = enc
}

/************************************************************
Encoding functions - set Atom.data bytes from go type
************************************************************/

// StringToUI01Bytes writes a string value to a byte slice pointer as ADE UI01 binary data.
func StringToUI01Bytes(buf *[]byte, v string) (e error) {
	if len(*buf) != 4 {
		*buf = make([]byte, 4)
	}
	switch v {
	case "false", "0", "+0", "-0":
		binary.BigEndian.PutUint32(*buf, uint32(0))
	case "true", "1", "+1":
		binary.BigEndian.PutUint32(*buf, uint32(1))
	default:
		e = errStrInvalid("UI01", v)
	}
	return
}

// BoolToUI01Bytes writes a bool value to a byte slice pointer as ADE UI01 binary data.
func BoolToUI01Bytes(buf *[]byte, v bool) (e error) {
	if len(*buf) != 4 {
		*buf = make([]byte, 4)
	}
	if v {
		binary.BigEndian.PutUint32(*buf, uint32(1))
	} else {
		binary.BigEndian.PutUint32(*buf, uint32(0))
	}
	return
}

// Uint64ToUI01Bytes writes a uint64 value to the byte slice pointer as ADE UI01 binary data .
func Uint64ToUI01Bytes(buf *[]byte, v uint64) (e error) {
	if len(*buf) != 4 {
		*buf = make([]byte, 4)
	}
	if v == 1 {
		binary.BigEndian.PutUint32(*buf, uint32(1))
	} else if v == 0 {
		binary.BigEndian.PutUint32(*buf, uint32(0))
	} else {
		e = errRange("UI01", v)
	}
	return
}

// encode of unsigned integer types

// StringToUI08Bytes writes a string value to a byte slice pointer as ADE UI08 binary data.
func StringToUI08Bytes(buf *[]byte, v string) (e error) {
	if len(*buf) != 1 {
		*buf = make([]byte, 1)
	}
	var i uint64
	i, e = strconv.ParseUint(v, 0, 8)
	if e != nil {
		return errStrInvalid("UI08", v)
	}
	return Uint64ToUI08Bytes(buf, i)
}

// Uint64ToUI08Bytes writes a uint64 value to a byte slice pointer as ADE UI08 binary data.
func Uint64ToUI08Bytes(buf *[]byte, v uint64) (e error) {
	if len(*buf) != 1 {
		*buf = make([]byte, 1)
	}
	if v > math.MaxUint8 {
		e = errRange("UI08", v)
		return
	}
	(*buf)[0] = uint8(v)
	return
}

// StringToUI16Bytes writes a string value to a byte slice pointer as ADE UI16 binary data.
func StringToUI16Bytes(buf *[]byte, v string) (e error) {
	if len(*buf) != 2 {
		*buf = make([]byte, 2)
	}
	var i uint64
	i, e = strconv.ParseUint(v, 0, 16)
	if e != nil {
		return errStrInvalid("UI16", v)
	}
	binary.BigEndian.PutUint16(*buf, uint16(i))
	return
}

// Uint64ToUI16Bytes writes a uint64 value to a byte slice pointer as ADE UI16 binary data.
func Uint64ToUI16Bytes(buf *[]byte, v uint64) (e error) {
	if len(*buf) != 2 {
		*buf = make([]byte, 2)
	}
	if v > math.MaxUint16 {
		return errRange("UI16", v)
	}
	binary.BigEndian.PutUint16(*buf, uint16(v))
	return
}

// StringToUI32Bytes writes a string value to a byte slice pointer as ADE UI32 binary data.
func StringToUI32Bytes(buf *[]byte, v string) (e error) {
	if len(*buf) != 4 {
		*buf = make([]byte, 4)
	}
	var i uint64
	i, e = strconv.ParseUint(v, 0, 32)
	if e != nil {
		return errStrInvalid("UI32", v)
	}
	binary.BigEndian.PutUint32(*buf, uint32(i))
	return
}

// Uint64ToUI32Bytes writes a uint64 value to a byte slice pointer as ADE UI32 binary data.
func Uint64ToUI32Bytes(buf *[]byte, v uint64) (e error) {
	if len(*buf) != 4 {
		*buf = make([]byte, 4)
	}
	if v > math.MaxUint32 {
		return errRange("UI32", v)
	}
	binary.BigEndian.PutUint32(*buf, uint32(v))
	return
}

// StringToUI64Bytes writes a string value to a byte slice pointer as ADE UI64 binary data.
func StringToUI64Bytes(buf *[]byte, v string) (e error) {
	if len(*buf) != 8 {
		*buf = make([]byte, 8)
	}

	var i uint64
	i, e = strconv.ParseUint(v, 0, 64)
	if e != nil {
		return errStrInvalid("UI64", v)
	}
	binary.BigEndian.PutUint64(*buf, uint64(i))
	return
}

// Uint64ToUI64Bytes writes a uint64 value to a byte slice pointer as ADE UI64 binary data.
func Uint64ToUI64Bytes(buf *[]byte, v uint64) (e error) {
	if len(*buf) != 8 {
		*buf = make([]byte, 8)
	}
	binary.BigEndian.PutUint64(*buf, uint64(v))
	return
}

// encode of signed integer types

// StringToSI08Bytes writes a string value to a byte slice pointer as ADE SI08 binary data.
func StringToSI08Bytes(buf *[]byte, v string) (e error) {
	if len(*buf) != 1 {
		*buf = make([]byte, 1)
	}
	var i int64
	i, e = strconv.ParseInt(v, 0, 8)
	if e != nil {
		return errStrInvalid("SI08", v)
	}
	return Int64ToSI08Bytes(buf, i)
}

// Int64ToSI08Bytes writes a int64 value to a byte slice pointer as ADE SI08 binary data.
func Int64ToSI08Bytes(buf *[]byte, v int64) (e error) {
	if len(*buf) != 1 {
		*buf = make([]byte, 1)
	}
	if v < math.MinInt8 || v > math.MaxInt8 {
		return errRange("SI08", v)
	}
	(*buf)[0] = byte(v)
	return
}

// StringToSI16Bytes writes a string value to a byte slice pointer as ADE SI16 binary data.
func StringToSI16Bytes(buf *[]byte, v string) (e error) {
	if len(*buf) != 2 {
		*buf = make([]byte, 2)
	}
	var i int64
	i, e = strconv.ParseInt(v, 0, 16)
	if e != nil {
		return errStrInvalid("SI16", v)
	}
	binary.BigEndian.PutUint16(*buf, uint16(i))
	return
}

// Int64ToSI16Bytes and writes an int64 value to a byte slice pointer as ADE SI16 binary data.
func Int64ToSI16Bytes(buf *[]byte, v int64) (e error) {
	if len(*buf) != 2 {
		*buf = make([]byte, 2)
	}
	if v < math.MinInt16 || v > math.MaxInt16 {
		return errRange("SI16", v)
	}
	binary.BigEndian.PutUint16(*buf, uint16(v))
	return
}

// StringToSI32Bytes writes a string value to a byte slice pointer as ADE SI32 binary data.
func StringToSI32Bytes(buf *[]byte, v string) (e error) {
	if len(*buf) != 4 {
		*buf = make([]byte, 4)
	}
	var i int64
	i, e = strconv.ParseInt(v, 0, 32)
	if e != nil {
		return errStrInvalid("SI32", v)
	}
	binary.BigEndian.PutUint32(*buf, uint32(i))
	return
}

// Int64ToSI32Bytes writes an int64 value to a byte slice pointer as ADE SI32 binary data.
func Int64ToSI32Bytes(buf *[]byte, v int64) (e error) {
	if len(*buf) != 4 {
		*buf = make([]byte, 4)
	}
	if v < math.MinInt32 || v > math.MaxInt32 {
		return errRange("SI32", v)
	}
	binary.BigEndian.PutUint32(*buf, uint32(v))
	return
}

// StringToSI64Bytes writes a string value to a byte slice pointer as ADE SI64 binary data.
func StringToSI64Bytes(buf *[]byte, v string) (e error) {
	if len(*buf) != 8 {
		*buf = make([]byte, 8)
	}
	var i int64
	i, e = strconv.ParseInt(v, 0, 64)
	if e != nil {
		return errStrInvalid("SI64", v)
	}
	binary.BigEndian.PutUint64(*buf, uint64(i))
	return
}

// Int64ToSI64Bytes writes an int64 value to a byte slice pointer as ADE SI64 binary data.
func Int64ToSI64Bytes(buf *[]byte, v int64) (e error) {
	if len(*buf) != 8 {
		*buf = make([]byte, 8)
	}
	binary.BigEndian.PutUint64(*buf, uint64(v))
	return
}

// encode of unsigned fractional types

// StringToUR32Bytes writes a string value to a byte slice pointer as ADE UR32 binary data.
func StringToUR32Bytes(buf *[]byte, v string) (e error) {
	if len(*buf) != 4 {
		*buf = make([]byte, 4)
	}

	// The %s is to detect trailing garbage in the line. It should not match
	// anything in the normal case.
	var num, den uint64
	var extra string
	matched, err := fmt.Sscanf(v, "%d/%d%s", &num, &den, &extra)
	if err != io.EOF || matched != 2 {
		return errStrInvalid("UR32", v)
	}
	if den == 0 {
		return errZeroDenominator("UR32", v)
	}
	return SliceOfUint64ToUR32Bytes(buf, []uint64{num, den})
}

// SliceOfUint64ToUR32Bytes writes a []uint64 value to a byte slice pointer as ADE UR32 binary data.
// The input should be a length 2 slice with the first value being the numerator, and the second being the denominator.
func SliceOfUint64ToUR32Bytes(buf *[]byte, v []uint64) (e error) {
	if len(*buf) != 4 {
		*buf = make([]byte, 4)
	}
	var num, den uint64
	num = v[0]
	den = v[1]
	if den == 0 {
		return errZeroDenominator("UR32", "")
	}
	if num > math.MaxUint16 || den > math.MaxUint16 {
		return errRange("UR32", v)
	}

	value := (uint32(num) << 16) + uint32(den)
	binary.BigEndian.PutUint32(*buf, value)
	return
}

// StringToUR64Bytes writes a string value to a byte slice pointer as ADE UR64 binary data.
func StringToUR64Bytes(buf *[]byte, v string) (e error) {
	if len(*buf) != 8 {
		*buf = make([]byte, 8)
	}
	var num, den uint64
	var extra string
	matched, err := fmt.Sscanf(v, "%d/%d%s", &num, &den, &extra)
	if err != io.EOF || matched != 2 {
		return errStrInvalid("UR64", v)
	}
	if den == 0 {
		return errZeroDenominator("UR64", v)
	}
	return SliceOfUint64ToUR64Bytes(buf, []uint64{num, den})
}

// SliceOfUint64ToUR64Bytes writes a []uint64 value to a byte slice pointer as ADE UI01 binary data.
// The input should be a length 2 slice with the first value being the numerator, and the second being the denominator.
func SliceOfUint64ToUR64Bytes(buf *[]byte, v []uint64) (e error) {
	if len(*buf) != 8 {
		*buf = make([]byte, 8)
	}
	var num, den uint64
	num = v[0]
	den = v[1]
	if num > math.MaxUint32 || den > math.MaxUint32 {
		return errRange("UR64", v)
	}
	if den == 0 {
		return errZeroDenominator("UR64", "")
	}
	value := (num << 32) + den
	binary.BigEndian.PutUint64(*buf, value)
	return
}

// encode of signed fractional types

// StringToSR32Bytes writes a string value to a byte slice pointer as ADE SR32 binary data.
func StringToSR32Bytes(buf *[]byte, v string) (e error) {
	if len(*buf) != 4 {
		*buf = make([]byte, 4)
	}
	var num, den int64
	var extra string
	matched, err := fmt.Sscanf(v, "%d/%d%s", &num, &den, &extra)
	if err != io.EOF || matched != 2 {
		return errStrInvalid("SR32", v)
	}
	if den == 0 {
		return errZeroDenominator("SR32", v)
	}
	return SliceOfInt64ToSR32Bytes(buf, []int64{num, den})
}

// SliceOfInt64ToSR32Bytes writes a []int64 value to a byte slice pointer as ADE SR32 binary data.
// The input should be a length 2 slice with the first value being the numerator, and the second being the denominator.
func SliceOfInt64ToSR32Bytes(buf *[]byte, v []int64) (e error) {
	if len(*buf) != 4 {
		*buf = make([]byte, 4)
	}
	var num, den int64
	num = v[0]
	den = v[1]
	if num > math.MaxInt16 || den > math.MaxInt16 || num < math.MinInt16 || den < math.MinInt16 {
		return errRange("SR32", v)
	}
	if den == 0 {
		return errZeroDenominator("SR32", "")
	}
	value := (int32(num) << 16) + int32(den)
	binary.BigEndian.PutUint32(*buf, uint32(value))
	return
}

// StringToSR64Bytes writes a string value to a byte slice pointer as ADE SR64 binary data.
func StringToSR64Bytes(buf *[]byte, v string) (e error) {
	if len(*buf) != 8 {
		*buf = make([]byte, 8)
	}
	var num, den int64
	var extra string
	matched, err := fmt.Sscanf(v, "%d/%d%s", &num, &den, &extra)
	if err != io.EOF || matched != 2 {
		return errStrInvalid("SR64", v)
	}
	if den == 0 {
		return errZeroDenominator("SR64", v)
	}
	return SliceOfInt64ToSR64Bytes(buf, []int64{num, den})
}

// SliceOfInt64ToSR64Bytes writes a []int64 value to a byte slice pointer as ADE SR64 binary data.
// The input should be a length 2 slice with the first value being the numerator, and the second being the denominator.
func SliceOfInt64ToSR64Bytes(buf *[]byte, v []int64) (e error) {
	if len(*buf) != 8 {
		*buf = make([]byte, 8)
	}
	var num, den int64
	num = v[0]
	den = v[1]
	if num > math.MaxInt32 || den > math.MaxInt32 || num < math.MinInt32 || den < math.MinInt32 {
		return errRange("SR64", v)
	}
	if den == 0 {
		return errZeroDenominator("SR64", "")
	}

	value := (num << 32) + den
	binary.BigEndian.PutUint64(*buf, uint64(value))
	return
}

// encode of floating point types

// StringToFP32Bytes writes a string value to a byte slice pointer as ADE FP32 binary data.
func StringToFP32Bytes(buf *[]byte, v string) (e error) {
	if len(*buf) != 4 {
		*buf = make([]byte, 4)
	}
	var f float64
	f, e = strconv.ParseFloat(v, 32)
	if e != nil {
		return errStrInvalid("FP32", v)
	}
	return Float64ToFP32Bytes(buf, f)
}

// Float64ToFP32Bytes writes a float64 value to a byte slice pointer as ADE FP32 binary data.
func Float64ToFP32Bytes(buf *[]byte, v float64) (e error) {
	if len(*buf) != 4 {
		*buf = make([]byte, 4)
	}
	if v > math.MaxFloat32 || math.IsNaN(v) || math.IsInf(v, 0) {
		return errRange("FP32", v)
	}
	var bits = math.Float32bits(float32(v))
	binary.BigEndian.PutUint32(*buf, bits)
	return
}

// StringToFP64Bytes writes a string value to a byte slice pointer as ADE FP64 binary data.
func StringToFP64Bytes(buf *[]byte, v string) (e error) {
	if len(*buf) != 8 {
		*buf = make([]byte, 8)
	}
	var f float64
	f, e = strconv.ParseFloat(v, 64)
	if e != nil {
		return errStrInvalid("FP64", v)
	}
	return Float64ToFP64Bytes(buf, f)
}

// Float64ToFP64Bytes writes a float64 value to a byte slice pointer as ADE FP64 binary data.
func Float64ToFP64Bytes(buf *[]byte, v float64) (e error) {
	if len(*buf) != 8 {
		*buf = make([]byte, 8)
	}
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return errRange("FP64", v)
	}
	binary.BigEndian.PutUint64(*buf, uint64(v))
	var bits = math.Float64bits(v)
	binary.BigEndian.PutUint64(*buf, bits)
	return
}

// encode of fixed point types

// StringToUF32Bytes writes a string value to a byte slice pointer as ADE UF32 binary data.
func StringToUF32Bytes(buf *[]byte, v string) (e error) {
	if len(*buf) != 4 {
		*buf = make([]byte, 4)
	}
	var f float64
	f, e = strconv.ParseFloat(v, 64)
	if e != nil {
		return errStrInvalid("UF32", v)
	}
	if f >= (math.MaxUint16 + 1) {
		return errRange("UF32", f)
	}
	return Float64ToUF32Bytes(buf, f)
}

// Float64ToUF32Bytes writes a float64 value to a byte slice pointer as ADE UF32 binary data.
func Float64ToUF32Bytes(buf *[]byte, v float64) (e error) {
	if len(*buf) != 4 {
		*buf = make([]byte, 4)
	}
	if int(v) > math.MaxUint16 || math.IsNaN(v) || math.IsInf(v, 0) {
		return errRange("UF32", v)
	}
	binary.BigEndian.PutUint32(*buf, uint32(v*(1+math.MaxUint16)))
	return
}

// StringToUF64Bytes writes a string value to a byte slice pointer as ADE UF64 binary data.
//
// The doc states this range limit for ADE type UFIX64:
///    "The highest value is 0xFFFFFFFFFFFFFFFF
//      = 231 + 230 + … + 21 + 20 + 2-1 +2-2 + … + 2-31 + 2-32
//      = 4294967295.999999999767169"
//
// That is too little precision.  The actual highest positive value is:
//     0x7FFFFFFFFFFFFFFF = 4294967295.9999999997671694 (has a 4 appended.)
// The missing fractional number means that if you do the natural thing and
// write a unit test for this range based on the doc, you'll expect this:
//     4294967295.999999999767169 = 0x7FFFFFFFFFFFFFFF
// but you'll get this:
//     4294967295.999999999767169 = 0x7FFFFFFFFFFFFFFE
func StringToUF64Bytes(buf *[]byte, v string) (e error) {
	if len(*buf) != 8 {
		*buf = make([]byte, 8)
	}

	// split string into whole and fractional parts
	iDec := strings.Index(v, ".")

	// convert whole part to uint32
	var whole uint64
	if iDec == -1 {
		whole, e = strconv.ParseUint(v, 10, 64)
	} else if iDec == 0 {
		whole = 0
		if len(v) == 1 {
			return errStrInvalid("UF64", v)
		}
	} else {
		whole, e = strconv.ParseUint(v[:iDec], 10, 64)
	}
	if e != nil {
		return errStrInvalid("UF64", v)
	}
	if whole > math.MaxUint32 {
		return errRange("UF64", v)
	}

	// fractional part
	var fractF float64
	if iDec > -1 && iDec != len(v)-1 {
		iEnd := len(v)
		if iEnd-iDec > 17 {
			iEnd = iDec + 17
		}
		// send fractional string for conversion, including the decimal point
		//fmt.Printf("  f '%s' :'%s' ", v, v[iDec:iEnd])
		fractF, e = strconv.ParseFloat(v[iDec:iEnd], 64)
		if e != nil {
			return errStrInvalid("UF64", v)
		}
		//fmt.Printf(" => %0.15f\n", fractF)
	}

	//	fmt.Printf("  float64bits = %x\n", math.Float64bits(float64(fractF)))

	// Move the precision places to be kept to the left of the decimal place.
	fractF *= (math.MaxUint32 + 1)
	fractF = Round(fractF, 6) // match rounding magnitude of ADE ccat

	binary.BigEndian.PutUint32(*buf, uint32(whole))
	binary.BigEndian.PutUint32((*buf)[4:], uint32(fractF))

	return
}

// Float64ToUF64Bytes writes a float64 value to a byte slice pointer as ADE UF64 binary data.
func Float64ToUF64Bytes(buf *[]byte, v float64) (e error) {
	if len(*buf) != 8 {
		*buf = make([]byte, 8)
	}
	if v < 0 || math.IsNaN(v) || math.IsInf(v, 0) {
		return errRange("UF64", v)
	}
	var i = uint64(v * (1 + math.MaxUint32))
	binary.BigEndian.PutUint64(*buf, i)
	return
}

// StringToSF32Bytes writes a string value to a byte slice pointer as ADE SF32 binary data.
func StringToSF32Bytes(buf *[]byte, v string) (e error) {
	if len(*buf) != 4 {
		*buf = make([]byte, 4)
	}
	var f float64
	f, e = strconv.ParseFloat(v, 64)
	if e != nil {
		return errStrInvalid("SF32", v)
	}
	return Float64ToSF32Bytes(buf, f)
}

// Float64ToSF32Bytes writes a float64 value to a byte slice pointer as ADE SF32 binary data.
func Float64ToSF32Bytes(buf *[]byte, v float64) (e error) {
	if len(*buf) != 4 {
		*buf = make([]byte, 4)
	}
	if v < -32768.0 || v >= 32768.0 || math.IsNaN(v) || math.IsInf(v, 0) {
		return errRange("SF32", v)
	}
	binary.BigEndian.PutUint32(*buf, uint32(v*(1+math.MaxUint16)))
	return
}

// StringToSF64Bytes writes a string value to a byte slice pointer as ADE SF64 binary data.
//
// The doc states this:
//     The highest positive value is:
//     0x7FFFFFFFFFFFFFFF = 2147483647.999999999
// This is incorrect.  The actual highest positive value is:
//     0x7FFFFFFFFFFFFFFF =  2147483647.9999999997671694
// As a result, the correct conversion procedure results in an almost-right value like
// My measurements show that float value as too low.
func StringToSF64Bytes(buf *[]byte, v string) (e error) {
	if len(*buf) != 8 {
		*buf = make([]byte, 8)
	}

	if _, e = strconv.ParseFloat(v, 128); e != nil {
		return errStrInvalid("SF64", v)
	}

	iMinus := strings.IndexByte(v, '-')
	if iMinus == -1 {
		StringToUF64Bytes(buf, v)
		return
	}

	// Number is negative.
	// Discard sign and set as UFIX64
	StringToUF64Bytes(buf, v[iMinus+1:])
	signed, e := SI64ToInt64(*buf)
	if e != nil {
		return e
	}

	// Reapply sign by pretending bytes are an int64.
	// This sorts out the 2s complement.
	Int64ToSI64Bytes(buf, -1*signed)
	return nil
}

// Float64ToSF64Bytes writes a string value to a byte slice pointer as ADE UI01 binary data.
func Float64ToSF64Bytes(buf *[]byte, v float64) (e error) {
	if len(*buf) != 8 {
		*buf = make([]byte, 8)
	}
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return errRange("SF64", v)
	}
	whole, fract := math.Modf(v)
	if fract < 0 {
		fract *= -1
	}
	binary.BigEndian.PutUint32(*buf, uint32(whole))
	binary.BigEndian.PutUint32((*buf)[4:], uint32(fract*(1+math.MaxUint32)))
	return
}

// StringToFC32Bytes writes a string value to a byte slice pointer as ADE FC32 binary data.
func StringToFC32Bytes(buf *[]byte, v string) (e error) {
	if len(*buf) != 4 {
		*buf = make([]byte, 4)
	}

	// nonprintable chars are allowed in input string if they are hex-encoded.
	// raw unprintable chars must return error.
	var extra string
	switch len(v) {
	case 10: // 8 hex digits plus leading 0x
		if !strings.HasPrefix(v, "0x") {
			return fmt.Errorf("FC32 value is too long: (%s)", v)
		}
		matched, e := fmt.Sscanf(v, "0x%x%s", buf, &extra)
		if e != io.EOF || matched != 1 {
			return errStrInvalid("FC32", v)
		}
	case 8: // 8 hex digits
		matched, e := fmt.Sscanf(v, "%x%s", buf, &extra)
		if e != io.EOF || matched != 1 {
			return errStrInvalid("FC32", v)
		}
	case 6: // 4 printable chars, single quote delimited
		if !IsPrintableString(v) {
			return fmt.Errorf("FC32 value is not printable: 0x%x", v)
		}
		if v[0] != '\'' || v[5] != '\'' {
			return fmt.Errorf("FC32 value is too long: (%s)", v)
		}
		*buf = []byte(v)[1:5]
	case 4: // 4 printable chars, no delimiters
		*buf = []byte(v)
	default:
		return errStrInvalid("FC32", v)
	}
	return nil
}

// Uint64ToFC32Bytes writes a uint64 value to a byte slice pointer as ADE FC32 binary data.
func Uint64ToFC32Bytes(buf *[]byte, v uint64) (e error) {
	if len(*buf) != 4 {
		*buf = make([]byte, 4)
	}
	if v > math.MaxUint32 {
		return errRange("FC32", v)
	}
	binary.BigEndian.PutUint32(*buf, uint32(v))
	return
}

// StringToIP32Bytes writes a string value to a byte slice pointer as ADE IP32 binary data.
// IP32 is usually a simple 4 bytes = 4 octets type, but it also has a
// rarely used multi-width form used to define a range.
// The double-width form seems to be expressed solely in hex.
func StringToIP32Bytes(buf *[]byte, v string) (e error) {
	// Set data to zero value in case of error
	if len(*buf) != 4 {
		*buf = make([]byte, 4)
	}

	// handle multi-address form separately
	if strings.HasPrefix(v, "0x") {
		return HexStringToIP32Bytes(buf, v)
	}
	// Only a single IPv4 address is allowed from here on.

	// Extract 4 octets from string as decimal numbers
	var oct1, oct2, oct3, oct4 uint8
	var extra string
	matched, err := fmt.Sscanf(v, "%d.%d.%d.%d%s", &oct1, &oct2, &oct3, &oct4, &extra)
	if err != io.EOF || matched != 4 {
		return errStrInvalid("IP32", v)
	}
	copied := copy(*buf, []byte{oct1, oct2, oct3, oct4})
	if copied != 4 {
		e = fmt.Errorf("expected 4 bytes copied for IP32 value(%s), got %d: e", v, copied)
	}
	return
}

// HexStringToIP32Bytes writes a hexadecimal string value to a byte slice pointer as ADE IP32 binary data.
// Restrictions:
// string must start with "0x"
// following that must be only hex digits, in any number of sets of 8
func HexStringToIP32Bytes(buf *[]byte, v string) (e error) {
	if !strings.HasPrefix(v, "0x") {
		return errStrInvalid("IP32", v)
	}

	// require 8 hex digits
	size := len(v[2:])
	if 0 != size%8 || size == 0 {
		return errStrInvalid("IP32", v)
	}

	// allocate enough space
	if len(*buf) != size {
		*buf = make([]byte, size/2)
	}

	// scan each chunk of 8 hex digits, and store as 4 byte address
	for i := 2; i < len(v); i += 8 {
		addr, err := strconv.ParseUint(v[i:i+8], 16, 32)
		if err != nil {
			*buf = make([]byte, 4) // zero before returning error
			return errStrInvalid("IP32", v)
		}
		iByte := (i - 2) / 2 // number of bytes seen so far
		binary.BigEndian.PutUint32((*buf)[iByte:], uint32(addr))
	}
	return
}

// Uint64ToIP32Bytes writes a uint64 value to a byte slice pointer as ADE IP32 binary data.
func Uint64ToIP32Bytes(buf *[]byte, v uint64) (e error) {
	if v > math.MaxUint32 {
		// store as 2 IPv4 addresses in 8 bytes
		if len(*buf) != 8 {
			*buf = make([]byte, 8)
		}
		binary.BigEndian.PutUint64(*buf, v)
	} else {
		// store as a single IPv4 address in 4 bytes
		if len(*buf) != 4 {
			*buf = make([]byte, 4)
		}
		binary.BigEndian.PutUint32(*buf, uint32(v))
	}
	return
}

// StringToIPADBytes writes a string value to a byte slice pointer as ADE IPAD binary data.
func StringToIPADBytes(buf *[]byte, v string) (e error) {
	size := len(v)
	*buf = make([]byte, size)
	if len(v) < 3 && v != "::" {
		*buf = []byte(nil)
		return errStrInvalid("IPAD", v)
	}
	copy(*buf, v)

	// check for optional delimiters
	if (*buf)[0] == '"' && (*buf)[size-1] == '"' {
		*buf = (*buf)[1 : size-1] // ignore the delimiters from here on
	}

	// verify valid chars for IPv6
	chars := "0123456789abcdefABCDEF:."
	for _, b := range *buf {
		if !strings.ContainsRune(chars, rune(b)) {
			*buf = []byte(nil)
			return errStrInvalid("IPAD", v)
		}
	}

	*buf = append(*buf, '\x00') // add null terminator like a CSTR
	return
}

// StringToUUIDBytes writes a string value to a byte slice pointer as ADE UUID binary data.
// No NULL terminator is used for this type.
// Double-quote delimiters are optional on the input string.
func StringToUUIDBytes(buf *[]byte, v string) (e error) {
	if len(*buf) != 36 {
		*buf = make([]byte, 36)
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
	*buf = uuid.Bytes()
	return
}

// DelimitedStringToCSTRBytes writes a string value to a byte slice pointer as ADE CSTR binary data.
func DelimitedStringToCSTRBytes(buf *[]byte, v string) (e error) {
	L := len(v)
	if L < 2 || (v[0] != '"' || v[L-1] != '"') {
		return fmt.Errorf("CSTR input string must be double-quoted: (%s)", v)
	}
	return StringToCSTRBytes(buf, v[1:L-1])
}

// StringToCSTRBytes writes a string value to a byte slice pointer as ADE UI01 binary data.
// A NULL terminator is appended to the string value.
func StringToCSTRBytes(buf *[]byte, v string) (e error) {
	*buf, e = CSTRBytesFromEscapedString(v)
	return e
}

// DelimitedStringToUSTRBytes writes a string value to a byte slice pointer as ADE USTR binary data.
// It first strips off the double-quote delimiters.
func DelimitedStringToUSTRBytes(buf *[]byte, v string) (e error) {
	L := len(v)
	if L < 2 || (v[0] != '"' || v[L-1] != '"') {
		return fmt.Errorf("USTR input string must be double-quoted: (%s)", v)
	}
	return StringToUSTRBytes(buf, v[1:L-1])
}

// StringToUSTRBytes writes a string value to a byte slice pointer as ADE USTR binary data.
//
// The string is encoded as UTF32 big-endian (ie. 4 bytes for each rune, no
// variable-length encoding allowed.)
//
// No NULL terminator is used for this type, unlike CSTR.
func StringToUSTRBytes(buf *[]byte, v string) (e error) {
	bb := bytes.NewBuffer(make([]byte, 0, 4*len(v)))

	// iterate by rune:  The rune value is the unicode codepoint value, which is
	// useful because that's the same as the UTF32 encoding.
	var isEscaped, isHexEncode bool
	var hexRunes = make([]rune, 0, 2)
	var hexBytes []byte
	for _, r := range v {
		if isHexEncode {
			hexRunes = append(hexRunes, r)
			if len(hexRunes) < 2 {
				continue
			}
			if hexBytes, e = hex.DecodeString(string(hexRunes)); e != nil {
				return errInvalidEscape("USTR", fmt.Sprintf("\\x%s", string(hexRunes)), e.Error())
			}
			if len(hexBytes) == 2 {
				r = rune(binary.BigEndian.Uint16(hexBytes))
			} else {
				r = rune(hexBytes[0])
			}
			hexRunes = hexRunes[:0] // clear buffer without altering capacity
			isHexEncode = false
		} else if isEscaped {
			switch r {
			case 'n':
				r = '\n'
			case 'r':
				r = '\r'
			case '\\', '"':
			case 'x':
				isEscaped = false
				isHexEncode = true
				continue
			default:
				return errInvalidEscape("USTR", fmt.Sprintf("\\%c", r), "")
			}
			isEscaped = false
		} else if r == '\\' {
			isEscaped = true
			continue
		} else if adeMustEscapeRune(r) {
			e = errUnescaped("USTR", r)
			return
		}
		e := binary.Write(bb, binary.BigEndian, uint32(r))
		if e != nil {
			return e
		}
	}
	if isEscaped || isHexEncode {
		if isHexEncode {
			strInput := fmt.Sprint("\\x", string(hexRunes)) // drop [] delimiters
			return errInvalidEscape("USTR", strInput, "EOF during hex encoded character")
		}
		return errInvalidEscape("USTR", "\\", "EOF during escaped character")
	}
	*buf = bb.Bytes()
	return
}

// HexStringToDATABytes writes a hexadecimal byte string value to a byte slice pointer as bytes.
// A leading "0x" on input is optional.
func HexStringToDATABytes(buf *[]byte, v string) (e error) {

	// empty input string results in empty data section
	if len(v) == 0 {
		*buf = []byte{}
		return
	}

	// non-empty input must be strictly hex
	if !strings.HasPrefix(v, "0x") {
		return errStrInvalid("DATA", v)
	}
	buffer, e := hex.DecodeString(v[2:])
	if e != nil {
		return
	}
	*buf = buffer
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
func (u *uuidType) SetFromString(s string) (e error) {
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
	return u.SetFromUints(values)
}

func (u *uuidType) SetFromUints(values []uint64) (e error) {
	if len(values) != 5 {
		return fmt.Errorf("invalid integer values for type UUID: %v", values)
	}
	u.TimeLow = uint32(values[0])
	u.TimeMid = uint16(values[1])
	u.TimeHiAndVersion = uint16(values[2])
	u.ClkSeqHiRes = uint8(values[3] >> 8)
	u.ClkSeqLow = uint8(values[3] & 0x00000000000000FF)

	var buf = make([]byte, 8)
	binary.BigEndian.PutUint64(buf, values[4])
	copy(u.Node[:], buf[2:])
	return
}

// Bytes returns the UUID data as a slice of bytes.
func (u uuidType) Bytes() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, u)
	return buf.Bytes()
}

// String returns the UUID data as a hyphenated string.
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

// Round the given decimal floating point value at the given place.
func Round(val float64, places int) float64 {
	pow := math.Pow(10, float64(places))
	digit := pow * val
	_, div := math.Modf(digit)

	var round float64
	if val > 0 {
		if div >= 0.5 {
			round = math.Ceil(digit)
		} else {
			round = math.Floor(digit)
		}
	} else {
		if div >= 0.5 {
			round = math.Floor(digit)
		} else {
			round = math.Ceil(digit)
		}
	}

	return round / pow
}

// Data type introspection methods
// Clients with an Atom can use these to decide how to handle atom data.
// They are intended to provide hints as to how the data should be handled, rather
// than a straight mapping of what decoder funcs are provided for each type.
// (eg. every ADE type implements String() and FromString(), not just the ones
// that return true for IsString().)

// IsBool returns true if the receiver's Atom has a boolean type.
func (c Codec) IsBool() bool {
	return c.typ == "UI01"
}

// IsUint returns true if the receiver's Atom has an unsigned integer type.
func (c *Codec) IsUint() bool {
	switch c.typ {
	case "UI08", "UI16", "UI32", "UI64":
		return true
	}
	return false
}

// IsInt returns true if the receiver's Atom has an integer type.
func (c *Codec) IsInt() bool {
	switch c.typ {
	case "SI08", "SI16", "SI32", "SI64":
		return true
	}
	return false
}

// IsFloat returns true if the receiver's Atom has a floating point type.
func (c *Codec) IsFloat() bool {
	switch c.typ {
	case "FP32", "FP64", "UF32", "UF64", "SF32", "SF64":
		return true
	}
	return false
}

// IsString returns true if the receiver's Atom has a string type.
func (c *Codec) IsString() bool {
	switch c.typ {
	case "CSTR", "USTR", "FC32", "IP32", "IPAD", "DATA", "CNCT", "cnct", "UUID":
		return true
	}
	return false
}

// ZeroData sets an atom's data to the zero value of its ADE type.
// For fixed-size types, the byte slice capacity remains the same so that a new
// value can be set without needing memory allocation.
// For variable-sized types, data is set to nil and all memory released for
// garbage collection.
func (c *Codec) ZeroData() {
	switch c.typ {
	case UI08, SI08:
		zeroOrAllocateByteSlice(c.dataPtr, 1)
	case UI16, SI16:
		zeroOrAllocateByteSlice(c.dataPtr, 2)
	case UI01, UI32, SI32, FP32, UF32, SF32, SR32, UR32, FC32, IP32, ENUM:
		zeroOrAllocateByteSlice(c.dataPtr, 4)
	case UI64, SI64, FP64, UF64, SF64, UR64, SR64:
		zeroOrAllocateByteSlice(c.dataPtr, 8)
	case UUID:
		zeroOrAllocateByteSlice(c.dataPtr, 36)
	case IPAD, CSTR, USTR, DATA, CNCT, Cnct:
		*c.dataPtr = nil
	case CONT, NULL:
		*c.dataPtr = nil
	default:
		panic(fmt.Sprintf(`unknown ADE type: "%v"`, c.typ))
	}
}

// zeroOrAllocateByteSlice verifies that the give byte slice has
// the specified capacity, and zeroes it out.
// It avoids memory allocation when possible.
func zeroOrAllocateByteSlice(buf *[]byte, size int) {
	if cap(*buf) == size {
		// zero out the buffer, O(1)
		for i := range *buf {
			(*buf)[i] = 0
		}
	} else {
		// newly allocated mem is already zeroed
		*buf = make([]byte, size)
	}
}
