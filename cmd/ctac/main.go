// ctac converts ADE ContainerText into binary.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/gongfarmer/ntap/encoding/atom"
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: ctac <filename> <outfilename>\n")
	fmt.Fprintf(os.Stderr, "       cat <filename> | ctac <outfilename>\n")
	fmt.Fprintf(os.Stderr, "Purpose: Read atoms from ADE Container Text format, write them as binary containers.\n")
	fmt.Fprintf(os.Stderr, "         <outfilename> may be \"-\" to print binary chars as output.\n")
	//	fmt.Fprintln(os.Stderr, "Options:")
	//	flag.PrintDefaults()
	os.Exit(2)
}

func main() {
	flag.Usage = usage
	flag.Parse()
	log.SetFlags(0)
	log.SetPrefix("ctac: ")
	if flag.NArg() == 0 {
		usage()
	}

	// Set up input, output
	var args = os.Args[1:]
	var input io.Reader
	var output io.Writer
	input, args = setInput(args)
	output, args = setOutput(args)
	if len(args) != 0 {
		log.Fatalf("unused arguments: %s", strings.Join(args, " "))
	}

	// Read text frominput
	var bb bytes.Buffer
	bytesRead, err := bb.ReadFrom(input)
	if err != nil {
		log.Fatalf("unable to read input", err.Error())
	}
	if bytesRead == 0 {
		log.Fatalf("empty input")
	}

	// Convert text to atom
	var a atom.Atom
	if err = a.UnmarshalText(bb.Bytes()); err != nil {
		log.Fatalf("invalid input container: %s", err)
	}

	// Convert atom to binary
	buf, err := a.MarshalBinary()
	if err != nil {
		log.Fatalf("unable to convert container to binary: ", err)
	}

	// Write binary to file
	_, err = output.Write(buf)
	if err != nil {
		log.Fatalf("unable to write to file: ", err)
	}
}

// stdinIsEmpty returns true if there is nothing to read on STDIN
func stdinIsEmpty() bool {
	stat, _ := os.Stdin.Stat()
	return (stat.Mode() & os.ModeCharDevice) != 0
}

// Define how to get input text, based on command line arguments
func setInput(argv []string) (input io.Reader, args []string) {
	var err error
	if len(argv) == 0 {
		if stdinIsEmpty() {
			fmt.Fprintln(os.Stderr, "please provide input filename, or pipe in some text.")
			usage()
		} else {
			input = os.Stdin
		}
	} else {
		if input, err = os.Open(argv[0]); err != nil {
			log.Fatalf(err.Error())
		}
		args = argv[1:]
	}
	return
}

// Define where to send output, based on command line arguments
func setOutput(argv []string) (output io.Writer, args []string) {
	var err error
	if len(argv) == 0 {
		log.Fatalf("please provide output filename, or - to get binary on STDOUT")
	} else {
		if argv[0] == "-" {
			output = os.Stdout
		} else {
			output, err = os.OpenFile(argv[0], os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
			if err != nil {
				log.Fatalf(err.Error())
			}
		}
		args = argv[1:]
	}
	return
}
