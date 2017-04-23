package ade

//
// Verify that UnmarshalText successfully reads all text test files.
//
// Verify that MarshalText successfully writes atoms to text, and that the
// text matches the canonical text files.
//

import (
	"crypto/sha1"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// testWriteDebugFiles toggles this behaviour: when test fails, write expected
// output file to /tmp/ along with actual result, for easy diffing.
const testWriteDebugFiles = true

// where to write output files from failed tests, if we're doing that
const failedOutputDir = "/tmp/test-atom/"

func TestUnmarshalText(t *testing.T) {
	var err error

	for _, test := range Tests {
		// test unmarshal
		a := new(Atom)
		if err = a.UnmarshalText(test.txtBytes); err != nil {
			t.Errorf("UnmarshalText(%s): [.txt] expect no error, got %s", test.Name(), err.Error())
		}
		if test.inPath == "" {
			continue // test lacks the optional *.in file
		}
		if err = a.UnmarshalText(test.inBytes); err != nil {
			t.Errorf("UnmarshalText(%s): [.in] expect no error, got %s", test.Name(), err.Error())
		}
	}
}

func checkFailedTest(t *testing.T) {
	if t.Failed() && testWriteDebugFiles {
		fmt.Println("text_test.go: failed test results are available for inspection here: ", failedOutputDir)
	}
}

// NOTE: marshaled text output is not guaranteed to always match its input, as
// odd but valid inputs may be normalized.
func TestMarshalText(t *testing.T) {
	defer checkFailedTest(t)
	var got []byte
	var err error
	if testWriteDebugFiles {
		oldFiles := func(glob string) (out []string) { out, _ = filepath.Glob(failedOutputDir + "/*"); return }
		// Empty out the test result dir. Don't remove and recreate the dir,
		// because it's useful to have a shell open there to run tests. Don't want
		// to blow away its $PWD and force the user to cd back to the same path.
		for _, f := range oldFiles(failedOutputDir + "/*") {
			os.RemoveAll(f)
		}
	}

	// Assumes testfiles and TestAtoms have matching order
	for _, test := range Tests {
		// Test that MarshalText succeeds
		if got, err = test.atom.MarshalText(); err != nil {
			t.Errorf("MarshalText(%s): expect no error, got %s", test.Name(), err.Error())
		}

		// write files
		if len(got) != len(test.txtBytes) {
			t.Errorf("MarshalText: Text size differs from original.  Got %d, want %d:  %s", len(got), len(test.txtBytes), test.Name())
			if testWriteDebugFiles {
				writeDebugFiles(got, test.txtBytes, test.Name(), failedOutputDir, "txt")
			}
		}

		// Verify that original matches marshaled text
		gotSum := sha1.Sum(got)
		wantSum := sha1.Sum(test.txtBytes)
		if gotSum != wantSum {
			t.Errorf("MarshalText: Text output differs from original: %s", test.Name())

			if testWriteDebugFiles {
				writeDebugFiles(got, test.txtBytes, test.Name(), failedOutputDir, "txt")
			}
		}
	}
}

// writeDebugFiles is for when a test has failed and the output must be made
// available for inspection.
// Arguments are byte slices containing wanted and actual output, and a
// filename on which to base output names.
// Write the wanted and actual outputs to files in the output  dir.
func writeDebugFiles(got, want []byte, testName, outputDir string, ext string) {
	path := filepath.Join(outputDir, strings.TrimPrefix(testName, "testdata"))
	gotPath := strings.Join([]string{path, "-got.", ext}, "")
	wantPath := strings.Join([]string{path, "-want.", ext}, "")

	os.Mkdir(filepath.Dir(gotPath), 0766)
	err := ioutil.WriteFile(gotPath, got, 0666)
	if err != nil {
		fmt.Println("Failed to write output for inspection: ", err)
	}

	err = ioutil.WriteFile(wantPath, want, 0666)
	if err != nil {
		fmt.Println("Failed to write output for inspection: ", err)
	}
}
