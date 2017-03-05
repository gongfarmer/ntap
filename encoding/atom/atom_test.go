// Benchmark Marshal / Unmarshal functions
package atom

import (
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

var TestAtoms []*Atom
var TestBytes [][]byte
var TestTexts [][]byte

type fileList []string

func init() {
	arrBinFiles := binTestFiles()
	arrTxtFiles := txtTestFiles()
	baseNames := make(map[string]bool, len(arrBinFiles)+len(arrTxtFiles))

	// verify that arrTxtFiles, arrBinFiles reference the same atoms in the same order
	for _, f := range arrTxtFiles {
		baseNames[strings.TrimSuffix(f, ".txt")] = true
	}
	for _, f := range arrBinFiles {
		baseNames[strings.TrimSuffix(f, ".bin")] = true
	}
	for b := range baseNames {
		binFile := strings.Join([]string{b, "bin"}, ".")
		if !arrBinFiles.includes(binFile) {
			log.Fatalf(fmt.Sprint("unit test missing binary representation, expected to find file ", binFile))

		}
		txtFile := strings.Join([]string{b, "txt"}, ".")
		if !arrTxtFiles.includes(txtFile) {
			log.Fatalf(fmt.Sprint("unit test missing text representation, expected to find file ", txtFile))
		}
	}

	// read contents of each binary file
	for _, f := range arrBinFiles {
		buf, err := ioutil.ReadFile(f)
		if err != nil {
			log.Fatalf(err.Error())
		}
		TestBytes = append(TestBytes, buf)
	}

	// read contents of each text file
	for _, f := range arrTxtFiles {
		buf, err := ioutil.ReadFile(f)
		if err != nil {
			log.Fatalf(err.Error())
		}
		TestTexts = append(TestTexts, buf)
	}

	TestAtoms = ReadAtomsFromBinaryFiles(TestBytes)
	if len(TestAtoms) == 0 {
		log.Fatalf("No test atoms available")
	}
}

func (list fileList) includes(target string) bool {
	for _, s := range list {
		if s == target {
			return true
		}
	}
	return false
}

func BenchmarkMarshalBinary(b *testing.B) {
	for n := 0; n < b.N; n++ {
		for _, a := range TestAtoms {
			if _, err := a.MarshalBinary(); err != nil {
				panic(err)
			}
		}
	}
	b.ReportAllocs()
}
func BenchmarkUnmarshalBinary(b *testing.B) {
	var a = new(Atom)
	for n := 0; n < b.N; n++ {
		for _, buf := range TestBytes {
			if err := a.UnmarshalBinary(buf); err != nil {
				panic(err)
			}
		}
	}
	b.ReportAllocs()
}
func BenchmarkMarshalText(b *testing.B) {
	for n := 0; n < b.N; n++ {
		for _, a := range TestAtoms {
			if _, err := a.MarshalText(); err != nil {
				panic(err)
			}
		}
	}
	b.ReportAllocs()
}
func BenchmarkUnmarshalText(b *testing.B) {
	var a = new(Atom)
	for n := 0; n < b.N; n++ {
		for _, buf := range TestTexts {
			if err := a.UnmarshalText(buf); err != nil {
				panic(err)
			}
		}
	}
	b.ReportAllocs()
}

func txtTestFiles() (files fileList) {
	_, path, _, _ := runtime.Caller(1) // path to this source file
	testdir := filepath.Join(filepath.Dir(path), "testdata")

	moreFiles, _ := filepath.Glob(filepath.Join(testdir, "*.txt"))
	files = append(files, moreFiles...)
	moreFiles, _ = filepath.Glob(filepath.Join(testdir, "from_grid/*.txt"))
	files = append(files, moreFiles...)

	if len(files) == 0 {
		log.Fatalf("Found no text test files")
	}
	sort.Slice(files, func(i, j int) bool { return strings.Compare(files[i], files[j]) == -1 })
	return
}

func binTestFiles() (files fileList) {
	_, path, _, _ := runtime.Caller(1)
	testdir := filepath.Join(filepath.Dir(path), "testdata")

	moreFiles, _ := filepath.Glob(filepath.Join(testdir, "*.bin"))
	files = append(files, moreFiles...)
	moreFiles, _ = filepath.Glob(filepath.Join(testdir, "from_grid/*.bin"))
	files = append(files, moreFiles...)

	if len(files) == 0 {
		log.Fatalf("Found no binary test files")
	}
	sort.Slice(files, func(i, j int) bool { return strings.Compare(files[i], files[j]) == -1 })
	return
}

func ReadAtomsFromBinaryFiles(atomBytes [][]byte) (atoms []*Atom) {
	for _, buf := range atomBytes {
		a := new(Atom)
		if err := a.UnmarshalBinary(buf); err != nil {
			panic(err)
		}
		atoms = append(atoms, a)
	}
	return
}
func ReadAtomsFromTextFiles(texts [][]byte) (atoms []*Atom) {
	for _, buf := range texts {
		a := new(Atom)
		if err := a.UnmarshalText(buf); err != nil {
			panic(err)
		}
		atoms = append(atoms, a)
	}
	return
}
