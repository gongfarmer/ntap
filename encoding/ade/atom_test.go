// Benchmark Marshal / Unmarshal functions
package ade

import (
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/gongfarmer/ntap/encoding/ade/codec"
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

// Make some random uint32s. Use each one's FCHR32 value as an atom name, and
// test that it matches the output from NameAsUint32().
func TestNameAsUint32(t *testing.T) {

	var buf = make([]byte, 4)
	for i := 0; i <= 20; i++ {
		nameUint := rand.Uint32()
		binary.BigEndian.PutUint32(buf, nameUint)
		a, err := NewAtom(string(buf), codec.UI32, nameUint)
		if err != nil {
			panic(err)
		}

		//		a.Value.SetUint(uint64(nameUint))
		if a.NameAsUint32() != nameUint {
			t.Errorf(`TestNameAsUint32(): Atom name "%s"(%X), expected uint32 value %d, got %d`, a.Name(), a.name, nameUint, a.NameAsUint32())
		}
	}
}

// Find and return a test object where the test atom has the given name.
// Return nil if not found.
func findTest(tests []Test, name string) *Test {
	for _, t := range tests {
		if t.atom.Name() == name {
			return &t
		}
	}
	return nil
}

func TestNumChildren(t *testing.T) {
	tests := make(map[string]int)
	tests["GINF"] = 7
	tests["NENT"] = 9
	tests["BID0"] = 3
	for name, count := range tests {
		a := findTest(Tests, name).atom
		if a == nil {
			panic(fmt.Errorf("Could not find test bundle %s", name))
		}
		if a.NumChildren() != count {
			t.Errorf(`TestNumChildren(): For bundle "%s", expected result %d, got %d`, name, count, a.NumChildren())
		}
	}
}

var Bid0Text = `BID0:CONT:
	BVER:UI32:1
	BTIM:UI64:1484723701865260
	NEXT:UI64:15987198135227121664
END
`

func TestFromFile(t *testing.T) {
	tst := findTest(Tests, "BID0")
	a, err := FromFile(tst.binPath)
	buf, err := a.MarshalText()
	if err != nil {
		t.Errorf(`TestFromFile(): expected no error result from FromFile(), got %s`, err.Error())
	}
	if string(buf) != Bid0Text {
		t.Errorf(`TestFromFile(): Bundle from file "BID0" does not match expected got (%s), want(%s)`, string(buf), Bid0Text)
	}
}
