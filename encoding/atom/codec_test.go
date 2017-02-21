package atom

import (
	"fmt"
	"math"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

// implement function currying for err funcs so that I can specify the type and
// expected bytes at the top of the test func, and the amount of bytes provided
// in each test separately.
func (f errFunc) curry(strAdeType string, want int) func(int) error {
	return func(got int) error {
		return f(strAdeType, want, got)
	}
}

// Given a function as an argument, return the function's name
func GetFunctionName(i interface{}) string {
	fullName := runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
	parts := strings.Split(fullName, "/")
	return parts[len(parts)-1]
}

// *****************************************************
// 1. Test decoding funcs, which write to an Atom's data
// *****************************************************

// *** decode test framework

type (
	// A decodeFunc converts a byte slice to a golang native type
	decodeFunc func([]byte) (interface{}, error)

	// decoderTest defines input and expected output values for a decodeFunc
	decoderTest struct {
		Input     []byte
		WantValue interface{} // interfaces are comparable as long as the underlying type is comparable
		WantError error
	}
)

// runDecoderTests evaluates a decodeFunc against test data
func runDecoderTests(t *testing.T, tests []decoderTest, f decodeFunc) {
	for _, test := range tests {
		gotValue, gotErr := f(test.Input)

		funcName := GetFunctionName(f)
		switch {
		case gotErr == nil && test.WantError == nil:
		case gotErr != nil && test.WantError == nil:
			t.Errorf("%v(%b): got err %s, want err <nil>", funcName, test.Input, gotErr)
		case gotErr == nil && test.WantError != nil:
			t.Errorf("%v(%b): got err <nil>, want err %s", funcName, test.Input, test.WantError)
		case gotErr.Error() != test.WantError.Error():
			t.Errorf("%v(%b): got err %s, want err %s", funcName, test.Input, gotErr, test.WantError)
			return
		}

		// value compare with DeepEqual instead of == so we can compare slice types like UR32
		if !reflect.DeepEqual(gotValue, test.WantValue) {
			t.Errorf("%v(%x): got %T(%[3]v), want %[4]T(%[4]v)", funcName, test.Input, gotValue, test.WantValue)
		}
	}
}

// *** unit tests

func TestUI08ToUint64(t *testing.T) {
	byteCountErr := errFunc(errByteCount).curry("UI08", 1)
	tests := []decoderTest{
		decoderTest{[]byte("\x00"), uint64(0), nil},
		decoderTest{[]byte("\x01"), uint64(1), nil},
		decoderTest{[]byte("\x00"), uint64(0), nil},
		decoderTest{[]byte("\x0F"), uint64(15), nil},
		decoderTest{[]byte("\xF0"), uint64(240), nil},
		decoderTest{[]byte("\xFF"), uint64(255), nil},
		decoderTest{[]byte("\x00\x00"), uint64(0), byteCountErr(2)},
		decoderTest{[]byte("\xFF\xFF"), uint64(0), byteCountErr(2)},
		decoderTest{[]byte(""), uint64(0), byteCountErr(0)},
	}
	runDecoderTests(t, tests, func(input []byte) (interface{}, error) {
		return UI08ToUint64(input)
	})
}

func TestUI16ToUint64(t *testing.T) {
	byteCountErr := errFunc(errByteCount).curry("UI16", 2)
	tests := []decoderTest{
		decoderTest{[]byte("\x00\x00"), uint64(0), nil},
		decoderTest{[]byte("\x00\xFF"), uint64(255), nil},
		decoderTest{[]byte("\xFF\x00"), uint64(65280), nil},
		decoderTest{[]byte("\xFF\xFF"), uint64(65535), nil},
		decoderTest{[]byte{}, uint64(0), byteCountErr(0)},
		decoderTest{[]byte("\x00"), uint64(0), byteCountErr(1)},
		decoderTest{[]byte("\xFF"), uint64(0), byteCountErr(1)},
		decoderTest{[]byte("\x00\x00\x01"), uint64(0), byteCountErr(3)},
	}
	runDecoderTests(t, tests, func(input []byte) (interface{}, error) {
		return UI16ToUint64(input)
	})
}

func TestUI32ToUint64(t *testing.T) {
	byteCountErr := errFunc(errByteCount).curry("UI32", 4)
	tests := []decoderTest{
		decoderTest{[]byte("\x00\x00\x00\x00"), uint64(0), nil},
		decoderTest{[]byte("\x00\x00\x00\xFF"), uint64(0xFF), nil},
		decoderTest{[]byte("\x00\x00\xFF\x00"), uint64(0xFF00), nil},
		decoderTest{[]byte("\x00\xFF\x00\x00"), uint64(0xFF0000), nil},
		decoderTest{[]byte("\xFF\x00\x00\x00"), uint64(0xFF000000), nil},
		decoderTest{[]byte("\xFF\xFF\xFF\xFF"), uint64(0xFFFFFFFF), nil},

		decoderTest{[]byte{}, uint64(0), byteCountErr(0)},
		decoderTest{[]byte("\x01"), uint64(0), byteCountErr(1)},
		decoderTest{[]byte("\xFF\x01"), uint64(0), byteCountErr(2)},
		decoderTest{[]byte("\xFF\xFF\x01"), uint64(0), byteCountErr(3)},
		decoderTest{[]byte("\xFF\xFF\xFF\x01"), uint64(0xFFFFFF01), nil},
		decoderTest{[]byte("\xFF\xFF\xFF\xFF\x01"), uint64(0), byteCountErr(5)},
	}
	runDecoderTests(t, tests, func(input []byte) (interface{}, error) {
		return UI32ToUint64(input)
	})
}

func TestUI64ToUint64(t *testing.T) {
	byteCountErr := errFunc(errByteCount).curry("UI64", 8)
	tests := []decoderTest{
		decoderTest{[]byte("\x00\x00\x00\x00\x00\x00\x00\x00"), uint64(0), nil},
		decoderTest{[]byte("\x00\x00\x00\x00\x00\x00\x00\xFF"), uint64(0xFF), nil},
		decoderTest{[]byte("\x00\x00\x00\x00\x00\x00\xFF\x00"), uint64(0xFF00), nil},
		decoderTest{[]byte("\x00\x00\x00\x00\x00\xFF\x00\x00"), uint64(0xFF0000), nil},
		decoderTest{[]byte("\x00\x00\x00\x00\xFF\x00\x00\x00"), uint64(0xFF000000), nil},
		decoderTest{[]byte("\x00\x00\x00\xFF\x00\x00\x00\x00"), uint64(0xFF00000000), nil},
		decoderTest{[]byte("\x00\x00\xFF\x00\x00\x00\x00\x00"), uint64(0xFF0000000000), nil},
		decoderTest{[]byte("\x00\xFF\x00\x00\x00\x00\x00\x00"), uint64(0xFF000000000000), nil},
		decoderTest{[]byte("\xFF\x00\x00\x00\x00\x00\x00\x00"), uint64(0xFF00000000000000), nil},
		decoderTest{[]byte("\xFF\xFF\xFF\xFF\xFF\xFF\xFF\xFF"), uint64(0xFFFFFFFFFFFFFFFF), nil},

		decoderTest{[]byte{}, uint64(0), byteCountErr(0)},
		decoderTest{[]byte("\x01"), uint64(0), byteCountErr(1)},
		decoderTest{[]byte("\xFF\x01"), uint64(0), byteCountErr(2)},
		decoderTest{[]byte("\xFF\xFF\x01"), uint64(0), byteCountErr(3)},
		decoderTest{[]byte("\xFF\xFF\xFF\x01"), uint64(0), byteCountErr(4)},
		decoderTest{[]byte("\xFF\xFF\xFF\xFF\x01"), uint64(0), byteCountErr(5)},
		decoderTest{[]byte("\xFF\xFF\xFF\xFF\xFF\x01"), uint64(0), byteCountErr(6)},
		decoderTest{[]byte("\xFF\xFF\xFF\xFF\xFF\xFF\x01"), uint64(0), byteCountErr(7)},
		decoderTest{[]byte("\xFF\xFF\xFF\xFF\xFF\xFF\xFF\x01"), uint64(0xFFFFFFFFFFFFFF01), nil},
		decoderTest{[]byte("\xFF\xFF\xFF\xFF\xFF\xFF\xFF\xFF\x01"), uint64(0), byteCountErr(9)},
	}
	runDecoderTests(t, tests, func(input []byte) (interface{}, error) {
		return UI64ToUint64(input)
	})
}

func TestUI01ToBool(t *testing.T) {
	byteCountErr := errFunc(errByteCount).curry("UI01", 4)
	zero := false
	tests := []decoderTest{
		decoderTest{[]byte("\x00\x00\x00\x00"), false, nil},
		decoderTest{[]byte("\x00\x00\x00\x01"), true, nil},
		decoderTest{[]byte("\x00\x00\x00\x02"), zero, errRange("bool", 2)},
		decoderTest{[]byte("\x00\x00\x00\xFF"), zero, errRange("bool", 255)},
		decoderTest{[]byte("\x00\x00\xFF\x00"), zero, errRange("bool", 65280)},
		decoderTest{[]byte("\x00\xFF\x00\x00"), zero, errRange("bool", 16711680)},
		decoderTest{[]byte("\xFF\x00\x00\x00"), zero, errRange("bool", 4278190080)},
		decoderTest{[]byte(""), zero, byteCountErr(0)},
		decoderTest{[]byte("\x01"), zero, byteCountErr(1)},
		decoderTest{[]byte("\x00\x01"), zero, byteCountErr(2)},
		decoderTest{[]byte("\x00\x00\x01"), zero, byteCountErr(3)},
		decoderTest{[]byte("\x00\x00\x00\x00\x01"), zero, byteCountErr(5)},
		decoderTest{[]byte("\x00\x00\x00\x00\x00\x01"), zero, byteCountErr(6)},
	}
	runDecoderTests(t, tests, func(input []byte) (interface{}, error) {
		return UI01ToBool(input)
	})
}

func funcUI32ToUint32(t *testing.T) {
	byteCountErr := errFunc(errByteCount).curry("UI32", 4)
	zero := uint32(0)
	tests := []decoderTest{
		decoderTest{[]byte{}, zero, byteCountErr(0)},
		decoderTest{[]byte("\x00"), zero, byteCountErr(1)},
		decoderTest{[]byte("\x00\xFF"), zero, byteCountErr(2)},
		decoderTest{[]byte("\xFF\x00\xFF"), zero, byteCountErr(3)},
		decoderTest{[]byte("\x00\x00\x00\x00"), zero, nil},
		decoderTest{[]byte("\xFF\xFF\xFF\xFF"), math.MaxUint32, nil},
		decoderTest{[]byte("\x01\xFF\xFF\xFF\xFF"), zero, nil},
	}
	runDecoderTests(t, tests, func(input []byte) (interface{}, error) {
		return UI32ToUint32(input)
	})
}

func TestUI08ToString(t *testing.T) {
	byteCountErr := errFunc(errByteCount).curry("UI08", 1)
	zero := ""
	tests := []decoderTest{
		decoderTest{[]byte("\x00"), "0", nil},
		decoderTest{[]byte("\x01"), "1", nil},
		decoderTest{[]byte("\x00"), "0", nil},
		decoderTest{[]byte("\x0F"), "15", nil},
		decoderTest{[]byte("\xF0"), "240", nil},
		decoderTest{[]byte("\xFF"), "255", nil},
		decoderTest{[]byte("\x00\x00"), zero, byteCountErr(2)},
		decoderTest{[]byte("\xFF\xFF"), zero, byteCountErr(2)},
		decoderTest{[]byte(""), zero, byteCountErr(0)},
	}
	runDecoderTests(t, tests, func(input []byte) (interface{}, error) {
		return UI08ToString(input)
	})
}

func TestUI16ToString(t *testing.T) {
	byteCountErr := errFunc(errByteCount).curry("UI16", 2)
	zero := ""
	tests := []decoderTest{
		decoderTest{[]byte("\x00\x00"), "0", nil},
		decoderTest{[]byte("\x00\x01"), "1", nil},
		decoderTest{[]byte("\x00\xFF"), "255", nil},
		decoderTest{[]byte("\xFF\x00"), "65280", nil},
		decoderTest{[]byte("\xFF\xFF"), "65535", nil},
		decoderTest{[]byte(""), zero, byteCountErr(0)},
		decoderTest{[]byte("\x00"), zero, byteCountErr(1)},
		decoderTest{[]byte("\x00\x00\x00"), zero, byteCountErr(3)},
		decoderTest{[]byte("\x00\x00\x00\x00"), zero, byteCountErr(4)},
	}
	runDecoderTests(t, tests, func(input []byte) (interface{}, error) {
		return UI16ToString(input)
	})
}

func TestUI32ToString(t *testing.T) {
	byteCountErr := errFunc(errByteCount).curry("UI32", 4)
	zero := ""
	tests := []decoderTest{
		decoderTest{[]byte("\x00\x00\x00\x00"), "0", nil},
		decoderTest{[]byte("\x00\x00\x00\x01"), "1", nil},
		decoderTest{[]byte("\x00\x00\x00\xFF"), "255", nil},
		decoderTest{[]byte("\x00\x00\xFF\x00"), "65280", nil},
		decoderTest{[]byte("\x00\xFF\x00\x00"), "16711680", nil},
		decoderTest{[]byte("\xFF\x00\x00\x00"), "4278190080", nil},
		decoderTest{[]byte("\xFF\xFF\xFF\xFF"), "4294967295", nil},
		decoderTest{[]byte(""), zero, byteCountErr(0)},
		decoderTest{[]byte("\x00"), zero, byteCountErr(1)},
		decoderTest{[]byte("\x00\x00\x00"), zero, byteCountErr(3)},
		decoderTest{[]byte("\x00\x00\x00\x00\x00"), zero, byteCountErr(5)},
	}
	runDecoderTests(t, tests, func(input []byte) (interface{}, error) {
		return UI32ToString(input)
	})
}

func TestUI64ToString(t *testing.T) {
	byteCountErr := errFunc(errByteCount).curry("UI64", 8)
	zero := ""
	tests := []decoderTest{
		decoderTest{[]byte("\x00\x00\x00\x00\x00\x00\x00\x00"), "0", nil},
		decoderTest{[]byte("\x00\x00\x00\x00\x00\x00\x00\x01"), "1", nil},
		decoderTest{[]byte("\x00\x00\x00\x00\x00\x00\x00\xFF"), "255", nil},
		decoderTest{[]byte("\x00\x00\x00\x00\x00\x00\xFF\x00"), "65280", nil},
		decoderTest{[]byte("\x00\x00\x00\x00\x00\xFF\x00\x00"), "16711680", nil},
		decoderTest{[]byte("\x00\x00\x00\x00\xFF\x00\x00\x00"), "4278190080", nil},
		decoderTest{[]byte("\x00\x00\x00\xFF\x00\x00\x00\x00"), "1095216660480", nil},
		decoderTest{[]byte("\x00\x00\xFF\x00\x00\x00\x00\x00"), "280375465082880", nil},
		decoderTest{[]byte("\x00\xFF\x00\x00\x00\x00\x00\x00"), "71776119061217280", nil},
		decoderTest{[]byte("\xFF\x00\x00\x00\x00\x00\x00\x00"), "18374686479671623680", nil},
		decoderTest{[]byte("\xFF\xFF\xFF\xFF\xFF\xFF\xFF\xFF"), "18446744073709551615", nil},
		decoderTest{[]byte(""), zero, byteCountErr(0)},
		decoderTest{[]byte("\x00"), zero, byteCountErr(1)},
		decoderTest{[]byte("\x00\x00\x00\x00\x00"), zero, byteCountErr(5)},
		decoderTest{[]byte("\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"), zero, byteCountErr(10)},
	}
	runDecoderTests(t, tests, func(input []byte) (interface{}, error) {
		return UI64ToString(input)
	})
}

func TestSI08ToInt64(t *testing.T) {
	byteCountErr := errFunc(errByteCount).curry("SI08", 1)
	zero := int64(0)
	tests := []decoderTest{
		decoderTest{[]byte("\x00"), int64(0), nil},
		decoderTest{[]byte("\x01"), int64(1), nil},
		decoderTest{[]byte("\x0F"), int64(15), nil},
		decoderTest{[]byte("\x1F"), int64(31), nil},
		decoderTest{[]byte("\xFF"), int64(-1), nil},
		decoderTest{[]byte(""), zero, byteCountErr(0)},
		decoderTest{[]byte("\x00\x00"), zero, byteCountErr(2)},
		decoderTest{[]byte("\x00\x00\x00\x00"), zero, byteCountErr(4)},
	}
	runDecoderTests(t, tests, func(input []byte) (interface{}, error) {
		return SI08ToInt64(input)
	})
}

func TestSI16ToInt64(t *testing.T) {
	byteCountErr := errFunc(errByteCount).curry("SI16", 2)
	zero := int64(0)
	tests := []decoderTest{
		decoderTest{[]byte("\x00\x00"), int64(0), nil},
		decoderTest{[]byte("\x00\x01"), int64(1), nil},
		decoderTest{[]byte("\x80\x00"), int64(math.MinInt16), nil},
		decoderTest{[]byte("\x7F\xFF"), int64(math.MaxInt16), nil},
		decoderTest{[]byte("\xFF\xFF"), int64(-1), nil},
		decoderTest{[]byte(""), zero, byteCountErr(0)},
		decoderTest{[]byte("\x00"), zero, byteCountErr(1)},
		decoderTest{[]byte("\x00\x00\x00\x00"), zero, byteCountErr(4)},
	}
	runDecoderTests(t, tests, func(input []byte) (interface{}, error) {
		return SI16ToInt64(input)
	})
}

func TestSI32ToInt32(t *testing.T) {
	byteCountErr := errFunc(errByteCount).curry("SI32", 4)
	zero := int32(0)
	tests := []decoderTest{
		decoderTest{[]byte("\x00\x00\x00\x00"), int32(0), nil},
		decoderTest{[]byte("\x00\x00\x00\x01"), int32(1), nil},
		decoderTest{[]byte("\x00\x00\x00\xFF"), int32(255), nil},
		decoderTest{[]byte("\x00\x00\xFF\x01"), int32(65281), nil},
		decoderTest{[]byte("\x00\xFF\x00\x01"), int32(16711681), nil},
		decoderTest{[]byte("\xFF\x00\x00\x01"), int32(-16777215), nil},
		decoderTest{[]byte("\xFF\xFF\xFF\xFF"), int32(-1), nil},
		decoderTest{[]byte("\x80\x00\x00\x00"), int32(math.MinInt32), nil},
		decoderTest{[]byte("\x7F\xFF\xFF\xFF"), int32(math.MaxInt32), nil},
		decoderTest{[]byte(""), zero, byteCountErr(0)},
		decoderTest{[]byte("\x00"), zero, byteCountErr(1)},
		decoderTest{[]byte("\x00\x00"), zero, byteCountErr(2)},
		decoderTest{[]byte("\x00\x00\x00\x00\x00\x00\x00\x00"), zero, byteCountErr(8)},
	}
	runDecoderTests(t, tests, func(input []byte) (interface{}, error) {
		return SI32ToInt32(input)
	})
}

func TestSI32ToInt64(t *testing.T) {
	byteCountErr := errFunc(errByteCount).curry("SI32", 4)
	zero := int64(0)
	tests := []decoderTest{
		decoderTest{[]byte("\x00\x00\x00\x00"), int64(0), nil},
		decoderTest{[]byte("\x00\x00\x00\x01"), int64(1), nil},
		decoderTest{[]byte("\x00\x00\x00\xFF"), int64(255), nil},
		decoderTest{[]byte("\x00\x00\xFF\x01"), int64(65281), nil},
		decoderTest{[]byte("\x00\xFF\x00\x01"), int64(16711681), nil},
		decoderTest{[]byte("\xFF\x00\x00\x01"), int64(-16777215), nil},
		decoderTest{[]byte("\xFF\xFF\xFF\xFF"), int64(-1), nil},
		decoderTest{[]byte("\x80\x00\x00\x00"), int64(math.MinInt32), nil},
		decoderTest{[]byte("\x7F\xFF\xFF\xFF"), int64(math.MaxInt32), nil},
		decoderTest{[]byte(""), zero, byteCountErr(0)},
		decoderTest{[]byte("\x00"), zero, byteCountErr(1)},
		decoderTest{[]byte("\x00\x00"), zero, byteCountErr(2)},
		decoderTest{[]byte("\x00\x00\x00\x00\x00\x00\x00\x00"), zero, byteCountErr(8)},
	}
	runDecoderTests(t, tests, func(input []byte) (interface{}, error) {
		return SI32ToInt64(input)
	})
}

func TestSI64ToInt64(t *testing.T) {
	byteCountErr := errFunc(errByteCount).curry("SI64", 8)
	zero := int64(0)
	tests := []decoderTest{
		decoderTest{[]byte("\x00\x00\x00\x00\x00\x00\x00\x00"), int64(0), nil},
		decoderTest{[]byte("\x00\x00\x00\x00\x00\x00\x00\x01"), int64(1), nil},
		decoderTest{[]byte("\x00\x00\x00\x00\x00\x00\x00\xFF"), int64(255), nil},
		decoderTest{[]byte("\x00\x00\x00\x00\x00\x00\xFF\x00"), int64(65280), nil},
		decoderTest{[]byte("\x00\x00\x00\x00\x00\xFF\x00\x00"), int64(16711680), nil},
		decoderTest{[]byte("\x00\x00\x00\x00\xFF\x00\x00\x00"), int64(4278190080), nil},
		decoderTest{[]byte("\x00\x00\x00\xFF\x00\x00\x00\x00"), int64(1095216660480), nil},
		decoderTest{[]byte("\x00\x00\xFF\x00\x00\x00\x00\x00"), int64(280375465082880), nil},
		decoderTest{[]byte("\x00\xFF\x00\x00\x00\x00\x00\x00"), int64(71776119061217280), nil},
		decoderTest{[]byte("\xFF\x00\x00\x00\x00\x00\x00\x00"), int64(-72057594037927936), nil},
		decoderTest{[]byte("\xFF\xFF\xFF\xFF\xFF\xFF\xFF\xFF"), int64(-1), nil},
		decoderTest{[]byte("\x80\x00\x00\x00\x00\x00\x00\x00"), int64(math.MinInt64), nil},
		decoderTest{[]byte("\x7F\xFF\xFF\xFF\xFF\xFF\xFF\xFF"), int64(math.MaxInt64), nil},
		decoderTest{[]byte(""), zero, byteCountErr(0)},
		decoderTest{[]byte("\x00"), zero, byteCountErr(1)},
		decoderTest{[]byte("\x00\x00"), zero, byteCountErr(2)},
		decoderTest{[]byte("\x00\x00\x00\x00"), zero, byteCountErr(4)},
	}
	runDecoderTests(t, tests, func(input []byte) (interface{}, error) {
		return SI64ToInt64(input)
	})
}

func TestSI08ToString(t *testing.T) {
	byteCountErr := errFunc(errByteCount).curry("SI08", 1)
	zero := ""
	tests := []decoderTest{
		decoderTest{[]byte("\x00"), "0", nil},
		decoderTest{[]byte("\x01"), "1", nil},
		decoderTest{[]byte("\x0F"), "15", nil},
		decoderTest{[]byte("\x1F"), "31", nil},
		decoderTest{[]byte("\xFF"), "-1", nil},
		decoderTest{[]byte(""), zero, byteCountErr(0)},
		decoderTest{[]byte("\x00\x00"), zero, byteCountErr(2)},
		decoderTest{[]byte("\x00\x00\x00\x00"), zero, byteCountErr(4)},
	}
	runDecoderTests(t, tests, func(input []byte) (interface{}, error) {
		return SI08ToString(input)
	})
}

func TestSI16ToString(t *testing.T) {
	byteCountErr := errFunc(errByteCount).curry("SI16", 2)
	zero := ""
	tests := []decoderTest{
		decoderTest{[]byte("\x00\x00"), "0", nil},
		decoderTest{[]byte("\x00\x01"), "1", nil},
		decoderTest{[]byte("\x80\x00"), "-32768", nil},
		decoderTest{[]byte("\x7F\xFF"), "32767", nil},
		decoderTest{[]byte("\xFF\xFF"), "-1", nil},
		decoderTest{[]byte(""), zero, byteCountErr(0)},
		decoderTest{[]byte("\x00"), zero, byteCountErr(1)},
		decoderTest{[]byte("\x00\x00\x00\x00"), zero, byteCountErr(4)},
	}
	runDecoderTests(t, tests, func(input []byte) (interface{}, error) {
		return SI16ToString(input)
	})
}

func TestSI32ToString(t *testing.T) {
	byteCountErr := errFunc(errByteCount).curry("SI32", 4)
	zero := ""
	tests := []decoderTest{
		decoderTest{[]byte("\x00\x00\x00\x00"), "0", nil},
		decoderTest{[]byte("\x00\x00\x00\x01"), "1", nil},
		decoderTest{[]byte("\x00\x00\x00\xFF"), "255", nil},
		decoderTest{[]byte("\x00\x00\xFF\x01"), "65281", nil},
		decoderTest{[]byte("\x00\xFF\x00\x01"), "16711681", nil},
		decoderTest{[]byte("\xFF\x00\x00\x01"), "-16777215", nil},
		decoderTest{[]byte("\xFF\xFF\xFF\xFF"), "-1", nil},
		decoderTest{[]byte("\x80\x00\x00\x00"), "-2147483648", nil},
		decoderTest{[]byte("\x7F\xFF\xFF\xFF"), "2147483647", nil},
		decoderTest{[]byte(""), zero, byteCountErr(0)},
		decoderTest{[]byte("\x00"), zero, byteCountErr(1)},
		decoderTest{[]byte("\x00\x00"), zero, byteCountErr(2)},
		decoderTest{[]byte("\x00\x00\x00\x00\x00\x00\x00\x00"), zero, byteCountErr(8)},
	}
	runDecoderTests(t, tests, func(input []byte) (interface{}, error) {
		return SI32ToString(input)
	})
}

func TestSI64ToString(t *testing.T) {
	byteCountErr := errFunc(errByteCount).curry("SI64", 8)
	zero := ""
	tests := []decoderTest{
		decoderTest{[]byte("\x00\x00\x00\x00\x00\x00\x00\x00"), "0", nil},
		decoderTest{[]byte("\x00\x00\x00\x00\x00\x00\x00\x01"), "1", nil},
		decoderTest{[]byte("\x00\x00\x00\x00\x00\x00\x00\xFF"), "255", nil},
		decoderTest{[]byte("\x00\x00\x00\x00\x00\x00\xFF\x00"), "65280", nil},
		decoderTest{[]byte("\x00\x00\x00\x00\x00\xFF\x00\x00"), "16711680", nil},
		decoderTest{[]byte("\x00\x00\x00\x00\xFF\x00\x00\x00"), "4278190080", nil},
		decoderTest{[]byte("\x00\x00\x00\xFF\x00\x00\x00\x00"), "1095216660480", nil},
		decoderTest{[]byte("\x00\x00\xFF\x00\x00\x00\x00\x00"), "280375465082880", nil},
		decoderTest{[]byte("\x00\xFF\x00\x00\x00\x00\x00\x00"), "71776119061217280", nil},
		decoderTest{[]byte("\xFF\x00\x00\x00\x00\x00\x00\x00"), "-72057594037927936", nil},
		decoderTest{[]byte("\xFF\xFF\xFF\xFF\xFF\xFF\xFF\xFF"), "-1", nil},
		decoderTest{[]byte("\x80\x00\x00\x00\x00\x00\x00\x00"), "-9223372036854775808", nil},
		decoderTest{[]byte("\x7F\xFF\xFF\xFF\xFF\xFF\xFF\xFF"), "9223372036854775807", nil},
		decoderTest{[]byte(""), zero, byteCountErr(0)},
		decoderTest{[]byte("\x00"), zero, byteCountErr(1)},
		decoderTest{[]byte("\x00\x00"), zero, byteCountErr(2)},
		decoderTest{[]byte("\x00\x00\x00\x00"), zero, byteCountErr(4)},
	}
	runDecoderTests(t, tests, func(input []byte) (interface{}, error) {
		return SI64ToString(input)
	})
}

// FP32 has a range magnitude minimum of 1.1754E-38 and a range magnitude
// maximum of 3.4028E+38 (either can be positive or negative).
func TestFP32ToFloat32(t *testing.T) {
	byteCountErr := errFunc(errByteCount).curry("FP32", 4)
	zero := float32(0)
	tests := []decoderTest{
		decoderTest{[]byte("\x00\x00\x00\x00"), float32(0.0), nil},
		decoderTest{[]byte("\x00\x7F\xFD\x5F"), float32(1.1754E-38), nil},
		decoderTest{[]byte("\x2d\x59\x2f\xfe"), float32(1.2345678E-11), nil},
		decoderTest{[]byte("\x42\x03\x11\x68"), float32(32.766998), nil},
		decoderTest{[]byte("\x42\x82\x00\x83"), float32(65.000999), nil},
		decoderTest{[]byte("\x43\xa3\xd5\xc3"), float32(327.67001), nil},
		decoderTest{[]byte("\x47\x00\x00\x00"), float32(32768), nil},
		decoderTest{[]byte("\x4c\x23\xd7\x0a"), float32(42949672), nil},
		decoderTest{[]byte("\x4d\x9c\x40\x00"), float32(3.2768E+08), nil},
		decoderTest{[]byte("\x7f\x7f\xff\x8b"), float32(3.4027999E+38), nil},
		decoderTest{[]byte("\x7F\x7F\xFF\x8B"), float32(3.4028E+38), nil},
		decoderTest{[]byte("\x80\x7f\xfd\x5f"), float32(-1.1754E-38), nil},
		decoderTest{[]byte("\xc0\x51\xb5\x74"), float32(-3.2767), nil},
		decoderTest{[]byte("\xc4\x9a\x52\x2b"), float32(-1234.5677), nil},
		decoderTest{[]byte("\xc5\xcb\x20\x00"), float32(-6500), nil},
		decoderTest{[]byte(""), zero, byteCountErr(0)},
		decoderTest{[]byte("\x00"), zero, byteCountErr(1)},
		decoderTest{[]byte("\x00\x00"), zero, byteCountErr(2)},
		decoderTest{[]byte("\x00\x00\x00\x00\x00\x00\x00\x00"), zero, byteCountErr(8)},
	}
	runDecoderTests(t, tests, func(input []byte) (interface{}, error) {
		return FP32ToFloat32(input)
	})
}

func TestFP32ToFloat64(t *testing.T) {
	byteCountErr := errFunc(errByteCount).curry("FP32", 4)
	zero := float64(0)
	tests := []decoderTest{
		// must cast expected result to float32 first, otherwise the float64 has
		// too much precision to match the real result
		decoderTest{[]byte("\x00\x00\x00\x00"), float64(float32(0)), nil},
		decoderTest{[]byte("\x00\x7F\xFD\x5F"), float64(float32(1.1754E-38)), nil},
		decoderTest{[]byte("\x2d\x59\x2f\xfe"), float64(float32(1.2345678E-11)), nil},
		decoderTest{[]byte("\x42\x03\x11\x68"), float64(float32(32.766998)), nil},
		decoderTest{[]byte("\x42\x82\x00\x83"), float64(float32(65.000999)), nil},
		decoderTest{[]byte("\x43\xa3\xd5\xc3"), float64(float32(327.67001)), nil},
		decoderTest{[]byte("\x47\x00\x00\x00"), float64(float32(32768)), nil},
		decoderTest{[]byte("\x4c\x23\xd7\x0a"), float64(float32(42949672)), nil},
		decoderTest{[]byte("\x4d\x9c\x40\x00"), float64(float32(3.2768E+08)), nil},
		decoderTest{[]byte("\x7f\x7f\xff\x8b"), float64(float32(3.4027999E+38)), nil},
		decoderTest{[]byte("\x7F\x7F\xFF\x8B"), float64(float32(3.4028E+38)), nil},
		decoderTest{[]byte("\x80\x7f\xfd\x5f"), float64(float32(-1.1754E-38)), nil},
		decoderTest{[]byte("\xc0\x51\xb5\x74"), float64(float32(-3.2767)), nil},
		decoderTest{[]byte("\xc4\x9a\x52\x2b"), float64(float32(-1234.5677)), nil},
		decoderTest{[]byte("\xc5\xcb\x20\x00"), float64(float32(-6500)), nil},
		decoderTest{[]byte(""), zero, byteCountErr(0)},
		decoderTest{[]byte("\x00"), zero, byteCountErr(1)},
		decoderTest{[]byte("\x00\x00"), zero, byteCountErr(2)},
		decoderTest{[]byte("\x00\x00\x00\x00\x00\x00\x00\x00"), zero, byteCountErr(8)},
	}
	runDecoderTests(t, tests, func(input []byte) (interface{}, error) {
		return FP32ToFloat64(input)
	})
}

func TestFP64ToFloat64(t *testing.T) {
	byteCountErr := errFunc(errByteCount).curry("FP64", 8)
	zero := float64(0)
	tests := []decoderTest{
		decoderTest{[]byte("\xc1\xd2\x65\x80\xb4\x87\xe6\xb7"), float64(-1.23456789012345672E+09), nil},
		decoderTest{[]byte("\x40\x40\x62\x2d\x0e\x56\x04\x19"), float64(3.27670000000000030E+01), nil},
		decoderTest{[]byte("\x40\x74\x7a\xb8\x51\xeb\x85\x1f"), float64(3.27670000000000016E+02), nil},
		decoderTest{[]byte("\x40\x50\x40\x10\x62\x4d\xd2\xf2"), float64(6.50010000000000048E+01), nil},
		decoderTest{[]byte("\xc0\x74\x6c\xcc\xcc\xcc\xcc\xcd"), float64(-3.26800000000000011E+02), nil},
		decoderTest{[]byte("\xc0\x0a\x36\xae\x7d\x56\x6c\xf4"), float64(-3.27669999999999995E+00), nil},
		decoderTest{[]byte("\xc0\xb9\x64\x00\x00\x00\x00\x00"), float64(-6.50000000000000000E+03), nil},
		decoderTest{[]byte("\x00\x0f\xff\xdd\x31\xa0\x0c\x6d"), float64(2.22499999999999987E-308), nil},
		decoderTest{[]byte("\x00\x0f\xff\xdd\x31\xa0\x0c\x6d"), float64(2.22499999999999987E-308), nil},
		decoderTest{[]byte("\x7f\xef\xff\x93\x59\xcc\x81\x04"), float64(1.79760000000000007E+308), nil},
		decoderTest{[]byte("\x00\x00\x00\x00\x00\x00\x00\x00"), float64(0.00000000000000000E+00), nil},
		decoderTest{[]byte("\x40\xe0\x00\x00\x00\x00\x00\x00"), float64(3.27680000000000000E+04), nil},
		decoderTest{[]byte("\x41\xb3\x88\x00\x01\x00\x00\x00"), float64(3.27680001000000000E+08), nil},
		decoderTest{[]byte("\x41\x84\x7a\xe1\x40\x00\x00\x00"), float64(4.29496720000000000E+07), nil},
		decoderTest{[]byte(""), zero, byteCountErr(0)},
		decoderTest{[]byte("\x00"), zero, byteCountErr(1)},
		decoderTest{[]byte("\x00\x00"), zero, byteCountErr(2)},
		decoderTest{[]byte("\x00\x00\x00\x00"), zero, byteCountErr(4)},
	}
	runDecoderTests(t, tests, func(input []byte) (interface{}, error) {
		return FP64ToFloat64(input)
	})
}

func TestFP32ToString(t *testing.T) {
	byteCountErr := errFunc(errByteCount).curry("FP32", 4)
	zero := ""
	tests := []decoderTest{
		decoderTest{[]byte("\x00\x00\x00\x00"), "0", nil},
		decoderTest{[]byte("\x00\x7F\xFD\x5F"), "1.1754E-38", nil},
		decoderTest{[]byte("\x2d\x59\x2f\xfe"), "1.2345678E-11", nil},
		decoderTest{[]byte("\x42\x03\x11\x68"), "32.766998", nil},
		decoderTest{[]byte("\x42\x82\x00\x83"), "65.000999", nil},
		decoderTest{[]byte("\x43\xa3\xd5\xc3"), "327.67001", nil},
		decoderTest{[]byte("\x47\x00\x00\x00"), "32768", nil},
		decoderTest{[]byte("\x4c\x23\xd7\x0a"), "42949672", nil},
		decoderTest{[]byte("\x4d\x9c\x40\x00"), "3.2768E+08", nil},
		decoderTest{[]byte("\x7f\x7f\xff\x8b"), "3.4027999E+38", nil},
		//FIXME		decoderTest{[]byte("\x7F\x7F\xFF\x8B"), "3.4028E+38", nil},
		decoderTest{[]byte("\x80\x7f\xfd\x5f"), "-1.1754E-38", nil},
		decoderTest{[]byte("\xc0\x51\xb5\x74"), "-3.2767", nil},
		decoderTest{[]byte("\xc4\x9a\x52\x2b"), "-1234.5677", nil},
		decoderTest{[]byte("\xc5\xcb\x20\x00"), "-6500", nil},
		decoderTest{[]byte(""), zero, byteCountErr(0)},
		decoderTest{[]byte("\x00"), zero, byteCountErr(1)},
		decoderTest{[]byte("\x00\x00"), zero, byteCountErr(2)},
		decoderTest{[]byte("\x00\x00\x00\x00\x00\x00\x00\x00"), zero, byteCountErr(8)},
	}
	runDecoderTests(t, tests, func(input []byte) (interface{}, error) {
		return FP32ToString(input)
	})
}

func TestFP64ToString(t *testing.T) {
	byteCountErr := errFunc(errByteCount).curry("FP64", 8)
	zero := ""
	tests := []decoderTest{
		decoderTest{[]byte("\xc1\xd2\x65\x80\xb4\x87\xe6\xb7"), "-1.23456789012345672E+09", nil},
		decoderTest{[]byte("\x40\x40\x62\x2d\x0e\x56\x04\x19"), "3.27670000000000030E+01", nil},
		decoderTest{[]byte("\x40\x74\x7a\xb8\x51\xeb\x85\x1f"), "3.27670000000000016E+02", nil},
		decoderTest{[]byte("\x40\x50\x40\x10\x62\x4d\xd2\xf2"), "6.50010000000000048E+01", nil},
		decoderTest{[]byte("\xc0\x74\x6c\xcc\xcc\xcc\xcc\xcd"), "-3.26800000000000011E+02", nil},
		decoderTest{[]byte("\xc0\x0a\x36\xae\x7d\x56\x6c\xf4"), "-3.27669999999999995E+00", nil},
		decoderTest{[]byte("\xc0\xb9\x64\x00\x00\x00\x00\x00"), "-6.50000000000000000E+03", nil},
		decoderTest{[]byte("\x00\x0f\xff\xdd\x31\xa0\x0c\x6d"), "2.22499999999999987E-308", nil},
		decoderTest{[]byte("\x00\x0f\xff\xdd\x31\xa0\x0c\x6d"), "2.22499999999999987E-308", nil},
		decoderTest{[]byte("\x7f\xef\xff\x93\x59\xcc\x81\x04"), "1.79760000000000007E+308", nil},
		decoderTest{[]byte("\x00\x00\x00\x00\x00\x00\x00\x00"), "0.00000000000000000E+00", nil},
		decoderTest{[]byte("\x40\xe0\x00\x00\x00\x00\x00\x00"), "3.27680000000000000E+04", nil},
		decoderTest{[]byte("\x41\xb3\x88\x00\x01\x00\x00\x00"), "3.27680001000000000E+08", nil},
		decoderTest{[]byte("\x41\x84\x7a\xe1\x40\x00\x00\x00"), "4.29496720000000000E+07", nil},
		decoderTest{[]byte(""), zero, byteCountErr(0)},
		decoderTest{[]byte("\x00"), zero, byteCountErr(1)},
		decoderTest{[]byte("\x00\x00"), zero, byteCountErr(2)},
		decoderTest{[]byte("\x00\x00\x00\x00"), zero, byteCountErr(4)},
	}
	runDecoderTests(t, tests, func(input []byte) (interface{}, error) {
		return FP64ToString(input)
	})
}

// // FIXME
// func TestUF32ToFloat64(t *testing.T) {
// 	byteCountErr := errFunc(errByteCount).curry("UF32", 4)
// 	tests := []decoderTest{
// 		decoderTest{[]byte("\x00\x00\x00\x00"), float64(float32(0.0000), nil},
// 		decoderTest{[]byte("\xff\xff\xff\xf9"), float64(float32(65535.9999), nil},
// 		decoderTest{[]byte("\xff\xff\xff\xf9"), float64(float32(65535.9999), nil},
// 		decoderTest{[]byte(""), "", byteCountErr(0)},
// 		decoderTest{[]byte("\x00"), "", byteCountErr(1)},
// 		decoderTest{[]byte("\x00\x00"), "", byteCountErr(2)},
// 		decoderTest{[]byte("\x00\x00\x00\x00\x00\x00\x00\x00"), "", byteCountErr(8)},
// 	}
// 	runDecoderTests(t, tests, func(input []byte) (interface{}, error) {
// 		return UF32ToFloat64(input)
// 	})
// }

func TestUF64ToFloat64(t *testing.T) {
	byteCountErr := errFunc(errByteCount).curry("UF64", 8)
	zero := float64(0)
	tests := []decoderTest{
		decoderTest{[]byte("\xff\xff\xff\xff\xff\xff\xff\xfb"), float64(4294967295.999999999), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x00\x00\x00\x00"), float64(4294967295.000000000), nil},
		decoderTest{[]byte("\xff\xff\xff\xfe\x00\x00\x00\x00"), float64(4294967294.000000000), nil},
		decoderTest{[]byte("\xff\xff\xff\xfd\x00\x00\x00\x00"), float64(4294967293.000000000), nil},
		decoderTest{[]byte("\xff\xff\xff\xfc\x00\x00\x00\x00"), float64(4294967292.000000000), nil},
		decoderTest{[]byte("\xff\xff\xff\xfb\x00\x00\x00\x00"), float64(4294967291.000000000), nil},
		decoderTest{[]byte("\xff\xff\xff\xfa\x00\x00\x00\x00"), float64(4294967290.000000000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x19\x99\x99\x99"), float64(4294967295.100000000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x33\x33\x33\x33"), float64(4294967295.200000000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x4c\xcc\xcc\xcc"), float64(4294967295.300000000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x66\x66\x66\x66"), float64(4294967295.400000000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x80\x00\x00\x00"), float64(4294967295.500000000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x99\x99\x99\x99"), float64(4294967295.600000000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\xb3\x33\x33\x33"), float64(4294967295.700000000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\xcc\xcc\xcc\xcc"), float64(4294967295.800000000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\xe6\x66\x66\x66"), float64(4294967295.900000000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x02\x8f\x5c\x28"), float64(4294967295.010000000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x05\x1e\xb8\x51"), float64(4294967295.020000000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x07\xae\x14\x7a"), float64(4294967295.030000000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x0a\x3d\x70\xa3"), float64(4294967295.040000000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x0c\xcc\xcc\xcc"), float64(4294967295.050000000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x0f\x5c\x28\xf5"), float64(4294967295.060000000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x11\xeb\x85\x1e"), float64(4294967295.070000000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x14\x7a\xe1\x47"), float64(4294967295.080000000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x17\x0a\x3d\x70"), float64(4294967295.090000000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x00\x41\x89\x37"), float64(4294967295.001000000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x00\x83\x12\x6e"), float64(4294967295.002000000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x00\xc4\x9b\xa5"), float64(4294967295.003000000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x01\x06\x24\xdd"), float64(4294967295.004000000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x01\x47\xae\x14"), float64(4294967295.005000000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x01\x89\x37\x4b"), float64(4294967295.006000000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x01\xca\xc0\x83"), float64(4294967295.007000000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x02\x0c\x49\xba"), float64(4294967295.008000000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x02\x4d\xd2\xf1"), float64(4294967295.009000000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x01\x89\x37\x4b"), float64(4294967295.006000000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x00\x06\x8d\xb8"), float64(4294967295.000100000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x00\x0d\x1b\x71"), float64(4294967295.000200000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x00\x13\xa9\x2a"), float64(4294967295.000300000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x00\x1a\x36\xe2"), float64(4294967295.000400000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x00\x20\xc4\x9b"), float64(4294967295.000500000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x00\x27\x52\x54"), float64(4294967295.000600000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x00\x2d\xe0\x0d"), float64(4294967295.000700000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x00\x34\x6d\xc5"), float64(4294967295.000800000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x00\x3a\xfb\x7e"), float64(4294967295.000900000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x00\x27\x52\x54"), float64(4294967295.000600000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x00\x00\xa7\xc5"), float64(4294967295.000010000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x00\x01\x4f\x8b"), float64(4294967295.000020000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x00\x01\xf7\x51"), float64(4294967295.000030000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x00\x02\x9f\x16"), float64(4294967295.000040000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x00\x03\x46\xdc"), float64(4294967295.000050000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x00\x03\xee\xa2"), float64(4294967295.000060000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x00\x04\x96\x67"), float64(4294967295.000070000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x00\x05\x3e\x2d"), float64(4294967295.000080000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x00\x05\xe5\xf3"), float64(4294967295.000090000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x00\x03\xee\xa2"), float64(4294967295.000060000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x00\x00\x10\xc6"), float64(4294967295.000001000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x00\x00\x21\x8d"), float64(4294967295.000002000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x00\x00\x32\x54"), float64(4294967295.000003000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x00\x00\x43\x1b"), float64(4294967295.000004000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x00\x00\x53\xe2"), float64(4294967295.000005000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x00\x00\x64\xa9"), float64(4294967295.000006000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x00\x00\x75\x70"), float64(4294967295.000007000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x00\x00\x86\x37"), float64(4294967295.000008000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x00\x00\x96\xfe"), float64(4294967295.000009000), nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x00\x00\x64\xa9"), float64(4294967295.000006000), nil},
		decoderTest{[]byte("\x00\x00\x00\x01\x00\x00\x00\x00"), float64(1.000000000), nil},
		// FIXME this stupid type should be simply two married UINT32s.
		//    decoderTest{[]byte("\x00\x00\x00\x01\x19\x99\x99\x99"), float64(1.100000000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\x33\x33\x33\x33"), float64(1.200000000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\x4c\xcc\xcc\xcc"), float64(1.300000000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\x66\x66\x66\x66"), float64(1.400000000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\x80\x00\x00\x00"), float64(1.500000000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\x99\x99\x99\x99"), float64(1.600000000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\xb3\x33\x33\x33"), float64(1.700000000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\xcc\xcc\xcc\xcc"), float64(1.800000000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\xe6\x66\x66\x66"), float64(1.900000000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\x02\x8f\x5c\x28"), float64(1.010000000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\x05\x1e\xb8\x51"), float64(1.020000000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\x07\xae\x14\x7a"), float64(1.030000000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\x0a\x3d\x70\xa3"), float64(1.040000000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\x0c\xcc\xcc\xcc"), float64(1.050000000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\x0f\x5c\x28\xf5"), float64(1.060000000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\x11\xeb\x85\x1e"), float64(1.070000000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\x14\x7a\xe1\x47"), float64(1.080000000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\x17\x0a\x3d\x70"), float64(1.090000000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\x00\x41\x89\x37"), float64(1.001000000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\x00\x83\x12\x6e"), float64(1.002000000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\x00\xc4\x9b\xa5"), float64(1.003000000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\x01\x06\x24\xdd"), float64(1.004000000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\x01\x47\xae\x14"), float64(1.005000000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\x01\x89\x37\x4b"), float64(1.006000000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\x01\xca\xc0\x83"), float64(1.007000000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\x02\x0c\x49\xba"), float64(1.008000000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\x02\x4d\xd2\xf1"), float64(1.009000000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\x01\x89\x37\x4b"), float64(1.006000000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\x00\x06\x8d\xb8"), float64(1.000100000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\x00\x0d\x1b\x71"), float64(1.000200000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\x00\x13\xa9\x2a"), float64(1.000300000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\x00\x1a\x36\xe2"), float64(1.000400000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\x00\x20\xc4\x9b"), float64(1.000500000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\x00\x27\x52\x54"), float64(1.000600000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\x00\x2d\xe0\x0d"), float64(1.000700000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\x00\x34\x6d\xc5"), float64(1.000800000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\x00\x3a\xfb\x7e"), float64(1.000900000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\x00\x27\x52\x54"), float64(1.000600000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\x00\x00\xa7\xc5"), float64(1.000010000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\x00\x01\x4f\x8b"), float64(1.000020000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\x00\x01\xf7\x51"), float64(1.000030000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\x00\x02\x9f\x16"), float64(1.000040000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\x00\x03\x46\xdc"), float64(1.000050000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\x00\x03\xee\xa2"), float64(1.000060000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\x00\x04\x96\x67"), float64(1.000070000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\x00\x05\x3e\x2d"), float64(1.000080000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\x00\x05\xe5\xf3"), float64(1.000090000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\x00\x03\xee\xa2"), float64(1.000060000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\x00\x00\x10\xc6"), float64(1.000001000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\x00\x00\x21\x8d"), float64(1.000002000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\x00\x00\x32\x54"), float64(1.000003000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\x00\x00\x43\x1b"), float64(1.000004000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\x00\x00\x53\xe2"), float64(1.000005000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\x00\x00\x64\xa9"), float64(1.000006000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\x00\x00\x75\x70"), float64(1.000007000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\x00\x00\x86\x37"), float64(1.000008000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\x00\x00\x96\xfe"), float64(1.000009000), nil},
		//		decoderTest{[]byte("\x00\x00\x00\x01\x00\x00\x64\xa9"), float64(1.000006000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x00\x00\x00\x00"), float64(65596.000000000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x00\x00\x00\x00"), float64(65596.000000000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x00\x00\x00\x00"), float64(65596.000000000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x00\x00\x00\x00"), float64(65596.000000000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x00\x00\x00\x00"), float64(65596.000000000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x00\x00\x00\x00"), float64(65596.000000000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x00\x00\x00\x00"), float64(65596.000000000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x00\x00\x00\x00"), float64(65596.000000000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x19\x99\x99\x99"), float64(65596.100000000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x33\x33\x33\x33"), float64(65596.200000000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x4c\xcc\xcc\xcc"), float64(65596.300000000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x66\x66\x66\x66"), float64(65596.400000000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x80\x00\x00\x00"), float64(65596.500000000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x99\x99\x99\x99"), float64(65596.600000000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\xb3\x33\x33\x33"), float64(65596.700000000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\xcc\xcc\xcc\xcc"), float64(65596.800000000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\xe6\x66\x66\x66"), float64(65596.900000000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x02\x8f\x5c\x28"), float64(65596.010000000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x05\x1e\xb8\x51"), float64(65596.020000000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x07\xae\x14\x7a"), float64(65596.030000000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x0a\x3d\x70\xa3"), float64(65596.040000000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x0c\xcc\xcc\xcc"), float64(65596.050000000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x0f\x5c\x28\xf5"), float64(65596.060000000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x11\xeb\x85\x1e"), float64(65596.070000000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x14\x7a\xe1\x47"), float64(65596.080000000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x17\x0a\x3d\x70"), float64(65596.090000000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x00\x41\x89\x37"), float64(65596.001000000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x00\x83\x12\x6e"), float64(65596.002000000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x00\xc4\x9b\xa5"), float64(65596.003000000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x01\x06\x24\xdd"), float64(65596.004000000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x01\x47\xae\x14"), float64(65596.005000000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x01\x89\x37\x4b"), float64(65596.006000000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x01\xca\xc0\x83"), float64(65596.007000000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x02\x0c\x49\xba"), float64(65596.008000000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x02\x4d\xd2\xf1"), float64(65596.009000000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x01\x89\x37\x4b"), float64(65596.006000000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x00\x06\x8d\xb8"), float64(65596.000100000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x00\x0d\x1b\x71"), float64(65596.000200000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x00\x13\xa9\x2a"), float64(65596.000300000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x00\x1a\x36\xe2"), float64(65596.000400000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x00\x20\xc4\x9b"), float64(65596.000500000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x00\x27\x52\x54"), float64(65596.000600000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x00\x2d\xe0\x0d"), float64(65596.000700000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x00\x34\x6d\xc5"), float64(65596.000800000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x00\x3a\xfb\x7e"), float64(65596.000900000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x00\x27\x52\x54"), float64(65596.000600000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x00\x00\xa7\xc5"), float64(65596.000010000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x00\x01\x4f\x8b"), float64(65596.000020000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x00\x01\xf7\x51"), float64(65596.000030000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x00\x02\x9f\x16"), float64(65596.000040000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x00\x03\x46\xdc"), float64(65596.000050000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x00\x03\xee\xa2"), float64(65596.000060000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x00\x04\x96\x67"), float64(65596.000070000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x00\x05\x3e\x2d"), float64(65596.000080000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x00\x05\xe5\xf3"), float64(65596.000090000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x00\x03\xee\xa2"), float64(65596.000060000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x00\x00\x10\xc6"), float64(65596.000001000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x00\x00\x21\x8d"), float64(65596.000002000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x00\x00\x32\x54"), float64(65596.000003000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x00\x00\x43\x1b"), float64(65596.000004000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x00\x00\x53\xe2"), float64(65596.000005000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x00\x00\x64\xa9"), float64(65596.000006000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x00\x00\x75\x70"), float64(65596.000007000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x00\x00\x86\x37"), float64(65596.000008000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x00\x00\x96\xfe"), float64(65596.000009000), nil},
		//		decoderTest{[]byte("\x00\x01\x00\x3c\x00\x00\x64\xa9"), float64(65596.000006000), nil},
		decoderTest{[]byte(""), zero, byteCountErr(0)},
		decoderTest{[]byte("\x00"), zero, byteCountErr(1)},
		decoderTest{[]byte("\x00\x00"), zero, byteCountErr(2)},
		decoderTest{[]byte("\x00\x00\x00\x00"), zero, byteCountErr(4)},
	}
	runDecoderTests(t, tests, func(input []byte) (interface{}, error) {
		return UF64ToFloat64(input)
	})
}

// FIXME
//func UF32ToString(buf []byte) (v string, e error) {
//func UF64ToString(buf []byte) (v string, e error) {
//func SF32ToFloat64(buf []byte) (v float64, e error) {
//func SF64ToFloat64(buf []byte) (v float64, e error) {
//func SF32ToString(buf []byte) (v string, e error) {
//func SF64ToString(buf []byte) (v string, e error) {

func TestUR32ToSliceOfUint(t *testing.T) {
	byteCountErr := errFunc(errByteCount).curry("UR32", 4)
	zero := []uint64(nil)
	tests := []decoderTest{
		decoderTest{[]byte("\x00\x01\x00\x01"), []uint64{1, 1}, nil},
		decoderTest{[]byte("\x00\x01\x00\x02"), []uint64{1, 2}, nil},
		decoderTest{[]byte("\x01\x00\x01\x00"), []uint64{256, 256}, nil},
		decoderTest{[]byte("\x00\x00\x00\x00"), []uint64{0, 0}, nil},
		decoderTest{[]byte("\x19\x99\x99\x99"), []uint64{6553, 39321}, nil},
		decoderTest{[]byte("\x02\x8f\x5c\x28"), []uint64{655, 23592}, nil},
		decoderTest{[]byte("\xff\xff\x00\x05"), []uint64{65535, 5}, nil},
		decoderTest{[]byte("\xff\xff\x00\x02"), []uint64{65535, 2}, nil},
		decoderTest{[]byte("\xff\xff\xff\xff"), []uint64{65535, 65535}, nil},
		decoderTest{[]byte(""), zero, byteCountErr(0)},
		decoderTest{[]byte("\x00"), zero, byteCountErr(1)},
		decoderTest{[]byte("\x00\x00"), zero, byteCountErr(2)},
		decoderTest{[]byte("\x00\x00\x00\x00\x00\x00\x00\x00"), zero, byteCountErr(8)},
	}
	runDecoderTests(t, tests, func(input []byte) (interface{}, error) {
		return UR32ToSliceOfUint(input)
	})
}

func TestUR64ToSliceOfUint(t *testing.T) {
	byteCountErr := errFunc(errByteCount).curry("UR64", 8)
	zero := []uint64(nil)
	tests := []decoderTest{
		decoderTest{[]byte("\x00\x00\x00\x01\x00\x00\x00\x01"), []uint64{1, 1}, nil},
		decoderTest{[]byte("\x00\x00\x00\x01\x00\x00\x00\x02"), []uint64{1, 2}, nil},
		decoderTest{[]byte("\x01\x02\x03\x04\x05\x06\x07\x08"), []uint64{16909060, 84281096}, nil},
		decoderTest{[]byte("\x10\x20\x30\x40\x50\x60\x70\x80"), []uint64{270544960, 1348497536}, nil},
		decoderTest{[]byte("\x19\x99\x99\x99\x19\x99\x99\x99"), []uint64{429496729, 429496729}, nil},
		decoderTest{[]byte("\xff\xff\x00\x02\xff\xff\xcc\xee"), []uint64{4294901762, 4294954222}, nil},
		decoderTest{[]byte("\xff\xff\xff\xff\xff\xff\xff\xff"), []uint64{4294967295, 4294967295}, nil},
		decoderTest{[]byte(""), zero, byteCountErr(0)},
		decoderTest{[]byte("\x00"), zero, byteCountErr(1)},
		decoderTest{[]byte("\x00\x00"), zero, byteCountErr(2)},
		decoderTest{[]byte("\x00\x00\x00\x00"), zero, byteCountErr(4)},
		decoderTest{[]byte("\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"), zero, byteCountErr(12)},
	}
	runDecoderTests(t, tests, func(input []byte) (interface{}, error) {
		return UR64ToSliceOfUint(input)
	})
}

func TestUR32ToString(t *testing.T) {
	byteCountErr := errFunc(errByteCount).curry("UR32", 4)
	zero := ""
	tests := []decoderTest{
		decoderTest{[]byte("\x00\x01\x00\x01"), "1/1", nil},
		decoderTest{[]byte("\x00\x01\x00\x02"), "1/2", nil},
		decoderTest{[]byte("\x01\x00\x01\x00"), "256/256", nil},
		decoderTest{[]byte("\x00\x00\x00\x00"), "0/0", nil},
		decoderTest{[]byte("\x19\x99\x99\x99"), "6553/39321", nil},
		decoderTest{[]byte("\x02\x8f\x5c\x28"), "655/23592", nil},
		decoderTest{[]byte("\xff\xff\x00\x05"), "65535/5", nil},
		decoderTest{[]byte("\xff\xff\x00\x02"), "65535/2", nil},
		decoderTest{[]byte("\xff\xff\xff\xff"), "65535/65535", nil},
		decoderTest{[]byte(""), zero, byteCountErr(0)},
		decoderTest{[]byte("\x00"), zero, byteCountErr(1)},
		decoderTest{[]byte("\x00\x00"), zero, byteCountErr(2)},
		decoderTest{[]byte("\x00\x00\x00\x00\x00\x00\x00\x00"), zero, byteCountErr(8)},
	}
	runDecoderTests(t, tests, func(input []byte) (interface{}, error) {
		return UR32ToString(input)
	})
}

func TestUR64ToString(t *testing.T) {
	byteCountErr := errFunc(errByteCount).curry("UR64", 8)
	zero := ""
	tests := []decoderTest{
		decoderTest{[]byte("\x00\x00\x00\x01\x00\x00\x00\x01"), "1/1", nil},
		decoderTest{[]byte("\x00\x00\x00\x01\x00\x00\x00\x02"), "1/2", nil},
		decoderTest{[]byte("\x01\x02\x03\x04\x05\x06\x07\x08"), "16909060/84281096", nil},
		decoderTest{[]byte("\x10\x20\x30\x40\x50\x60\x70\x80"), "270544960/1348497536", nil},
		decoderTest{[]byte("\x19\x99\x99\x99\x19\x99\x99\x99"), "429496729/429496729", nil},
		decoderTest{[]byte("\xff\xff\x00\x02\xff\xff\xcc\xee"), "4294901762/4294954222", nil},
		decoderTest{[]byte("\xff\xff\xff\xff\xff\xff\xff\xff"), "4294967295/4294967295", nil},
		decoderTest{[]byte(""), zero, byteCountErr(0)},
		decoderTest{[]byte("\x00"), zero, byteCountErr(1)},
		decoderTest{[]byte("\x00\x00"), zero, byteCountErr(2)},
		decoderTest{[]byte("\x00\x00\x00\x00"), zero, byteCountErr(4)},
		decoderTest{[]byte("\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"), zero, byteCountErr(12)},
	}
	runDecoderTests(t, tests, func(input []byte) (interface{}, error) {
		return UR64ToString(input)
	})
}
func TestSR32ToSliceOfInt(t *testing.T) {
	byteCountErr := errFunc(errByteCount).curry("SR32", 4)
	zero := []int64(nil)
	tests := []decoderTest{
		decoderTest{[]byte("\x00\x01\x00\x01"), []int64{1, 1}, nil},
		decoderTest{[]byte("\x00\x01\xff\xff"), []int64{1, -1}, nil},
		decoderTest{[]byte("\xff\xff\x00\x01"), []int64{-1, 1}, nil},
		decoderTest{[]byte("\x00\x01\x00\x01"), []int64{1, 1}, nil},
		decoderTest{[]byte("\x00\x01\x00\x02"), []int64{1, 2}, nil},
		decoderTest{[]byte("\x00\x01\xff\xfe"), []int64{1, -2}, nil},
		decoderTest{[]byte("\xff\xff\x00\x02"), []int64{-1, 2}, nil},
		decoderTest{[]byte("\x00\x01\x00\x02"), []int64{1, 2}, nil},
		decoderTest{[]byte("\x00\x01\x00\x01"), []int64{1, 1}, nil},
		decoderTest{[]byte("\x80\x00\x7f\xff"), []int64{-32768, 32767}, nil},
		decoderTest{[]byte("\x7f\xff\x80\x00"), []int64{32767, -32768}, nil},
		decoderTest{[]byte("\x00\x01\x00\x01"), []int64{1, 1}, nil},
		decoderTest{[]byte("\x00\x01\x7f\xff"), []int64{1, 32767}, nil},
		decoderTest{[]byte("\xff\xff\x7f\xff"), []int64{-1, 32767}, nil},
		decoderTest{[]byte("\x00\x01\x80\x00"), []int64{1, -32768}, nil},
		decoderTest{[]byte(""), zero, byteCountErr(0)},
		decoderTest{[]byte("\x00"), zero, byteCountErr(1)},
		decoderTest{[]byte("\x00\x00"), zero, byteCountErr(2)},
		decoderTest{[]byte("\x00\x00\x00\x00\x00\x00\x00\x00"), zero, byteCountErr(8)},
	}
	runDecoderTests(t, tests, func(input []byte) (interface{}, error) {
		return SR32ToSliceOfInt(input)
	})
}
func TestSR64ToSliceOfInt(t *testing.T) {
	byteCountErr := errFunc(errByteCount).curry("SR64", 8)
	zero := []int64(nil)
	tests := []decoderTest{
		decoderTest{[]byte("\x00\x00\x00\x01\x00\x00\x00\x01"), []int64{1, 1}, nil},
		decoderTest{[]byte("\x00\x00\x00\x01\xff\xff\xff\xff"), []int64{1, -1}, nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x00\x00\x00\x01"), []int64{-1, 1}, nil},
		decoderTest{[]byte("\x00\x00\x00\x01\x00\x00\x00\x01"), []int64{1, 1}, nil},
		decoderTest{[]byte("\x00\x00\x00\x01\x00\x00\x00\x02"), []int64{1, 2}, nil},
		decoderTest{[]byte("\x00\x00\x00\x01\xff\xff\xff\xfe"), []int64{1, -2}, nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x00\x00\x00\x02"), []int64{-1, 2}, nil},
		decoderTest{[]byte("\x00\x00\x00\x01\x00\x00\x00\x02"), []int64{1, 2}, nil},
		decoderTest{[]byte("\x00\x00\x00\x01\x00\x00\x00\x01"), []int64{1, 1}, nil},
		decoderTest{[]byte("\x80\x00\x00\x00\x7f\xff\xff\xff"), []int64{-2147483648, 2147483647}, nil},
		decoderTest{[]byte("\x7f\xff\xff\xff\x80\x00\x00\x00"), []int64{2147483647, -2147483648}, nil},
		decoderTest{[]byte("\x00\x00\x00\x01\x00\x00\x00\x01"), []int64{1, 1}, nil},
		decoderTest{[]byte("\x00\x00\x00\x01\x7f\xff\xff\xff"), []int64{1, 2147483647}, nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x7f\xff\xff\xff"), []int64{-1, 2147483647}, nil},
		decoderTest{[]byte("\x00\x00\x00\x01\x80\x00\x00\x00"), []int64{1, -2147483648}, nil},
		decoderTest{[]byte(""), zero, byteCountErr(0)},
		decoderTest{[]byte("\x00"), zero, byteCountErr(1)},
		decoderTest{[]byte("\x00\x00"), zero, byteCountErr(2)},
		decoderTest{[]byte("\x00\x00\x00\x00"), zero, byteCountErr(4)},
		decoderTest{[]byte("\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"), zero, byteCountErr(12)},
	}
	runDecoderTests(t, tests, func(input []byte) (interface{}, error) {
		return SR64ToSliceOfInt(input)
	})
}

func TestSR32ToString(t *testing.T) {
	byteCountErr := errFunc(errByteCount).curry("SR32", 4)
	zero := ""
	tests := []decoderTest{
		decoderTest{[]byte("\x00\x01\x00\x01"), "1/1", nil},
		decoderTest{[]byte("\x00\x01\xff\xff"), "1/-1", nil},
		decoderTest{[]byte("\xff\xff\x00\x01"), "-1/1", nil},
		decoderTest{[]byte("\x00\x01\x00\x01"), "1/1", nil},
		decoderTest{[]byte("\x00\x01\x00\x02"), "1/2", nil},
		decoderTest{[]byte("\x00\x01\xff\xfe"), "1/-2", nil},
		decoderTest{[]byte("\xff\xff\x00\x02"), "-1/2", nil},
		decoderTest{[]byte("\x00\x01\x00\x02"), "1/2", nil},
		decoderTest{[]byte("\x00\x01\x00\x01"), "1/1", nil},
		decoderTest{[]byte("\x80\x00\x7f\xff"), "-32768/32767", nil},
		decoderTest{[]byte("\x7f\xff\x80\x00"), "32767/-32768", nil},
		decoderTest{[]byte("\x00\x01\x00\x01"), "1/1", nil},
		decoderTest{[]byte("\x00\x01\x7f\xff"), "1/32767", nil},
		decoderTest{[]byte("\xff\xff\x7f\xff"), "-1/32767", nil},
		decoderTest{[]byte("\x00\x01\x80\x00"), "1/-32768", nil},
		decoderTest{[]byte(""), zero, byteCountErr(0)},
		decoderTest{[]byte("\x00"), zero, byteCountErr(1)},
		decoderTest{[]byte("\x00\x00"), zero, byteCountErr(2)},
		decoderTest{[]byte("\x00\x00\x00\x00\x00\x00\x00\x00"), zero, byteCountErr(8)},
	}
	runDecoderTests(t, tests, func(input []byte) (interface{}, error) {
		return SR32ToString(input)
	})
}
func TestSR64ToString(t *testing.T) {
	byteCountErr := errFunc(errByteCount).curry("SR64", 8)
	zero := ""
	tests := []decoderTest{
		decoderTest{[]byte("\x00\x00\x00\x01\x00\x00\x00\x01"), "1/1", nil},
		decoderTest{[]byte("\x00\x00\x00\x01\xff\xff\xff\xff"), "1/-1", nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x00\x00\x00\x01"), "-1/1", nil},
		decoderTest{[]byte("\x00\x00\x00\x01\x00\x00\x00\x01"), "1/1", nil},
		decoderTest{[]byte("\x00\x00\x00\x01\x00\x00\x00\x02"), "1/2", nil},
		decoderTest{[]byte("\x00\x00\x00\x01\xff\xff\xff\xfe"), "1/-2", nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x00\x00\x00\x02"), "-1/2", nil},
		decoderTest{[]byte("\x00\x00\x00\x01\x00\x00\x00\x02"), "1/2", nil},
		decoderTest{[]byte("\x00\x00\x00\x01\x00\x00\x00\x01"), "1/1", nil},
		decoderTest{[]byte("\x80\x00\x00\x00\x7f\xff\xff\xff"), "-2147483648/2147483647", nil},
		decoderTest{[]byte("\x7f\xff\xff\xff\x80\x00\x00\x00"), "2147483647/-2147483648", nil},
		decoderTest{[]byte("\x00\x00\x00\x01\x00\x00\x00\x01"), "1/1", nil},
		decoderTest{[]byte("\x00\x00\x00\x01\x7f\xff\xff\xff"), "1/2147483647", nil},
		decoderTest{[]byte("\xff\xff\xff\xff\x7f\xff\xff\xff"), "-1/2147483647", nil},
		decoderTest{[]byte("\x00\x00\x00\x01\x80\x00\x00\x00"), "1/-2147483648", nil},
		decoderTest{[]byte(""), zero, byteCountErr(0)},
		decoderTest{[]byte("\x00"), zero, byteCountErr(1)},
		decoderTest{[]byte("\x00\x00"), zero, byteCountErr(2)},
		decoderTest{[]byte("\x00\x00\x00\x00"), zero, byteCountErr(4)},
		decoderTest{[]byte("\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"), zero, byteCountErr(12)},
	}
	runDecoderTests(t, tests, func(input []byte) (interface{}, error) {
		return SR64ToString(input)
	})
}

func funcFC32ToUint32(t *testing.T) {
	byteCountErr := errFunc(errByteCount).curry("FC32", 4)
	zero := 0
	tests := []decoderTest{
		decoderTest{[]byte("\x20\x7e\x7d\x7c"), uint32(0x207e7d7c), nil},
		decoderTest{[]byte("\x21\x20\x7e\x7d"), uint32(0x21207e7d), nil},
		decoderTest{[]byte("\x5c\x21\x20\x7e"), uint32(0x5c21207e), nil},
		decoderTest{[]byte("\x23\x5c\x21\x20"), uint32(0x235c2120), nil},
		decoderTest{[]byte("\x24\x23\x5c\x21"), uint32(0x24235c21), nil},
		decoderTest{[]byte("\x25\x24\x23\x5c"), uint32(0x2524235c), nil},
		decoderTest{[]byte("\x26\x25\x24\x23"), uint32(0x26252423), nil},
		decoderTest{[]byte("\x27\x26\x25\x24"), uint32(0x27262524), nil},
		decoderTest{[]byte("\x28\x27\x26\x25"), uint32(0x28272625), nil},
		decoderTest{[]byte("\x29\x28\x27\x26"), uint32(0x29282726), nil},
		decoderTest{[]byte("\x2a\x29\x28\x27"), uint32(0x2a292827), nil},
		decoderTest{[]byte("\x2b\x2a\x29\x28"), uint32(0x2b2a2928), nil},
		decoderTest{[]byte("\x2c\x2b\x2a\x29"), uint32(0x2c2b2a29), nil},
		decoderTest{[]byte("\x2d\x2c\x2b\x2a"), uint32(0x2d2c2b2a), nil},
		decoderTest{[]byte("\x2e\x2d\x2c\x2b"), uint32(0x2e2d2c2b), nil},
		decoderTest{[]byte("\x2f\x2e\x2d\x2c"), uint32(0x2f2e2d2c), nil},
		decoderTest{[]byte("\x30\x2f\x2e\x2d"), uint32(0x302f2e2d), nil},
		decoderTest{[]byte("\x31\x30\x2f\x2e"), uint32(0x31302f2e), nil},
		decoderTest{[]byte("\x32\x31\x30\x2f"), uint32(0x3231302f), nil},
		decoderTest{[]byte("\x5b\x5a\x59\x58"), uint32(0x5b5a5958), nil},
		decoderTest{[]byte("\x5c\x5b\x5a\x59"), uint32(0x5c5b5a59), nil},
		decoderTest{[]byte("\x5d\x5c\x5b\x5a"), uint32(0x5d5c5b5a), nil},
		decoderTest{[]byte("\x5e\x5d\x5c\x5b"), uint32(0x5e5d5c5b), nil},
		decoderTest{[]byte("\x5f\x5e\x5d\x5c"), uint32(0x5f5e5d5c), nil},
		decoderTest{[]byte("\x60\x5f\x5e\x5d"), uint32(0x605f5e5d), nil},
		decoderTest{[]byte("\x61\x60\x5f\x5e"), uint32(0x61605f5e), nil},
		decoderTest{[]byte("\x62\x61\x60\x5f"), uint32(0x6261605f), nil},
		decoderTest{[]byte("\x63\x62\x61\x60"), uint32(0x63626160), nil},
		decoderTest{[]byte("\x7b\x7a\x79\x78"), uint32(0x7b7a7978), nil},
		decoderTest{[]byte("\x7c\x7b\x7a\x79"), uint32(0x7c7b7a79), nil},
		decoderTest{[]byte("\x7d\x7c\x7b\x7a"), uint32(0x7d7c7b7a), nil},
		decoderTest{[]byte("\x7e\x7d\x7c\x7b"), uint32(0x7e7d7c7b), nil},
		decoderTest{[]byte("\x20\x20\x20\x20"), uint32(0x20202020), nil},
		decoderTest{[]byte("\x00\x00\x00\x00"), uint32(0x00000000), nil},
		decoderTest{[]byte("\x00\x00\x00\x01"), uint32(0x00000001), nil},
		decoderTest{[]byte("\x00\x00\x00\x02"), uint32(0x00000002), nil},
		decoderTest{[]byte("\x00\x00\x00\x03"), uint32(0x00000003), nil},
		decoderTest{[]byte("\x00\x00\x00\x04"), uint32(0x00000004), nil},
		decoderTest{[]byte("\x00\x00\x00\x05"), uint32(0x00000005), nil},
		decoderTest{[]byte("\x00\x00\x00\x06"), uint32(0x00000006), nil},
		decoderTest{[]byte("\x00\x00\x00\x07"), uint32(0x00000007), nil},
		decoderTest{[]byte("\x00\x00\x00\x08"), uint32(0x00000008), nil},
		decoderTest{[]byte("\x00\x00\x00\x09"), uint32(0x00000009), nil},
		decoderTest{[]byte("\x00\x00\x00\x0a"), uint32(0x0000000A), nil},
		decoderTest{[]byte("\x00\x00\x00\x0b"), uint32(0x0000000B), nil},
		decoderTest{[]byte("\x00\x00\x00\x0c"), uint32(0x0000000C), nil},
		decoderTest{[]byte("\x00\x00\x00\x0d"), uint32(0x0000000D), nil},
		decoderTest{[]byte("\x00\x00\x00\x0e"), uint32(0x0000000E), nil},
		decoderTest{[]byte("\x00\x00\x00\x0f"), uint32(0x0000000F), nil},
		decoderTest{[]byte("\x01\x00\x00\x00"), uint32(0x01000000), nil},
		decoderTest{[]byte("\x02\x00\x00\x00"), uint32(0x02000000), nil},
		decoderTest{[]byte("\x03\x00\x00\x00"), uint32(0x03000000), nil},
		decoderTest{[]byte("\x04\x00\x00\x00"), uint32(0x04000000), nil},
		decoderTest{[]byte("\x05\x00\x00\x00"), uint32(0x05000000), nil},
		decoderTest{[]byte("\x06\x00\x00\x00"), uint32(0x06000000), nil},
		decoderTest{[]byte("\x07\x00\x00\x00"), uint32(0x07000000), nil},
		decoderTest{[]byte("\x08\x00\x00\x00"), uint32(0x08000000), nil},
		decoderTest{[]byte("\x09\x00\x00\x00"), uint32(0x09000000), nil},
		decoderTest{[]byte("\x0a\x00\x00\x00"), uint32(0x0A000000), nil},
		decoderTest{[]byte("\x0b\x00\x00\x00"), uint32(0x0B000000), nil},
		decoderTest{[]byte("\x0c\x00\x00\x00"), uint32(0x0C000000), nil},
		decoderTest{[]byte("\x0d\x00\x00\x00"), uint32(0x0D000000), nil},
		decoderTest{[]byte("\x0e\x00\x00\x00"), uint32(0x0E000000), nil},
		decoderTest{[]byte("\x0f\x00\x00\x00"), uint32(0x0F000000), nil},
		decoderTest{[]byte(""), zero, byteCountErr(0)},
		decoderTest{[]byte("\x00"), zero, byteCountErr(1)},
		decoderTest{[]byte("\x00\x00"), zero, byteCountErr(2)},
		decoderTest{[]byte("\x00\x00\x00\x00\x00\x00\x00\x00"), zero, byteCountErr(8)},
	}
	runDecoderTests(t, tests, func(input []byte) (interface{}, error) {
		return UI32ToUint32(input)
	})
}

func TestFC32ToStringDelimited(t *testing.T) {
	byteCountErr := errFunc(errByteCount).curry("FC32", 4)
	zero := ""
	tests := []decoderTest{
		decoderTest{[]byte("\x20\x7e\x7d\x7c"), "0x207E7D7C", nil},
		decoderTest{[]byte("\x21\x20\x7e\x7d"), "0x21207E7D", nil},
		decoderTest{[]byte("\x5c\x21\x20\x7e"), "0x5C21207E", nil},
		decoderTest{[]byte("\x23\x5c\x21\x20"), "0x235C2120", nil},
		decoderTest{[]byte("\x24\x23\x5c\x21"), `'$#\!'`, nil},
		decoderTest{[]byte("\x25\x24\x23\x5c"), `'%$#\'`, nil},
		decoderTest{[]byte("\x26\x25\x24\x23"), `'&%$#'`, nil},
		decoderTest{[]byte("\x27\x26\x25\x24"), "0x27262524", nil}, // starts with '
		decoderTest{[]byte("\x28\x27\x26\x25"), `'('&%'`, nil},
		decoderTest{[]byte("\x29\x28\x27\x26"), `')('&'`, nil},
		decoderTest{[]byte("\x2a\x29\x28\x27"), `'*)(''`, nil},
		decoderTest{[]byte("\x2b\x2a\x29\x28"), `'+*)('`, nil},
		decoderTest{[]byte("\x2c\x2b\x2a\x29"), `',+*)'`, nil},
		decoderTest{[]byte("\x2d\x2c\x2b\x2a"), `'-,+*'`, nil},
		decoderTest{[]byte("\x2e\x2d\x2c\x2b"), `'.-,+'`, nil},
		decoderTest{[]byte("\x2f\x2e\x2d\x2c"), `'/.-,'`, nil},
		decoderTest{[]byte("\x30\x2f\x2e\x2d"), `'0/.-'`, nil},
		decoderTest{[]byte("\x31\x30\x2f\x2e"), `'10/.'`, nil},
		decoderTest{[]byte("\x32\x31\x30\x2f"), `'210/'`, nil},
		decoderTest{[]byte("\x5b\x5a\x59\x58"), `'[ZYX'`, nil},
		decoderTest{[]byte("\x5c\x5b\x5a\x59"), `'\[ZY'`, nil},
		decoderTest{[]byte("\x5d\x5c\x5b\x5a"), `']\[Z'`, nil},
		decoderTest{[]byte("\x5e\x5d\x5c\x5b"), `'^]\['`, nil},
		decoderTest{[]byte("\x5f\x5e\x5d\x5c"), `'_^]\'`, nil},
		decoderTest{[]byte("\x60\x5f\x5e\x5d"), "'`_^]'", nil},
		decoderTest{[]byte("\x61\x60\x5f\x5e"), "'a`_^'", nil},
		decoderTest{[]byte("\x62\x61\x60\x5f"), "'ba`_'", nil},
		decoderTest{[]byte("\x63\x62\x61\x60"), "'cba`'", nil},
		decoderTest{[]byte("\x7b\x7a\x79\x78"), `'{zyx'`, nil},
		decoderTest{[]byte("\x7c\x7b\x7a\x79"), `'|{zy'`, nil},
		decoderTest{[]byte("\x7d\x7c\x7b\x7a"), `'}|{z'`, nil},
		decoderTest{[]byte("\x7e\x7d\x7c\x7b"), `'~}|{'`, nil},
		decoderTest{[]byte("\x20\x20\x20\x20"), "0x20202020", nil},
		decoderTest{[]byte("\x00\x00\x00\x00"), "0x00000000", nil},
		decoderTest{[]byte("\x00\x00\x00\x01"), "0x00000001", nil},
		decoderTest{[]byte("\x00\x00\x00\x02"), "0x00000002", nil},
		decoderTest{[]byte("\x00\x00\x00\x03"), "0x00000003", nil},
		decoderTest{[]byte("\x00\x00\x00\x04"), "0x00000004", nil},
		decoderTest{[]byte("\x00\x00\x00\x05"), "0x00000005", nil},
		decoderTest{[]byte("\x00\x00\x00\x06"), "0x00000006", nil},
		decoderTest{[]byte("\x00\x00\x00\x07"), "0x00000007", nil},
		decoderTest{[]byte("\x00\x00\x00\x08"), "0x00000008", nil},
		decoderTest{[]byte("\x00\x00\x00\x09"), "0x00000009", nil},
		decoderTest{[]byte("\x00\x00\x00\x0a"), "0x0000000A", nil},
		decoderTest{[]byte("\x00\x00\x00\x0b"), "0x0000000B", nil},
		decoderTest{[]byte("\x00\x00\x00\x0c"), "0x0000000C", nil},
		decoderTest{[]byte("\x00\x00\x00\x0d"), "0x0000000D", nil},
		decoderTest{[]byte("\x00\x00\x00\x0e"), "0x0000000E", nil},
		decoderTest{[]byte("\x00\x00\x00\x0f"), "0x0000000F", nil},
		decoderTest{[]byte("\x01\x00\x00\x00"), "0x01000000", nil},
		decoderTest{[]byte("\x02\x00\x00\x00"), "0x02000000", nil},
		decoderTest{[]byte("\x03\x00\x00\x00"), "0x03000000", nil},
		decoderTest{[]byte("\x04\x00\x00\x00"), "0x04000000", nil},
		decoderTest{[]byte("\x05\x00\x00\x00"), "0x05000000", nil},
		decoderTest{[]byte("\x06\x00\x00\x00"), "0x06000000", nil},
		decoderTest{[]byte("\x07\x00\x00\x00"), "0x07000000", nil},
		decoderTest{[]byte("\x08\x00\x00\x00"), "0x08000000", nil},
		decoderTest{[]byte("\x09\x00\x00\x00"), "0x09000000", nil},
		decoderTest{[]byte("\x0a\x00\x00\x00"), "0x0A000000", nil},
		decoderTest{[]byte("\x0b\x00\x00\x00"), "0x0B000000", nil},
		decoderTest{[]byte("\x0c\x00\x00\x00"), "0x0C000000", nil},
		decoderTest{[]byte("\x0d\x00\x00\x00"), "0x0D000000", nil},
		decoderTest{[]byte("\x0e\x00\x00\x00"), "0x0E000000", nil},
		decoderTest{[]byte("\x0f\x00\x00\x00"), "0x0F000000", nil},
		decoderTest{[]byte(""), zero, byteCountErr(0)},
		decoderTest{[]byte("\x00"), zero, byteCountErr(1)},
		decoderTest{[]byte("\x00\x00"), zero, byteCountErr(2)},
		decoderTest{[]byte("\x00\x00\x00\x00\x00\x00\x00\x00"), zero, byteCountErr(8)},
	}
	runDecoderTests(t, tests, func(input []byte) (interface{}, error) {
		return FC32ToStringDelimited(input)
	})
}
func TestFC32ToString(t *testing.T) {
	byteCountErr := errFunc(errByteCount).curry("FC32", 4)
	zero := ""
	tests := []decoderTest{
		decoderTest{[]byte("\x20\x7e\x7d\x7c"), "0x207E7D7C", nil},
		decoderTest{[]byte("\x21\x20\x7e\x7d"), "0x21207E7D", nil},
		decoderTest{[]byte("\x5c\x21\x20\x7e"), "0x5C21207E", nil},
		decoderTest{[]byte("\x23\x5c\x21\x20"), "0x235C2120", nil},
		decoderTest{[]byte("\x24\x23\x5c\x21"), `$#\!`, nil},
		decoderTest{[]byte("\x25\x24\x23\x5c"), `%$#\`, nil},
		decoderTest{[]byte("\x26\x25\x24\x23"), "&%$#", nil},
		decoderTest{[]byte("\x27\x26\x25\x24"), "0x27262524", nil}, // starts with '
		decoderTest{[]byte("\x28\x27\x26\x25"), "('&%", nil},
		decoderTest{[]byte("\x29\x28\x27\x26"), ")('&", nil},
		decoderTest{[]byte("\x2a\x29\x28\x27"), "*)('", nil},
		decoderTest{[]byte("\x2b\x2a\x29\x28"), "+*)(", nil},
		decoderTest{[]byte("\x2c\x2b\x2a\x29"), ",+*)", nil},
		decoderTest{[]byte("\x2d\x2c\x2b\x2a"), "-,+*", nil},
		decoderTest{[]byte("\x2e\x2d\x2c\x2b"), ".-,+", nil},
		decoderTest{[]byte("\x2f\x2e\x2d\x2c"), "/.-,", nil},
		decoderTest{[]byte("\x30\x2f\x2e\x2d"), "0/.-", nil},
		decoderTest{[]byte("\x31\x30\x2f\x2e"), "10/.", nil},
		decoderTest{[]byte("\x32\x31\x30\x2f"), "210/", nil},
		decoderTest{[]byte("\x5b\x5a\x59\x58"), "[ZYX", nil},
		decoderTest{[]byte("\x5c\x5b\x5a\x59"), `\[ZY`, nil},
		decoderTest{[]byte("\x5d\x5c\x5b\x5a"), `]\[Z`, nil},
		decoderTest{[]byte("\x5e\x5d\x5c\x5b"), `^]\[`, nil},
		decoderTest{[]byte("\x5f\x5e\x5d\x5c"), `_^]\`, nil},
		decoderTest{[]byte("\x60\x5f\x5e\x5d"), "`_^]", nil},
		decoderTest{[]byte("\x61\x60\x5f\x5e"), "a`_^", nil},
		decoderTest{[]byte("\x62\x61\x60\x5f"), "ba`_", nil},
		decoderTest{[]byte("\x63\x62\x61\x60"), "cba`", nil},
		decoderTest{[]byte("\x7b\x7a\x79\x78"), "{zyx", nil},
		decoderTest{[]byte("\x7c\x7b\x7a\x79"), "|{zy", nil},
		decoderTest{[]byte("\x7d\x7c\x7b\x7a"), "}|{z", nil},
		decoderTest{[]byte("\x7e\x7d\x7c\x7b"), "~}|{", nil},
		decoderTest{[]byte("\x20\x20\x20\x20"), "0x20202020", nil},
		decoderTest{[]byte("\x00\x00\x00\x00"), "0x00000000", nil},
		decoderTest{[]byte("\x00\x00\x00\x01"), "0x00000001", nil},
		decoderTest{[]byte("\x00\x00\x00\x02"), "0x00000002", nil},
		decoderTest{[]byte("\x0a\x00\x00\x00"), "0x0A000000", nil},
		decoderTest{[]byte("\x0b\x00\x00\x00"), "0x0B000000", nil},
		decoderTest{[]byte("\x0c\x00\x00\x00"), "0x0C000000", nil},
		decoderTest{[]byte("\x0d\x00\x00\x00"), "0x0D000000", nil},
		decoderTest{[]byte("\x0e\x00\x00\x00"), "0x0E000000", nil},
		decoderTest{[]byte("\x0f\x00\x00\x00"), "0x0F000000", nil},
		decoderTest{[]byte(""), zero, byteCountErr(0)},
		decoderTest{[]byte("\x00"), zero, byteCountErr(1)},
		decoderTest{[]byte("\x00\x00"), zero, byteCountErr(2)},
		decoderTest{[]byte("\x00\x00\x00\x00\x00\x00\x00\x00"), zero, byteCountErr(8)},
	}
	runDecoderTests(t, tests, func(input []byte) (interface{}, error) {
		return FC32ToString(input)
	})
}

func TestUUIDToString(t *testing.T) {
	byteCountErr := errFunc(errByteCount).curry("UUID", 16)
	zero := ""
	tests := []decoderTest{
		decoderTest{[]byte("\x64\x88\x14\x31\xb6\xdc\x47\x8e\xb7\xee\xed\x30\x66\x19\xc7\x97"), "64881431-B6DC-478E-B7EE-ED306619C797", nil},
		decoderTest{[]byte("\xa3\xbf\xff\x54\xf4\x74\x42\xe9\xab\x53\x01\xd9\x13\xd1\x18\xb1"), "A3BFFF54-F474-42E9-AB53-01D913D118B1", nil},
		decoderTest{[]byte("\x64\x88\x14\x31\xb6\xdc\x47\x8e\xb7\xee\xed\x30\x66\x19\xc7\x97"), "64881431-B6DC-478E-B7EE-ED306619C797", nil},
		decoderTest{[]byte("\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"), "00000000-0000-0000-0000-000000000000", nil},
		decoderTest{[]byte("\xFF\xFF\xFF\xFF\xFF\xFF\xFF\xFF\xFF\xFF\xFF\xFF\xFF\xFF\xFF\xFF"), "FFFFFFFF-FFFF-FFFF-FFFF-FFFFFFFFFFFF", nil},
		decoderTest{[]byte(""), zero, byteCountErr(0)},
		decoderTest{[]byte("\x00"), zero, byteCountErr(1)},
		decoderTest{[]byte("\x00\x00"), zero, byteCountErr(2)},
		decoderTest{[]byte("\x00\x00\x00\x00"), zero, byteCountErr(4)},
		decoderTest{[]byte("\x00\x00\x00\x00\x00\x00\x00\x00"), zero, byteCountErr(8)},
		decoderTest{[]byte("\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"), zero, byteCountErr(20)},
	}
	runDecoderTests(t, tests, func(input []byte) (interface{}, error) {
		return UUIDToString(input)
	})
}

func TestIP32ToString(t *testing.T) {
	byteCountErr := errFunc(errByteCount).curry("IP32", 4)
	zero := ""
	tests := []decoderTest{
		decoderTest{[]byte("\x00\x00\x00\x00"), "0.0.0.0", nil},
		decoderTest{[]byte("\x11\x22\x33\x44"), "17.34.51.68", nil},
		decoderTest{[]byte("\xC0\xA8\x01\x80"), "192.168.1.128", nil},
		decoderTest{[]byte("\xF1\xAB\xCD\xEF"), "241.171.205.239", nil},
		decoderTest{[]byte("\xff\xff\xff\xff"), "255.255.255.255", nil},
		decoderTest{[]byte("\x00\x00\x00\x00\xff\xff\xff\xff"), "0.0.0.0-255.255.255.255", nil},
		decoderTest{[]byte(""), zero, byteCountErr(0)},
		decoderTest{[]byte("\x00"), zero, byteCountErr(1)},
		decoderTest{[]byte("\x00\x00"), zero, byteCountErr(2)},
	}
	runDecoderTests(t, tests, func(input []byte) (interface{}, error) {
		return IP32ToString(input)
	})
}

func TestIPADToString(t *testing.T) {
	tests := []decoderTest{
		decoderTest{[]byte("\x30\x2e\x30\x2e\x30\x2e\x30\x00"), "\"0.0.0.0\"", nil},
		decoderTest{[]byte("\x31\x2e\x31\x2e\x31\x2e\x31\x00"), "\"1.1.1.1\"", nil},
		decoderTest{[]byte("\x31\x2e\x32\x35\x35\x2e\x33\x2e\x34\x00"), "\"1.255.3.4\"", nil},
		decoderTest{[]byte("\x31\x30\x2e\x32\x35\x35\x2e\x32\x35\x35\x2e\x32\x35\x34\x00"), "\"10.255.255.254\"", nil},
		decoderTest{[]byte("\x31\x32\x37\x2e\x30\x2e\x30\x2e\x31\x00"), "\"127.0.0.1\"", nil},
		decoderTest{[]byte("\x31\x37\x32\x2e\x31\x38\x2e\x35\x2e\x34\x00"), "\"172.18.5.4\"", nil},
		decoderTest{[]byte("\x31\x39\x32\x2e\x31\x36\x38\x2e\x30\x2e\x31\x00"), "\"192.168.0.1\"", nil},
		decoderTest{[]byte("\x31\x39\x32\x2e\x31\x36\x38\x2e\x31\x2e\x30\x00"), "\"192.168.1.0\"", nil},
		decoderTest{[]byte("\x32\x30\x30\x31\x3a\x30\x30\x30\x30\x3a\x34\x31\x33\x36\x3a\x65\x33\x37\x38\x3a\x38\x30\x30\x30\x3a\x36\x33\x62\x66\x3a\x33\x66\x66\x66\x3a\x66\x64\x64\x32\x00"), "\"2001:0000:4136:e378:8000:63bf:3fff:fdd2\"", nil},
		decoderTest{[]byte("\x32\x30\x30\x31\x3a\x30\x30\x30\x30\x3a\x34\x31\x33\x36\x3a\x65\x33\x37\x38\x3a\x38\x30\x30\x30\x3a\x36\x33\x62\x66\x3a\x33\x66\x66\x66\x3a\x66\x64\x64\x32\x00"), "\"2001:0000:4136:e378:8000:63bf:3fff:fdd2\"", nil},
		decoderTest{[]byte("\x32\x30\x30\x31\x3a\x30\x30\x30\x32\x3a\x36\x63\x3a\x3a\x34\x33\x30\x00"), "\"2001:0002:6c::430\"", nil},
		decoderTest{[]byte("\x32\x30\x30\x31\x3a\x31\x30\x3a\x32\x34\x30\x3a\x61\x62\x3a\x3a\x61\x00"), "\"2001:10:240:ab::a\"", nil},
		decoderTest{[]byte("\x32\x30\x30\x31\x3a\x3a\x31\x00"), "\"2001::1\"", nil},
		decoderTest{[]byte("\x32\x30\x30\x31\x3a\x3a\x31\x00"), "\"2001::1\"", nil},
		decoderTest{[]byte("\x32\x30\x30\x31\x3a\x64\x62\x38\x3a\x38\x3a\x34\x3a\x3a\x32\x00"), "\"2001:db8:8:4::2\"", nil},
		decoderTest{[]byte("\x32\x30\x30\x32\x3a\x63\x62\x30\x61\x3a\x33\x63\x64\x64\x3a\x31\x3a\x3a\x31\x00"), "\"2002:cb0a:3cdd:1::1\"", nil},
		decoderTest{[]byte("\x32\x35\x35\x2e\x30\x2e\x30\x2e\x31\x00"), "\"255.0.0.1\"", nil},
		decoderTest{[]byte("\x32\x35\x35\x2e\x32\x35\x35\x2e\x32\x35\x35\x2e\x32\x35\x35\x00"), "\"255.255.255.255\"", nil},
		decoderTest{[]byte("\x38\x2e\x38\x2e\x34\x2e\x34\x00"), "\"8.8.4.4\"", nil},
		decoderTest{[]byte("\x3a\x3a\x00"), "\"::\"", nil},
		decoderTest{[]byte("\x3a\x3a\x66\x66\x66\x66\x3a\x35\x2e\x36\x2e\x37\x2e\x38\x00"), "\"::ffff:5.6.7.8\"", nil},
		decoderTest{[]byte("\x66\x64\x66\x38\x3a\x66\x35\x33\x62\x3a\x38\x32\x65\x34\x3a\x3a\x35\x33\x00"), "\"fdf8:f53b:82e4::53\"", nil},
		decoderTest{[]byte("\x66\x64\x66\x38\x3a\x66\x35\x33\x62\x3a\x38\x32\x65\x34\x3a\x3a\x35\x33\x00"), "\"fdf8:f53b:82e4::53\"", nil},
		decoderTest{[]byte("\x66\x65\x38\x30\x3a\x3a\x32\x30\x30\x3a\x35\x61\x65\x65\x3a\x66\x65\x61\x61\x3a\x32\x30\x61\x32\x00"), "\"fe80::200:5aee:feaa:20a2\"", nil},
		decoderTest{[]byte("\x66\x66\x30\x31\x3a\x30\x3a\x30\x3a\x30\x3a\x30\x3a\x30\x3a\x30\x3a\x32\x00"), "\"ff01:0:0:0:0:0:0:2\"", nil},
	}
	runDecoderTests(t, tests, func(input []byte) (interface{}, error) {
		return IPADToString(input)
	})
}
func TestCSTRToString(t *testing.T) {
	tests := []decoderTest{
		decoderTest{[]byte(""), "", fmt.Errorf("Illegal CSTR data lacks null byte terminator")},
		decoderTest{[]byte("\x20\x20\x20\x20\x01\x02\x03\x00"), "    \x01\x02\x03", nil},
		decoderTest{[]byte("\x04\x05\x06\x07\x00"), "\x04\x05\x06\x07", nil},
		decoderTest{[]byte("\x08\x09\x0a\x0b\x00"), "\x08\x09\n\x0B", nil},
		decoderTest{[]byte("\x0c\x0d\x0e\x0f\x00"), "\x0C\r\x0E\x0F", nil},
		decoderTest{[]byte("\x10\x11\x12\x13\x00"), "\x10\x11\x12\x13", nil},
		decoderTest{[]byte("\x14\x15\x16\x17\x00"), "\x14\x15\x16\x17", nil},
		decoderTest{[]byte("\x18\x19\x1a\x1b\x00"), "\x18\x19\x1A\x1B", nil},
		decoderTest{[]byte("\x1c\x1d\x1e\x1f\x00"), "\x1C\x1D\x1E\x1F", nil},
		decoderTest{[]byte("\x20\x21\x22\x23\x00"), ` !"#`, nil},
		decoderTest{[]byte("\x24\x25\x26\x27\x00"), "$%&'", nil},
		decoderTest{[]byte("\x28\x29\x2a\x2b\x00"), "()*+", nil},
		decoderTest{[]byte("\x2c\x2d\x2e\x2f\x00"), ",-./", nil},
		decoderTest{[]byte("\x30\x31\x32\x33\x00"), "0123", nil},
		decoderTest{[]byte("\x34\x35\x36\x37\x00"), "4567", nil},
		decoderTest{[]byte("\x38\x39\x3a\x3b\x00"), "89:;", nil},
		decoderTest{[]byte("\x3c\x3d\x3e\x3f\x00"), "<=>?", nil},
		decoderTest{[]byte("\x40\x41\x42\x43\x00"), "@ABC", nil},
		decoderTest{[]byte("\x44\x45\x46\x47\x00"), "DEFG", nil},
		decoderTest{[]byte("\x48\x49\x4a\x4b\x00"), "HIJK", nil},
		decoderTest{[]byte("\x4c\x4d\x4e\x4f\x00"), "LMNO", nil},
		decoderTest{[]byte("\x50\x51\x52\x53\x00"), "PQRS", nil},
		decoderTest{[]byte("\x54\x55\x56\x57\x00"), "TUVW", nil},
		decoderTest{[]byte("\x58\x59\x5a\x5b\x00"), "XYZ[", nil},
		decoderTest{[]byte("\x5c\x5d\x5e\x5f\x00"), `\]^_`, nil},
		decoderTest{[]byte("\x60\x61\x62\x63\x00"), "`abc", nil},
		decoderTest{[]byte("\x64\x65\x66\x67\x00"), "defg", nil},
		decoderTest{[]byte("\x68\x69\x6a\x6b\x00"), "hijk", nil},
		decoderTest{[]byte("\x6c\x6d\x6e\x6f\x00"), "lmno", nil},
		decoderTest{[]byte("\x70\x71\x72\x73\x00"), "pqrs", nil},
		decoderTest{[]byte("\x74\x75\x76\x77\x00"), "tuvw", nil},
		decoderTest{[]byte("\x78\x79\x7a\x7b\x00"), "xyz{", nil},
		decoderTest{[]byte("\x7c\x7d\x7e\x7f\x00"), "|}~\x7F", nil},
	}
	runDecoderTests(t, tests, func(input []byte) (interface{}, error) {
		return CSTRToString(input)
	})
}
func TestUSTRToString(t *testing.T) {
	tests := []decoderTest{
		decoderTest{[]byte(""), ``, nil},
		decoderTest{[]byte("\x00\x00\x00\x00\x00\x00\x00\x40"), "\x00@", nil},
		decoderTest{[]byte("\x00\x00\x00\x01\x00\x00\x00\x41"), "\x01A", nil},
		decoderTest{[]byte("\x00\x00\x00\x02\x00\x00\x00\x42"), "\x02B", nil},
		decoderTest{[]byte("\x00\x00\x00\x03\x00\x00\x00\x43"), "\x03C", nil},
		decoderTest{[]byte("\x00\x00\x00\x04\x00\x00\x00\x44"), "\x04D", nil},
		decoderTest{[]byte("\x00\x00\x00\x05\x00\x00\x00\x45"), "\x05E", nil},
		decoderTest{[]byte("\x00\x00\x00\x06\x00\x00\x00\x46"), "\x06F", nil},
		decoderTest{[]byte("\x00\x00\x00\x07\x00\x00\x00\x47"), "\x07G", nil},
		decoderTest{[]byte("\x00\x00\x00\x08\x00\x00\x00\x48"), "\x08H", nil},
		decoderTest{[]byte("\x00\x00\x00\x09\x00\x00\x00\x49"), "\x09I", nil},
		decoderTest{[]byte("\x00\x00\x00\x0A\x00\x00\x00\x4A"), "\x0AJ", nil},
		decoderTest{[]byte("\x00\x00\x00\x0B\x00\x00\x00\x4B"), "\x0BK", nil},
		decoderTest{[]byte("\x00\x00\x00\x0C\x00\x00\x00\x4C"), "\x0CL", nil},
		decoderTest{[]byte("\x00\x00\x00\x0D\x00\x00\x00\x4D"), "\x0DM", nil},
		decoderTest{[]byte("\x00\x00\x00\x0E\x00\x00\x00\x4E"), "\x0EN", nil},
		decoderTest{[]byte("\x00\x00\x00\x0F\x00\x00\x00\x4F"), "\x0FO", nil},
		decoderTest{[]byte("\x00\x00\x00\x10\x00\x00\x00\x50"), "\x10P", nil},
		decoderTest{[]byte("\x00\x00\x00\x11\x00\x00\x00\x51"), "\x11Q", nil},
		decoderTest{[]byte("\x00\x00\x00\x12\x00\x00\x00\x52"), "\x12R", nil},
		decoderTest{[]byte("\x00\x00\x00\x13\x00\x00\x00\x53"), "\x13S", nil},
		decoderTest{[]byte("\x00\x00\x00\x14\x00\x00\x00\x54"), "\x14T", nil},
		decoderTest{[]byte("\x00\x00\x00\x15\x00\x00\x00\x55"), "\x15U", nil},
		decoderTest{[]byte("\x00\x00\x00\x16\x00\x00\x00\x56"), "\x16V", nil},
		decoderTest{[]byte("\x00\x00\x00\x17\x00\x00\x00\x57"), "\x17W", nil},
		decoderTest{[]byte("\x00\x00\x00\x18\x00\x00\x00\x58"), "\x18X", nil},
		decoderTest{[]byte("\x00\x00\x00\x19\x00\x00\x00\x59"), "\x19Y", nil},
		decoderTest{[]byte("\x00\x00\x00\x1A\x00\x00\x00\x5A"), "\x1AZ", nil},
		decoderTest{[]byte("\x00\x00\x00\x1B\x00\x00\x00\x5B"), "\x1B[", nil},
		decoderTest{[]byte("\x00\x00\x00\x1C\x00\x00\x00\x5C"), "\x1C\\", nil},
		decoderTest{[]byte("\x00\x00\x00\x1D\x00\x00\x00\x5D"), "\x1D]", nil},
		decoderTest{[]byte("\x00\x00\x00\x1E\x00\x00\x00\x5E"), "\x1E^", nil},
		decoderTest{[]byte("\x00\x00\x00\x1F\x00\x00\x00\x5F"), "\x1F_", nil},
		decoderTest{[]byte("\x00\x00\x00\x20\x00\x00\x00\x60"), "\x20`", nil},
		decoderTest{[]byte("\x00\x00\x00\x21\x00\x00\x00\x61"), "\x21a", nil},
		decoderTest{[]byte("\x00\x00\x00\x22\x00\x00\x00\x62"), "\x22b", nil},
		decoderTest{[]byte("\x00\x00\x00\x23\x00\x00\x00\x63"), "#c", nil},
		decoderTest{[]byte("\x00\x00\x00\x24\x00\x00\x00\x64"), "$d", nil},
		decoderTest{[]byte("\x00\x00\x00\x25\x00\x00\x00\x65"), "%e", nil},
		decoderTest{[]byte("\x00\x00\x00\x26\x00\x00\x00\x66"), "&f", nil},
		decoderTest{[]byte("\x00\x00\x00\x27\x00\x00\x00\x67"), "'g", nil},
		decoderTest{[]byte("\x00\x00\x00\x28\x00\x00\x00\x68"), "(h", nil},
		decoderTest{[]byte("\x00\x00\x00\x29\x00\x00\x00\x69"), ")i", nil},
		decoderTest{[]byte("\x00\x00\x00\x2A\x00\x00\x00\x6A"), "*j", nil},
		decoderTest{[]byte("\x00\x00\x00\x2B\x00\x00\x00\x6B"), "+k", nil},
		decoderTest{[]byte("\x00\x00\x00\x2C\x00\x00\x00\x6C"), ",l", nil},
		decoderTest{[]byte("\x00\x00\x00\x2D\x00\x00\x00\x6D"), "-m", nil},
		decoderTest{[]byte("\x00\x00\x00\x2E\x00\x00\x00\x6E"), ".n", nil},
		decoderTest{[]byte("\x00\x00\x00\x2F\x00\x00\x00\x6F"), "/o", nil},
		decoderTest{[]byte("\x00\x00\x00\x30\x00\x00\x00\x70"), "0p", nil},
		decoderTest{[]byte("\x00\x00\x00\x31\x00\x00\x00\x71"), "1q", nil},
		decoderTest{[]byte("\x00\x00\x00\x32\x00\x00\x00\x72"), "2r", nil},
		decoderTest{[]byte("\x00\x00\x00\x33\x00\x00\x00\x73"), "3s", nil},
		decoderTest{[]byte("\x00\x00\x00\x34\x00\x00\x00\x74"), "4t", nil},
		decoderTest{[]byte("\x00\x00\x00\x35\x00\x00\x00\x75"), "5u", nil},
		decoderTest{[]byte("\x00\x00\x00\x36\x00\x00\x00\x76"), "6v", nil},
		decoderTest{[]byte("\x00\x00\x00\x37\x00\x00\x00\x77"), "7w", nil},
		decoderTest{[]byte("\x00\x00\x00\x38\x00\x00\x00\x78"), "8x", nil},
		decoderTest{[]byte("\x00\x00\x00\x39\x00\x00\x00\x79"), "9y", nil},
		decoderTest{[]byte("\x00\x00\x00\x3A\x00\x00\x00\x7A"), ":z", nil},
		decoderTest{[]byte("\x00\x00\x00\x3B\x00\x00\x00\x7B"), ";{", nil},
		decoderTest{[]byte("\x00\x00\x00\x3C\x00\x00\x00\x7C"), "<|", nil},
		decoderTest{[]byte("\x00\x00\x00\x3D\x00\x00\x00\x7D"), "=}", nil},
		decoderTest{[]byte("\x00\x00\x00\x3E\x00\x00\x00\x7E"), ">~", nil},
		decoderTest{[]byte("\x00\x00\x00\x3F\x00\x00\x00\x7F"), "?\x7F", nil},
		// FIXME
		//    decoderTest{[]byte("\x00\x02\xf8\x00\x00\x02\xf8\x01"), "丽丸", nil},
		//		decoderTest{[]byte("\x00\x02\xf8\x02\x00\x02\xf8\x03"), "乁𠄢", nil},
		//		decoderTest{[]byte("\x00\x02\xf8\x04\x00\x02\xf8\x05"), "你侮", nil},
	}
	runDecoderTests(t, tests, func(input []byte) (interface{}, error) {
		return USTRToString(input)
	})
}

func TestCSTRToStringEscaped(t *testing.T) {
	tests := []decoderTest{
		decoderTest{[]byte(""), "", fmt.Errorf("Illegal CSTR data lacks null byte terminator")},
		decoderTest{[]byte("\x61\x62\x63\x0a\x64\x65\x66\x00"), `"abc\ndef"`, nil},
		decoderTest{[]byte("\x61\x62\x63\x0d\x64\x65\x66\x00"), `"abc\rdef"`, nil},
		decoderTest{[]byte("\x61\x62\x63\x5c\x64\x65\x66\x00"), `"abc\\def"`, nil},
		decoderTest{[]byte("\x61\x62\x63\x22\x64\x65\x66\x00"), `"abc\"def"`, nil},
		decoderTest{[]byte("\x61\x62\x63\x7f\x64\x65\x66\x00"), `"abc\x7Fdef"`, nil},
	}
	runDecoderTests(t, tests, func(input []byte) (interface{}, error) {
		return CSTRToStringEscaped(input)
	})
}

func TestUSTRToStringEscaped(t *testing.T) {
	tests := []decoderTest{
		decoderTest{[]byte("\x00\x00\x00\x0a"), `"\n"`, nil},
		decoderTest{[]byte("\x00\x00\x00\x0d"), `"\r"`, nil},
		decoderTest{[]byte("\x00\x00\x00\x5c"), `"\\"`, nil},
		decoderTest{[]byte("\x00\x00\x00\x22"), `"\""`, nil},
		decoderTest{[]byte("\x00\x00\x00\x7f"), `"\x7F"`, nil},
	}
	runDecoderTests(t, tests, func(input []byte) (interface{}, error) {
		return USTRToStringEscaped(input)
	})
}

func TestBytesToHexString(t *testing.T) {
	tests := []decoderTest{
		decoderTest{[]byte{}, "", nil},
		decoderTest{[]byte("\x00"), "0x00", nil},
		decoderTest{[]byte("\x00\x00"), "0x0000", nil},
		decoderTest{[]byte("\x00\x00\x00\x00"), "0x00000000", nil},
		decoderTest{[]byte("\x00\x00\x00\x00\x00\x00\x00\x00"), "0x0000000000000000", nil},
		decoderTest{[]byte("\xFF"), "0xFF", nil},
		decoderTest{[]byte("\xFF\xFF"), "0xFFFF", nil},
		decoderTest{[]byte("\xFF\xFF\xFF\xFF"), "0xFFFFFFFF", nil},
		decoderTest{[]byte("\xFF\xFF\xFF\xFF\xFF\xFF\xFF\xFF"), "0xFFFFFFFFFFFFFFFF", nil},
	}
	runDecoderTests(t, tests, func(input []byte) (interface{}, error) {
		return BytesToHexString(input)
	})
}
func TestAdeStringEscape(t *testing.T) {
	tests := []decoderTest{
		decoderTest{[]byte(""), "", nil},
		decoderTest{[]byte("\x61\x62\x63\x0a\x64\x65\x66\x00"), `abc\ndef\x00`, nil},
		decoderTest{[]byte("\x61\x62\x63\x0d\x64\x65\x66\x00"), `abc\rdef\x00`, nil},
		decoderTest{[]byte("\x61\x62\x63\x5c\x64\x65\x66"), `abc\\def`, nil},
		decoderTest{[]byte("\x61\x62\x63\x22\x64\x65\x66"), `abc\"def`, nil},
		decoderTest{[]byte("\x61\x62\x63\x7f\x64\x65\x66"), `abc\x7Fdef`, nil},
	}
	runDecoderTests(t, tests, func(input []byte) (interface{}, error) {
		return adeStringEscape(string(input)), nil
	})
}

// *****************************************************
// 2. Test encoding funcs, which write to an Atom's data
// *****************************************************

// *** encode test framework
type (
	// An encodeFunc converts a golang native type to a byte slice at Atom.data
	encodeFunc func(*Atom, interface{}) error

	// encoderTest defines input and expected output values for an encodeFunc
	encoderTest struct {
		Input     interface{}
		WantValue []byte
		WantError error
	}
)

// runEncoderTests evaluates an encodeFunc against test data
func runEncoderTests(t *testing.T, tests []encoderTest, f encodeFunc) {
	for _, test := range tests {
		funcName := GetFunctionName(f)
		var inputAtom = new(Atom)
		var gotErr error = f(inputAtom, test.Input)
		var gotValue []byte = inputAtom.data

		switch {
		case gotErr == nil && test.WantError == nil:
		case gotErr != nil && test.WantError == nil:
			t.Errorf("%v(%b): got err {%s}, want err <nil>", funcName, test.Input, gotErr)
			return
		case gotErr == nil && test.WantError != nil:
			t.Errorf("%v(%b): got err <nil>, want err {%s}", funcName, test.Input, test.WantError)
			return
		case gotErr.Error() != test.WantError.Error():
			t.Errorf("%v(%v): got err {%s}, want err {%s}", funcName, test.Input, gotErr, test.WantError)
			return
		}

		// Instead of ==, compare with DeepEqual because it can compare slices of bytes
		if !reflect.DeepEqual(gotValue, test.WantValue) {
			t.Errorf("%v(Atom, %v): got %T(%[3]v), want %[4]T(%[4]v)", funcName, test.Input, gotValue, test.WantValue)
		}
	}
}

// *** unit tests

func TestSetUI01FromString(t *testing.T) {
	typ := "UI01"
	zero := make([]byte, 4)
	tests := []encoderTest{
		encoderTest{"false", []byte("\x00\x00\x00\x00"), nil},
		encoderTest{"true", []byte("\x00\x00\x00\x01"), nil},
		encoderTest{"0", []byte("\x00\x00\x00\x00"), nil},
		encoderTest{"1", []byte("\x00\x00\x00\x01"), nil},
		encoderTest{" 0", zero, errStrInvalid(typ, " 0")},
		encoderTest{"1 ", zero, errStrInvalid(typ, "1 ")},
		encoderTest{"", zero, errStrInvalid(typ, "")},
		encoderTest{" ", zero, errStrInvalid(typ, " ")},
		encoderTest{"00", zero, errStrInvalid(typ, "00")},
		encoderTest{"01", zero, errStrInvalid(typ, "01")},
		encoderTest{"10", zero, errStrInvalid(typ, "10")},
		encoderTest{"0x01", zero, errStrInvalid(typ, "0x01")},
		encoderTest{"dog", zero, errStrInvalid(typ, "dog")},
	}
	runEncoderTests(t, tests, func(atom *Atom, input interface{}) error {
		return SetUI01FromString(atom, input.(string))
	})
}
func TestSetUI01FromBool(t *testing.T) {
	tests := []encoderTest{
		encoderTest{false, []byte("\x00\x00\x00\x00"), nil},
		encoderTest{true, []byte("\x00\x00\x00\x01"), nil},
	}
	runEncoderTests(t, tests, func(atom *Atom, input interface{}) error {
		return SetUI01FromBool(atom, input.(bool))
	})
}
func TestSetUI01FromUint64(t *testing.T) {
	typ := "UI01"
	zero := make([]byte, 4)
	tests := []encoderTest{
		encoderTest{uint64(0), []byte("\x00\x00\x00\x00"), nil},
		encoderTest{uint64(1), []byte("\x00\x00\x00\x01"), nil},
		encoderTest{uint64(0x00), []byte("\x00\x00\x00\x00"), nil},
		encoderTest{uint64(0x01), []byte("\x00\x00\x00\x01"), nil},
		encoderTest{uint64(2), zero, errRange(typ, uint64(2))},
		encoderTest{uint64(10), zero, errRange(typ, uint64(10))},
	}
	runEncoderTests(t, tests, func(atom *Atom, input interface{}) error {
		return SetUI01FromUint64(atom, input.(uint64))
	})
}
func TestSetUI08FromString(t *testing.T) {
	typ := "UI08"
	zero := make([]byte, 1)
	tests := []encoderTest{
		encoderTest{"0", []byte("\x00"), nil},
		encoderTest{"15", []byte("\x0F"), nil},
		encoderTest{"240", []byte("\xF0"), nil},
		encoderTest{"255", []byte("\xFF"), nil},
		encoderTest{"3000", zero, errStrInvalid(typ, "3000")},
		encoderTest{"-1", zero, errStrInvalid(typ, "-1")},
		encoderTest{"dog", zero, errStrInvalid(typ, "dog")},
		encoderTest{"", zero, errStrInvalid(typ, "")},
		encoderTest{" ", zero, errStrInvalid(typ, " ")},
	}
	runEncoderTests(t, tests, func(atom *Atom, input interface{}) error {
		return SetUI08FromString(atom, input.(string))
	})
}
func TestSetUI08FromUint64(t *testing.T) {
	typ := "UI08"
	zero := make([]byte, 1)
	tests := []encoderTest{
		encoderTest{uint64(0), []byte("\x00"), nil},
		encoderTest{uint64(15), []byte("\x0F"), nil},
		encoderTest{uint64(240), []byte("\xF0"), nil},
		encoderTest{uint64(255), []byte("\xFF"), nil},
		encoderTest{uint64(3000), zero, errRange(typ, uint64(3000))},
	}
	runEncoderTests(t, tests, func(atom *Atom, input interface{}) error {
		return SetUI08FromUint64(atom, input.(uint64))
	})
}
func TestSetUI16FromString(t *testing.T) {
	typ := "UI16"
	zero := make([]byte, 2)
	tests := []encoderTest{
		encoderTest{"0", []byte("\x00\x00"), nil},
		encoderTest{"255", []byte("\x00\xFF"), nil},
		encoderTest{"65280", []byte("\xFF\x00"), nil},
		encoderTest{"65535", []byte("\xFF\xFF"), nil},
		encoderTest{"65536", zero, errStrInvalid(typ, "65536")},
		encoderTest{"-1", zero, errStrInvalid(typ, "-1")},
		encoderTest{"dog", zero, errStrInvalid(typ, "dog")},
		encoderTest{"", zero, errStrInvalid(typ, "")},
		encoderTest{" ", zero, errStrInvalid(typ, " ")},
	}
	runEncoderTests(t, tests, func(atom *Atom, input interface{}) error {
		return SetUI16FromString(atom, input.(string))
	})
}
func TestSetUI16FromUint64(t *testing.T) {
	typ := "UI16"
	zero := make([]byte, 2)
	tests := []encoderTest{
		encoderTest{uint64(0), []byte("\x00\x00"), nil},
		encoderTest{uint64(255), []byte("\x00\xFF"), nil},
		encoderTest{uint64(65280), []byte("\xFF\x00"), nil},
		encoderTest{uint64(65535), []byte("\xFF\xFF"), nil},
		encoderTest{uint64(65536), zero, errRange(typ, uint64(65536))},
	}
	runEncoderTests(t, tests, func(atom *Atom, input interface{}) error {
		return SetUI16FromUint64(atom, input.(uint64))
	})
}
func TestSetUI32FromString(t *testing.T) {
	typ := "UI32"
	zero := make([]byte, 4)
	tests := []encoderTest{
		encoderTest{"0", []byte("\x00\x00\x00\x00"), nil},
		encoderTest{"0x00000000", []byte("\x00\x00\x00\x00"), nil},
		encoderTest{"255", []byte("\x00\x00\x00\xFF"), nil},
		encoderTest{"0x000000FF", []byte("\x00\x00\x00\xFF"), nil},
		encoderTest{"0x0000FF00", []byte("\x00\x00\xFF\x00"), nil},
		encoderTest{"0x00FF0000", []byte("\x00\xFF\x00\x00"), nil},
		encoderTest{"0xFF000000", []byte("\xFF\x00\x00\x00"), nil},
		encoderTest{"0xFFFFFFFF", []byte("\xFF\xFF\xFF\xFF"), nil},
		encoderTest{"0x0100000000", zero, errStrInvalid(typ, "0x0100000000")},
		encoderTest{"-1", zero, errStrInvalid(typ, "-1")},
		encoderTest{"dog", zero, errStrInvalid(typ, "dog")},
		encoderTest{"", zero, errStrInvalid(typ, "")},
		encoderTest{" ", zero, errStrInvalid(typ, " ")},
	}
	runEncoderTests(t, tests, func(atom *Atom, input interface{}) error {
		return SetUI32FromString(atom, input.(string))
	})
}
func TestSetUI32FromUint64(t *testing.T) {
	typ := "UI32"
	zero := make([]byte, 4)
	tests := []encoderTest{
		encoderTest{uint64(0x00000000), []byte("\x00\x00\x00\x00"), nil},
		encoderTest{uint64(0x000000FF), []byte("\x00\x00\x00\xFF"), nil},
		encoderTest{uint64(0x0000FF00), []byte("\x00\x00\xFF\x00"), nil},
		encoderTest{uint64(0x00FF0000), []byte("\x00\xFF\x00\x00"), nil},
		encoderTest{uint64(0xFF000000), []byte("\xFF\x00\x00\x00"), nil},
		encoderTest{uint64(0xFFFFFFFF), []byte("\xFF\xFF\xFF\xFF"), nil},
		encoderTest{uint64(0xFFFFFFFF + 1), zero, errRange(typ, 0xFFFFFFFF+1)},
	}
	runEncoderTests(t, tests, func(atom *Atom, input interface{}) error {
		return SetUI32FromUint64(atom, input.(uint64))
	})
}
func TestSetUI64FromString(t *testing.T) {
	typ := "UI64"
	zero := make([]byte, 8)
	tests := []encoderTest{
		encoderTest{"0", []byte("\x00\x00\x00\x00\x00\x00\x00\x00"), nil},
		encoderTest{"1", []byte("\x00\x00\x00\x00\x00\x00\x00\x01"), nil},
		encoderTest{"255", []byte("\x00\x00\x00\x00\x00\x00\x00\xFF"), nil},
		encoderTest{"0x0000000000000000", []byte("\x00\x00\x00\x00\x00\x00\x00\x00"), nil},
		encoderTest{"0x00000000000000FF", []byte("\x00\x00\x00\x00\x00\x00\x00\xFF"), nil},
		encoderTest{"0x000000000000FF00", []byte("\x00\x00\x00\x00\x00\x00\xFF\x00"), nil},
		encoderTest{"0x0000000000FF0000", []byte("\x00\x00\x00\x00\x00\xFF\x00\x00"), nil},
		encoderTest{"0x00000000FF000000", []byte("\x00\x00\x00\x00\xFF\x00\x00\x00"), nil},
		encoderTest{"0x000000FF00000000", []byte("\x00\x00\x00\xFF\x00\x00\x00\x00"), nil},
		encoderTest{"0x0000FF0000000000", []byte("\x00\x00\xFF\x00\x00\x00\x00\x00"), nil},
		encoderTest{"0x00FF000000000000", []byte("\x00\xFF\x00\x00\x00\x00\x00\x00"), nil},
		encoderTest{"0xFF00000000000000", []byte("\xFF\x00\x00\x00\x00\x00\x00\x00"), nil},
		encoderTest{"0xFFFFFFFFFFFFFFFF", []byte("\xFF\xFF\xFF\xFF\xFF\xFF\xFF\xFF"), nil},
		encoderTest{"0x010000000000000000", zero, errStrInvalid(typ, "0x010000000000000000")},
		encoderTest{"-1", zero, errStrInvalid(typ, "-1")},
		encoderTest{"dog", zero, errStrInvalid(typ, "dog")},
		encoderTest{"", zero, errStrInvalid(typ, "")},
		encoderTest{" ", zero, errStrInvalid(typ, " ")},
	}
	runEncoderTests(t, tests, func(atom *Atom, input interface{}) error {
		return SetUI64FromString(atom, input.(string))
	})
}
func TestSetUI64FromUint64(t *testing.T) {
	tests := []encoderTest{
		encoderTest{uint64(0), []byte("\x00\x00\x00\x00\x00\x00\x00\x00"), nil},
		encoderTest{uint64(1), []byte("\x00\x00\x00\x00\x00\x00\x00\x01"), nil},
		encoderTest{uint64(255), []byte("\x00\x00\x00\x00\x00\x00\x00\xFF"), nil},
		encoderTest{uint64(0x0000000000000000), []byte("\x00\x00\x00\x00\x00\x00\x00\x00"), nil},
		encoderTest{uint64(0x00000000000000FF), []byte("\x00\x00\x00\x00\x00\x00\x00\xFF"), nil},
		encoderTest{uint64(0x000000000000FF00), []byte("\x00\x00\x00\x00\x00\x00\xFF\x00"), nil},
		encoderTest{uint64(0x0000000000FF0000), []byte("\x00\x00\x00\x00\x00\xFF\x00\x00"), nil},
		encoderTest{uint64(0x00000000FF000000), []byte("\x00\x00\x00\x00\xFF\x00\x00\x00"), nil},
		encoderTest{uint64(0x000000FF00000000), []byte("\x00\x00\x00\xFF\x00\x00\x00\x00"), nil},
		encoderTest{uint64(0x0000FF0000000000), []byte("\x00\x00\xFF\x00\x00\x00\x00\x00"), nil},
		encoderTest{uint64(0x00FF000000000000), []byte("\x00\xFF\x00\x00\x00\x00\x00\x00"), nil},
		encoderTest{uint64(0xFF00000000000000), []byte("\xFF\x00\x00\x00\x00\x00\x00\x00"), nil},
		encoderTest{uint64(0xFFFFFFFFFFFFFFFF), []byte("\xFF\xFF\xFF\xFF\xFF\xFF\xFF\xFF"), nil},
	}
	runEncoderTests(t, tests, func(atom *Atom, input interface{}) error {
		return SetUI64FromUint64(atom, input.(uint64))
	})
}

func TestSetSI08FromString(t *testing.T) {
	typ := "SI08"
	zero := make([]byte, 1)
	tests := []encoderTest{
		encoderTest{"-128", []byte("\x80"), nil},
		encoderTest{"-1", []byte("\xFF"), nil},
		encoderTest{"0", []byte("\x00"), nil},
		encoderTest{"00", []byte("\x00"), nil},
		encoderTest{"64", []byte("\x40"), nil},
		encoderTest{"127", []byte("\x7F"), nil},
		encoderTest{"128", zero, errStrInvalid(typ, "128")},
		encoderTest{"-129", zero, errStrInvalid(typ, "-129")},
		encoderTest{"dog", zero, errStrInvalid(typ, "dog")},
		encoderTest{"", zero, errStrInvalid(typ, "")},
		encoderTest{" ", zero, errStrInvalid(typ, " ")},
	}
	runEncoderTests(t, tests, func(atom *Atom, input interface{}) error {
		return SetSI08FromString(atom, input.(string))
	})
}
func TestSetSI08FromInt64(t *testing.T) {
	typ := "SI08"
	zero := make([]byte, 1)
	tests := []encoderTest{
		encoderTest{int64(-128), []byte("\x80"), nil},
		encoderTest{int64(-1), []byte("\xFF"), nil},
		encoderTest{int64(0), []byte("\x00"), nil},
		encoderTest{int64(64), []byte("\x40"), nil},
		encoderTest{int64(127), []byte("\x7F"), nil},

		encoderTest{int64(128), zero, errRange(typ, 128)},
		encoderTest{int64(-129), zero, errRange(typ, -129)},
	}
	runEncoderTests(t, tests, func(atom *Atom, input interface{}) error {
		return SetSI08FromInt64(atom, input.(int64))
	})
}
func TestSetSI16FromString(t *testing.T) {
	typ := "SI16"
	zero := make([]byte, 2)
	tests := []encoderTest{
		encoderTest{"-32769", zero, errStrInvalid(typ, "-32769")},
		encoderTest{"-32768", []byte("\x80\x00"), nil},
		encoderTest{"-255", []byte("\xFF\x01"), nil},
		encoderTest{"-1", []byte("\xFF\xFF"), nil},
		encoderTest{"0", []byte("\x00\x00"), nil},
		encoderTest{"255", []byte("\x00\xFF"), nil},
		encoderTest{"32767", []byte("\x7F\xFF"), nil},
		encoderTest{"32768", zero, errStrInvalid(typ, "32768")},
		encoderTest{"dog", zero, errStrInvalid(typ, "dog")},
		encoderTest{"", zero, errStrInvalid(typ, "")},
		encoderTest{" ", zero, errStrInvalid(typ, " ")},
	}
	runEncoderTests(t, tests, func(atom *Atom, input interface{}) error {
		return SetSI16FromString(atom, input.(string))
	})
}
func TestSetSI16FromInt64(t *testing.T) {
	typ := "SI16"
	zero := make([]byte, 2)
	tests := []encoderTest{
		encoderTest{int64(-32769), zero, errRange(typ, -32769)},
		encoderTest{int64(-32768), []byte("\x80\x00"), nil},
		encoderTest{int64(-255), []byte("\xFF\x01"), nil},
		encoderTest{int64(0), []byte("\x00\x00"), nil},
		encoderTest{int64(255), []byte("\x00\xFF"), nil},
		encoderTest{int64(32767), []byte("\x7F\xFF"), nil},
		encoderTest{int64(32768), zero, errRange(typ, 32768)},
	}
	runEncoderTests(t, tests, func(atom *Atom, input interface{}) error {
		return SetSI16FromInt64(atom, input.(int64))
	})
}
func TestSetSI32FromString(t *testing.T) {
	typ := "SI32"
	zero := make([]byte, 4)
	tests := []encoderTest{
		encoderTest{"-2147483648", []byte("\x80\x00\x00\x00"), nil},
		encoderTest{"-1", []byte("\xFF\xFF\xFF\xFF"), nil},
		encoderTest{"0", []byte("\x00\x00\x00\x00"), nil},
		encoderTest{"0x00000000", []byte("\x00\x00\x00\x00"), nil},
		encoderTest{"255", []byte("\x00\x00\x00\xFF"), nil},
		encoderTest{"0x000000FF", []byte("\x00\x00\x00\xFF"), nil},
		encoderTest{"0x0000FF00", []byte("\x00\x00\xFF\x00"), nil},
		encoderTest{"0x00FF0000", []byte("\x00\xFF\x00\x00"), nil},
		encoderTest{"2147483647", []byte("\x7F\xFF\xFF\xFF"), nil},
		encoderTest{"0xFF000000", zero, errStrInvalid(typ, "0xFF000000")},
		encoderTest{"0xFFFFFFFF", zero, errStrInvalid(typ, "0xFFFFFFFF")},
		encoderTest{"dog", zero, errStrInvalid(typ, "dog")},
		encoderTest{"", zero, errStrInvalid(typ, "")},
		encoderTest{" ", zero, errStrInvalid(typ, " ")},
	}
	runEncoderTests(t, tests, func(atom *Atom, input interface{}) error {
		return SetSI32FromString(atom, input.(string))
	})
}

func TestSetSI32FromInt64(t *testing.T) {
	typ := "SI32"
	zero := make([]byte, 4)
	tests := []encoderTest{
		encoderTest{int64(-2147483648), []byte("\x80\x00\x00\x00"), nil},
		encoderTest{int64(-1), []byte("\xFF\xFF\xFF\xFF"), nil},
		encoderTest{int64(0), []byte("\x00\x00\x00\x00"), nil},
		encoderTest{int64(255), []byte("\x00\x00\x00\xFF"), nil},
		encoderTest{int64(0x000000FF), []byte("\x00\x00\x00\xFF"), nil},
		encoderTest{int64(0x0000FF00), []byte("\x00\x00\xFF\x00"), nil},
		encoderTest{int64(0x00FF0000), []byte("\x00\xFF\x00\x00"), nil},
		encoderTest{int64(2147483647), []byte("\x7F\xFF\xFF\xFF"), nil},
		encoderTest{int64(0xFF000000), zero, errRange(typ, 0xFF000000)},
		encoderTest{int64(0xFFFFFFFF + 1), zero, errRange(typ, 0xFFFFFFFF+1)},
	}
	runEncoderTests(t, tests, func(atom *Atom, input interface{}) error {
		return SetSI32FromInt64(atom, input.(int64))
	})
}

func TestSetSI64FromString(t *testing.T) {
	typ := "SI64"
	zero := make([]byte, 8)
	tests := []encoderTest{
		encoderTest{"-9223372036854775808", []byte("\x80\x00\x00\x00\x00\x00\x00\x00"), nil},
		encoderTest{"-1", []byte("\xFF\xFF\xFF\xFF\xFF\xFF\xFF\xFF"), nil},
		encoderTest{"0", []byte("\x00\x00\x00\x00\x00\x00\x00\x00"), nil},
		encoderTest{"0x00000000", []byte("\x00\x00\x00\x00\x00\x00\x00\x00"), nil},
		encoderTest{"255", []byte("\x00\x00\x00\x00\x00\x00\x00\xFF"), nil},
		encoderTest{"0x000000FF", []byte("\x00\x00\x00\x00\x00\x00\x00\xFF"), nil},
		encoderTest{"0x0000FF00", []byte("\x00\x00\x00\x00\x00\x00\xFF\x00"), nil},
		encoderTest{"0x00FF0000", []byte("\x00\x00\x00\x00\x00\xFF\x00\x00"), nil},
		encoderTest{"2147483647", []byte("\x00\x00\x00\x00\x7F\xFF\xFF\xFF"), nil},
		encoderTest{"9223372036854775807", []byte("\x7F\xFF\xFF\xFF\xFF\xFF\xFF\xFF"), nil},
		encoderTest{"9223372036854775808", zero, errStrInvalid(typ, "9223372036854775808")},
		encoderTest{"dog", zero, errStrInvalid(typ, "dog")},
		encoderTest{"", zero, errStrInvalid(typ, "")},
		encoderTest{" ", zero, errStrInvalid(typ, " ")},
	}
	runEncoderTests(t, tests, func(atom *Atom, input interface{}) error {
		return SetSI64FromString(atom, input.(string))
	})
}
func TestSetSI64FromInt64(t *testing.T) {
	tests := []encoderTest{
		encoderTest{int64(0), []byte("\x00\x00\x00\x00\x00\x00\x00\x00"), nil},
		encoderTest{int64(1), []byte("\x00\x00\x00\x00\x00\x00\x00\x01"), nil},
		encoderTest{int64(-9223372036854775808), []byte("\x80\x00\x00\x00\x00\x00\x00\x00"), nil},
		encoderTest{int64(9223372036854775807), []byte("\x7F\xFF\xFF\xFF\xFF\xFF\xFF\xFF"), nil},
	}
	runEncoderTests(t, tests, func(atom *Atom, input interface{}) error {
		return SetSI64FromInt64(atom, input.(int64))
	})
}
func TestSetUR32FromString(t *testing.T) {
	typ := "UR32"
	zero := make([]byte, 4)
	tests := []encoderTest{
		encoderTest{"0/0", []byte("\x00\x00\x00\x00"), nil},
		encoderTest{"1/0", []byte("\x00\x01\x00\x00"), nil},
		encoderTest{"0/1", []byte("\x00\x00\x00\x01"), nil},
		encoderTest{"1/1", []byte("\x00\x01\x00\x01"), nil},
		encoderTest{"1/1 ", []byte("\x00\x01\x00\x01"), nil},
		encoderTest{"1/ 1", []byte("\x00\x01\x00\x01"), nil},
		encoderTest{" 1/1", []byte("\x00\x01\x00\x01"), nil},
		encoderTest{"65535/65535", []byte("\xFF\xFF\xFF\xFF"), nil},
		encoderTest{"65536/65535", zero, errRange(typ, "[65536 65535]")},
		encoderTest{"65535/65536", zero, errRange(typ, "[65535 65536]")},
		encoderTest{"65536/65536", zero, errRange(typ, "[65536 65536]")},
	}
	var arrInvalid = []string{"0xFF/0x00", "0x00/0xFF", "0xFF/0x00", "0xFF/0xFF",
		"1.0/1", "1/1.0", "1/1/1", "1.1/1", "-1/-1", "-1/1", "1//1", "1/-1",
		"1 /1", "1/", "/1", "dog", "1", "/", " ", "",
	}
	for _, str := range arrInvalid {
		tests = append(tests, encoderTest{str, zero, errStrInvalid(typ, str)})
	}
	runEncoderTests(t, tests, func(atom *Atom, input interface{}) error {
		return SetUR32FromString(atom, input.(string))
	})
}
func TestSetUR32FromSliceOfUint(t *testing.T) {
	typ := "UR32"
	zero := make([]byte, 4)
	tests := []encoderTest{
		encoderTest{[]uint64{0, 0}, []byte("\x00\x00\x00\x00"), nil},
		encoderTest{[]uint64{1, 0}, []byte("\x00\x01\x00\x00"), nil},
		encoderTest{[]uint64{0, 1}, []byte("\x00\x00\x00\x01"), nil},
		encoderTest{[]uint64{1, 1}, []byte("\x00\x01\x00\x01"), nil},
		encoderTest{[]uint64{65535, 65535}, []byte("\xFF\xFF\xFF\xFF"), nil},
		encoderTest{[]uint64{65536, 65535}, zero, errRange(typ, "[65536 65535]")},
		encoderTest{[]uint64{65535, 65536}, zero, errRange(typ, "[65535 65536]")},
		encoderTest{[]uint64{65536, 65536}, zero, errRange(typ, "[65536 65536]")},
	}
	runEncoderTests(t, tests, func(atom *Atom, input interface{}) error {
		return SetUR32FromSliceOfUint(atom, input.([]uint64))
	})
}

func TestSetUR64FromString(t *testing.T) {
	typ := "UR64"
	zero := make([]byte, 8)
	tests := []encoderTest{
		encoderTest{"0/0", []byte("\x00\x00\x00\x00\x00\x00\x00\x00"), nil},
		encoderTest{"1/0", []byte("\x00\x00\x00\x01\x00\x00\x00\x00"), nil},
		encoderTest{"0/1", []byte("\x00\x00\x00\x00\x00\x00\x00\x01"), nil},
		encoderTest{"1/1", []byte("\x00\x00\x00\x01\x00\x00\x00\x01"), nil},
		encoderTest{"1/1 ", []byte("\x00\x00\x00\x01\x00\x00\x00\x01"), nil},
		encoderTest{"1/ 1", []byte("\x00\x00\x00\x01\x00\x00\x00\x01"), nil},
		encoderTest{" 1/1", []byte("\x00\x00\x00\x01\x00\x00\x00\x01"), nil},
		encoderTest{"4294967295/4294967295", []byte("\xFF\xFF\xFF\xFF\xFF\xFF\xFF\xFF"), nil},
		encoderTest{"4294967297/4294967296", zero, errRange(typ, "[4294967297 4294967296]")},
		encoderTest{"4294967296/4294967297", zero, errRange(typ, "[4294967296 4294967297]")},
		encoderTest{"4294967297/4294967297", zero, errRange(typ, "[4294967297 4294967297]")},
	}
	var arrInvalid = []string{"0xFF/0x00", "0x00/0xFF", "0xFF/0x00", "0xFF/0xFF",
		"1.0/1", "1/1.0", "1/1/1", "1.1/1", "-1/-1", "-1/1", "1//1", "1/-1",
		"1 /1", "1/", "/1", "dog", "1", "/", " ", "",
	}
	for _, str := range arrInvalid {
		tests = append(tests, encoderTest{str, zero, errStrInvalid(typ, str)})
	}
	runEncoderTests(t, tests, func(atom *Atom, input interface{}) error {
		return SetUR64FromString(atom, input.(string))
	})
}
func TestSetUR64FromSliceOfUint(t *testing.T) {
	typ := "UR64"
	zero := make([]byte, 8)
	tests := []encoderTest{
		encoderTest{[]uint64{0, 0}, []byte("\x00\x00\x00\x00\x00\x00\x00\x00"), nil},
		encoderTest{[]uint64{1, 0}, []byte("\x00\x00\x00\x01\x00\x00\x00\x00"), nil},
		encoderTest{[]uint64{0, 1}, []byte("\x00\x00\x00\x00\x00\x00\x00\x01"), nil},
		encoderTest{[]uint64{1, 1}, []byte("\x00\x00\x00\x01\x00\x00\x00\x01"), nil},
		encoderTest{[]uint64{4294967295, 4294967295}, []byte("\xFF\xFF\xFF\xFF\xFF\xFF\xFF\xFF"), nil},
		encoderTest{[]uint64{4294967297, 4294967296}, zero, errRange(typ, "[4294967297 4294967296]")},
		encoderTest{[]uint64{4294967296, 4294967297}, zero, errRange(typ, "[4294967296 4294967297]")},
		encoderTest{[]uint64{4294967297, 4294967297}, zero, errRange(typ, "[4294967297 4294967297]")},
	}
	runEncoderTests(t, tests, func(atom *Atom, input interface{}) error {
		return SetUR64FromSliceOfUint(atom, input.([]uint64))
	})
}

func TestSetSR32FromString(t *testing.T) {
	typ := "SR32"
	zero := make([]byte, 4)
	tests := []encoderTest{
		encoderTest{"+0/+0", []byte("\x00\x00\x00\x00"), nil},
		encoderTest{"+1/+0", []byte("\x00\x01\x00\x00"), nil},
		encoderTest{"+0/+1", []byte("\x00\x00\x00\x01"), nil},
		encoderTest{"+1/+1", []byte("\x00\x01\x00\x01"), nil},
		encoderTest{"+0/-0", []byte("\x00\x00\x00\x00"), nil},
		encoderTest{"-1/+0", []byte("\xFF\xFF\x00\x00"), nil},
		encoderTest{"+0/-1", []byte("\xFF\xFF\xFF\xFF"), nil},
		encoderTest{"-1/+1", []byte("\xFF\xFF\x00\x01"), nil},
		encoderTest{"-0/-0", []byte("\x00\x00\x00\x00"), nil},
		encoderTest{"-1/-1", []byte("\xFF\xFE\xFF\xFF"), nil},
		encoderTest{"0/-0", []byte("\x00\x00\x00\x00"), nil},
		encoderTest{"-0/0", []byte("\x00\x00\x00\x00"), nil},
		encoderTest{" 1/1", []byte("\x00\x01\x00\x01"), nil},
		encoderTest{"1/-1", []byte("\x00\x00\xFF\xFF"), nil},
		encoderTest{"-1/1", []byte("\xFF\xFF\x00\x01"), nil},
		encoderTest{"1/1 ", []byte("\x00\x01\x00\x01"), nil},
		encoderTest{"1/ 1", []byte("\x00\x01\x00\x01"), nil},
		encoderTest{"0/0", []byte("\x00\x00\x00\x00"), nil},
		encoderTest{"0/0", []byte("\x00\x00\x00\x00"), nil},
		encoderTest{"1/0", []byte("\x00\x01\x00\x00"), nil},
		encoderTest{"0/1", []byte("\x00\x00\x00\x01"), nil},
		encoderTest{"1/1", []byte("\x00\x01\x00\x01"), nil},
		encoderTest{"32767/32767", []byte("\x7F\xFF\x7F\xFF"), nil},
		encoderTest{"32767/-32768", []byte("\x7F\xFE\x80\x00"), nil},
		encoderTest{"-32768/32767", []byte("\x80\x00\x7F\xFF"), nil},
		encoderTest{"+32767/+32767", []byte("\x7F\xFF\x7F\xFF"), nil},
		encoderTest{"+32767/-32768", []byte("\x7F\xFE\x80\x00"), nil},
		encoderTest{"-32768/-32768", []byte("\x7F\xFF\x80\x00"), nil},
		encoderTest{"32768/32767", zero, errRange(typ, "[32768 32767]")},
		encoderTest{"32767/32768", zero, errRange(typ, "[32767 32768]")},
		encoderTest{"32768/32768", zero, errRange(typ, "[32768 32768]")},
	}
	var arrInvalid = []string{"0xFF/0x00", "0x00/0xFF", "0xFF/0x00", "0xFF/0xFF",
		"1.0/1", "1/1.0", "1/1/1", "1.1/1", "1//1", "1 /1", "1/", "/1",
		"dog", "1", "/", " ", "",
	}
	for _, str := range arrInvalid {
		tests = append(tests, encoderTest{str, zero, errStrInvalid(typ, str)})
	}
	runEncoderTests(t, tests, func(atom *Atom, input interface{}) error {
		return SetSR32FromString(atom, input.(string))
	})
}
func TestSetSR32FromSliceOfInt(t *testing.T) {
	typ := "SR32"
	zero := make([]byte, 4)
	tests := []encoderTest{
		encoderTest{[]int64{-0, -0}, []byte("\x00\x00\x00\x00"), nil},
		encoderTest{[]int64{-1, -1}, []byte("\xFF\xFE\xFF\xFF"), nil},
		encoderTest{[]int64{0, -0}, []byte("\x00\x00\x00\x00"), nil},
		encoderTest{[]int64{-0, 0}, []byte("\x00\x00\x00\x00"), nil},
		encoderTest{[]int64{1, 1}, []byte("\x00\x01\x00\x01"), nil},
		encoderTest{[]int64{1, -1}, []byte("\x00\x00\xFF\xFF"), nil},
		encoderTest{[]int64{-1, 1}, []byte("\xFF\xFF\x00\x01"), nil},
		encoderTest{[]int64{1, 1}, []byte("\x00\x01\x00\x01"), nil},
		encoderTest{[]int64{1, 1}, []byte("\x00\x01\x00\x01"), nil},
		encoderTest{[]int64{0, 0}, []byte("\x00\x00\x00\x00"), nil},
		encoderTest{[]int64{0, 0}, []byte("\x00\x00\x00\x00"), nil},
		encoderTest{[]int64{1, 0}, []byte("\x00\x01\x00\x00"), nil},
		encoderTest{[]int64{0, 1}, []byte("\x00\x00\x00\x01"), nil},
		encoderTest{[]int64{1, 1}, []byte("\x00\x01\x00\x01"), nil},
		encoderTest{[]int64{32767, 32767}, []byte("\x7F\xFF\x7F\xFF"), nil},
		encoderTest{[]int64{32767, -32768}, []byte("\x7F\xFE\x80\x00"), nil},
		encoderTest{[]int64{-32768, 32767}, []byte("\x80\x00\x7F\xFF"), nil},
		encoderTest{[]int64{-32768, -32768}, []byte("\x7F\xFF\x80\x00"), nil},
		encoderTest{[]int64{32768, 32767}, zero, errRange(typ, "[32768 32767]")},
		encoderTest{[]int64{32767, 32768}, zero, errRange(typ, "[32767 32768]")},
		encoderTest{[]int64{32768, 32768}, zero, errRange(typ, "[32768 32768]")},
		encoderTest{[]int64{32767, -32769}, zero, errRange(typ, "[32767 -32769]")},
		encoderTest{[]int64{-32769, 32767}, zero, errRange(typ, "[-32769 32767]")},
		encoderTest{[]int64{-32769, -32769}, zero, errRange(typ, "[-32769 -32769]")},
	}
	runEncoderTests(t, tests, func(atom *Atom, input interface{}) error {
		return SetSR32FromSliceOfInt(atom, input.([]int64))
	})
}

func TestSetSR64FromString(t *testing.T) {
	typ := "SR64"
	zero := make([]byte, 8)
	tests := []encoderTest{
		encoderTest{"+0/+0", []byte("\x00\x00\x00\x00\x00\x00\x00\x00"), nil},
		encoderTest{"+1/+0", []byte("\x00\x00\x00\x01\x00\x00\x00\x00"), nil},
		encoderTest{"+0/+1", []byte("\x00\x00\x00\x00\x00\x00\x00\x01"), nil},
		encoderTest{"+1/+1", []byte("\x00\x00\x00\x01\x00\x00\x00\x01"), nil},
		encoderTest{"+0/-0", []byte("\x00\x00\x00\x00\x00\x00\x00\x00"), nil},
		encoderTest{"-1/+0", []byte("\xFF\xFF\xFF\xFF\x00\x00\x00\x00"), nil},
		encoderTest{"+0/-1", []byte("\xFF\xFF\xFF\xFF\xFF\xFF\xFF\xFF"), nil},
		encoderTest{"-1/+1", []byte("\xFF\xFF\xFF\xFF\x00\x00\x00\x01"), nil},
		encoderTest{"-0/-0", []byte("\x00\x00\x00\x00\x00\x00\x00\x00"), nil},
		encoderTest{"-1/-1", []byte("\xFF\xFF\xFF\xFE\xFF\xFF\xFF\xFF"), nil},
		encoderTest{"0/-0", []byte("\x00\x00\x00\x00\x00\x00\x00\x00"), nil},
		encoderTest{"-0/0", []byte("\x00\x00\x00\x00\x00\x00\x00\x00"), nil},
		encoderTest{" 1/1", []byte("\x00\x00\x00\x01\x00\x00\x00\x01"), nil},
		encoderTest{"1/-1", []byte("\x00\x00\x00\x00\xFF\xFF\xFF\xFF"), nil},
		encoderTest{"-1/1", []byte("\xFF\xFF\xFF\xFF\x00\x00\x00\x01"), nil},
		encoderTest{"1/1 ", []byte("\x00\x00\x00\x01\x00\x00\x00\x01"), nil},
		encoderTest{"1/ 1", []byte("\x00\x00\x00\x01\x00\x00\x00\x01"), nil},
		encoderTest{"0/0", []byte("\x00\x00\x00\x00\x00\x00\x00\x00"), nil},
		encoderTest{"0/0", []byte("\x00\x00\x00\x00\x00\x00\x00\x00"), nil},
		encoderTest{"1/0", []byte("\x00\x00\x00\x01\x00\x00\x00\x00"), nil},
		encoderTest{"0/1", []byte("\x00\x00\x00\x00\x00\x00\x00\x01"), nil},
		encoderTest{"1/1", []byte("\x00\x00\x00\x01\x00\x00\x00\x01"), nil},
		encoderTest{"2147483647/2147483647", []byte("\x7F\xFF\xFF\xFF\x7F\xFF\xFF\xFF"), nil},
		encoderTest{"2147483647/-2147483648", []byte("\x7F\xFF\xFF\xFE\x80\x00\x00\x00"), nil},
		encoderTest{"-2147483648/2147483647", []byte("\x80\x00\x00\x00\x7F\xFF\xFF\xFF"), nil},
		encoderTest{"+2147483647/+2147483647", []byte("\x7F\xFF\xFF\xFF\x7F\xFF\xFF\xFF"), nil},
		encoderTest{"+2147483647/-2147483648", []byte("\x7F\xFF\xFF\xFE\x80\x00\x00\x00"), nil},
		encoderTest{"-2147483648/-2147483648", []byte("\x7F\xFF\xFF\xFF\x80\x00\x00\x00"), nil},
		encoderTest{"2147483648/2147483647", zero, errRange(typ, "[2147483648 2147483647]")},
		encoderTest{"2147483647/2147483648", zero, errRange(typ, "[2147483647 2147483648]")},
		encoderTest{"2147483648/2147483648", zero, errRange(typ, "[2147483648 2147483648]")},
	}
	var arrInvalid = []string{"0xFF/0x00", "0x00/0xFF", "0xFF/0x00", "0xFF/0xFF",
		"1.0/1", "1/1.0", "1/1/1", "1.1/1", "1//1", "1 /1", "1/", "/1",
		"dog", "1", "/", " ", "",
	}
	for _, str := range arrInvalid {
		tests = append(tests, encoderTest{str, zero, errStrInvalid(typ, str)})
	}
	runEncoderTests(t, tests, func(atom *Atom, input interface{}) error {
		return SetSR64FromString(atom, input.(string))
	})
}
func TestSetSR64FromSliceOfInt(t *testing.T) {
	typ := "SR64"
	zero := make([]byte, 8)
	tests := []encoderTest{
		encoderTest{[]int64{-0, -0}, []byte("\x00\x00\x00\x00\x00\x00\x00\x00"), nil},
		encoderTest{[]int64{-1, -1}, []byte("\xFF\xFF\xFF\xFE\xFF\xFF\xFF\xFF"), nil},
		encoderTest{[]int64{0, -0}, []byte("\x00\x00\x00\x00\x00\x00\x00\x00"), nil},
		encoderTest{[]int64{-0, 0}, []byte("\x00\x00\x00\x00\x00\x00\x00\x00"), nil},
		encoderTest{[]int64{1, 1}, []byte("\x00\x00\x00\x01\x00\x00\x00\x01"), nil},
		encoderTest{[]int64{1, -1}, []byte("\x00\x00\x00\x00\xFF\xFF\xFF\xFF"), nil},
		encoderTest{[]int64{-1, 1}, []byte("\xFF\xFF\xFF\xFF\x00\x00\x00\x01"), nil},
		encoderTest{[]int64{1, 1}, []byte("\x00\x00\x00\x01\x00\x00\x00\x01"), nil},
		encoderTest{[]int64{1, 1}, []byte("\x00\x00\x00\x01\x00\x00\x00\x01"), nil},
		encoderTest{[]int64{0, 0}, []byte("\x00\x00\x00\x00\x00\x00\x00\x00"), nil},
		encoderTest{[]int64{0, 0}, []byte("\x00\x00\x00\x00\x00\x00\x00\x00"), nil},
		encoderTest{[]int64{1, 0}, []byte("\x00\x00\x00\x01\x00\x00\x00\x00"), nil},
		encoderTest{[]int64{0, 1}, []byte("\x00\x00\x00\x00\x00\x00\x00\x01"), nil},
		encoderTest{[]int64{1, 1}, []byte("\x00\x00\x00\x01\x00\x00\x00\x01"), nil},
		encoderTest{[]int64{2147483647, 2147483647}, []byte("\x7F\xFF\xFF\xFF\x7F\xFF\xFF\xFF"), nil},
		encoderTest{[]int64{2147483647, -2147483648}, []byte("\x7F\xFF\xFF\xFE\x80\x00\x00\x00"), nil},
		encoderTest{[]int64{-2147483648, 2147483647}, []byte("\x80\x00\x00\x00\x7F\xFF\xFF\xFF"), nil},
		encoderTest{[]int64{-2147483648, -2147483648}, []byte("\x7F\xFF\xFF\xFF\x80\x00\x00\x00"), nil},
		encoderTest{[]int64{2147483648, 2147483647}, zero, errRange(typ, "[2147483648 2147483647]")},
		encoderTest{[]int64{2147483647, 2147483648}, zero, errRange(typ, "[2147483647 2147483648]")},
		encoderTest{[]int64{2147483648, 2147483648}, zero, errRange(typ, "[2147483648 2147483648]")},
	}
	runEncoderTests(t, tests, func(atom *Atom, input interface{}) error {
		return SetSR64FromSliceOfInt(atom, input.([]int64))
	})
}
