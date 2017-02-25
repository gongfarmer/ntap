package main

import (
	"io/ioutil"
	"log"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gongfarmer/ntap/encoding/atom"
)

var TestAtoms []*atom.Atom
var TestBytes [][]byte
var atomPrinterFunc = formatWriter(printAtomText).formatter(ioutil.Discard)

func init() {
	for _, f := range findTestFiles() {
		buf, err := ioutil.ReadFile(f)
		if err != nil {
			log.Fatalf(err.Error())
		}
		TestBytes = append(TestBytes, buf)
	}
}

func findTestFiles() []string {
	_, dir, _, _ := runtime.Caller(1)
	testdir := filepath.Join(dir, "../../encoding/atom/testdata/from_grid/")
	files, _ := filepath.Glob(filepath.Join(testdir, "*.bin"))
	return files
}

func BenchmarkUnmarshalBinary(b *testing.B) {
	var a = new(atom.Atom)
	for n := 0; n < b.N; n++ {
		for _, buf := range TestBytes {
			if err := a.UnmarshalBinary(buf); err != nil {
				panic(err)
			}
		}
	}
}

func BenchmarkMarshalText(b *testing.B) {
	for n := 0; n < b.N; n++ {
		for _, a := range TestAtoms {
			if _, err := a.MarshalText(); err != nil {
				panic(err)
			}
		}
	}
}

func ReadAtoms(atomBytes [][]byte) (atoms []*atom.Atom) {
	for _, buf := range atomBytes {
		a := new(atom.Atom)
		if err := a.UnmarshalBinary(buf); err != nil {
			panic(err)
		}
		atoms = append(atoms, a)
	}
	return
}
