package ade

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var xmlGINF = strings.TrimSpace(`
<?xml version="1.0" encoding="UTF-8"?>
<containerxml version="1" xmlns="http://www.bycast.com/schemas/XML-container-1.0.0">
	<container name="GINF">
		<atom name="BVER" type="UI32" value="4"></atom>
		<atom name="BTIM" type="UI64" value="1484723582627327"></atom>
		<container name="GIDV">
			<atom name="AVER" type="UI32" value="2"></atom>
			<atom name="ATIM" type="UI64" value="1"></atom>
			<atom name="AVTP" type="FC32" value="UI32"></atom>
			<atom name="APER" type="FC32" value="READ"></atom>
			<container name="AVAL">
				<atom name="0x00000000" type="UI32" value="2"></atom>
				<atom name="0x00000001" type="UI32" value="908767"></atom>
			</container>
		</container>
		<container name="GPVD">
			<atom name="AVER" type="UI32" value="2"></atom>
			<atom name="ATIM" type="UI64" value="1"></atom>
			<atom name="AVTP" type="FC32" value="UI64"></atom>
			<atom name="APER" type="FC32" value="READ"></atom>
			<container name="AVAL">
				<atom name="0x00000000" type="UI32" value="2"></atom>
				<atom name="0x00000001" type="UI64" value="1484722540084888"></atom>
			</container>
		</container>
		<container name="GVND">
			<atom name="AVER" type="UI32" value="2"></atom>
			<atom name="ATIM" type="UI64" value="1"></atom>
			<atom name="AVTP" type="FC32" value="CSTR"></atom>
			<atom name="APER" type="FC32" value="READ"></atom>
			<container name="AVAL">
				<atom name="0x00000000" type="UI32" value="2"></atom>
				<atom name="0x00000001" type="CSTR" value="{OID=&#39;2.16.124.113590.3.1.3.3.1&#39;}"></atom>
			</container>
		</container>
		<container name="GSIV">
			<atom name="AVER" type="UI32" value="2"></atom>
			<atom name="ATIM" type="UI64" value="1"></atom>
			<atom name="AVTP" type="FC32" value="CSTR"></atom>
			<atom name="APER" type="FC32" value="READ"></atom>
			<container name="AVAL">
				<atom name="0x00000000" type="UI32" value="2"></atom>
				<atom name="0x00000001" type="CSTR" value="10.4.0"></atom>
			</container>
		</container>
	</container>
</containerxml>
`)

// test hardcoded atom value here to xml that is not a complete doc
func TestMarshalXML(t *testing.T) {
	fn := "MarshalXML"
	a := TestAtomGINF
	var buf []byte
	var e error
	if buf, e = AtomToXMLDocumentText(a); e != nil {
		t.Errorf("%s(%s): expect no error, got %s", fn, "GINF", e.Error())
		return
	} else {
		got := strings.TrimSpace(string(buf))
		want := strings.TrimSpace(xmlGINF)
		if got != want {
			t.Errorf("%s(%s): xml result does not match expected.\nGot:\n<<<%s>>>\nWant:\n<<<%s>>>", fn, "GINF", got, want)
		}
	}
}

func TestAtomToXMLDocumentText(t *testing.T) {
	defer checkFailedTest(t)
	var got []byte
	var err error
	var fn = "AtomToXMLDocumentText"
	//	if testWriteDebugFiles {
	//		os.RemoveAll(failedOutputDir)
	//		os.Mkdir(failedOutputDir, 0766)
	//	}

	// Assumes testfiles and TestAtoms have matching order
	for _, test := range Tests {
		// Test that AtomToXMLDocumentText succeeds
		if got, err = AtomToXMLDocumentText(test.atom); err != nil {
			t.Errorf("%s(%s): expect no error, got %s", fn, test.Name(), err.Error())
			if testWriteDebugFiles {
				writeDebugFiles(got, test.xmlBytes, test.Name(), failedOutputDir, "xml")
			}
		}

		if result := compareXML(got, test.xmlBytes); result != nil {
			t.Errorf("%s: generated XML does not match expected for test %s", fn, test.Name())
			break
		}
	}
}

func compareXML(xdoc1_bytes, xdoc2_bytes []byte) error {
	var filepaths = []string{
		filepath.Join(failedOutputDir, "xdoc1-raw.xml"),
		filepath.Join(failedOutputDir, "xdoc2-raw.xml"),
		filepath.Join(failedOutputDir, "xdoc1-canonical.xml"),
		filepath.Join(failedOutputDir, "xdoc2-canonical.xml"),
	}
	var err error
	//	os.Mkdir(failedOutputDir, 0766) // dir might already exist, ignore error

	// Write xml to file
	if err = ioutil.WriteFile(filepaths[0], xdoc1_bytes, 0666); err != nil {
		log.Fatalf("Unable to write XML file for comparison")
	}
	if err = ioutil.WriteFile(filepaths[1], xdoc2_bytes, 0666); err != nil {
		log.Fatalf("Unable to write XML file for comparison")
	}

	// Canonicalize xml
	_, err = runCommand(fmt.Sprintf("xmllint --c14n %s > %s", filepaths[0], filepaths[2]))
	if err != nil {
		fmt.Println("Got error ", err)
		panic("Unable to canonicalize XML")
	}
	_, err = runCommand(fmt.Sprintf("xmllint --c14n %s > %s", filepaths[1], filepaths[3]))
	if err != nil {
		fmt.Println("Got error ", err)
		panic("Unable to canonicalize XML")
	}

	_, err = runCommand(fmt.Sprintf("diff %s %s", filepaths[2], filepaths[3]))
	return err
}

func runCommand(cmdString string) (output string, err error) {
	// using bash -c allows usage of redirect operators in cmd string, and lets
	// us delegate command string splitting to bash instead of doing it here.
	cmd := exec.Command("bash", "-c", cmdString)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err = cmd.Run()
	return out.String(), err
}
