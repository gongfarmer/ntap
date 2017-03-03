// ccat converts binary AtomContainer data into text.
package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/gongfarmer/ntap/encoding/atom"
)

var (
	FlagFilename    = flag.String("o", "", "write output to file")
	FlagOutputXML   = flag.Bool("x", false, "print atom as xml")
	FlagOutputHex   = flag.Bool("X", false, "print atom as hex string")
	FlagOutputDebug = flag.Bool("d", false, "print atoms in verbose debug format")
)

func usage() {
	fmt.Fprintln(os.Stderr, "Usage: ccat [options] [<file> ...]")
	fmt.Fprintln(os.Stderr, "       cat <file> | ccat [options]")
	fmt.Fprintln(os.Stderr, "Purpose:")
	fmt.Fprintln(os.Stderr, "       Read atoms from ADE binary container format, write them as various text formats.")
	fmt.Fprintln(os.Stderr, "       Reads input from STDIN if no filenames given.")
	fmt.Fprintln(os.Stderr, "       Input may also be a hex representation of the binary format.")
	fmt.Fprintln(os.Stderr, "Options:")
	flag.PrintDefaults()
	os.Exit(2)
}

type formatWriter func(io.Writer, *atom.Atom)

func (f formatWriter) formatter(w io.Writer) func(*atom.Atom) {
	return func(a *atom.Atom) { f(w, a) }
}

func main() {
	flag.Usage = usage
	flag.Parse()
	log.SetFlags(0)
	log.SetPrefix("ccat: ")
	if flag.NArg() == 0 && stdinIsEmpty() {
		usage()
	}

	// Read atom data
	var files = filter(os.Args[1:], func(s string) bool { return !strings.HasPrefix(s, "-") && s != *FlagFilename })

	atoms, err := ReadAtomsFromInput(files)
	if err != nil {
		log.Fatalf(err.Error())
	}

	// Make Writer for output stream
	var output io.Writer
	if "" == *FlagFilename {
		output = os.Stdout
	} else {
		output, err = os.OpenFile(*FlagFilename, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			log.Fatalf(err.Error())
		}
	}

	// Choose output format, pair with output stream
	var atomPrinterFunc func(*atom.Atom)
	if true == *FlagOutputDebug {
		atomPrinterFunc = formatWriter(printAtomDebug).formatter(output)
	} else if true == *FlagOutputHex {
		atomPrinterFunc = formatWriter(printAtomHex).formatter(output)
	} else if true == *FlagOutputXML {
		panic("XML output not implemented yet")
	} else {
		atomPrinterFunc = formatWriter(printAtomText).formatter(output)
	}

	WriteAtoms(atoms, atomPrinterFunc)
	os.Exit(0)
}

// print atoms in grossly verbose format showing atom data in hex
func printAtomDebug(w io.Writer, a *atom.Atom) {
	atoms := a.AtomList()
	for _, a := range atoms {
		fmt.Fprintf(w, "name: \"%s\"\n", a.Name)
		fmt.Fprintf(w, "type: \"%s\"\n", a.Type())
		strData, err := a.Value.String()
		if err != nil {
			log.Fatalf(err.Error())
		}
		bytesData, err := a.Value.SliceOfByte()
		if err != nil {
			log.Fatalf(err.Error())
		}
		fmt.Fprintf(w, "data: \"%s\"\n", strData) // value as string
		fmt.Fprintf(w, "      % x\n", bytesData)  // value as bytes
		fmt.Fprintln(w, "--")
	}
}

// Print atom as ADE Container Text
func printAtomText(w io.Writer, a *atom.Atom) {
	buf, err := a.MarshalText()
	if err != nil {
		log.Printf("failed to print AtomContainer: %s\n", err)
		return
	}
	fmt.Fprint(w, string(buf))
}

// Print atom as hex representation of binary-form bytes
func printAtomHex(w io.Writer, a *atom.Atom) {
	buf, err := a.MarshalBinary()
	if err != nil {
		log.Printf("failed to print AtomContainer: %s\n", err)
		return
	}
	fmt.Fprintf(w, "0x%X\n", buf)
}

func stdinIsEmpty() bool {
	stat, _ := os.Stdin.Stat()
	return (stat.Mode() & os.ModeCharDevice) != 0
}

// ReadAtomsFromInput takes in a possibly empty list of files.
// If files are provided, read each file as an ADE binary atom, returning the
// results as a slice of atomPtrs.
// If no files are provided, then attempt to read a single binary atom from STDIN.
// An empty array and nil error are returned if no input is found.
// A non-nil error is returned if invalid input is encountered.
func ReadAtomsFromInput(files []string) (atoms []*atom.Atom, err error) {
	if len(files) == 0 && stdinIsEmpty() {
		return
	}

	var buffer []byte
	var someAtoms []*atom.Atom

	if len(files) == 0 {
		// Read STDIN if no files provided
		buffer, err = ioutil.ReadAll(os.Stdin)
		if err != nil && err != io.EOF {
			return
		}

		// Convert input to atoms
		if uint32(len(buffer)) == binary.BigEndian.Uint32(buffer[0:4]) {
			someAtoms, err = atom.ReadAtomsFromBinary(bytes.NewReader(buffer))
		} else if string(buffer[0:2]) == "0x" {
			someAtoms, err = atom.ReadAtomsFromHex(bytes.NewReader(buffer))
		} else {
			log.Fatalf("STDIN length (%d) does not match encoded size(%d) , not a binary atom container.", len(buffer), binary.BigEndian.Uint32(buffer[0:4]))
		}
		if err != nil {
			log.Fatalf("failed to parse STDIN as atom container: %s", err.Error())
		}
		atoms = append(atoms, someAtoms...)
	}

	// Read each file, expecting ADE binary data
	for _, path := range files {
		buffer, err = ioutil.ReadFile(path)
		if err != nil && err != io.EOF {
			return atoms, err // no need to add filepath, it's in the error
		}

		// convert to atoms.
		if uint32(len(buffer)) == binary.BigEndian.Uint32(buffer[0:4]) {
			someAtoms, err = atom.ReadAtomsFromBinary(bytes.NewReader(buffer))
		} else if string(buffer[0:1]) == "0x" {
			someAtoms, err = atom.ReadAtomsFromHex(bytes.NewReader(buffer))
		} else {
			log.Fatalf("file size (%d) does not match encoded size(%d), this is not a binary atom container: %s", len(buffer), binary.BigEndian.Uint32(buffer[0:4]), path)
		}
		if err != nil {
			log.Fatalf("unable to parse file '%s' as a binary atom container: %s", path, err.Error())
		}

		atoms = append(atoms, someAtoms...)
	}
	return
}

// Filter array items based on test function
func filter(ss []string, testFunc func(string) bool) (out []string) {
	for _, s := range ss {
		if testFunc(s) {
			out = append(out, s)
		}
	}
	return out
}

// WriteAtoms writes each atom using the given print function, which includes
// an output stream writer and an output format.
//
// This bit of code is a separate function so it can be a target for test and
// benchmark code.  See main_test.go.
func WriteAtoms(atoms []*atom.Atom, printFunc func(*atom.Atom)) {
	for _, a := range atoms {
		printFunc(a)
	}
}
