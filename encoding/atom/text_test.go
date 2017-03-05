package atom

//
// Verify that UnmarshalText successfully reads all text test files.
//
// Verify that MarshalText successfully writes atoms to text, and that the
// text matches the original files.
//

import (
	"crypto/sha1"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const txtWriteDebugFiles = true
const failedOutputDir = "/tmp/test-atom/"

func TestUnmarshalText(t *testing.T) {
	var buf []byte
	var err error

	for _, f := range txtTestFiles() {

		// read file contents
		if buf, err = ioutil.ReadFile(f); err != nil {
			log.Fatalf("Unable to read test file: %s", err.Error())
		}

		// test unmarshal
		a := new(Atom)
		if err := a.UnmarshalText(buf); err != nil {
			t.Errorf("UnmarshalText(%s): expect no error, got %s", f, err.Error())
		}
	}
}

// NOTE: marshaled text output is not guaranteed to always match its input, as
// odd but valid inputs may be normalized.
// The test files use the file extension *.txt for files that can round-trip
// from text->binary->text without being altered.
// The extension *.in represents a file that is valid but will be altered
// during marshaling.
func TestMarshalText(t *testing.T) {
	var a Atom
	var got, want []byte
	var err error
	if txtWriteDebugFiles {
		os.RemoveAll(failedOutputDir)
		os.Mkdir(failedOutputDir, 0766)
		fmt.Println("failed test results are available for inspection here: ", failedOutputDir)
	}

	// Assumes testfiles and TestAtoms have matching order
	for _, f := range txtTestFiles() {
		a, TestAtoms = *TestAtoms[0], TestAtoms[1:] // shift first elt from slice

		// Test that MarshalText succeeds
		if got, err = a.MarshalText(); err != nil {
			t.Errorf("MarshalText(%s): expect no error, got %s", f, err.Error())
		}

		// Read original file
		if want, err = ioutil.ReadFile(f); err != nil {
			log.Fatalf("Unable to read test file: %s", err.Error())
		}

		// write files
		if len(got) != len(want) {
			t.Errorf("MarshalText: Text size differs from original.  Got %d, want %d:  %s", len(got), len(want), filepath.Base(f))

			if txtWriteDebugFiles {
				writeDebugFiles(got, want, f, failedOutputDir)
			}

		}

		// Verify that original matches marshaled text
		gotSum := sha1.Sum(got)
		wantSum := sha1.Sum(want)
		if gotSum != wantSum {
			t.Errorf("MarshalText: Text output differs from original: %s", filepath.Base(f))

			if txtWriteDebugFiles {
				writeDebugFiles(got, want, f, failedOutputDir)
			}
		}
	}
}

// writeDebugFiles is for when a test has failed and the output must be made
// available for inspection.
// Arguments are byte slices containing wanted and actual output, and a filename to base output names on.
// Write the wanted and actual outputs to files in the output  dir.
func writeDebugFiles(got, want []byte, filename, outputDir string) {

	base := filepath.Base(filename)
	base = filepath.Join(outputDir, base[:strings.LastIndex(base, ".")])
	gotPath := strings.Join([]string{base, "-got.txt"}, "")
	wantPath := strings.Join([]string{base, "-want.txt"}, "")

	err := ioutil.WriteFile(gotPath, got, 0666)
	if err != nil {
		fmt.Println("Failed to write output for inspection: ", err)
	}

	err = ioutil.WriteFile(wantPath, want, 0666)
	if err != nil {
		fmt.Println("Failed to write output for inspection: ", err)
	}
}
