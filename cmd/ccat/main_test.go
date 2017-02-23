package main

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gongfarmer/ntap/encoding/atom"
)

var TestFiles []string
var TestAtoms []*atom.Atom

func init() {
	TestFiles = findTestFiles()
}

func findTestFiles() []string {
	_, dir, _, _ := runtime.Caller(1)
	testdir := filepath.Join(dir, "../../encoding/atom/testdata/from_grid/")
	files, _ := filepath.Glob(filepath.Join(testdir, "*.bin"))
	return files
}

func BenchmarkMakeAtoms(b *testing.B) {
	for n := 0; n < b.N; n++ {
		TestAtoms, _ = MakeAtoms(TestFiles)
	}
}
func BenchmarkPrintAtoms(b *testing.B) {
	for n := 0; n < b.N; n++ {
		PrintAtoms(TestAtoms)
	}
}
