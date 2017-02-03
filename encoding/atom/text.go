package atom

import (
	"bytes"
	"fmt"
	"strings"
	"unicode/utf8"
)

// Enable reading and writing of text format ADE AtomContainers by fulfilling
// these interfaces from stdlib encoding/:

// TextMarshaler is the interface implemented by an object that can marshal
// itself into a textual form.
//
// MarshalText encodes the receiver into UTF-8-encoded text and returns the result.
//
//     type TextMarshaler interface {
//         MarshalText() (text []byte, err error)
//     }

// TextUnmarshaler is the interface implemented by an object that can unmarshal
// a textual representation of itself.
//
// UnmarshalText must be able to decode the form generated by MarshalText.
// UnmarshalText must copy the text if it wishes to retain the text after
// returning.
//
//     type TextUnmarshaler interface {
//     	   UnmarshalText(text []byte) error
//     }

/**********************************************************/
// Marshaling from Atom to text
/**********************************************************/
// Write Atom object to a byte slice in ADE ContainerText format.
func (a *Atom) MarshalText() (text []byte, err error) {
	buf := atomToTextBuffer(a, 0)
	return buf.Bytes(), err
}

func atomToTextBuffer(a *Atom, depth int) bytes.Buffer {
	var output bytes.Buffer

	// write atom name,type,data
	fmt.Fprintf(&output, "% *s%s:%s:", depth*4, "", a.Name, a.Type)
	s, err := a.Value.String()
	if err != nil {
		panic(fmt.Errorf("conversion of atom to text failed: %s", err))
	}
	fmt.Fprintln(&output, s)

	// write children
	if a.Type == CONT {
		for _, childPtr := range a.Children {
			buf := atomToTextBuffer(childPtr, depth+1)
			output.Write(buf.Bytes())
		}
		fmt.Fprintf(&output, "% *sEND\n", depth*4, "")
	}
	return output
}

/**********************************************************/
// Unmarshaling from text to Atom
/**********************************************************/

// UnmarshalText gets called on a zero-value Atom receiver, and populates it
// based on the contents of the argument string, which contains an ADE
// ContainerText reprentation with a single top-level CONT atom.
// "#" comments are not allowed within this text string.
func (a *Atom) UnmarshalText(input []byte) error {
	var err error
	var l *lexer = lex("UnmarshalText", string(input))
	for ok := true; ok; {
		var x item
		select {
		case x, ok = <-l.items:
			if !ok {
				break
			}
			fmt.Println(x)
		}
	}
	return err
}

// Lexer / parser design is based on a talk from Rob Pike.
//   https://talks.golang.org/2011/lex.slide
// That describes an early version of go standard lib text/template/parse/lex.go

// The lexer is a state machine with each state implemented as a function
// (stateFn) which takes the lexer state as an argument, and returns the next
// state function which should run.
// The lexer and parser run concurrently in separate goroutines. This is done
// for lexer/parser code separation, not for performance.
// The lexer sends tokens to the parser over a channel.

const (
	digits                       = "0123456789"
	hexDigits                    = "0123456789abcdefABCDEF"
	whitespaceChars              = "\t\r "
	eof                          = -1
	_                   itemEnum = iota
	itemAtomName                 // atom name field
	itemAtomType                 // atom type field
	itemFractionDivider          // separator ":"
	itemNumber                   // number within data field
	itemUUID                     // uuid value
	itemNULL                     // uuid value
	itemIP32                     // IP32 value (1 byte per octet IPv4)
	itemString                   // string value
	itemContainerStart           // AtomContainer start
	itemContainerEnd             // AtomContainer end
	itemFourCharCode             // FCHR32 value
	itemError                    // error occured, value is text of error
	itemEOF                      // end of input
)

var printableChars = strPrintableChars()
var alphaNumericChars = strAlphaNumeric()

// returns string of all printable chars < ascii 127, excludes whitespace
func strPrintableChars() string {
	var b []byte = make([]byte, 0, 0x7f-0x21) // ascii char values
	for c := byte(0x21); c < 0x7f; c++ {
		b = append(b, c)
	}
	return string(b)
}

