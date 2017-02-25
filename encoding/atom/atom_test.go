// Benchmark Marshal / Unmarshal functions
package atom

import (
	"io/ioutil"
	"log"
	"path/filepath"
	"runtime"
	"testing"
)

var TestAtoms []*Atom
var TestBytes [][]byte
var TestTexts [][]byte

func init() {
	for _, f := range binTestFiles() {
		buf, err := ioutil.ReadFile(f)
		if err != nil {
			log.Fatalf(err.Error())
		}
		TestBytes = append(TestBytes, buf)
	}
	for _, f := range txtTestFiles() {
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

func txtTestFiles() (files []string) {
	_, path, _, _ := runtime.Caller(1)
	testdir := filepath.Join(filepath.Dir(path), "testdata")

	moreFiles, _ := filepath.Glob(filepath.Join(testdir, "*.txt"))
	files = append(files, moreFiles...)
	moreFiles, _ = filepath.Glob(filepath.Join(testdir, "from_grid/*.txt"))
	files = append(files, moreFiles...)

	if len(files) == 0 {
		log.Fatalf("Found no text test files")
	}
	return files
}

func binTestFiles() (files []string) {
	_, path, _, _ := runtime.Caller(1)
	testdir := filepath.Join(filepath.Dir(path), "testdata")

	moreFiles, _ := filepath.Glob(filepath.Join(testdir, "*.bin"))
	files = append(files, moreFiles...)
	moreFiles, _ = filepath.Glob(filepath.Join(testdir, "from_grid/*.bin"))
	files = append(files, moreFiles...)

	if len(files) == 0 {
		log.Fatalf("Found no binary test files")
	}
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
