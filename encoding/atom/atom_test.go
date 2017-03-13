// Benchmark Marshal / Unmarshal functions
package atom

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

var Tests []Test

// Test represents test data containing different representations of the
// same Atom.  Test data is from the test input files in the testdata/ directory.
// The test files all have the same name but different extension:
//  .in:  text file containing valid but non-canonical representations
//  .txt: text file containing valid, canonical, round-trippable representation
//  .bin: binary file representation
//  .xml: xml file representation
// These are all read at init() time and kept in memory to avoid disk reads
// during benchmarking.
type (
	Test struct {
		atom     *Atom  // Atom object
		inBytes  []byte // bytes from in file
		binBytes []byte // bytes from bin file
		txtBytes []byte // bytes from txt file
		xmlBytes []byte // bytes from xml file
		inPath   string // path to *.in file
		binPath  string // path to *.bin file
		txtPath  string // path to *.txt file
		xmlPath  string // path to *.xml file
	}

	PathTest struct {
		Input     string
		WantValue []string
		WantError error
	}
)

// NewTest creates a new Test object from a given base path.
// It assumes that all 4 related test files would share the same base pathname, differing only in file extensions.
// It verifies that all 4 expected files exist, if not it returns a nil
// Test and a non-nil error.
func NewTest(basePath string) (t Test, err error) {
	t = Test{
		inPath:  strings.Join([]string{basePath, "in"}, "."),
		binPath: strings.Join([]string{basePath, "bin"}, "."),
		txtPath: strings.Join([]string{basePath, "txt"}, "."),
		xmlPath: strings.Join([]string{basePath, "xml"}, "."),
	}

	// verify that required files exist
	missing := []string{}
	for _, f := range t.Files() {
		if _, err = os.Stat(f); os.IsNotExist(err) {
			if strings.HasSuffix(f, ".in") { // *.in file is optional
				t.inPath = "" // hint to clients to ignore .in for this test
			} else {
				missing = append(missing, filepath.Ext(f))
			}
		}
		if len(missing) > 0 {
			msg := fmt.Sprintf("incomplete Test \"%s\" is missing", t.Name())
			err = fmt.Errorf("%s %s representations", msg, strings.Join(missing, ","))
			return Test{}, err
		}
	}

	// Read in test data and create Atom object

	// read *.in bytes
	if t.inPath != "" {
		t.inBytes, err = ioutil.ReadFile(t.inPath)
		if err != nil {
			panic(err.Error())
		}
	}
	// read text bytes
	t.txtBytes, err = ioutil.ReadFile(t.txtPath)
	if err != nil {
		panic(err.Error())
	}
	// read xml bytes
	t.xmlBytes, err = ioutil.ReadFile(t.xmlPath)
	if err != nil {
		panic(err.Error())
	}
	// read binary bytes
	t.binBytes, err = ioutil.ReadFile(t.binPath)
	if err != nil {
		panic(err.Error())
	}
	// make atom object
	t.atom = new(Atom)
	if err := t.atom.UnmarshalBinary(t.binBytes); err != nil {
		panic(err.Error())
	}

	return
}

// Files returns a list of files in this test file set as a slice.
func (t Test) Files() []string {
	return []string{
		t.inPath,
		t.binPath,
		t.txtPath,
		t.xmlPath,
	}
}

// Name returns the base name that is common to all files in the test.
// The basename is shorted by stripping everything in the absolute path that
// precedes "testdata".
func (t Test) Name() string {
	iTestdata := strings.LastIndex(t.binPath, "testdata/")
	iDot := strings.LastIndex(t.binPath, ".")
	return t.binPath[iTestdata:iDot]
}

