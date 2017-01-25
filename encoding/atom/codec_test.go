package atom

import (
	"bytes"
	"encoding/binary"
	"math"
	"reflect"
	"testing"
)

type decodeTest struct {
	Input []byte
	Want  reflect.Value
}

// The specification says explicitly not to store UINT01 as a bool.
// 112-0002_r4.0B_StorageGRID_Data_Types
func TestDecUI01(t *testing.T) {
	tests := []decodeTest{
		// Not a mistake, we really do use 4 bytes for this type
		decodeTest{[]byte{0x00, 0x00, 0x00, 0x00}, reflect.ValueOf(uint32(0))},
		decodeTest{[]byte{0x00, 0x00, 0x00, 0x01}, reflect.ValueOf(uint32(1))},
	}
	for _, test := range tests {
		got := decUI01(test.Input).Interface()
		want := test.Want.Interface()
		if got != want {
			t.Errorf("decUI01(%q) = %T(%[2]v), want %T(%[3]v)", test.Input, got, want)
		}
	}
}

func TestDecUI08(t *testing.T) {
	tests := []decodeTest{
		decodeTest{[]byte{0x00}, reflect.ValueOf(byte(0))},
		decodeTest{[]byte{0x0F}, reflect.ValueOf(byte(15))},
		decodeTest{[]byte{0xF0}, reflect.ValueOf(byte(240))},
		decodeTest{[]byte{0xFF}, reflect.ValueOf(byte(255))},
	}
	for _, test := range tests {
		got := decUI08(test.Input).Interface()
		want := test.Want.Interface()
		if got != want {
			t.Errorf("decUI08(%q) = %T(%[2]v), want %T(%[3]v)", test.Input, got, want)
		}
	}
}

func TestDecUI16(t *testing.T) {
	tests := []decodeTest{
		decodeTest{[]byte{0x00, 0x00}, reflect.ValueOf(uint16(0))},
		decodeTest{[]byte{0x00, 0xFF}, reflect.ValueOf(uint16(255))},
		decodeTest{[]byte{0xFF, 0x00}, reflect.ValueOf(uint16(65280))},
		decodeTest{[]byte{0xFF, 0xFF}, reflect.ValueOf(uint16(65535))},
	}
	for _, test := range tests {
		got := decUI16(test.Input).Interface()
		want := test.Want.Interface()
		if got != want {
			t.Errorf("decUI16(%q) = %T(%[2]v), want %T(%[3]v)", test.Input, got, want)
		}
	}
}

func TestDecUI32(t *testing.T) {
	tests := []decodeTest{
		decodeTest{[]byte{0x00, 0x00, 0x00, 0x00}, reflect.ValueOf(uint32(0x00000000))},
		decodeTest{[]byte{0x00, 0x00, 0x00, 0xFF}, reflect.ValueOf(uint32(0x000000FF))},
		decodeTest{[]byte{0x00, 0x00, 0xFF, 0x00}, reflect.ValueOf(uint32(0x0000FF00))},
		decodeTest{[]byte{0x00, 0xFF, 0x00, 0x00}, reflect.ValueOf(uint32(0x00FF0000))},
		decodeTest{[]byte{0xFF, 0x00, 0x00, 0x00}, reflect.ValueOf(uint32(0xFF000000))},
		decodeTest{[]byte{0xFF, 0xFF, 0xFF, 0xFF}, reflect.ValueOf(uint32(0xFFFFFFFF))},
	}
	for _, test := range tests {
		got := decUI32(test.Input).Interface()
		want := test.Want.Interface()
		if got != want {
			t.Errorf("decUI32(%q) = %T(%[2]v), want %T(%[3]v)", test.Input, got, want)
		}
	}
}