// returns string of all alphanumeric chars < ascii 127
func strAlphaNumeric() string {
	var b []byte = make([]byte, 62)
	for c := '0'; c < '9'; c++ {
		b = append(b, byte(c))
	}
	for c := 'a'; c < 'z'; c++ {
		b = append(b, byte(c))
	}
	for c := 'A'; c < 'Z'; c++ {
		b = append(b, byte(c))
	}
	return string(b)
}

type (
	itemEnum int
	stateFn  func(*lexer) stateFn

	// item represents a token returned from the scanenr
	item struct {
		typ  itemEnum // type of item, such as itemAtomName/itemAtomType
		val  string   // Value, such as "23.2"
		line uint32   // line number at the start of this line
	}

	// lexer holds the state of the scanner
	lexer struct {
		name  string    // used only for error reports
		input string    // the string being scanned
		start uint32    // start position of this item
		width int       // width of last rune read from input
		items chan item // channel of scanned items
		pos   uint32    // current string offset
		line  uint32    // 1+number of newlines seen
	}
)

func (i item) String() string {
	switch {
	case i.typ == itemEOF:
		return "EOF"
	case i.typ == itemError:
		return i.val
	case len(i.val) > 40:
		return fmt.Sprintf("%.40q...", i.val)
	}
	return fmt.Sprintf("%q", i.val)
}

func lex(name, input string) *lexer {
	l := &lexer{
		name:  name,
		input: input,
		items: make(chan item),
	}
	go l.run() // Concurrently run state machine
	return l
}

// run lexes the input by executing state functions until the state is nil.
func (l *lexer) run() {
	for state := lexLine; state != nil; {
		state = state(l)
	}
	close(l.items) // No more tokens will be delivered
}

// next returns the next rune in the input.
func (l *lexer) next() (r rune) {
	if int(l.pos) >= len(l.input) {
		l.width = 0
		return eof
	}
	r, l.width = utf8.DecodeRuneInString(l.input[l.pos:])
	l.pos += uint32(l.width)
	if r == '\n' {
		l.line++
	}
	return
}

// ignore skips over the pending input before this point.
func (l *lexer) ignore() {
	l.start = l.pos
}

// backup steps back one rune.
// Can only be called once per call of next.
func (l *lexer) backup() {
	l.pos -= uint32(l.width)
}