// Create slice of Test objects from testdata dir contents
func init() {
	// Find all test files under the test root
	_, path, _, _ := runtime.Caller(1)
	testroot := filepath.Join(filepath.Dir(path), "testdata")
	testFileExt := map[string]bool{
		".in":  true,
		".bin": true,
		".txt": true,
		".xml": true,
	}
	testNames := make(map[string]bool) // map prevents duplicate test names
	filepath.Walk(testroot,
		func(path string, info os.FileInfo, _ error) error {
			if info.IsDir() && filepath.Base(path) == "invalid" {
				return filepath.SkipDir
			}
			if info.IsDir() || !testFileExt[filepath.Ext(path)] {
				return nil
			}
			testNames[strings.TrimSuffix(path, filepath.Ext(path))] = true
			return nil
		})

	// Build master test list from path list
	Tests = make([]Test, 0, len(testNames))
	for basepath := range testNames {
		t, err := NewTest(basepath)
		if err != nil {
			panic(err.Error())
		}
		Tests = append(Tests, t)
	}
}

func BenchmarkMarshalBinary(b *testing.B) {
	for n := 0; n < b.N; n++ {
		for _, t := range Tests {
			if _, err := t.atom.MarshalBinary(); err != nil {
				panic(err)
			}
		}
	}
	b.ReportAllocs()
}
func BenchmarkUnmarshalBinary(b *testing.B) {
	for n := 0; n < b.N; n++ {
		for _, t := range Tests {
			a := new(Atom)
			if err := a.UnmarshalBinary(t.binBytes); err != nil {
				panic(err)
			}
		}
	}
	b.ReportAllocs()
}
func BenchmarkMarshalText(b *testing.B) {
	for n := 0; n < b.N; n++ {
		for _, t := range Tests {
			if _, err := t.atom.MarshalText(); err != nil {
				panic(err)
			}
		}
	}
	b.ReportAllocs()
}
func BenchmarkUnmarshalText(b *testing.B) {
	for n := 0; n < b.N; n++ {
		for _, t := range Tests {
			a := new(Atom)
			if err := a.UnmarshalText(t.txtBytes); err != nil {
				panic(err)
			}
		}
	}
	b.ReportAllocs()
}

// Tests of atom path matching
// TODO:
// test case where target name appears earlier in the path too
var TestAtom = new(Atom)

func init() {
	TestAtom.UnmarshalText([]byte(`
ROOT:CONT:
  CN1A:CONT:
		DOGS:UI32:1
    CN2A:CONT:
      CN3A:CONT:
        CN4A:CONT:
          LF5A:UI32:1
          LF5B:CSTR:"hello from depth 5"
        END
        LF4B:CSTR:"hello from depth 4"
      END
    END
  END
  CN1B:CONT:
		DOGS:UI32:2
    NODE:CONT:
      NODE:CONT:
        NODE:CONT:
          NODE:CONT:
            NODE:CONT:
              NODE:CONT:
                NODE:USTR:"branch1 result"
              END
            END
          END
          NODE:CONT:
            NODE:CONT:
              NODE:CONT:
                NODE:USTR:"branch2 result"
              END
            END
          END
          NODE:CONT:
            NODE:CONT:
              NODE:CONT:
                NODE:USTR:"branch3 result"
              END
            END
          END
          NODE:USTR:"too much NODE"
        END
      END
    END
  END
  CN1C:CONT:
    DOGS:UI32:3
  END
	GINF:CONT:
		BVER:UI32:4
		BTIM:UI64:1484723582627327
		GIDV:CONT:
			AVER:UI32:2
			ATIM:UI64:1
			AVTP:FC32:'UI32'
			APER:FC32:'READ'
			AVAL:CONT:
				0x00000000:UI32:2
				0x00000001:UI32:908767
			END
		END
		GPVD:CONT:
			AVER:UI32:2
			ATIM:UI64:1
			AVTP:FC32:'UI64'
			APER:FC32:'READ'
			AVAL:CONT:
				0x00000000:UI32:2
				0x00000001:UI64:1484722540084888
			END
		END
		GVND:CONT:
			AVER:UI32:2
			ATIM:UI64:1
			AVTP:FC32:'CSTR'
			APER:FC32:'READ'
			AVAL:CONT:
				0x00000000:UI32:2
				0x00000001:CSTR:"{OID='2.16.124.113590.3.1.3.3.1'}"
			END
		END
		GSIV:CONT:
			AVER:UI32:2
			ATIM:UI64:1
			AVTP:FC32:'CSTR'
			APER:FC32:'READ'
			AVAL:CONT:
				0x00000000:UI32:2
				0x00000001:CSTR:"10.4.0"
			END
		END
	END
END
`))
}

