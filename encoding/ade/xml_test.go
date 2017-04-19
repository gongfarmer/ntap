package ade

import (
	"crypto/sha1"
	"fmt"
	"os"
	"testing"
)

// test hardcoded atom value here to xml that is not a complete doc
func TestMarshalXML(t *testing.T) {
	a := TestAtomGINF
	var buf []byte
	var e error
	if buf, e = AtomToXMLDocumentText(a); e != nil {
		t.Errorf("Failed to convert atom to XML")
		return
	}

	fmt.Println(string(buf))
}

func TestAtomToXMLDocumentText(t *testing.T) {
	defer checkFailedTest(t)
	var got []byte
	var err error
	var fn = "AtomToXMLDocumentText"
	if testWriteDebugFiles {
		os.RemoveAll(failedOutputDir)
		os.Mkdir(failedOutputDir, 0766)
	}

	// Assumes testfiles and TestAtoms have matching order
	for _, test := range Tests {
		// Test that AtomToXMLDocumentText succeeds
		if got, err = AtomToXMLDocumentText(test.atom); err != nil {
			t.Errorf("%s(%s): expect no error, got %s", fn, test.Name(), err.Error())
			if testWriteDebugFiles {
				writeDebugFiles(got, test.xmlBytes, test.Name(), failedOutputDir, "xml")
			}
		}

		// write files
		if len(got) != len(test.xmlBytes) {
			t.Errorf("%s: XML size differs from original.  Got %d, want %d:  %s", fn, len(got), len(test.xmlBytes), test.Name())
			if testWriteDebugFiles {
				writeDebugFiles(got, test.xmlBytes, test.Name(), failedOutputDir, "xml")
			}
		}

		// Verify that original matches marshaled text
		gotSum := sha1.Sum(got)
		wantSum := sha1.Sum(test.xmlBytes)
		if gotSum != wantSum {
			t.Errorf("%s: XML output differs from original: %s", fn, test.Name())
		}
	}
}