func TestDecUI64(t *testing.T) {
	tests := []decodeTest{
		decodeTest{[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, reflect.ValueOf(uint64(0x0000000000000000))},
		decodeTest{[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xFF}, reflect.ValueOf(uint64(0x00000000000000FF))},
		decodeTest{[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xFF, 0x00}, reflect.ValueOf(uint64(0x000000000000FF00))},
		decodeTest{[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0xFF, 0x00, 0x00}, reflect.ValueOf(uint64(0x0000000000FF0000))},
		decodeTest{[]byte{0x00, 0x00, 0x00, 0x00, 0xFF, 0x00, 0x00, 0x00}, reflect.ValueOf(uint64(0x00000000FF000000))},
		decodeTest{[]byte{0x00, 0x00, 0x00, 0xFF, 0x00, 0x00, 0x00, 0x00}, reflect.ValueOf(uint64(0x000000FF00000000))},
		decodeTest{[]byte{0x00, 0x00, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00}, reflect.ValueOf(uint64(0x0000FF0000000000))},
		decodeTest{[]byte{0x00, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, reflect.ValueOf(uint64(0x00FF000000000000))},
		decodeTest{[]byte{0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, reflect.ValueOf(uint64(0xFF00000000000000))},
		decodeTest{[]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}, reflect.ValueOf(uint64(0xFFFFFFFFFFFFFFFF))},
	}
	for _, test := range tests {
		got := decUI64(test.Input).Interface()
		want := test.Want.Interface()
		if got != want {
			t.Errorf("decUI64(%q) = %T(%[2]v), want %T(%[3]v)", test.Input, got, want)
		}
	}
}

func TestDecSF32(t *testing.T) {
	tests := []decodeTest{
		decodeTest{[]byte{0x00, 0x00, 0x00, 0x00}, reflect.ValueOf(float32(0))},
		decodeTest{[]byte{0x00, 0x00, 0x00, 0x01}, reflect.ValueOf(float32(1.5258789e-05))},
		decodeTest{[]byte{0x00, 0x00, 0x00, 0xFF}, reflect.ValueOf(float32(0.0038909912))},
		decodeTest{[]byte{0x00, 0x00, 0xFF, 0x00}, reflect.ValueOf(float32(0.99609375))},
		decodeTest{[]byte{0x00, 0xFF, 0x00, 0x00}, reflect.ValueOf(float32(255.0))},
		decodeTest{[]byte{0xFF, 0xFF, 0xFF, 0xFF}, reflect.ValueOf(float32(-1.5258789e-05))},
	}
	for _, test := range tests {
		got := decSF32(test.Input).Interface()
		want := test.Want.Interface()
		if got != want {
			t.Errorf("decSF32(%q) = %T(%[2]v), want %T(%[3]v)", test.Input, got, want)
		}
	}
}

func TestDecSF64(t *testing.T) {
	tests := []decodeTest{
		decodeTest{[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, reflect.ValueOf(float64(0))},
		decodeTest{[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}, reflect.ValueOf(float64(2.3283064365386963e-10))},
		decodeTest{[]byte{0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01}, reflect.ValueOf(float64(1.684300900392157e+07))},
		decodeTest{[]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}, reflect.ValueOf(float64(-2.3283064365386963e-10))},
	}
	for _, test := range tests {
		got := decSF64(test.Input).Interface()
		want := test.Want.Interface()
		if got != want {
			t.Errorf("decSF64(%q) = %T(%[2]v), want %T(%[3]v)", test.Input, got, want)
		}
	}
}
func TestDecSI08(t *testing.T) {
	Min := int8(math.MinInt8)

	tests := []decodeTest{
		decodeTest{[]byte{0}, reflect.ValueOf(int8(0))},
		decodeTest{[]byte{math.MaxInt8}, reflect.ValueOf(int8(127))},
	}

	// add test of min value for this type
	var buf = bytes.NewBuffer(make([]byte, 0, 2))
	binary.Write(buf, binary.BigEndian, &Min)
	tests = append(tests, decodeTest{buf.Bytes(), reflect.ValueOf(Min)})

	for _, test := range tests {
		got := decSI08(test.Input).Interface()
		want := test.Want.Interface()
		if got != want {
			t.Errorf("decSI08(%q) = %T(%[2]v), want %T(%[3]v)", test.Input, got, want)
		}
	}
}
func TestDecSI16(t *testing.T) {
	Min := int16(math.MinInt16)
	Max := int16(math.MaxInt16)

	// sanity check of some handcoded values
	tests := []decodeTest{
		decodeTest{[]byte{0x00, 0x00}, reflect.ValueOf(int16(0))},
		decodeTest{[]byte{0x00, 0x01}, reflect.ValueOf(int16(1))},
		decodeTest{[]byte{0x00, 0xFF}, reflect.ValueOf(int16(255))},
		decodeTest{[]byte{0xFF, 0x00}, reflect.ValueOf(int16(-256))},
		decodeTest{[]byte{0xFF, 0xFF}, reflect.ValueOf(int16(-1))},
	}

	// add test of min value for this type
	var buf = bytes.NewBuffer(make([]byte, 0, 2))
	binary.Write(buf, binary.BigEndian, &Min)
	tests = append(tests, decodeTest{buf.Bytes(), reflect.ValueOf(Min)})

	// add test of max value for this type
	buf = bytes.NewBuffer(make([]byte, 0, 2))
	binary.Write(buf, binary.BigEndian, &Max)
	tests = append(tests, decodeTest{buf.Bytes(), reflect.ValueOf(Max)})

	for _, test := range tests {
		got := decSI16(test.Input).Interface()
		want := test.Want.Interface()
		if got != want {
			t.Errorf("decSI16(%q) = %T(%[2]v), want %T(%[3]v)", test.Input, got, want)
		}
	}
}

func TestDecSI32(t *testing.T) {
	Min := int32(math.MinInt32)
	Max := int32(math.MaxInt32)

	// sanity check of some handcoded values
	tests := []decodeTest{
		decodeTest{[]byte{0x00, 0x00, 0x00, 0x00}, reflect.ValueOf(int32(0))},
		decodeTest{[]byte{0x00, 0x00, 0x00, 0x01}, reflect.ValueOf(int32(1))},
		decodeTest{[]byte{0x00, 0xFF, 0x00, 0xFF}, reflect.ValueOf(int32(16711935))},
		decodeTest{[]byte{0xFF, 0x00, 0x00, 0x00}, reflect.ValueOf(int32(-16777216))},
		decodeTest{[]byte{0xFF, 0xFF, 0xFF, 0xFF}, reflect.ValueOf(int32(-1))},
	}

	// add test of min value for this type
	buf := bytes.NewBuffer(make([]byte, 0, 2))
	binary.Write(buf, binary.BigEndian, &Min)
	tests = append(tests, decodeTest{buf.Bytes(), reflect.ValueOf(Min)})

	// add test of max value for this type
	buf = bytes.NewBuffer(make([]byte, 0, 2))
	binary.Write(buf, binary.BigEndian, &Max)
	tests = append(tests, decodeTest{buf.Bytes(), reflect.ValueOf(Max)})

	for _, test := range tests {
		got := decSI32(test.Input).Interface()
		want := test.Want.Interface()
		if got != want {
			t.Errorf("decSI32(%q) = %T(%[2]v), want %T(%[3]v)", test.Input, got, want)
		}
	}
}

func TestDecSI64(t *testing.T) {
	Min := int64(math.MinInt64)
	Max := int64(math.MaxInt64)

	// sanity check of some handcoded values
	tests := []decodeTest{
		decodeTest{[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, reflect.ValueOf(int64(0))},
		decodeTest{[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}, reflect.ValueOf(int64(1))},
		decodeTest{[]byte{0x00, 0xFF, 0x00, 0xFF, 0x00, 0xFF, 0x00, 0xFF}, reflect.ValueOf(int64(0x00FF00FF00FF00FF))},
		decodeTest{[]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}, reflect.ValueOf(int64(-1))},
	}

	// add test of min value for this type
	buf := bytes.NewBuffer(make([]byte, 0, 2))
	binary.Write(buf, binary.BigEndian, &Min)
	tests = append(tests, decodeTest{buf.Bytes(), reflect.ValueOf(Min)})

	// add test of max value for this type
	buf = bytes.NewBuffer(make([]byte, 0, 2))
	binary.Write(buf, binary.BigEndian, &Max)
	tests = append(tests, decodeTest{buf.Bytes(), reflect.ValueOf(Max)})

	for _, test := range tests {
		got := decSI64(test.Input).Interface()
		want := test.Want.Interface()
		if got != want {
			t.Errorf("decSI64(%q) = %T(%[2]v), want %T(%[3]v)", test.Input, got, want)
		}
	}
}

/*
func TestDecFP32(t *testing.T) {
func TestDecFP64(t *testing.T) {
func TestDecUF32(t *testing.T) {
func TestDecUF64(t *testing.T) {
func TestDecUR32(t *testing.T) {
func TestDecUR64(t *testing.T) {
func TestDecSR32(t *testing.T) {
func TestDecSR64(t *testing.T) {
func TestDecFC32(t *testing.T) {
func TestDecIP32(t *testing.T) {
func TestDecIPAD(t *testing.T) {
func TestDecCSTR(t *testing.T) {
func TestDecUSTR(t *testing.T) {
func TestDecDATA(t *testing.T) {
func TestDecNULL(t *testing.T) {
func TestStrUI01(t *testing.T) {
func TestStrUI08(t *testing.T) {
func TestStrUI16(t *testing.T) {
func TestStrUI32(t *testing.T) {
func TestStrUI64(t *testing.T) {
func TestStrSI08(t *testing.T) {
func TestStrSI16(t *testing.T) {
func TestStrSI32(t *testing.T) {
func TestStrSI64(t *testing.T) {
func TestStrFP32(t *testing.T) {
func TestStrFP64(t *testing.T) {
func TestStrUF32(t *testing.T) {
func TestStrUF64(t *testing.T) {
func TestStrSF32(t *testing.T) {
func TestStrSF64(t *testing.T) {
func TestStrUR32(t *testing.T) {
func TestStrUR64(t *testing.T) {
func TestStrSR32(t *testing.T) {
func TestStrSR64(t *testing.T) {
func TestStrFC32(t *testing.T) {
func TestStrIP32(t *testing.T) {
func TestStrCSTR(t *testing.T) {
func TestStrUSTR(t *testing.T) {
func TestStrDATA(t *testing.T) {
func TestStrUUID(t *testing.T) {
func TestStrNULL(t *testing.T) {
func TestAsPrintableString(t *testing.T) {
func Test(t *testing.T) {
func Test(t *testing.T) {
*/
