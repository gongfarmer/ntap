package atom

import (
	"fmt"
	"math"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

// Test conversion of Atom data as bytes to golang native types
type testBytesToUint struct {
	Input     []byte
	WantValue uint64
	WantError error
}

type fromBytesFunc func([]byte) (uint64, error)

// Given a function as an argument, return the function's name
func GetFunctionName(i interface{}) string {
	fullName := runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
	parts := strings.Split(fullName, "/")
	return parts[len(parts)-1]
}

func runTestsUint(t *testing.T, tests []testBytesToUint, f fromBytesFunc) {
	for _, test := range tests {
		got_value, got_err := f(test.Input)

		funcName := GetFunctionName(f)
		switch {
		case got_err == nil && test.WantError == nil:
		case got_err != nil && test.WantError == nil:
			fallthrough
		case got_err == nil && test.WantError != nil:
			fallthrough
		case got_err.Error() != test.WantError.Error():
			t.Errorf("%v(%q): got err %s, want err %s", funcName, test.Input, got_err, test.WantError)
			return
		}

		if got_value != test.WantValue {
			t.Errorf("%v(%q): got value %T(%[3]v), want %[4]T(%[4]v)", funcName, test.Input, got_value, test.WantValue)
		}
	}
}

func TestUI08ToUint64(t *testing.T) {
	fmtByteCount := fmt.Sprintf("invalid byte count for ADE type %s: want %d, got %%d", "UI08", 1)
	tests := []testBytesToUint{
		testBytesToUint{[]byte("\x00"), 0, nil},
		testBytesToUint{[]byte("\x01"), 1, nil},
		testBytesToUint{[]byte("\x00"), 0, nil},
		testBytesToUint{[]byte("\x0F"), 15, nil},
		testBytesToUint{[]byte("\xF0"), 240, nil},
		testBytesToUint{[]byte("\xFF"), 255, nil},
		testBytesToUint{[]byte("\x00\x00"), 0, fmt.Errorf(fmtByteCount, 2)},
		testBytesToUint{[]byte("\xFF\xFF"), 0, fmt.Errorf(fmtByteCount, 2)},
		testBytesToUint{[]byte(""), 0, fmt.Errorf(fmtByteCount, 0)},
	}
	runTestsUint(t, tests, UI08ToUint64)
}

func TestUI16ToUint64(t *testing.T) {
	fmtByteCount := fmt.Sprintf("invalid byte count for ADE type %s: want %d, got %%d", "UI16", 2)
	tests := []testBytesToUint{
		testBytesToUint{[]byte("\x00\x00"), 0, nil},
		testBytesToUint{[]byte("\x00\xFF"), 255, nil},
		testBytesToUint{[]byte("\xFF\x00"), 65280, nil},
		testBytesToUint{[]byte("\xFF\xFF"), 65535, nil},
		testBytesToUint{[]byte("\x00\x00\x01"), 0, fmt.Errorf(fmtByteCount, 3)},
		testBytesToUint{[]byte("\xFF"), 0, fmt.Errorf(fmtByteCount, 1)},
		testBytesToUint{[]byte("\x00"), 0, fmt.Errorf(fmtByteCount, 1)},
		testBytesToUint{[]byte{}, 0, fmt.Errorf(fmtByteCount, 0)},
	}
	runTestsUint(t, tests, UI16ToUint64)
}

func TestUI01ToBool(t *testing.T) {
	type testBytesToBool struct {
		Input     []byte
		WantValue bool
		WantError error
	}
	fmtTooBig := "value %d overflows type bool"
	fmtByteCount := fmt.Sprintf("invalid byte count for ADE type %s: want %d, got %%d", "UI01", 4)
	tests := []testBytesToBool{
		testBytesToBool{[]byte("\x00\x00\x00\x00"), false, nil},
		testBytesToBool{[]byte("\x00\x00\x00\x01"), true, nil},
		testBytesToBool{[]byte("\x00\x00\x00\x02"), false, fmt.Errorf(fmtTooBig, 2)},
		testBytesToBool{[]byte("\x00\x00\x00\xFF"), false, fmt.Errorf(fmtTooBig, 255)},
		testBytesToBool{[]byte(""), false, fmt.Errorf(fmtByteCount, 0)},
		testBytesToBool{[]byte("\x01"), false, fmt.Errorf(fmtByteCount, 1)},
		testBytesToBool{[]byte("\x00\x01"), false, fmt.Errorf(fmtByteCount, 2)},
		testBytesToBool{[]byte("\x00\x00\x01"), false, fmt.Errorf(fmtByteCount, 3)},
		testBytesToBool{[]byte("\x00\x00\x00\x00\x01"), false, fmt.Errorf(fmtByteCount, 5)},
		testBytesToBool{[]byte("\x00\x00\x00\x00\x00\x01"), false, fmt.Errorf(fmtByteCount, 6)},
	}
	funcName := "UI01ToBool"
	for _, test := range tests {
		got_value, got_err := UI01ToBool(test.Input)
		switch {
		case got_err == nil && test.WantError == nil:
		case got_err != nil && test.WantError == nil:
			fallthrough
		case got_err == nil && test.WantError != nil:
			fallthrough
		case got_err.Error() != test.WantError.Error():
			t.Errorf("%v(%q): got err <<%s>>, want err <<%s>>", funcName, test.Input, got_err, test.WantError)
			return
		}

		if got_value != test.WantValue {
			t.Errorf("%v(%q): got value %T(%[3]v), want %[4]T(%[4]v)", funcName, test.Input, got_value, test.WantValue)
		}
	}
}

func funcUI32ToUint32(t *testing.T) {
	type testBytesToUint32 struct {
		Input     []byte
		WantValue uint32
		WantError error
	}
	fmtByteCount := fmt.Sprintf("invalid byte count for ADE type %s: want %d, got %%d", "UI32", 4)
	tests := []testBytesToUint32{
		testBytesToUint32{[]byte{}, 0, fmt.Errorf(fmtByteCount, 0)},
		testBytesToUint32{[]byte("\x00"), 0, fmt.Errorf(fmtByteCount, 1)},
		testBytesToUint32{[]byte("\x00\xFF"), 0, fmt.Errorf(fmtByteCount, 2)},
		testBytesToUint32{[]byte("\xFF\x00\xFF"), 0, fmt.Errorf(fmtByteCount, 3)},
		testBytesToUint32{[]byte("\x00\x00\x00\x00"), 0, nil},
		testBytesToUint32{[]byte("\xFF\xFF\xFF\xFF"), math.MaxUint32, nil},
		testBytesToUint32{[]byte("\x01\xFF\xFF\xFF\xFF"), 0, nil},
	}
	funcName := "UI32ToUint32"
	for _, test := range tests {
		got_value, got_err := UI32ToUint32(test.Input)
		switch {
		case got_err == nil && test.WantError == nil:
		case got_err != nil && test.WantError == nil:
			fallthrough
		case got_err == nil && test.WantError != nil:
			fallthrough
		case got_err.Error() != test.WantError.Error():
			t.Errorf("%v(%q): got err <<%s>>, want err <<%s>>", funcName, test.Input, got_err, test.WantError)
			return
		}
		if got_value != test.WantValue {
			t.Errorf("%v(%q): got value %T(%[3]v), want %[4]T(%[4]v)", funcName, test.Input, got_value, test.WantValue)
		}
	}
}

//func UI32ToUint64(buf []byte) (v uint64, e error) {
//func UI64ToUint64(buf []byte) (v uint64, e error) {
//func UI08ToString(buf []byte) (v string, e error) {
//func UI16ToString(buf []byte) (v string, e error) {
//func UI32ToString(buf []byte) (v string, e error) {
//func UI64ToString(buf []byte) (v string, e error) {
//func SI08ToInt64(buf []byte) (v int64, e error) {
//func SI16ToInt64(buf []byte) (v int64, e error) {
//func SI32ToInt32(buf []byte) (v int32, e error) {
//func SI32ToInt64(buf []byte) (v int64, e error) {
//func SI64ToInt64(buf []byte) (v int64, e error) {
//func SI08ToString(buf []byte) (v string, e error) {
//func SI16ToString(buf []byte) (v string, e error) {
//func SI32ToString(buf []byte) (v string, e error) {
//func SI64ToString(buf []byte) (v string, e error) {
//func FP32ToFloat32(buf []byte) (v float32, e error) {
//func FP32ToFloat64(buf []byte) (v float64, e error) {
//func FP64ToFloat64(buf []byte) (v float64, e error) {
//func FP32ToString(buf []byte) (v string, e error) {
//func FP64ToString(buf []byte) (v string, e error) {
//func UF32ToFloat64(buf []byte) (v float64, e error) {
//func UF64ToFloat64(buf []byte) (v float64, e error) {
//func UF32ToString(buf []byte) (v string, e error) {
//func UF64ToString(buf []byte) (v string, e error) {
//func SF32ToFloat64(buf []byte) (v float64, e error) {
//func SF64ToFloat64(buf []byte) (v float64, e error) {
//func SF32ToString(buf []byte) (v string, e error) {
//func SF64ToString(buf []byte) (v string, e error) {
//func UR32ToSliceOfUint(buf []byte) (v []uint64, e error) {
//func UR64ToSliceOfUint(buf []byte) (v []uint64, e error) {
//func UR32ToString(buf []byte) (v string, e error) {
//func UR64ToString(buf []byte) (v string, e error) {
//func SR32ToSliceOfInt(buf []byte) (v []int64, e error) {
//func SR64ToSliceOfInt(buf []byte) (v []int64, e error) {
//func SR32ToString(buf []byte) (v string, e error) {
//func SR64ToString(buf []byte) (v string, e error) {
//func FC32ToString(buf []byte) (v string, e error) {
//func UUIDToString(buf []byte) (v string, e error) {
//func IP32ToString(buf []byte) (v string, e error) {
//func IPADToString(buf []byte) (v string, e error) {
//func CSTRToString(buf []byte) (v string, e error) {
//func CSTRToStringEscaped(buf []byte) (v string, e error) {
//func USTRToString(buf []byte) (v string, e error) {
//func USTRToStringEscaped(buf []byte) (v string, e error) {
//func BytesToHexString(buf []byte) (v string, e error) {
//func asPrintableString(buf []byte) string {
//func adeCstrEscape(s string) string {
//func init() {
//	enc.SetString = func(_ *Atom, _ string) (e error) { return }
//	enc.SetString = func(_ *Atom, _ string) (e error) { return }
//Encoding functions - set Atom.data bytes from go type
//func SetUI01FromString(a *Atom, v string) (e error) {
//func SetUI01FromBool(a *Atom, v bool) (e error) {
//func SetUI01FromUint64(a *Atom, v uint64) (e error) {
//func SetUI08FromString(a *Atom, v string) (e error) {
//func SetUI08FromUint64(a *Atom, v uint64) (e error) {
//func SetUI16FromString(a *Atom, v string) (e error) {
//func SetUI16FromUint64(a *Atom, v uint64) (e error) {
//func SetUI32FromString(a *Atom, v string) (e error) {
//func SetUI32FromUint64(a *Atom, v uint64) (e error) {
//func SetUI64FromString(a *Atom, v string) (e error) {
//func SetUI64FromUint64(a *Atom, v uint64) (e error) {
//func SetSI08FromString(a *Atom, v string) (e error) {
//func SetSI08FromInt64(a *Atom, v int64) (e error) {
//func SetSI16FromString(a *Atom, v string) (e error) {
//func SetSI16FromInt64(a *Atom, v int64) (e error) {
//func SetSI32FromString(a *Atom, v string) (e error) {
//func SetSI32FromInt64(a *Atom, v int64) (e error) {
//func SetSI64FromString(a *Atom, v string) (e error) {
//func SetSI64FromInt64(a *Atom, v int64) (e error) {
//func SetUR32FromString(a *Atom, v string) (e error) {
//func SetUR32FromSliceOfUint(a *Atom, v []uint64) (e error) {
//func SetUR64FromString(a *Atom, v string) (e error) {
//func SetUR64FromSliceOfUint(a *Atom, v []uint64) (e error) {
//func SetSR32FromString(a *Atom, v string) (e error) {
//func SetSR32FromSliceOfInt(a *Atom, v []int64) (e error) {
//func SetSR64FromString(a *Atom, v string) (e error) {
//func SetSR64FromSliceOfInt(a *Atom, v []int64) (e error) {
//func SetFP32FromString(a *Atom, v string) (e error) {
//func SetFP32FromFloat64(a *Atom, v float64) (e error) {
//func SetFP64FromString(a *Atom, v string) (e error) {
//func SetFP64FromFloat64(a *Atom, v float64) (e error) {
//func SetUF32FromString(a *Atom, v string) (e error) {
//func SetUF32FromFloat64(a *Atom, v float64) (e error) {
//func SetUF64FromString(a *Atom, v string) (e error) {
//func SetUF64FromFloat64(a *Atom, v float64) (e error) {
//func SetSF32FromString(a *Atom, v string) (e error) {
//func SetSF32FromFloat64(a *Atom, v float64) (e error) {
//func SetSF64FromString(a *Atom, v string) (e error) {
//func SetSF64FromFloat64(a *Atom, v float64) (e error) {
//func SetFC32FromString(a *Atom, v string) (e error) {
//func SetFC32FromUint(a *Atom, v uint64) (e error) {
//func SetIP32FromString(a *Atom, v string) (e error) {
//func SetIP32FromUint(a *Atom, v uint64) (e error) {
//func SetIPADFromString(a *Atom, v string) (e error) {
//func SetUUIDFromString(a *Atom, v string) (e error) {
//func SetCSTRFromEscapedString(a *Atom, v string) (e error) {
//func SetUSTRFromEscapedString(a *Atom, v string) (e error) {
//func SetDATAFromHexString(a *Atom, v string) (e error) {

/////
/////
///// /*
///// func TestDecUI01(t *testing.T) {
///// 	tests := []decodeTest{
///// 		// Yes, we really do use 4 bytes for this type!
///// 		decodeTest{[]byte{0x00, 0x00, 0x00, 0x00}, reflect.ValueOf(uint32(0))},
///// 		decodeTest{[]byte{0x00, 0x00, 0x00, 0x01}, reflect.ValueOf(uint32(1))},
///// 	}
///// 	for _, test := range tests {
///// 		got := decUI01(test.Input).Interface()
///// 		want := test.Want.Interface()
///// 		if got != want {
///// 			t.Errorf("decUI01(%q) = %T(%[2]v), want %T(%[3]v)", test.Input, got, want)
///// 		}
///// 	}
///// }
/////
///// func TestDecUI08(t *testing.T) {
///// 	tests := []decodeTest{
///// 		decodeTest{[]byte{0x00}, reflect.ValueOf(byte(0))},
///// 		decodeTest{[]byte{0x0F}, reflect.ValueOf(byte(15))},
///// 		decodeTest{[]byte{0xF0}, reflect.ValueOf(byte(240))},
///// 		decodeTest{[]byte{0xFF}, reflect.ValueOf(byte(255))},
///// 	}
///// 	for _, test := range tests {
///// 		got := decUI08(test.Input).Interface()
///// 		want := test.Want.Interface()
///// 		if got != want {
///// 			t.Errorf("decUI08(%q) = %T(%[2]v), want %T(%[3]v)", test.Input, got, want)
///// 		}
///// 	}
///// }
/////
///// func TestDecUI16(t *testing.T) {
///// 	tests := []decodeTest{
///// 		decodeTest{[]byte{0x00, 0x00}, reflect.ValueOf(uint16(0))},
///// 		decodeTest{[]byte{0x00, 0xFF}, reflect.ValueOf(uint16(255))},
///// 		decodeTest{[]byte{0xFF, 0x00}, reflect.ValueOf(uint16(65280))},
///// 		decodeTest{[]byte{0xFF, 0xFF}, reflect.ValueOf(uint16(65535))},
///// 	}
///// 	for _, test := range tests {
///// 		got := decUI16(test.Input).Interface()
///// 		want := test.Want.Interface()
///// 		if got != want {
///// 			t.Errorf("decUI16(%q) = %T(%[2]v), want %T(%[3]v)", test.Input, got, want)
///// 		}
///// 	}
///// }
/////
///// func TestDecUI32(t *testing.T) {
///// 	tests := []decodeTest{
///// 		decodeTest{[]byte{0x00, 0x00, 0x00, 0x00}, reflect.ValueOf(uint32(0x00000000))},
///// 		decodeTest{[]byte{0x00, 0x00, 0x00, 0xFF}, reflect.ValueOf(uint32(0x000000FF))},
///// 		decodeTest{[]byte{0x00, 0x00, 0xFF, 0x00}, reflect.ValueOf(uint32(0x0000FF00))},
///// 		decodeTest{[]byte{0x00, 0xFF, 0x00, 0x00}, reflect.ValueOf(uint32(0x00FF0000))},
///// 		decodeTest{[]byte{0xFF, 0x00, 0x00, 0x00}, reflect.ValueOf(uint32(0xFF000000))},
///// 		decodeTest{[]byte{0xFF, 0xFF, 0xFF, 0xFF}, reflect.ValueOf(uint32(0xFFFFFFFF))},
///// 	}
///// 	for _, test := range tests {
///// 		got := decUI32(test.Input).Interface()
///// 		want := test.Want.Interface()
///// 		if got != want {
///// 			t.Errorf("decUI32(%q) = %T(%[2]v), want %T(%[3]v)", test.Input, got, want)
///// 		}
///// 	}
///// }
/////
///// func TestDecUI64(t *testing.T) {
///// 	tests := []decodeTest{
///// 		decodeTest{[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, reflect.ValueOf(uint64(0x0000000000000000))},
///// 		decodeTest{[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xFF}, reflect.ValueOf(uint64(0x00000000000000FF))},
///// 		decodeTest{[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xFF, 0x00}, reflect.ValueOf(uint64(0x000000000000FF00))},
///// 		decodeTest{[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0xFF, 0x00, 0x00}, reflect.ValueOf(uint64(0x0000000000FF0000))},
///// 		decodeTest{[]byte{0x00, 0x00, 0x00, 0x00, 0xFF, 0x00, 0x00, 0x00}, reflect.ValueOf(uint64(0x00000000FF000000))},
///// 		decodeTest{[]byte{0x00, 0x00, 0x00, 0xFF, 0x00, 0x00, 0x00, 0x00}, reflect.ValueOf(uint64(0x000000FF00000000))},
///// 		decodeTest{[]byte{0x00, 0x00, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00}, reflect.ValueOf(uint64(0x0000FF0000000000))},
///// 		decodeTest{[]byte{0x00, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, reflect.ValueOf(uint64(0x00FF000000000000))},
///// 		decodeTest{[]byte{0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, reflect.ValueOf(uint64(0xFF00000000000000))},
///// 		decodeTest{[]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}, reflect.ValueOf(uint64(0xFFFFFFFFFFFFFFFF))},
///// 	}
///// 	for _, test := range tests {
///// 		got := decUI64(test.Input).Interface()
///// 		want := test.Want.Interface()
///// 		if got != want {
///// 			t.Errorf("decUI64(%q) = %T(%[2]v), want %T(%[3]v)", test.Input, got, want)
///// 		}
///// 	}
///// }
/////
///// func TestDecSF32(t *testing.T) {
///// 	tests := []decodeTest{
///// 		decodeTest{[]byte{0x00, 0x00, 0x00, 0x00}, reflect.ValueOf(float32(0))},
///// 		decodeTest{[]byte{0x00, 0x00, 0x00, 0x01}, reflect.ValueOf(float32(1.5258789e-05))},
///// 		decodeTest{[]byte{0x00, 0x00, 0x00, 0xFF}, reflect.ValueOf(float32(0.0038909912))},
///// 		decodeTest{[]byte{0x00, 0x00, 0xFF, 0x00}, reflect.ValueOf(float32(0.99609375))},
///// 		decodeTest{[]byte{0x00, 0xFF, 0x00, 0x00}, reflect.ValueOf(float32(255.0))},
///// 		decodeTest{[]byte{0xFF, 0xFF, 0xFF, 0xFF}, reflect.ValueOf(float32(-1.5258789e-05))},
///// 	}
///// 	for _, test := range tests {
///// 		got := decSF32(test.Input).Interface()
///// 		want := test.Want.Interface()
///// 		if got != want {
///// 			t.Errorf("decSF32(%q) = %T(%[2]v), want %T(%[3]v)", test.Input, got, want)
///// 		}
///// 	}
///// }
/////
///// func TestDecSF64(t *testing.T) {
///// 	tests := []decodeTest{
///// 		decodeTest{[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, reflect.ValueOf(float64(0))},
///// 		decodeTest{[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}, reflect.ValueOf(float64(2.3283064365386963e-10))},
///// 		decodeTest{[]byte{0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01}, reflect.ValueOf(float64(1.684300900392157e+07))},
///// 		decodeTest{[]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}, reflect.ValueOf(float64(-2.3283064365386963e-10))},
///// 	}
///// 	for _, test := range tests {
///// 		got := decSF64(test.Input).Interface()
///// 		want := test.Want.Interface()
///// 		if got != want {
///// 			t.Errorf("decSF64(%q) = %T(%[2]v), want %T(%[3]v)", test.Input, got, want)
///// 		}
///// 	}
///// }
///// func TestDecSI08(t *testing.T) {
/////
///// 	tests := []decodeTest{
///// 		decodeTest{[]byte{0}, reflect.ValueOf(int8(0))},
///// 		decodeTest{[]byte{math.MaxInt8}, reflect.ValueOf(int8(127))},
///// 	}
/////
///// 	// test min value for this type
///// 	// (buffer is needed to force a signed int8 to be an unsigned byte.)
///// 	var Min int8 = math.MinInt8
///// 	var buf = bytes.NewBuffer(make([]byte, 0, 2))
///// 	binary.Write(buf, binary.BigEndian, &Min)
///// 	tests = append(tests, decodeTest{buf.Bytes(), reflect.ValueOf(Min)})
/////
///// 	for _, test := range tests {
///// 		got := decSI08(test.Input).Interface()
///// 		want := test.Want.Interface()
///// 		if got != want {
///// 			t.Errorf("decSI08(%q) = %T(%[2]v), want %T(%[3]v)", test.Input, got, want)
///// 		}
///// 	}
///// }
///// func TestDecSI16(t *testing.T) {
///// 	tests := []decodeTest{
///// 		decodeTest{[]byte{0x00, 0x00}, reflect.ValueOf(int16(0))},
///// 		decodeTest{[]byte{0x00, 0x01}, reflect.ValueOf(int16(1))},
///// 		decodeTest{[]byte{0x00, 0xFF}, reflect.ValueOf(int16(255))},
///// 		decodeTest{[]byte{0xFF, 0x00}, reflect.ValueOf(int16(-256))},
///// 		decodeTest{[]byte{0xFF, 0xFF}, reflect.ValueOf(int16(-1))},
///// 	}
/////
///// 	// test min value
///// 	var Min int16 = math.MinInt16
///// 	var buf = bytes.NewBuffer(make([]byte, 0, 2))
///// 	binary.Write(buf, binary.BigEndian, &Min)
///// 	tests = append(tests, decodeTest{buf.Bytes(), reflect.ValueOf(Min)})
/////
///// 	// test max value
///// 	var Max int16 = math.MaxInt16
///// 	buf = bytes.NewBuffer(make([]byte, 0, 2))
///// 	binary.Write(buf, binary.BigEndian, &Max)
///// 	tests = append(tests, decodeTest{buf.Bytes(), reflect.ValueOf(Max)})
/////
///// 	for _, test := range tests {
///// 		got := decSI16(test.Input).Interface()
///// 		want := test.Want.Interface()
///// 		if got != want {
///// 			t.Errorf("decSI16(%q) = %T(%[2]v), want %T(%[3]v)", test.Input, got, want)
///// 		}
///// 	}
///// }
///// func TestDecSI32(t *testing.T) {
///// 	tests := []decodeTest{
///// 		decodeTest{[]byte{0x00, 0x00, 0x00, 0x00}, reflect.ValueOf(int32(0))},
///// 		decodeTest{[]byte{0x00, 0x00, 0x00, 0x01}, reflect.ValueOf(int32(1))},
///// 		decodeTest{[]byte{0x00, 0xFF, 0x00, 0xFF}, reflect.ValueOf(int32(16711935))},
///// 		decodeTest{[]byte{0xFF, 0x00, 0x00, 0x00}, reflect.ValueOf(int32(-16777216))},
///// 		decodeTest{[]byte{0xFF, 0xFF, 0xFF, 0xFF}, reflect.ValueOf(int32(-1))},
///// 	}
/////
///// 	// test min value
///// 	var Min int32 = math.MinInt32
///// 	buf := bytes.NewBuffer(make([]byte, 0, 2))
///// 	binary.Write(buf, binary.BigEndian, &Min)
///// 	tests = append(tests, decodeTest{buf.Bytes(), reflect.ValueOf(Min)})
/////
///// 	// test max value
///// 	var Max int32 = math.MaxInt32
///// 	buf = bytes.NewBuffer(make([]byte, 0, 2))
///// 	binary.Write(buf, binary.BigEndian, &Max)
///// 	tests = append(tests, decodeTest{buf.Bytes(), reflect.ValueOf(Max)})
/////
///// 	for _, test := range tests {
///// 		got := decSI32(test.Input).Interface()
///// 		want := test.Want.Interface()
///// 		if got != want {
///// 			t.Errorf("decSI32(%q) = %T(%[2]v), want %T(%[3]v)", test.Input, got, want)
///// 		}
///// 	}
///// }
/////
///// func TestDecSI64(t *testing.T) {
///// 	tests := []decodeTest{
///// 		decodeTest{[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, reflect.ValueOf(int64(0))},
///// 		decodeTest{[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}, reflect.ValueOf(int64(1))},
///// 		decodeTest{[]byte{0x00, 0xFF, 0x00, 0xFF, 0x00, 0xFF, 0x00, 0xFF}, reflect.ValueOf(int64(0x00FF00FF00FF00FF))},
///// 		decodeTest{[]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}, reflect.ValueOf(int64(-1))},
///// 	}
/////
///// 	// test min value
///// 	var Min int64 = math.MinInt64
///// 	buf := bytes.NewBuffer(make([]byte, 0, 2))
///// 	binary.Write(buf, binary.BigEndian, &Min)
///// 	tests = append(tests, decodeTest{buf.Bytes(), reflect.ValueOf(Min)})
/////
///// 	// test max value
///// 	var Max int64 = math.MaxInt64
///// 	buf = bytes.NewBuffer(make([]byte, 0, 2))
///// 	binary.Write(buf, binary.BigEndian, &Max)
///// 	tests = append(tests, decodeTest{buf.Bytes(), reflect.ValueOf(Max)})
/////
///// 	for _, test := range tests {
///// 		got := decSI64(test.Input).Interface()
///// 		want := test.Want.Interface()
///// 		if got != want {
///// 			t.Errorf("decSI64(%q) = %T(%[2]v), want %T(%[3]v)", test.Input, got, want)
///// 		}
///// 	}
///// }
///// func TestDecFP32(t *testing.T) {
///// 	tests := []decodeTest{
///// 		decodeTest{[]byte{0x00, 0x00, 0x00, 0x00}, reflect.ValueOf(float32(0))},
///// 		decodeTest{[]byte{0x00, 0x00, 0x00, 0xFF}, reflect.ValueOf(float32(3.57e-43))},
///// 	}
/////
///// 	// test max value
///// 	var Max float32 = math.MaxFloat32
///// 	buf := bytes.NewBuffer(make([]byte, 0, 4))
///// 	binary.Write(buf, binary.BigEndian, &Max)
///// 	tests = append(tests, decodeTest{buf.Bytes(), reflect.ValueOf(Max)})
/////
///// 	// test min value
///// 	var Min float32 = math.SmallestNonzeroFloat32
///// 	buf = bytes.NewBuffer(make([]byte, 0, 4))
///// 	binary.Write(buf, binary.BigEndian, &Min)
///// 	tests = append(tests, decodeTest{buf.Bytes(), reflect.ValueOf(Min)})
/////
///// 	for _, test := range tests {
///// 		got := decFP32(test.Input).Interface()
///// 		want := test.Want.Interface()
///// 		if got != want {
///// 			t.Errorf("decFP32(%q) = %T(%[2]v), want %T(%[3]v)", test.Input, got, want)
///// 		}
///// 	}
///// }
/////
///// func TestDecFP64(t *testing.T) {
///// 	tests := []decodeTest{
///// 		decodeTest{[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, reflect.ValueOf(float64(0))},
///// 		decodeTest{[]byte{0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01}, reflect.ValueOf(float64(7.748604185489348e-304))},
///// 	}
/////
///// 	// test max value
///// 	var Max float64 = math.MaxFloat64
///// 	buf := bytes.NewBuffer(make([]byte, 0, 4))
///// 	binary.Write(buf, binary.BigEndian, &Max)
///// 	tests = append(tests, decodeTest{buf.Bytes(), reflect.ValueOf(Max)})
/////
///// 	// test min value
///// 	var Min float64 = math.SmallestNonzeroFloat64
///// 	buf = bytes.NewBuffer(make([]byte, 0, 4))
///// 	binary.Write(buf, binary.BigEndian, &Min)
///// 	tests = append(tests, decodeTest{buf.Bytes(), reflect.ValueOf(Min)})
/////
///// 	for _, test := range tests {
///// 		got := decFP64(test.Input).Interface()
///// 		want := test.Want.Interface()
///// 		if got != want {
///// 			t.Errorf("decFP64(%q) = %T(%[2]v), want %T(%[3]v)", test.Input, got, want)
///// 		}
///// 	}
///// }
/////
///// func TestDecUF32(t *testing.T) {
///// 	tests := []decodeTest{
///// 		decodeTest{[]byte("\x00\x00\x00\x00"), reflect.ValueOf(float64(0))},
///// 		decodeTest{[]byte("\x00\x00\x00\x01"), reflect.ValueOf(float64(1.52587890625e-05))},
///// 		decodeTest{[]byte("\x00\x00\x00\xff"), reflect.ValueOf(float64(0.0038909912109375))},
///// 		decodeTest{[]byte("\x00\x00\xff\x00"), reflect.ValueOf(float64(0.99609375))},
///// 		decodeTest{[]byte("\x00\xff\x00\x00"), reflect.ValueOf(float64(255.0))},
///// 		decodeTest{[]byte("\xff\xff\xff\xff"), reflect.ValueOf(float64(65535.99998474121))},
///// 	}
///// 	for _, test := range tests {
///// 		got := decUF32(test.Input).Interface()
///// 		want := test.Want.Interface()
///// 		if got != want {
///// 			t.Errorf("decUF32(%q) = %T(%[2]v), want %T(%[3]v)", test.Input, got, want)
///// 		}
///// 	}
///// }
/////
///// func TestDecUF64(t *testing.T) {
///// 	tests := []decodeTest{
///// 		decodeTest{[]byte("\x00\x00\x00\x00\x00\x00\x00\x00"), reflect.ValueOf(float64(0))},
///// 		decodeTest{[]byte("\x00\x00\x00\x01\x00\x00\x00\x00"), reflect.ValueOf(float64(1.000000000))},
///// 		//		decodeTest{[]byte("\x00\x01\x00\x3c\x00\x00\x96\xfe"), reflect.ValueOf(float64(65596.000009000))},
///// 		decodeTest{[]byte("\xff\xff\xff\xff\x00\x00\x00\x00"), reflect.ValueOf(float64(4294967295.000000000))},
///// 		decodeTest{[]byte("\xff\xff\xff\xfe\x00\x00\x00\x00"), reflect.ValueOf(float64(4294967294.000000000))},
///// 		decodeTest{[]byte("\xff\xff\xff\xff\x19\x99\x99\x99"), reflect.ValueOf(float64(4294967295.100000000))},
///// 		decodeTest{[]byte("\xff\xff\xff\xff\x02\x8f\x5c\x28"), reflect.ValueOf(float64(4294967295.010000000))},
///// 		decodeTest{[]byte("\xff\xff\xff\xff\x00\x41\x89\x37"), reflect.ValueOf(float64(4294967295.001000000))},
///// 		decodeTest{[]byte("\xff\xff\xff\xff\x00\x06\x8d\xb8"), reflect.ValueOf(float64(4294967295.000100000))},
///// 		decodeTest{[]byte("\xff\xff\xff\xff\x00\x00\xa7\xc5"), reflect.ValueOf(float64(4294967295.000010000))},
///// 		decodeTest{[]byte("\xff\xff\xff\xff\x00\x00\x10\xc6"), reflect.ValueOf(float64(4294967295.000001000))},
///// 		decodeTest{[]byte("\xff\xff\xff\xff\xff\xff\xff\xfb"), reflect.ValueOf(float64(4294967295.999999999))},
///// 	}
/////
///// 	for _, test := range tests {
///// 		got := decUF64(test.Input).Interface()
///// 		want := test.Want.Interface()
///// 		if got != want {
///// 			t.Errorf("decUF64(% x)  got %T(%[2]v), want %T(%[3]v)", test.Input, got, want)
///// 		}
///// 	}
///// }
/////
///// func TestDecUR32(t *testing.T) {
///// 	tests := []decodeTest{
///// 		decodeTest{[]byte("\x00\x01\x00\x01"), reflect.ValueOf([2]uint16{1, 1})},
///// 		decodeTest{[]byte("\x00\x01\x00\x02"), reflect.ValueOf([2]uint16{1, 2})},
///// 		decodeTest{[]byte("\x01\x00\x01\x00"), reflect.ValueOf([2]uint16{256, 256})},
///// 		decodeTest{[]byte("\x00\x00\x00\x00"), reflect.ValueOf([2]uint16{0, 0})},
///// 		decodeTest{[]byte("\x19\x99\x99\x99"), reflect.ValueOf([2]uint16{6553, 39321})},
///// 		decodeTest{[]byte("\x02\x8f\x5c\x28"), reflect.ValueOf([2]uint16{655, 23592})},
///// 		decodeTest{[]byte("\xff\xff\x00\x05"), reflect.ValueOf([2]uint16{65535, 5})},
///// 		decodeTest{[]byte("\xff\xff\x00\x02"), reflect.ValueOf([2]uint16{65535, 2})},
///// 		decodeTest{[]byte("\xff\xff\xff\xff"), reflect.ValueOf([2]uint16{65535, 65535})},
///// 	}
/////
///// 	for _, test := range tests {
///// 		got := decUR32(test.Input).Interface()
///// 		want := test.Want.Interface()
///// 		if got != want {
///// 			t.Errorf("decUR32(% x)  got %T(%[2]v), want %T(%[3]v)", test.Input, got, want)
///// 		}
///// 	}
///// }
/////
///// func TestDecUR64(t *testing.T) {
///// 	tests := []decodeTest{
///// 		decodeTest{[]byte("\x00\x00\x00\x01\x00\x00\x00\x01"), reflect.ValueOf([2]uint32{1, 1})},
///// 		decodeTest{[]byte("\x00\x00\x00\x01\x00\x00\x00\x02"), reflect.ValueOf([2]uint32{1, 2})},
///// 		decodeTest{[]byte("\x01\x02\x03\x04\x05\x06\x07\x08"), reflect.ValueOf([2]uint32{16909060, 84281096})},
///// 		decodeTest{[]byte("\x10\x20\x30\x40\x50\x60\x70\x80"), reflect.ValueOf([2]uint32{270544960, 1348497536})},
///// 		decodeTest{[]byte("\x19\x99\x99\x99\x19\x99\x99\x99"), reflect.ValueOf([2]uint32{429496729, 429496729})},
///// 		decodeTest{[]byte("\xff\xff\x00\x02\xff\xff\xcc\xee"), reflect.ValueOf([2]uint32{4294901762, 4294954222})},
///// 		decodeTest{[]byte("\xff\xff\xff\xff\xff\xff\xff\xff"), reflect.ValueOf([2]uint32{4294967295, 4294967295})},
///// 	}
/////
///// 	for _, test := range tests {
///// 		got := decUR64(test.Input).Interface()
///// 		want := test.Want.Interface()
///// 		if got != want {
///// 			t.Errorf("decUR64(% x)  got %T(%[2]v), want %T(%[3]v)", test.Input, got, want)
///// 		}
///// 	}
///// }
/////
///// func TestDecSR32(t *testing.T) {
///// 	tests := []decodeTest{
///// 		decodeTest{[]byte("\x00\x01\x00\x01"), reflect.ValueOf([2]int16{1, 1})},
///// 		decodeTest{[]byte("\x00\x01\xff\xff"), reflect.ValueOf([2]int16{1, -1})},
///// 		decodeTest{[]byte("\xff\xff\x00\x01"), reflect.ValueOf([2]int16{-1, 1})},
///// 		decodeTest{[]byte("\x00\x01\x00\x01"), reflect.ValueOf([2]int16{1, 1})},
///// 		decodeTest{[]byte("\x00\x01\x00\x02"), reflect.ValueOf([2]int16{1, 2})},
///// 		decodeTest{[]byte("\x00\x01\xff\xfe"), reflect.ValueOf([2]int16{1, -2})},
///// 		decodeTest{[]byte("\xff\xff\x00\x02"), reflect.ValueOf([2]int16{-1, 2})},
///// 		decodeTest{[]byte("\x00\x01\x00\x02"), reflect.ValueOf([2]int16{1, 2})},
///// 		decodeTest{[]byte("\x00\x01\x00\x01"), reflect.ValueOf([2]int16{1, 1})},
///// 		decodeTest{[]byte("\x80\x00\x7f\xff"), reflect.ValueOf([2]int16{-32768, 32767})},
///// 		decodeTest{[]byte("\x7f\xff\x80\x00"), reflect.ValueOf([2]int16{32767, -32768})},
///// 		decodeTest{[]byte("\x00\x01\x00\x01"), reflect.ValueOf([2]int16{1, 1})},
///// 		decodeTest{[]byte("\x00\x01\x7f\xff"), reflect.ValueOf([2]int16{1, 32767})},
///// 		decodeTest{[]byte("\xff\xff\x7f\xff"), reflect.ValueOf([2]int16{-1, 32767})},
///// 		decodeTest{[]byte("\x00\x01\x80\x00"), reflect.ValueOf([2]int16{1, -32768})},
///// 	}
/////
///// 	for _, test := range tests {
///// 		got := decSR32(test.Input).Interface()
///// 		want := test.Want.Interface()
///// 		if got != want {
///// 			t.Errorf("decSR32(% x)  got %T(%[2]v), want %T(%[3]v)", test.Input, got, want)
///// 		}
///// 	}
///// }
/////
///// func TestDecSR64(t *testing.T) {
///// 	tests := []decodeTest{
///// 		decodeTest{[]byte("\x00\x00\x00\x01\x00\x00\x00\x01"), reflect.ValueOf([2]int32{1, 1})},
///// 		decodeTest{[]byte("\x00\x00\x00\x01\xff\xff\xff\xff"), reflect.ValueOf([2]int32{1, -1})},
///// 		decodeTest{[]byte("\xff\xff\xff\xff\x00\x00\x00\x01"), reflect.ValueOf([2]int32{-1, 1})},
///// 		decodeTest{[]byte("\x00\x00\x00\x01\x00\x00\x00\x01"), reflect.ValueOf([2]int32{1, 1})},
///// 		decodeTest{[]byte("\x00\x00\x00\x01\x00\x00\x00\x02"), reflect.ValueOf([2]int32{1, 2})},
///// 		decodeTest{[]byte("\x00\x00\x00\x01\xff\xff\xff\xfe"), reflect.ValueOf([2]int32{1, -2})},
///// 		decodeTest{[]byte("\xff\xff\xff\xff\x00\x00\x00\x02"), reflect.ValueOf([2]int32{-1, 2})},
///// 		decodeTest{[]byte("\x00\x00\x00\x01\x00\x00\x00\x02"), reflect.ValueOf([2]int32{1, 2})},
///// 		decodeTest{[]byte("\x00\x00\x00\x01\x00\x00\x00\x01"), reflect.ValueOf([2]int32{1, 1})},
///// 		decodeTest{[]byte("\x80\x00\x00\x00\x7f\xff\xff\xff"), reflect.ValueOf([2]int32{-2147483648, 2147483647})},
///// 		decodeTest{[]byte("\x7f\xff\xff\xff\x80\x00\x00\x00"), reflect.ValueOf([2]int32{2147483647, -2147483648})},
///// 		decodeTest{[]byte("\x00\x00\x00\x01\x00\x00\x00\x01"), reflect.ValueOf([2]int32{1, 1})},
///// 		decodeTest{[]byte("\x00\x00\x00\x01\x7f\xff\xff\xff"), reflect.ValueOf([2]int32{1, 2147483647})},
///// 		decodeTest{[]byte("\xff\xff\xff\xff\x7f\xff\xff\xff"), reflect.ValueOf([2]int32{-1, 2147483647})},
///// 		decodeTest{[]byte("\x00\x00\x00\x01\x80\x00\x00\x00"), reflect.ValueOf([2]int32{1, -2147483648})},
///// 	}
/////
///// 	for _, test := range tests {
///// 		got := decSR64(test.Input).Interface()
///// 		want := test.Want.Interface()
///// 		if got != want {
///// 			t.Errorf("decSR64(% x)  got %T(%[2]v), want %T(%[3]v)", test.Input, got, want)
///// 		}
///// 	}
///// }
/////
///// func TestDecFC32(t *testing.T) {
///// 	tests := []decodeTest{
///// 		// test printable chars
///// 		decodeTest{[]byte("\x20\x7e\x7d\x7c"), reflect.ValueOf(uint32(0x207e7d7c))},
///// 		decodeTest{[]byte("\x21\x20\x7e\x7d"), reflect.ValueOf(uint32(0x21207e7d))},
///// 		decodeTest{[]byte("\x5c\x21\x20\x7e"), reflect.ValueOf(uint32(0x5c21207e))},
///// 		decodeTest{[]byte("\x23\x5c\x21\x20"), reflect.ValueOf(uint32(0x235c2120))},
///// 		decodeTest{[]byte("\x24\x23\x5c\x21"), reflect.ValueOf(uint32(0x24235c21))},
///// 		decodeTest{[]byte("\x25\x24\x23\x5c"), reflect.ValueOf(uint32(0x2524235c))},
///// 		decodeTest{[]byte("\x26\x25\x24\x23"), reflect.ValueOf(uint32(0x26252423))},
///// 		decodeTest{[]byte("\x27\x26\x25\x24"), reflect.ValueOf(uint32(0x27262524))},
///// 		decodeTest{[]byte("\x28\x27\x26\x25"), reflect.ValueOf(uint32(0x28272625))},
///// 		decodeTest{[]byte("\x29\x28\x27\x26"), reflect.ValueOf(uint32(0x29282726))},
///// 		decodeTest{[]byte("\x2a\x29\x28\x27"), reflect.ValueOf(uint32(0x2a292827))},
///// 		decodeTest{[]byte("\x2b\x2a\x29\x28"), reflect.ValueOf(uint32(0x2b2a2928))},
///// 		decodeTest{[]byte("\x2c\x2b\x2a\x29"), reflect.ValueOf(uint32(0x2c2b2a29))},
///// 		decodeTest{[]byte("\x2d\x2c\x2b\x2a"), reflect.ValueOf(uint32(0x2d2c2b2a))},
///// 		decodeTest{[]byte("\x2e\x2d\x2c\x2b"), reflect.ValueOf(uint32(0x2e2d2c2b))},
///// 		decodeTest{[]byte("\x2f\x2e\x2d\x2c"), reflect.ValueOf(uint32(0x2f2e2d2c))},
///// 		decodeTest{[]byte("\x30\x2f\x2e\x2d"), reflect.ValueOf(uint32(0x302f2e2d))},
///// 		decodeTest{[]byte("\x31\x30\x2f\x2e"), reflect.ValueOf(uint32(0x31302f2e))},
///// 		decodeTest{[]byte("\x32\x31\x30\x2f"), reflect.ValueOf(uint32(0x3231302f))},
///// 		decodeTest{[]byte("\x5b\x5a\x59\x58"), reflect.ValueOf(uint32(0x5b5a5958))},
///// 		decodeTest{[]byte("\x5c\x5b\x5a\x59"), reflect.ValueOf(uint32(0x5c5b5a59))},
///// 		decodeTest{[]byte("\x5d\x5c\x5b\x5a"), reflect.ValueOf(uint32(0x5d5c5b5a))},
///// 		decodeTest{[]byte("\x5e\x5d\x5c\x5b"), reflect.ValueOf(uint32(0x5e5d5c5b))},
///// 		decodeTest{[]byte("\x5f\x5e\x5d\x5c"), reflect.ValueOf(uint32(0x5f5e5d5c))},
///// 		decodeTest{[]byte("\x60\x5f\x5e\x5d"), reflect.ValueOf(uint32(0x605f5e5d))},
///// 		decodeTest{[]byte("\x61\x60\x5f\x5e"), reflect.ValueOf(uint32(0x61605f5e))},
///// 		decodeTest{[]byte("\x62\x61\x60\x5f"), reflect.ValueOf(uint32(0x6261605f))},
///// 		decodeTest{[]byte("\x63\x62\x61\x60"), reflect.ValueOf(uint32(0x63626160))},
///// 		decodeTest{[]byte("\x7b\x7a\x79\x78"), reflect.ValueOf(uint32(0x7b7a7978))},
///// 		decodeTest{[]byte("\x7c\x7b\x7a\x79"), reflect.ValueOf(uint32(0x7c7b7a79))},
///// 		decodeTest{[]byte("\x7d\x7c\x7b\x7a"), reflect.ValueOf(uint32(0x7d7c7b7a))},
///// 		decodeTest{[]byte("\x7e\x7d\x7c\x7b"), reflect.ValueOf(uint32(0x7e7d7c7b))},
///// 		decodeTest{[]byte("\x20\x20\x20\x20"), reflect.ValueOf(uint32(0x20202020))},
///// 		// test a few nonprintable chars
///// 		decodeTest{[]byte("\x00\x00\x00\x00"), reflect.ValueOf(uint32(0x00000000))},
///// 		decodeTest{[]byte("\x00\x00\x00\x01"), reflect.ValueOf(uint32(0x00000001))},
///// 		decodeTest{[]byte("\x00\x00\x00\x02"), reflect.ValueOf(uint32(0x00000002))},
///// 		decodeTest{[]byte("\x00\x00\x00\x03"), reflect.ValueOf(uint32(0x00000003))},
///// 		decodeTest{[]byte("\x00\x00\x00\x04"), reflect.ValueOf(uint32(0x00000004))},
///// 		decodeTest{[]byte("\x00\x00\x00\x05"), reflect.ValueOf(uint32(0x00000005))},
///// 		decodeTest{[]byte("\x00\x00\x00\x06"), reflect.ValueOf(uint32(0x00000006))},
///// 		decodeTest{[]byte("\x00\x00\x00\x07"), reflect.ValueOf(uint32(0x00000007))},
///// 		decodeTest{[]byte("\x00\x00\x00\x08"), reflect.ValueOf(uint32(0x00000008))},
///// 		decodeTest{[]byte("\x00\x00\x00\x09"), reflect.ValueOf(uint32(0x00000009))},
///// 		decodeTest{[]byte("\x00\x00\x00\x0a"), reflect.ValueOf(uint32(0x0000000A))},
///// 		decodeTest{[]byte("\x00\x00\x00\x0b"), reflect.ValueOf(uint32(0x0000000B))},
///// 		decodeTest{[]byte("\x00\x00\x00\x0c"), reflect.ValueOf(uint32(0x0000000C))},
///// 		decodeTest{[]byte("\x00\x00\x00\x0d"), reflect.ValueOf(uint32(0x0000000D))},
///// 		decodeTest{[]byte("\x00\x00\x00\x0e"), reflect.ValueOf(uint32(0x0000000E))},
///// 		decodeTest{[]byte("\x00\x00\x00\x0f"), reflect.ValueOf(uint32(0x0000000F))},
///// 		decodeTest{[]byte("\x01\x00\x00\x00"), reflect.ValueOf(uint32(0x01000000))},
///// 		decodeTest{[]byte("\x02\x00\x00\x00"), reflect.ValueOf(uint32(0x02000000))},
///// 		decodeTest{[]byte("\x03\x00\x00\x00"), reflect.ValueOf(uint32(0x03000000))},
///// 		decodeTest{[]byte("\x04\x00\x00\x00"), reflect.ValueOf(uint32(0x04000000))},
///// 		decodeTest{[]byte("\x05\x00\x00\x00"), reflect.ValueOf(uint32(0x05000000))},
///// 		decodeTest{[]byte("\x06\x00\x00\x00"), reflect.ValueOf(uint32(0x06000000))},
///// 		decodeTest{[]byte("\x07\x00\x00\x00"), reflect.ValueOf(uint32(0x07000000))},
///// 		decodeTest{[]byte("\x08\x00\x00\x00"), reflect.ValueOf(uint32(0x08000000))},
///// 		decodeTest{[]byte("\x09\x00\x00\x00"), reflect.ValueOf(uint32(0x09000000))},
///// 		decodeTest{[]byte("\x0a\x00\x00\x00"), reflect.ValueOf(uint32(0x0A000000))},
///// 		decodeTest{[]byte("\x0b\x00\x00\x00"), reflect.ValueOf(uint32(0x0B000000))},
///// 		decodeTest{[]byte("\x0c\x00\x00\x00"), reflect.ValueOf(uint32(0x0C000000))},
///// 		decodeTest{[]byte("\x0d\x00\x00\x00"), reflect.ValueOf(uint32(0x0D000000))},
///// 		decodeTest{[]byte("\x0e\x00\x00\x00"), reflect.ValueOf(uint32(0x0E000000))},
///// 		decodeTest{[]byte("\x0f\x00\x00\x00"), reflect.ValueOf(uint32(0x0F000000))},
///// 	}
///// 	for _, test := range tests {
///// 		got := decFC32(test.Input).Interface()
///// 		want := test.Want.Interface()
///// 		if got != want {
///// 			t.Errorf("decFC32(% x)  got %T(%[2]v), want %T(%[3]v)", test.Input, got, want)
///// 		}
///// 	}
///// }
/////
///// func TestDecIP32(t *testing.T) {
///// 	tests := []decodeTest{
///// 		decodeTest{[]byte("\x00\x00\x00\x00"), reflect.ValueOf([]byte{0, 0, 0, 0})},
///// 		decodeTest{[]byte("\x11\x22\x33\x44"), reflect.ValueOf([]byte{17, 34, 51, 68})},
///// 		decodeTest{[]byte("\xC0\xA8\x01\x80"), reflect.ValueOf([]byte{192, 168, 1, 128})},
///// 		decodeTest{[]byte("\xF1\xAB\xCD\xEF"), reflect.ValueOf([]byte{241, 171, 205, 239})},
///// 		decodeTest{[]byte("\xff\xff\xff\xff"), reflect.ValueOf([]byte{255, 255, 255, 255})},
///// 	}
///// 	for _, test := range tests {
///// 		got := decIP32(test.Input).Interface().([]byte)
///// 		want := test.Want.Interface().([]byte)
///// 		if got[0] != want[0] || got[1] != want[1] || got[2] != want[2] || got[3] != want[3] {
///// 			t.Errorf(
///// 				"decIP32(%q)  got (%d.%d.%d.%d), want (%d.%d.%d.%d)",
///// 				test.Input,
///// 				got[0], got[1], got[2], got[3],
///// 				want[0], want[1], want[2], want[3])
///// 		}
///// 	}
///// }
/////
///// func TestDecIPAD(t *testing.T) {
///// 	tests := []decodeTest{
///// 		decodeTest{[]byte("\x30\x2e\x30\x2e\x30\x2e\x30\x00"), reflect.ValueOf(string("0.0.0.0"))},
///// 		decodeTest{[]byte("\x31\x2e\x31\x2e\x31\x2e\x31\x00"), reflect.ValueOf(string("1.1.1.1"))},
///// 		decodeTest{[]byte("\x31\x39\x32\x2e\x31\x36\x38\x2e\x30\x2e\x31\x00"), reflect.ValueOf(string("192.168.0.1"))},
///// 		decodeTest{[]byte("\x32\x35\x35\x2e\x32\x35\x35\x2e\x32\x35\x35\x2e\x32\x35\x35\x00"), reflect.ValueOf(string("255.255.255.255"))},
///// 		decodeTest{[]byte("\x31\x39\x32\x2e\x31\x36\x38\x2e\x31\x2e\x30\x00"), reflect.ValueOf(string("192.168.1.0"))},
///// 		decodeTest{[]byte("\x31\x30\x2e\x32\x35\x35\x2e\x32\x35\x35\x2e\x32\x35\x34\x00"), reflect.ValueOf(string("10.255.255.254"))},
///// 		decodeTest{[]byte("\x31\x37\x32\x2e\x31\x38\x2e\x35\x2e\x34\x00"), reflect.ValueOf(string("172.18.5.4"))},
///// 		decodeTest{[]byte("\x38\x2e\x38\x2e\x34\x2e\x34\x00"), reflect.ValueOf(string("8.8.4.4"))},
///// 		decodeTest{[]byte("\x31\x32\x37\x2e\x30\x2e\x30\x2e\x31\x00"), reflect.ValueOf(string("127.0.0.1"))},
///// 		decodeTest{[]byte("\x31\x2e\x32\x35\x35\x2e\x33\x2e\x34\x00"), reflect.ValueOf(string("1.255.3.4"))},
///// 		decodeTest{[]byte("\x32\x35\x35\x2e\x30\x2e\x30\x2e\x31\x00"), reflect.ValueOf(string("255.0.0.1"))},
///// 		decodeTest{[]byte("\x3a\x3a\x00"), reflect.ValueOf(string("::"))},
///// 		decodeTest{[]byte("\x3a\x3a\x66\x66\x66\x66\x3a\x35\x2e\x36\x2e\x37\x2e\x38\x00"), reflect.ValueOf(string("::ffff:5.6.7.8"))},
///// 		decodeTest{[]byte("\x66\x64\x66\x38\x3a\x66\x35\x33\x62\x3a\x38\x32\x65\x34\x3a\x3a\x35\x33\x00"), reflect.ValueOf(string("fdf8:f53b:82e4::53"))},
///// 		decodeTest{[]byte("\x66\x65\x38\x30\x3a\x3a\x32\x30\x30\x3a\x35\x61\x65\x65\x3a\x66\x65\x61\x61\x3a\x32\x30\x61\x32\x00"), reflect.ValueOf(string("fe80::200:5aee:feaa:20a2"))},
///// 		decodeTest{[]byte("\x32\x30\x30\x31\x3a\x3a\x31\x00"), reflect.ValueOf(string("2001::1"))},
///// 		decodeTest{[]byte("\x32\x30\x30\x31\x3a\x30\x30\x30\x30\x3a\x34\x31\x33\x36\x3a\x65\x33\x37\x38\x3a\x38\x30\x30\x30\x3a\x36\x33\x62\x66\x3a\x33\x66\x66\x66\x3a\x66\x64\x64\x32\x00"), reflect.ValueOf(string("2001:0000:4136:e378:8000:63bf:3fff:fdd2"))},
///// 		decodeTest{[]byte("\x32\x30\x30\x31\x3a\x30\x30\x30\x32\x3a\x36\x63\x3a\x3a\x34\x33\x30\x00"), reflect.ValueOf(string("2001:0002:6c::430"))},
///// 		decodeTest{[]byte("\x32\x30\x30\x31\x3a\x31\x30\x3a\x32\x34\x30\x3a\x61\x62\x3a\x3a\x61\x00"), reflect.ValueOf(string("2001:10:240:ab::a"))},
///// 		decodeTest{[]byte("\x32\x30\x30\x32\x3a\x63\x62\x30\x61\x3a\x33\x63\x64\x64\x3a\x31\x3a\x3a\x31\x00"), reflect.ValueOf(string("2002:cb0a:3cdd:1::1"))},
///// 		decodeTest{[]byte("\x32\x30\x30\x31\x3a\x64\x62\x38\x3a\x38\x3a\x34\x3a\x3a\x32\x00"), reflect.ValueOf(string("2001:db8:8:4::2"))},
///// 		decodeTest{[]byte("\x66\x66\x30\x31\x3a\x30\x3a\x30\x3a\x30\x3a\x30\x3a\x30\x3a\x30\x3a\x32\x00"), reflect.ValueOf(string("ff01:0:0:0:0:0:0:2"))},
///// 		decodeTest{[]byte("\x66\x64\x66\x38\x3a\x66\x35\x33\x62\x3a\x38\x32\x65\x34\x3a\x3a\x35\x33\x00"), reflect.ValueOf(string("fdf8:f53b:82e4::53"))},
///// 		decodeTest{[]byte("\x32\x30\x30\x31\x3a\x3a\x31\x00"), reflect.ValueOf(string("2001::1"))},
///// 		decodeTest{[]byte("\x32\x30\x30\x31\x3a\x30\x30\x30\x30\x3a\x34\x31\x33\x36\x3a\x65\x33\x37\x38\x3a\x38\x30\x30\x30\x3a\x36\x33\x62\x66\x3a\x33\x66\x66\x66\x3a\x66\x64\x64\x32\x00"), reflect.ValueOf(string("2001:0000:4136:e378:8000:63bf:3fff:fdd2"))},
///// 	}
///// 	for _, test := range tests {
///// 		got := decIPAD(test.Input).String()
///// 		want := test.Want.String()
///// 		if got != want {
///// 			t.Errorf("decIPAD(%q)  got '%s', want '%s'", test.Input, got, want)
///// 		}
///// 	}
///// }
/////
///// func TestDecCSTR(t *testing.T) {
///// 	testData := make(map[string]string)
///// 	// tests for ascii 1-127, 0 is defined invalid
///// 	testData = map[string]string{
///// 		"    \x01\x02\x03": "    \x01\x02\x03",
///// 		"\x04\x05\x06\x07": "\x04\x05\x06\x07",
///// 		"\x08\x09\x0a\x0b": "\x08\x09\n\x0B",
///// 		"\x0c\x0d\x0e\x0f": "\x0C\r\x0E\x0F",
///// 		"\x10\x11\x12\x13": "\x10\x11\x12\x13",
///// 		"\x14\x15\x16\x17": "\x14\x15\x16\x17",
///// 		"\x18\x19\x1a\x1b": "\x18\x19\x1A\x1B",
///// 		"\x1c\x1d\x1e\x1f": "\x1C\x1D\x1E\x1F",
///// 		"\x20\x21\x22\x23": " !\"#",
///// 		"\x24\x25\x26\x27": "$%&'",
///// 		"\x28\x29\x2a\x2b": "()*+",
///// 		"\x2c\x2d\x2e\x2f": ",-./",
///// 		"\x30\x31\x32\x33": "0123",
///// 		"\x34\x35\x36\x37": "4567",
///// 		"\x38\x39\x3a\x3b": "89:;",
///// 		"\x3c\x3d\x3e\x3f": "<=>?",
///// 		"\x40\x41\x42\x43": "@ABC",
///// 		"\x44\x45\x46\x47": "DEFG",
///// 		"\x48\x49\x4a\x4b": "HIJK",
///// 		"\x4c\x4d\x4e\x4f": "LMNO",
///// 		"\x50\x51\x52\x53": "PQRS",
///// 		"\x54\x55\x56\x57": "TUVW",
///// 		"\x58\x59\x5a\x5b": "XYZ[",
///// 		"\x5c\x5d\x5e\x5f": "\\]^_",
///// 		"\x60\x61\x62\x63": "`abc",
///// 		"\x64\x65\x66\x67": "defg",
///// 		"\x68\x69\x6a\x6b": "hijk",
///// 		"\x6c\x6d\x6e\x6f": "lmno",
///// 		"\x70\x71\x72\x73": "pqrs",
///// 		"\x74\x75\x76\x77": "tuvw",
///// 		"\x78\x79\x7a\x7b": "xyz{",
///// 		"\x7c\x7d\x7e\x7f": "|}~\x7F",
///// 	}
///// 	tests := []decodeTest{}
///// 	for input, expect := range testData {
///// 		test := decodeTest{[]byte(input), reflect.ValueOf(expect)}
///// 		tests = append(tests, test)
///// 	}
///// 	for _, test := range tests {
///// 		got := fmt.Sprintf("%x", decCSTR(test.Input).Interface())
///// 		want := fmt.Sprintf("%x", test.Want.Interface())
///// 		if got != want {
///// 			t.Errorf("decCSTR(%q) = %T(%[2]v), want %T(%[3]v)", test.Input, got, want)
///// 		}
///// 	}
///// }
/////
///// func TestDecUSTR(t *testing.T) {
///// 	testData := make(map[string]string)
///// 	// tests for ascii 0-127
///// 	testData = map[string]string{
///// 		"\x00\x00\x00\x00\x00\x00\x00\x40": "\x00@",
///// 		"\x00\x00\x00\x01\x00\x00\x00\x41": "\x01A",
///// 		"\x00\x00\x00\x02\x00\x00\x00\x42": "\x02B",
///// 		"\x00\x00\x00\x03\x00\x00\x00\x43": "\x03C",
///// 		"\x00\x00\x00\x04\x00\x00\x00\x44": "\x04D",
///// 		"\x00\x00\x00\x05\x00\x00\x00\x45": "\x05E",
///// 		"\x00\x00\x00\x06\x00\x00\x00\x46": "\x06F",
///// 		"\x00\x00\x00\x07\x00\x00\x00\x47": "\x07G",
///// 		"\x00\x00\x00\x08\x00\x00\x00\x48": "\x08H",
///// 		"\x00\x00\x00\x09\x00\x00\x00\x49": "\x09I",
///// 		"\x00\x00\x00\x0A\x00\x00\x00\x4A": "\x0AJ",
///// 		"\x00\x00\x00\x0B\x00\x00\x00\x4B": "\x0BK",
///// 		"\x00\x00\x00\x0C\x00\x00\x00\x4C": "\x0CL",
///// 		"\x00\x00\x00\x0D\x00\x00\x00\x4D": "\x0DM",
///// 		"\x00\x00\x00\x0E\x00\x00\x00\x4E": "\x0EN",
///// 		"\x00\x00\x00\x0F\x00\x00\x00\x4F": "\x0FO",
///// 		"\x00\x00\x00\x10\x00\x00\x00\x50": "\x10P",
///// 		"\x00\x00\x00\x11\x00\x00\x00\x51": "\x11Q",
///// 		"\x00\x00\x00\x12\x00\x00\x00\x52": "\x12R",
///// 		"\x00\x00\x00\x13\x00\x00\x00\x53": "\x13S",
///// 		"\x00\x00\x00\x14\x00\x00\x00\x54": "\x14T",
///// 		"\x00\x00\x00\x15\x00\x00\x00\x55": "\x15U",
///// 		"\x00\x00\x00\x16\x00\x00\x00\x56": "\x16V",
///// 		"\x00\x00\x00\x17\x00\x00\x00\x57": "\x17W",
///// 		"\x00\x00\x00\x18\x00\x00\x00\x58": "\x18X",
///// 		"\x00\x00\x00\x19\x00\x00\x00\x59": "\x19Y",
///// 		"\x00\x00\x00\x1A\x00\x00\x00\x5A": "\x1AZ",
///// 		"\x00\x00\x00\x1B\x00\x00\x00\x5B": "\x1B[",
///// 		"\x00\x00\x00\x1C\x00\x00\x00\x5C": "\x1C\\",
///// 		"\x00\x00\x00\x1D\x00\x00\x00\x5D": "\x1D]",
///// 		"\x00\x00\x00\x1E\x00\x00\x00\x5E": "\x1E^",
///// 		"\x00\x00\x00\x1F\x00\x00\x00\x5F": "\x1F_",
///// 		"\x00\x00\x00\x20\x00\x00\x00\x60": "\x20`",
///// 		"\x00\x00\x00\x21\x00\x00\x00\x61": "\x21a",
///// 		"\x00\x00\x00\x22\x00\x00\x00\x62": "\x22b",
///// 		"\x00\x00\x00\x23\x00\x00\x00\x63": "#c",
///// 		"\x00\x00\x00\x24\x00\x00\x00\x64": "$d",
///// 		"\x00\x00\x00\x25\x00\x00\x00\x65": "%e",
///// 		"\x00\x00\x00\x26\x00\x00\x00\x66": "&f",
///// 		"\x00\x00\x00\x27\x00\x00\x00\x67": "'g",
///// 		"\x00\x00\x00\x28\x00\x00\x00\x68": "(h",
///// 		"\x00\x00\x00\x29\x00\x00\x00\x69": ")i",
///// 		"\x00\x00\x00\x2A\x00\x00\x00\x6A": "*j",
///// 		"\x00\x00\x00\x2B\x00\x00\x00\x6B": "+k",
///// 		"\x00\x00\x00\x2C\x00\x00\x00\x6C": ",l",
///// 		"\x00\x00\x00\x2D\x00\x00\x00\x6D": "-m",
///// 		"\x00\x00\x00\x2E\x00\x00\x00\x6E": ".n",
///// 		"\x00\x00\x00\x2F\x00\x00\x00\x6F": "/o",
///// 		"\x00\x00\x00\x30\x00\x00\x00\x70": "0p",
///// 		"\x00\x00\x00\x31\x00\x00\x00\x71": "1q",
///// 		"\x00\x00\x00\x32\x00\x00\x00\x72": "2r",
///// 		"\x00\x00\x00\x33\x00\x00\x00\x73": "3s",
///// 		"\x00\x00\x00\x34\x00\x00\x00\x74": "4t",
///// 		"\x00\x00\x00\x35\x00\x00\x00\x75": "5u",
///// 		"\x00\x00\x00\x36\x00\x00\x00\x76": "6v",
///// 		"\x00\x00\x00\x37\x00\x00\x00\x77": "7w",
///// 		"\x00\x00\x00\x38\x00\x00\x00\x78": "8x",
///// 		"\x00\x00\x00\x39\x00\x00\x00\x79": "9y",
///// 		"\x00\x00\x00\x3A\x00\x00\x00\x7A": ":z",
///// 		"\x00\x00\x00\x3B\x00\x00\x00\x7B": ";{",
///// 		"\x00\x00\x00\x3C\x00\x00\x00\x7C": "<|",
///// 		"\x00\x00\x00\x3D\x00\x00\x00\x7D": "=}",
///// 		"\x00\x00\x00\x3E\x00\x00\x00\x7E": ">~",
///// 		"\x00\x00\x00\x3F\x00\x00\x00\x7F": "?\x7F",
///// 	}
///// 	tests := []decodeTest{}
///// 	for input, expect := range testData {
///// 		test := decodeTest{[]byte(input), reflect.ValueOf(expect)}
///// 		tests = append(tests, test)
///// 	}
///// 	for _, test := range tests {
///// 		got := fmt.Sprintf("%x", decUSTR(test.Input).Interface())
///// 		want := fmt.Sprintf("%x", test.Want.Interface())
///// 		if got != want {
///// 			t.Errorf("decUSTR(%q) = %T(%[2]v), want %T(%[3]v)", test.Input, got, want)
///// 		}
///// 	}
///// }
/////
///// // test ascii 0-127
///// func TestDecDATA(t *testing.T) {
///// 	testData := [][]byte{
///// 		[]byte("\x00\x01\x02\x03\x04\x05\x06\x07"),
///// 		[]byte("\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f"),
///// 		[]byte("\x10\x11\x12\x13\x14\x15\x16\x17"),
///// 		[]byte("\x18\x19\x1a\x1b\x1c\x1d\x1e\x1f"),
///// 		[]byte("\x20\x21\x22\x23\x24\x25\x26\x27"),
///// 		[]byte("\x28\x29\x2a\x2b\x2c\x2d\x2e\x2f"),
///// 		[]byte("\x30\x31\x32\x33\x34\x35\x36\x37"),
///// 		[]byte("\x38\x39\x3a\x3b\x3c\x3d\x3e\x3f"),
///// 		[]byte("\x40\x41\x42\x43\x44\x45\x46\x47"),
///// 		[]byte("\x48\x49\x4a\x4b\x4c\x4d\x4e\x4f"),
///// 		[]byte("\x50\x51\x52\x53\x54\x55\x56\x57"),
///// 		[]byte("\x58\x59\x5a\x5b\x5c\x5d\x5e\x5f"),
///// 		[]byte("\x60\x61\x62\x63\x64\x65\x66\x67"),
///// 		[]byte("\x68\x69\x6a\x6b\x6c\x6d\x6e\x6f"),
///// 		[]byte("\x70\x71\x72\x73\x74\x75\x76\x77"),
///// 		[]byte("\x78\x79\x7a\x7b\x7c\x7d\x7e\x7f"),
///// 	}
///// 	tests := []decodeTest{}
///// 	for _, buf := range testData {
///// 		tests = append(tests, decodeTest{buf, reflect.ValueOf(buf)})
///// 	}
///// 	for _, test := range tests {
///// 		got := decDATA(test.Input).String()
///// 		want := test.Want.String()
///// 		if got != want {
///// 			t.Errorf("decDATA(%q)  got '%s', want '%s'", test.Input, got, want)
///// 		}
///// 	}
///// }
/////
///// func TestDecNULL(t *testing.T) {
///// 	testData := [][]byte{
///// 		[]byte{},
///// 		[]byte{0x00},
///// 		[]byte{0x01},
///// 		[]byte(nil),
///// 		[]byte(""),
///// 		[]byte("abcdefghijk"),
///// 		[]byte("\x00"),
///// 	}
///// 	tests := []decodeTest{}
///// 	for _, buf := range testData {
///// 		tests = append(tests, decodeTest{buf, reflect.ValueOf(nil)})
///// 	}
///// 	for _, test := range tests {
///// 		got := decNULL(test.Input)
///// 		want := test.Want
///// 		if got != want {
///// 			t.Errorf("decNULL(%q)  got '%s', want '%s'", test.Input, got, want)
///// 		}
///// 	}
///// }
/////
///// /**********************************************************/
///// // test methods for conversion to escaped string
///// /**********************************************************/
///// /*
///// func TestStrUI01(t *testing.T) {
///// func TestStrUI08(t *testing.T) {
///// func TestStrUI16(t *testing.T) {
///// func TestStrUI32(t *testing.T) {
///// func TestStrUI64(t *testing.T) {
///// func TestStrSI08(t *testing.T) {
///// func TestStrSI16(t *testing.T) {
///// func TestStrSI32(t *testing.T) {
///// func TestStrSI64(t *testing.T) {
///// func TestStrFP32(t *testing.T) {
///// func TestStrFP64(t *testing.T) {
///// func TestStrUF32(t *testing.T) {
///// */
///// func TestStrUF64(t *testing.T) {
///// 	tests := []stringTest{
///// 		stringTest{[]byte("\x00\x00\x00\x00\x00\x00\x00\x00"), "0.000000000"},
///// 		stringTest{[]byte("\x00\x00\x00\x01\x00\x00\x00\x00"), "1.000000000"},
///// 		stringTest{[]byte("\x00\x01\x00\x3c\x00\x00\x96\xfe"), "65596.000009000"},
///// 		stringTest{[]byte("\xff\xff\xff\xff\x00\x00\x00\x00"), "4294967295.000000000"},
///// 		stringTest{[]byte("\xff\xff\xff\xfe\x00\x00\x00\x00"), "4294967294.000000000"},
///// 		stringTest{[]byte("\xff\xff\xff\xff\x19\x99\x99\x99"), "4294967295.100000000"},
///// 		stringTest{[]byte("\xff\xff\xff\xff\x02\x8f\x5c\x28"), "4294967295.010000000"},
///// 		stringTest{[]byte("\xff\xff\xff\xff\x00\x41\x89\x37"), "4294967295.001000000"},
///// 		stringTest{[]byte("\xff\xff\xff\xff\x00\x06\x8d\xb8"), "4294967295.000100000"},
///// 		stringTest{[]byte("\xff\xff\xff\xff\x00\x00\xa7\xc5"), "4294967295.000010000"},
///// 		stringTest{[]byte("\xff\xff\xff\xff\x00\x00\x10\xc6"), "4294967295.000001000"},
///// 		stringTest{[]byte("\xff\xff\xff\xff\xff\xff\xff\xfb"), "4294967295.999999999"},
///// 	}
/////
///// 	for _, test := range tests {
///// 		got := strUF64(test.Input)
///// 		want := test.Want
///// 		if got != want {
///// 			t.Errorf("strUF64(% x)  got '%v', want '%v'", test.Input, got, want)
///// 		}
///// 	}
///// }
/////
///// /*
///// func TestStrSF32(t *testing.T) {
///// func TestStrSF64(t *testing.T) {
///// func TestStrUR32(t *testing.T) {
///// func TestStrUR64(t *testing.T) {
///// func TestStrSR32(t *testing.T) {
///// func TestStrSR64(t *testing.T) {
///// func TestStrFC32(t *testing.T) {
///// func TestStrIP32(t *testing.T) {
///// func TestStrCSTR(t *testing.T) {
///// */
///// func TestStrUSTR(t *testing.T) {
///// 	testData := make(map[string]string)
///// 	testData = map[string]string{
///// 		"\x00\x00\x00\x01": "\\x01",
///// 		"\x00\x00\x00\x02": "\\x02",
///// 		"\x00\x00\x00\x03": "\\x03",
///// 		"\x00\x00\x00\x04": "\\x04",
///// 		"\x00\x00\x00\x05": "\\x05",
///// 		"\x00\x00\x00\x06": "\\x06",
///// 		"\x00\x00\x00\x07": "\\x07",
///// 		"\x00\x00\x00\x08": "\\x08",
///// 		"\x00\x00\x00\x09": "\\x09",
///// 		"\x00\x00\x00\x0A": "\\n",
///// 		"\x00\x00\x00\x0B": "\\x0B",
///// 		"\x00\x00\x00\x0C": "\\x0C",
///// 		"\x00\x00\x00\x0D": "\\r",
///// 		"\x00\x00\x00\x0E": "\\x0E",
///// 		"\x00\x00\x00\x0F": "\\x0F",
///// 		"\x00\x00\x00\x10": "\\x10",
///// 		"\x00\x00\x00\x11": "\\x11",
///// 		"\x00\x00\x00\x12": "\\x12",
///// 		"\x00\x00\x00\x13": "\\x13",
///// 		"\x00\x00\x00\x14": "\\x14",
///// 		"\x00\x00\x00\x15": "\\x15",
///// 		"\x00\x00\x00\x16": "\\x16",
///// 		"\x00\x00\x00\x17": "\\x17",
///// 		"\x00\x00\x00\x18": "\\x18",
///// 		"\x00\x00\x00\x19": "\\x19",
///// 		"\x00\x00\x00\x1A": "\\x1A",
///// 		"\x00\x00\x00\x1B": "\\x1B",
///// 		"\x00\x00\x00\x1C": "\\x1C",
///// 		"\x00\x00\x00\x1D": "\\x1D",
///// 		"\x00\x00\x00\x1E": "\\x1E",
///// 		"\x00\x00\x00\x1F": "\\x1F",
///// 		"\x00\x00\x00\x20": " ",
///// 		"\x00\x00\x00\x21": "!",
///// 		"\x00\x00\x00\x22": "\\\"",
///// 		"\x00\x00\x00\x23": "#",
///// 		"\x00\x00\x00\x24": "$",
///// 		"\x00\x00\x00\x25": "%",
///// 		"\x00\x00\x00\x26": "&",
///// 		"\x00\x00\x00\x27": "'",
///// 		"\x00\x00\x00\x28": "(",
///// 		"\x00\x00\x00\x29": ")",
///// 		"\x00\x00\x00\x2A": "*",
///// 		"\x00\x00\x00\x2B": "+",
///// 		"\x00\x00\x00\x2C": ",",
///// 		"\x00\x00\x00\x2D": "-",
///// 		"\x00\x00\x00\x2E": ".",
///// 		"\x00\x00\x00\x2F": "/",
///// 		"\x00\x00\x00\x30": "0",
///// 		"\x00\x00\x00\x31": "1",
///// 		"\x00\x00\x00\x32": "2",
///// 		"\x00\x00\x00\x33": "3",
///// 		"\x00\x00\x00\x34": "4",
///// 		"\x00\x00\x00\x35": "5",
///// 		"\x00\x00\x00\x36": "6",
///// 		"\x00\x00\x00\x37": "7",
///// 		"\x00\x00\x00\x38": "8",
///// 		"\x00\x00\x00\x39": "9",
///// 		"\x00\x00\x00\x3A": ":",
///// 		"\x00\x00\x00\x3B": ";",
///// 		"\x00\x00\x00\x3C": "<",
///// 		"\x00\x00\x00\x3D": "=",
///// 		"\x00\x00\x00\x3E": ">",
///// 		"\x00\x00\x00\x3F": "?",
///// 		"\x00\x00\x00\x40": "@",
///// 		"\x00\x00\x00\x41": "A",
///// 		"\x00\x00\x00\x42": "B",
///// 		"\x00\x00\x00\x43": "C",
///// 		"\x00\x00\x00\x44": "D",
///// 		"\x00\x00\x00\x45": "E",
///// 		"\x00\x00\x00\x46": "F",
///// 		"\x00\x00\x00\x47": "G",
///// 		"\x00\x00\x00\x48": "H",
///// 		"\x00\x00\x00\x49": "I",
///// 		"\x00\x00\x00\x4A": "J",
///// 		"\x00\x00\x00\x4B": "K",
///// 		"\x00\x00\x00\x4C": "L",
///// 		"\x00\x00\x00\x4D": "M",
///// 		"\x00\x00\x00\x4E": "N",
///// 		"\x00\x00\x00\x4F": "O",
///// 		"\x00\x00\x00\x50": "P",
///// 		"\x00\x00\x00\x51": "Q",
///// 		"\x00\x00\x00\x52": "R",
///// 		"\x00\x00\x00\x53": "S",
///// 		"\x00\x00\x00\x54": "T",
///// 		"\x00\x00\x00\x55": "U",
///// 		"\x00\x00\x00\x56": "V",
///// 		"\x00\x00\x00\x57": "W",
///// 		"\x00\x00\x00\x58": "X",
///// 		"\x00\x00\x00\x59": "Y",
///// 		"\x00\x00\x00\x5A": "Z",
///// 		"\x00\x00\x00\x5B": "[",
///// 		"\x00\x00\x00\x5C": "\\\\",
///// 		"\x00\x00\x00\x5D": "]",
///// 		"\x00\x00\x00\x5E": "^",
///// 		"\x00\x00\x00\x5F": "_",
///// 		"\x00\x00\x00\x60": "`",
///// 		"\x00\x00\x00\x61": "a",
///// 		"\x00\x00\x00\x62": "b",
///// 		"\x00\x00\x00\x63": "c",
///// 		"\x00\x00\x00\x64": "d",
///// 		"\x00\x00\x00\x65": "e",
///// 		"\x00\x00\x00\x66": "f",
///// 		"\x00\x00\x00\x67": "g",
///// 		"\x00\x00\x00\x68": "h",
///// 		"\x00\x00\x00\x69": "i",
///// 		"\x00\x00\x00\x6A": "j",
///// 		"\x00\x00\x00\x6B": "k",
///// 		"\x00\x00\x00\x6C": "l",
///// 		"\x00\x00\x00\x6D": "m",
///// 		"\x00\x00\x00\x6E": "n",
///// 		"\x00\x00\x00\x6F": "o",
///// 		"\x00\x00\x00\x70": "p",
///// 		"\x00\x00\x00\x71": "q",
///// 		"\x00\x00\x00\x72": "r",
///// 		"\x00\x00\x00\x73": "s",
///// 		"\x00\x00\x00\x74": "t",
///// 		"\x00\x00\x00\x75": "u",
///// 		"\x00\x00\x00\x76": "v",
///// 		"\x00\x00\x00\x77": "w",
///// 		"\x00\x00\x00\x78": "x",
///// 		"\x00\x00\x00\x79": "y",
///// 		"\x00\x00\x00\x7A": "z",
///// 		"\x00\x00\x00\x7B": "{",
///// 		"\x00\x00\x00\x7C": "|",
///// 		"\x00\x00\x00\x7D": "}",
///// 		"\x00\x00\x00\x7E": "~",
///// 		"\x00\x00\x00\x7F": "\\x7F",
///// 	}
///// 	for in, out := range testData {
///// 		got := strUSTR([]byte(in))
///// 		want := fmt.Sprintf("\"%s\"", out)
///// 		if got != want {
///// 			fmt.Printf("hex(%q) got(%x)len(%d) want(%x)len(%d)\n", in, got, len(got), want, len(want))
///// 			t.Errorf("strUSTR(%q) got %[2]T(%[2]v), want %[3]T(%[3]v)", in, got, want)
///// 		}
///// 	}
///// }
/////
///// /*
///// func TestStrDATA(t *testing.T) {
///// func TestStrUUID(t *testing.T) {
///// func TestStrNULL(t *testing.T) {
///// func TestAsPrintableString(t *testing.T) {
///// func Test(t *testing.T) {
///// func Test(t *testing.T) {
///// */