func TestAtomsAtPath(t *testing.T) {
	zero := []string{}
	tests := []PathTest{
		PathTest{"CN1A/CN2A/CN3A/CN4A/LF5A",
			[]string{"LF5A:UI32:1"}, nil},
		PathTest{"CN1A/CN2A/CN3A/LF4B",
			[]string{`LF4B:CSTR:"hello from depth 4"`}, nil},
		PathTest{"CN1A/CN2A/CN3A/CN4A/LF5B",
			[]string{`LF5B:CSTR:"hello from depth 5"`}, nil},
		PathTest{"CN1B/NODE/NODE/NODE/NODE/NODE/NODE/NODE", []string{
			`NODE:USTR:"branch1 result"`,
			`NODE:USTR:"branch2 result"`,
			`NODE:USTR:"branch3 result"`}, nil,
		},
		PathTest{"*/DOGS", []string{
			`DOGS:UI32:1`,
			`DOGS:UI32:2`,
			`DOGS:UI32:3`}, nil,
		},
		PathTest{"GINF/*/AVAL/0x00000001", []string{
			`0x00000001:UI32:908767`,
			`0x00000001:UI64:1484722540084888`,
			`0x00000001:CSTR:"{OID='2.16.124.113590.3.1.3.3.1'}"`,
			`0x00000001:CSTR:"10.4.0"`}, nil,
		},
		PathTest{"GINF/*/AVAL/*", []string{
			`0x00000000:UI32:2`,
			`0x00000001:UI32:908767`,
			`0x00000000:UI32:2`,
			`0x00000001:UI64:1484722540084888`,
			`0x00000000:UI32:2`,
			`0x00000001:CSTR:"{OID='2.16.124.113590.3.1.3.3.1'}"`,
			`0x00000000:UI32:2`,
			`0x00000001:CSTR:"10.4.0"`}, nil,
		},

		PathTest{"THER/E IS/NOTH/INGH/ERE.", zero, fmt.Errorf("atom 'ROOT' has no child named 'THER'")},
		PathTest{"CN1A/CN2A/CN3A/LF4B/LEAF", zero, fmt.Errorf("atom 'ROOT/CN1A/CN2A/CN3A' has no container child named 'LF4B'")},
		PathTest{"CN1A/NONE", zero, fmt.Errorf("atom 'ROOT/CN1A' has no child named 'NONE'")},
	}
	runPathTests(t, tests)
}
func runPathTests(t *testing.T, tests []PathTest) {
	for _, test := range tests {
		atoms, gotErr := TestAtom.AtomsAtPath(test.Input)

		// check for expected error result
		switch {
		case gotErr == nil && test.WantError == nil:
		case gotErr != nil && test.WantError == nil:
			t.Errorf("%s: got err {%s}, want err <nil>", test.Input, gotErr)
		case gotErr == nil && test.WantError != nil:
			t.Errorf("%s: got err <nil>, want err {%s}", test.Input, test.WantError)
		case gotErr.Error() != test.WantError.Error():
			t.Errorf("%s: got err {%s}, want err {%s}", test.Input, gotErr, test.WantError)
		}

		// convert result atoms to string representations
		var results []string
		for _, a := range atoms {
			results = append(results, strings.TrimSpace(a.String()))
		}

		// compare each result atom in order
		for i, want := range test.WantValue {
			if want != results[i] {
				t.Errorf("%s: got %s, want %s", test.Input, results[i], want)
			}
		}
	}
}
