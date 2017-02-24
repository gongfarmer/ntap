package main

import (
	"io/ioutil"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gongfarmer/ntap/encoding/atom"
)

var TestFiles []string
var TestAtoms []*atom.Atom
var atomPrinterFunc = formatWriter(printAtomText).formatter(ioutil.Discard)

func init() {
	TestFiles = findTestFiles()
}

func findTestFiles() []string {
	_, dir, _, _ := runtime.Caller(1)
	testdir := filepath.Join(dir, "../../encoding/atom/testdata/from_grid/")
	files, _ := filepath.Glob(filepath.Join(testdir, "*.bin"))
	return files
}

func BenchmarkReadAtomsFromInput(b *testing.B) {
	for n := 0; n < b.N; n++ {
		TestAtoms, _ = ReadAtomsFromInput(TestFiles)
	}
}
func BenchmarkWriteAtoms(b *testing.B) {

	for n := 0; n < b.N; n++ {
		WriteAtoms(TestAtoms, atomPrinterFunc)
	}
}