// peek returns but does not consume the next rune in the input.
func (l *lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

// accept consumes the next rune if it's from the valid set.
func (l *lexer) accept(valid string) bool {
	if strings.IndexRune(valid, l.next()) >= 0 {
		return true
	}
	l.backup()
	return false
}

func (l *lexer) readToEndOfLine() {
	for c := l.next(); c != '\n'; c = l.next() {
	}
}

// acceptRun consumes a run of runes from the valid set.
// Returns a count of runes consumed.
func (l *lexer) acceptRun(valid string) int {
	i := 0
	for strings.IndexRune(valid, l.next()) >= 0 {
		i++
	}
	l.backup()
	return i
}

// token emitter
func (l *lexer) emit(t itemEnum) {
	l.items <- item{t, l.input[l.start:l.pos], l.line}
	l.start = l.pos
}

// chars returns a count of the chars seen in the current value
func (l *lexer) chars() int {
	return int(l.pos - l.start)
}

// Return the characters seen so far in the current value
func (l *lexer) buffer() string {
	return l.input[l.start:l.pos]
}

// first returns the first rune in the value
func (l *lexer) first() rune {
	if l.chars() == 0 {
		panic("Can't return first char from empty buffer")
	}
	return rune(l.input[0])
}

// error returns an error token and terminates the scan by passing back a nil
// pointer that will be the next state, terminating l.run.
func (l *lexer) errorf(format string, args ...interface{}) stateFn {
	l.items <- item{
		itemError,
		strings.Join([]string{
			fmt.Sprintf("error while lexing line %d: ", l.line),
			fmt.Sprintf(format, args...),
		}, ""),
		l.line,
	}
	return nil
}

func lexLine(l *lexer) stateFn {
	ok := true
	for ok {
		if l.chars() != 0 {
			s := fmt.Sprintf("Expecting empy buffer at start of line, got <<<%s>>>", l.buffer())
			panic(s)
		}
		r := l.next()
		switch {
		case isSpace(r):
			l.ignore()
		case r == eof:
			l.emit(itemEOF)
			ok = false
		case r == '#':
			l.readToEndOfLine()
			l.ignore()
		case isPrintableRune(r):
			l.backup()
			return lexAtomName
		default:
			return l.errorf("bad line start char: %q", l.buffer())
		}
	}
	// Correctly reached EOF.
	return nil // Stop the run loop
}

func lexAtomName(l *lexer) stateFn {

	// If Atom name starts with 0x, check for 8 byte hex string
	if l.accept("0") && l.accept("xX") {
		l.acceptRun(hexDigits)

		switch l.chars() {
		case 10: // got a complete hex atom name
			l.emit(itemAtomName)
			return lexAtomType
		case 4: // complete short atom name starts with 0x.  Weird, but OK.
			if l.peek() == ':' {
				l.emit(itemAtomName)
				return lexAtomType
			}
		case 2, 3: // < 2 is not possible in here
			// incomplete short atom name starts with 0x.  Weird, but OK.
		default:
			return l.errorf("badly formed atom name: %q", l.buffer())
		}
	}

	// Try to get 4 printable chars. May already have one.
	for i := l.chars(); i < 4; i++ {
		l.accept(printableChars)
	}
	if l.buffer() == "END" {
		l.emit(itemContainerEnd)
		return lexEndOfLine
	}
	if l.chars() == 4 && l.peek() == ':' {
		l.emit(itemAtomName)
		return lexAtomType
	}

	// Next char is not printable.
	l.next()
	return l.errorf("badly formed atom name: %q", l.buffer())
}

func lexAtomType(l *lexer) stateFn {
	if l.next() != ':' {
		return l.errorf("expected `:' after atom name, got `%q'", l.buffer())
	}
	l.ignore()

	// Try to get 4 printable chars.
	for i := 0; i < 4; i++ {
		l.next()
	}
	if l.chars() == 4 && l.peek() == ':' {
		atyp := l.buffer()
		l.emit(itemAtomType)
		l.next()
		l.ignore() // discard trailing colon

		switch atyp {
		case "CONT":
			l.emit(itemContainerStart)
			l.accept(":")
			return lexEndOfLine
		case "NULL":
			l.emit(itemNULL)
			l.accept(":")
			return lexEndOfLine
		case "UUID":
			return lexUUID
		case "UR32", "UR64", "SR32", "SR64":
			return lexFraction
		case "DATA", "CNCT", "Cnct":
			return lexHexData
		case "IPAD":
			return lexIPAD
		case "IP32":
			return lexIP32
		case "USTR", "CSTR":
			return lexString
		case "FC32":
			return lexFourCharCode
		default:
			return lexNumber
		}
	}

	return l.errorf("badly formed atom type: '%q'", l.buffer())
}

// example: uuid:UUID:64881431-B6DC-478E-B7EE-ED306619C797
func lexUUID(l *lexer) stateFn {
	l.acceptRun(hexDigits)
	l.accept("-")
	l.acceptRun(hexDigits)
	l.accept("-")
	l.acceptRun(hexDigits)
	l.accept("-")
	l.acceptRun(hexDigits)
	l.accept("-")
	l.acceptRun(hexDigits)
	if l.chars() == 36 { // size of well-formed UUID
		l.emit(itemUUID)
		return lexEndOfLine
	}
	return l.errorf("badly formed UUID value: `%q'", l.buffer())
}

// may be in hex
func lexIP32(l *lexer) stateFn {
	if l.accept("0") && l.accept("xX") { // Is it hex?
		l.acceptRun(hexDigits)
		if l.chars() < 3 {
			return l.errorf("badly formed IPv4 value: `%q'", l.buffer())
		}
		l.emit(itemIP32)
		return lexEndOfLine
	}
	l.acceptRun(digits)
	l.acceptRun(digits)
	l.accept(".")
	l.acceptRun(digits)
	l.accept(".")
	l.acceptRun(digits)
	l.accept(".")
	l.acceptRun(digits)
	if l.chars() > 15 || l.chars() < 7 { // min/max IPv4 string length
		return l.errorf("badly formed IPv4 value: `%q'", l.buffer())
	}
	l.emit(itemIP32)
	return lexEndOfLine
}

func lexFraction(l *lexer) stateFn {
	lexNumber(l)
	if !l.accept("/") {
		return l.errorf("fractional type lacks divider: ", l.buffer())
	}
	l.emit(itemFractionDivider)
	return lexNumber(l)
}

func lexIPAD(l *lexer) stateFn {
	if !l.accept("\"") {
		return l.errorf("IPAD type should start with double quote")
	}
	ipadChars := strings.Join([]string{hexDigits, ".:"}, "")
	l.acceptRun(ipadChars)
	l.accept("\"")
	l.emit(itemString)
	return lexEndOfLine
}

func lexHexData(l *lexer) stateFn {
	l.next()
	l.next()
	if l.buffer() != "0x" {
		return l.errorf("hex data type should start with 0x")
	}
	l.acceptRun(hexDigits)
	return lexEndOfLine
}

func lexNumber(l *lexer) stateFn {
	l.accept("+-") // Optional leading sign.

	digits := "0123456789"
	if l.accept("0") && l.accept("xX") { // Is it hex?
		digits = hexDigits
	}
	l.acceptRun(digits)
	if l.accept(".") {
		l.acceptRun(digits)
	}
	if l.accept("eE") {
		l.accept("+-")
		l.acceptRun("0123456789")
	}

	// Next thing mustn't be alphanumeric.
	if l.accept(alphaNumericChars) {
		return l.errorf("bad number syntax: %q", l.buffer())
	}
	l.emit(itemNumber)
	return lexEndOfLine
}

func lexFourCharCode(l *lexer) stateFn {
	// Read in single quote
	if l.next() != '\'' {
		fmt := "expected single quote to start four-char code value, got `%q'"
		return l.errorf(fmt, l.first())
	}

	// Read in chars
	for i := 0; i < 4; i++ {
		l.next()
	}
	if l.chars() < 4 {
		fmt := "insufficient chars for four-char code value, got `%q'"
		return l.errorf(fmt, l.buffer())
	}
	if !isPrintableString(l.buffer()) {
		fmt := "invalid chars for four-char code value, got these (shown in hex:) %X"
		return l.errorf(fmt, l.input[l.start+1:l.pos]) // skips leading single quote
	}

	// Read in single quote
	if l.next() != '\'' {
		fmt := "expected single quote to end four-char code value, got: %q"
		return l.errorf(fmt, l.buffer())
	}

	l.emit(itemFourCharCode)
	return lexEndOfLine
}

func lexEndOfLine(l *lexer) stateFn {
	l.acceptRun(whitespaceChars)
	if l.accept("\n") {
		l.ignore()
		return lexLine
	}

	fmt := "trailing characters at end of line: %q"
	return l.errorf(fmt, l.buffer())
}

func lexString(l *lexer) stateFn {
	// Read double quote
	if l.next() != '"' {
		fmt := "expected double quote to start string value, got `%q'"
		return l.errorf(fmt, l.first())
	}

	// Read in chars
	var r rune
	var done bool = false
	for !done {
		r = l.next()
		switch r {
		case '\\':
			r = l.next()
			if !strings.ContainsRune("\\\"nrx", r) {
				return l.errorf("string has invalid escape: %s", l.buffer())
			}
		case '"':
			done = true
		case '\n':
			return l.errorf("unterminated string, got %q", l.buffer())
		}
	}

	if r != '"' {
		return l.errorf("unterminated string data, got %q", l.buffer())
	}

	l.emit(itemString)
	return lexEndOfLine
}

func isSpace(r rune) bool {
	whitespaceChars := " \v\t\r\n" // tab or space
	return strings.ContainsRune(whitespaceChars, r)
}

func isAtomNameChar(r rune) bool {
	// multibyte UTF8 is allowed within strings, but not within an FC32
	if utf8.RuneLen(r) > 1 {
		return false
	}

	// check for byte value within ascii printable range
	b := byte(r)
	if b < 0x21 || b > 0x7f {
		return false
	}

	return true
}

func isAlphaNumeric(buf []byte) bool {
	for _, c := range buf {
		if !strings.ContainsRune(alphaNumericChars, rune(c)) {
			return false
		}
	}
	return true
}

func isPrintableRune(r rune) bool {
	return strings.ContainsRune(printableChars, r)
}
