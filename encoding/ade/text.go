package ade

import (
	"bytes"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/gongfarmer/ntap/encoding/ade/codec"
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

// MarshalText writes an Atom to a byte slice in ADE ContainerText format.
// it implements the encoding.TextMarshaler interface.
func (a *Atom) MarshalText() (text []byte, err error) {
	var buf bytes.Buffer
	buf, err = atomToTextBuffer(a, 0)
	if err != nil {
		return []byte{}, err
	}
	return buf.Bytes(), err
}

func atomToTextBuffer(a *Atom, depth int) (output bytes.Buffer, err error) {

	// write indentation
	for i := 0; i < depth; i++ {
		fmt.Fprintf(&output, "\t")
	}

	// write atom name,type,data
	fmt.Fprintf(&output, "%s:%s:", a.Name(), a.Type())
	s, err := a.Value.StringDelimited()
	if err != nil {
		return output, fmt.Errorf("conversion of atom to text failed for atom '%s:%s': %s", a.Name(), a.Type(), err)
	}
	fmt.Fprintln(&output, s)

	if a.typ == codec.CONT {
		// write children
		for _, childPtr := range a.children {
			buf, err := atomToTextBuffer(childPtr, depth+1)
			if err != nil {
				return output, err
			}
			output.Write(buf.Bytes())
		}

		// write END, with indentation
		for i := 0; i < depth; i++ {
			fmt.Fprintf(&output, "\t")
		}
		fmt.Fprintf(&output, "END\n")
	}
	return output, err
}

/**********************************************************
 Unmarshaling from text to Atom - Lexer
 Identifies token strings (and structure problems) in input text
**********************************************************/

// UnmarshalText sets its Atom receiver to a copy of a serialized atom given as
// an argument.
func (a *Atom) UnmarshalText(input []byte) (err error) {
	// Convert text into Atom values
	var atoms []*Atom
	var lexr = lex(string(input))
	atoms, err = parse(lexr.tokens)
	if err != nil {
		return
	}

	// Set receiver to the sole top-level AtomContainer
	switch len(atoms) {
	case 0:
		err = fmt.Errorf("no atoms found in text")
	case 1:
		a.Zero()
		*a = *atoms[0]
	default:
		err = fmt.Errorf("multiple top-level atoms (%d) found in text", len(atoms))
	}
	return
}

// Lexer / parser design is based on a talk from Rob Pike.
//   https://talks.golang.org/2011/lex.slide
// That describes an early version of go standard lib text/template/parse/lex.go

// The lexer is a state machine with each state implemented as a function
// (stateFn) which takes the lexer state as an argument, and returns the next
// state function which should run.
// The lexer and parser run concurrently in separate goroutines. This is done
// for code separation, not for performance.
// The lexer sends tokens to the parser over a channel.

const (
	digits            = "0123456789"
	numericChars      = ".0123456789"
	hexDigits         = "0123456789abcdefABCDEF"
	alphabetLowerCase = "abcdefghijklmnopqrstuvwxyz"
	alphabetUpperCase = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	whitespaceChars   = "\t\r "
	eof               = -1
	numOfADETypes     = 32
	tokenAtomName     = "tknName"     // atom name
	tokenAtomType     = "tknType"     // atom type
	tokenVinculum     = "tknVinc"     // fraction divider
	tokenNumber       = "tknNumber"   // number value
	tokenUUID         = "tknUUID"     // uuid value
	tokenNULL         = "tknNULL"     // null value
	tokenIP32         = "tknIP32"     // IPv4 address as 1 byte per octet
	tokenString       = "tknString"   // string value
	tokenContainerEnd = "tknEND"      // AtomContainer end
	tokenFC32Hex      = "tknFC32hex"  // FCHR32 value as 8 hexadecimal digits
	tokenFC32Quoted   = "tknFC32quot" // FCHR32 value as single quoted 4 chars
	tokenError        = "tknErr"      // error occurred, value is text of error
	tokenEOF          = "tknEOF"      // end of input
)

var alphaNumericChars = strAlphaNumeric()

type (
	tokenEnum string
	stateFn   func(*lexer) stateFn

	// token represents a token returned from the scanner
	token struct {
		typ   tokenEnum // type of token, such as tokenAtomName/tokenAtomType
		value string    // Value, such as "23.2"
		line  uint32    // line number at the start of this line
	}

	// lexer holds the state of the scanner
	lexer struct {
		input         string     // the string being scanned
		start         uint32     // start position of this token
		width         int        // width of last rune read from input
		tokens        chan token // channel of scanned tokens
		pos           uint32     // current string offset
		lineNumber    uint32     // 1+number of newlines seen
		prevTokenType tokenEnum  // type of previous token emitted
	}
)

func lex(input string) *lexer {
	l := &lexer{
		input:      input,
		tokens:     make(chan token),
		lineNumber: 1,
	}
	go l.run(lexLine) // Concurrently run state machine
	return l
}

// run lexes the input by executing state functions until the state is nil.
func (l *lexer) run(start stateFn) {
	for state := start; state != nil; {
		state = state(l)
	}
	close(l.tokens) // No more tokens will be delivered
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
		l.lineNumber++
	}
	return
}

// ignore skips over the pending input before this point.
func (l *lexer) ignore() {
	l.start = l.pos
}

// backup undoes the last call to next().
// Usually this means backing up the position by the byte width of the last
// consumed char. If the last call to next() did not consume a char, then do
// nothing.
func (l *lexer) backup() {
	if l.width == 0 {
		return // last next() failed to consume anything. Nothing to undo.
	}
	l.pos -= uint32(l.width)
	if l.input[l.pos] == '\n' {
		l.lineNumber--
	}
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

// acceptRun consumes a run of 0 or more runes from the valid set.
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
func (l *lexer) emit(t tokenEnum) {
	l.tokens <- token{t, l.input[l.start:l.pos], l.lineNumber}
	l.start = l.pos
	l.prevTokenType = t
}

// chars returns a count of the chars seen in the current value
func (l *lexer) bufferSize() int {
	return int(l.pos - l.start)
}

// Return the characters seen so far in the current value
func (l *lexer) buffer() string {
	return l.input[l.start:l.pos]
}

// Return the entire current line as a string
func (l *lexer) line() string {
	var iStart, iEnd uint32

	// find line end
	if l.input[l.pos] == '\n' {
		iEnd = l.pos
		l.backup() // at line end, take preceding line
	} else {
		for iEnd = l.pos; l.input[iEnd] != '\n'; iEnd++ {
		}
	}

	// find line start
	for iStart = l.pos; l.input[iStart] != '\n' && iStart != 0; iStart-- {
	}
	iStart++

	return l.input[iStart:iEnd]
}

// first returns the first rune in the value
func (l *lexer) first() (r rune) {
	if l.bufferSize() == 0 {
		l.errorf("Can't return first char from empty buffer")
		return r
	}
	r, _ = utf8.DecodeRuneInString(l.input[l.start:])
	return r
}

// error returns an error token and terminates the scan by passing back a nil
// pointer that will be the next state, terminating l.run.
func (l *lexer) errorf(format string, args ...interface{}) stateFn {
	l.tokens <- token{
		tokenError,
		strings.Join([]string{
			fmt.Sprintf(format, args...),
		}, ""),
		l.lineNumber,
	}
	return nil
}

func lexLine(l *lexer) stateFn {
	ok := true
	for ok {
		if l.bufferSize() != 0 {
			return l.errorf("expecting empty buffer at start of line, got <<<%s>>>", l.buffer())
		}
		r := l.next()
		switch {
		case isSpace(r):
			l.ignore()
		case r == eof:
			l.emit(tokenEOF)
			ok = false
		case r == '#':
			l.readToEndOfLine()
			l.ignore()
		case isPrintableRune(r):
			l.backup()
			return lexAtomName
		default:
			return l.errorf("invalid line: %s", l.line())
		}
	}
	// Correctly reached EOF.
	return nil // Stop the run loop
}

func lexAtomName(l *lexer) stateFn {

	// If Atom name starts with 0x, check for 8 byte hex string
	if l.accept("0") && l.accept("xX") {
		l.acceptRun(hexDigits)

		switch l.bufferSize() {
		case 10: // got a complete hex atom name
			l.emit(tokenAtomName)
			return lexAtomType
		case 4: // complete short atom name starts with 0x.  Weird, but OK.
			if l.peek() == ':' {
				l.emit(tokenAtomName)
				return lexAtomType
			}
		case 2, 3: // < 2 is not possible in here
			// incomplete short atom name starts with 0x.  Weird, but OK.
		default:
			return l.errorf("invalid atom name: %s", l.line())
		}
	}

	// Try to get 4 printable chars. May already have one.
	for i := l.bufferSize(); i < 4; i++ {
		l.accept(codec.PrintableChars)
	}
	if l.buffer() == "END" {
		l.emit(tokenContainerEnd)
		return lexEndOfLine
	}
	if l.bufferSize() == 4 && l.peek() == ':' {
		l.emit(tokenAtomName)
		return lexAtomType
	}

	// Next char is not printable.
	return l.errorf("invalid atom name: %s", l.line())
}

func lexAtomType(l *lexer) stateFn {
	if !l.accept(":") {
		return l.errorf("atom name should be followed by a colon: %s", l.line())
	}
	l.ignore()

	// Try to get 4 printable chars.
	for i := 0; i < 4; i++ {
		l.next()
	}
	if l.bufferSize() == 4 && l.peek() == ':' {
		atyp := l.buffer()
		l.emit(tokenAtomType)
		l.next()
		l.ignore() // discard trailing colon

		switch atyp {
		case "CONT":
			// NOTE: ade ccat accepts arbitrary chars until end of line. This won't.
			return lexNullValue
		case "NULL":
			return lexNullValue
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

	// Accept type codec.CONT even without trailing :
	if l.buffer() == "CONT" {
		l.emit(tokenAtomType)
		return lexNullValue
	}

	if l.bufferSize() == 4 {
		return l.errorf("atom type should be followed by a colon: %s", l.line())
	}

	return l.errorf("invalid atom type: %s", l.line())
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
	if l.bufferSize() == 36 { // size of well-formed UUID
		l.emit(tokenUUID)
		return lexEndOfLine
	}
	return l.errorf("invalid UUID value: %s", l.line())
}

// may be in hex
func lexIP32(l *lexer) stateFn {
	if l.accept("0") && l.accept("xX") { // Is it hex?
		l.acceptRun(hexDigits)
		if l.bufferSize() < 3 {
			return l.errorf("invalid IPv4 value: %s", l.line())
		}
		l.emit(tokenIP32)
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
	if l.bufferSize() > 15 || l.bufferSize() < 7 { // min/max IPv4 string length
		return l.errorf("invalid IPv4 value: %s", l.line())
	}
	l.emit(tokenIP32)
	return lexEndOfLine
}

func lexFraction(l *lexer) stateFn {
	lexNumber(l)
	if !l.accept("/") {
		return l.errorf("fractional type is missing seperator: %s", l.line())
	}
	l.emit(tokenVinculum)
	return lexNumber(l)
}

func lexIPAD(l *lexer) stateFn {
	if !l.accept("\"") {
		return l.errorf("IPAD type should start with double quote: %s", l.line())
	}
	ipadChars := strings.Join([]string{hexDigits, ".:"}, "")
	l.acceptRun(ipadChars)
	if !l.accept("\"") {
		return l.errorf("invalid IPAD value: %s", l.line())
	}
	l.emit(tokenString)
	return lexEndOfLine
}

func lexHexData(l *lexer) stateFn {
	if l.peek() != '\n' { // empty data section is legal
		l.next()
		l.next()
		if l.buffer() != "0x" {
			return l.errorf("hex data should start with 0x, got %s", l.line())
		}
		l.acceptRun(hexDigits)
	}
	l.emit(tokenString)
	return lexEndOfLine
}

func lexNumber(l *lexer) stateFn {
	if l.accept("+-") { // Optional leading sign.
		if l.buffer() == "+" { // discard leading +, keep leading -
			l.ignore()
		}
	}

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
		return l.errorf("invalid numeric value: %s", l.line())
	}
	l.emit(tokenNumber)
	return lexEndOfLine
}

func lexFourCharCode(l *lexer) stateFn {
	var err error
	var tokenType tokenEnum
	switch l.peek() {
	case '\'':
		tokenType = tokenFC32Quoted
		err = acceptFCHR32AsSingleQuotedString(l)
	case '0':
		tokenType = tokenFC32Hex
		err = acceptFCHR32AsHex(l)
	default:
		return l.errorf("invalid four-char code value: %s", l.line())
	}

	if err != nil {
		return l.errorf(err.Error())
	}

	l.emit(tokenType)
	return lexEndOfLine
}

// Make the lexer read characters.
// If 0x followed by 8 hex digits is found, leave it in the lexer's buffer and return nil.
// Otherwise, return error. Don't emit anything -- that's up to the caller.
func acceptFCHR32AsHex(l *lexer) error {
	// Read in initial 0x
	l.next()
	l.next()
	if l.buffer() != "0x" {
		l.backup()
		l.backup()
		return fmt.Errorf("hexadecimal four-char code value should start with 0x: %s", l.line())
	}

	l.acceptRun("0123456789ABCDEFabcdef")
	if l.bufferSize() != 10 { // 0x and 8 hex digits
		return fmt.Errorf("invalid four-char code value: %s", l.line())
	}
	return nil
}

// Make the lexer read characters.
// If a single-quoted 4-char string is found, leave it in the lexer's buffer and return nil.
// Otherwise, return error. Don't emit anything -- that's up to the caller.
func acceptFCHR32AsSingleQuotedString(l *lexer) error {
	// Read initial single quote
	if l.next() != '\'' {
		return fmt.Errorf("four-char code data should start with single-quote: %s", l.line())
	}

	// Read in chars
	for i := 0; i < 4; i++ {
		l.next()
	}
	if l.bufferSize() < 4 {
		return fmt.Errorf("four-char code data is too short: %s", l.line())
	}
	if !codec.IsPrintableString(l.buffer()) {
		return fmt.Errorf("four-char code data has invalid characters: %s", l.line())
	}

	// Check for single quote
	if l.next() != '\'' {
		return fmt.Errorf("four-char code value should end with single-quote: %s", l.line())
	}
	return nil
}

func lexEndOfLine(l *lexer) stateFn {
	l.acceptRun(whitespaceChars)
	if l.accept("\n") {
		l.ignore()
		return lexLine
	}
	return l.errorf("trailing characters at end of line: %s", l.line())
}

func lexString(l *lexer) stateFn {
	// Read double quote
	if l.next() != '"' {
		l.backup()
		return l.errorf("string data should start with double-quote: %s", l.line())
	}

	// Read in chars
	var r rune
	var done = false
	for !done {
		r = l.next()
		switch r {
		case '\\':
			r = l.next()
			if !strings.ContainsRune("\\\"nrx", r) {
				l.backup()
				return l.errorf("invalid escape in string data: %s", l.line())
			}
		case '"':
			done = true
		case '\n':
			l.backup()
			return l.errorf("unterminated string data: %s", l.line())
		}
	}

	if r != '"' {
		return l.errorf("unterminated string data: %s", l.line())
	}

	l.emit(tokenString)
	return lexEndOfLine
}

func lexNullValue(l *lexer) stateFn {
	l.emit(tokenNULL)
	l.accept(":")
	return lexEndOfLine
}

func isSpace(r rune) bool {
	whitespaceChars := " \v\t\r\n" // tab or space
	return strings.ContainsRune(whitespaceChars, r)
}

func isPrintableRune(r rune) bool {
	return strings.ContainsRune(codec.PrintableChars, r)
}

// returns string of all alphanumeric chars < ascii 127
func strAlphaNumeric() string {
	return strings.Join([]string{alphabetLowerCase, alphabetUpperCase, digits, "_"}, "")
}

/**********************************************************
 Unmarshaling from text to Atom - Parser
 Converts token strings into Atom instances, detects invalid values
**********************************************************/

type (
	parseFunc func(p *parser) parseFunc
	parser    struct {
		theAtom    *Atom        // atom currently being built
		containers atomStack    // containers kept in a stack to track hierarchy
		atoms      []*Atom      // array of output atoms
		line       uint32       // 1+number of newlines seen
		tokens     <-chan token // source of token text strings
		err        error        // indicates parsing succeeded or describes what failed
	}
	atomStack []*Atom
)

var parseType = make(map[codec.ADEType]parseFunc, numOfADETypes)

func init() {
	parseType[codec.UI01] = parseNumber
	parseType[codec.UI08] = parseNumber
	parseType[codec.UI16] = parseNumber
	parseType[codec.UI32] = parseNumber
	parseType[codec.UI64] = parseNumber
	parseType[codec.SI08] = parseNumber
	parseType[codec.SI16] = parseNumber
	parseType[codec.SI32] = parseNumber
	parseType[codec.SI64] = parseNumber
	parseType[codec.FP32] = parseNumber
	parseType[codec.FP64] = parseNumber
	parseType[codec.UF32] = parseNumber
	parseType[codec.UF64] = parseNumber
	parseType[codec.SF32] = parseNumber
	parseType[codec.SF64] = parseNumber
	parseType[codec.UR32] = parseFraction
	parseType[codec.UR64] = parseFraction
	parseType[codec.SR32] = parseFraction
	parseType[codec.SR64] = parseFraction
	parseType[codec.FC32] = parseFC32Value
	parseType[codec.IP32] = parseIP32Value
	parseType[codec.IPAD] = parseString
	parseType[codec.CSTR] = parseStringDelimited
	parseType[codec.USTR] = parseStringDelimited
	parseType[codec.UUID] = parseString
	parseType[codec.DATA] = parseString
	parseType[codec.CNCT] = parseString
	parseType[codec.Cnct] = parseString
	parseType[codec.ENUM] = parseNumber
	parseType[codec.NULL] = parseNULL
	parseType[codec.CONT] = parseNULL
}

func parse(ch <-chan token) (atoms []*Atom, err error) {
	var state = parser{tokens: ch}
	state.runParser()
	return state.atoms, state.err
}

func (p *parser) runParser() {
	for state := parseAtomName; state != nil; {
		state = state(p)
	}
}

func readToken(p *parser) (tk token) {
	var ok bool
	select {
	case tk, ok = <-p.tokens:
		if !ok {
			return token{
				typ:   tokenEOF,
				value: "EOF",
			}
		}
	}
	if tk.typ == tokenError {
		p.err = fmt.Errorf("line %d: %s", tk.line, tk.value)
	}
	p.line = tk.line
	return
}
func (p *parser) errorf(format string, args ...interface{}) parseFunc {
	p.err = fmt.Errorf(
		strings.Join([]string{
			fmt.Sprintf("parse error on line %d: ", p.line),
			fmt.Sprintf(format, args...),
		}, ""))
	return nil
}
func (ptr *atomStack) push(a *Atom) {
	*ptr = append(*ptr, a)
}
func (ptr *atomStack) pop() *Atom {
	var stack = *ptr
	size := len(stack)
	if size == 0 {
		return nil
	}
	lastAtom := stack[size-1]
	*ptr = stack[:size-1]
	return lastAtom
}
func (ptr *atomStack) empty() bool {
	return len(*ptr) == 0
}
func (ptr *atomStack) size() int {
	return len(*ptr)
}
func (ptr *atomStack) top() *Atom {
	size := len(*ptr)
	if size == 0 {
		return nil
	}
	return (*ptr)[size-1]
}

func parseAtomName(p *parser) parseFunc {
	p.theAtom = new(Atom)

	// get next token
	tk := readToken(p)
	switch tk.typ {
	case tokenAtomName: // may be hex or 4 printable chars
		if e := codec.StringToFC32Bytes(&p.theAtom.name, tk.value); e != nil {
			return p.errorf(fmt.Sprint("invalid atom name: ", tk.value))
		}
	case tokenError:
		return p.errorf(tk.value)
	case tokenEOF:
		return nil
	case tokenContainerEnd:
		return parseContainerEnd(p)
	default:
		return p.errorf("line %d: expecting atom name, got %s", tk.line, tk.typ)
	}
	return parseAtomType
}

func parseContainerEnd(p *parser) parseFunc {
	if p.containers.empty() {
		p.err = fmt.Errorf("got END but there are no open containers")
		return nil
	}
	cont := p.containers.pop()
	if p.containers.empty() {
		// on close, push parentless containers into output array
		p.atoms = append(p.atoms, cont)
	}
	return parseAtomName
}

func parseAtomType(p *parser) parseFunc {
	tk := readToken(p)
	if tk.typ == tokenError {
		return p.errorf(tk.value)
	}
	if tk.typ == tokenEOF {
		return p.errorf("end of input while parsing atom %s", p.theAtom.Name())
	}

	// verify token type
	if tk.typ != tokenAtomType {
		return p.errorf("expecting token type tokenAtomType, got %s", tk.typ)
	}
	p.theAtom.SetType(codec.ADEType(tk.value))

	// Add atom to children of parent, if any
	if p.containers.empty() { // No open containers = no parent. Add atom to output.
		if tk.value != "CONT" { // Containers get added to output when they close
			p.atoms = append(p.atoms, p.theAtom)
		}
	} else {
		// Add atom to children of currently open container
		p.containers.top().AddChild(p.theAtom)
	}

	// If container, make it the currently open container
	if p.theAtom.typ == codec.CONT {
		p.containers.push(p.theAtom)
	}

	return parseAtomData
}

func parseAtomData(p *parser) parseFunc {
	parseFunc := parseType[p.theAtom.typ]
	if parseFunc == nil {
		return p.errorf("no data parse function defined for type %s", p.theAtom.Type())
	}
	retval := parseFunc(p)
	if retval == nil { // nil function returned means error
		return nil
	}
	return parseAtomName // return next state
}

func parseNumber(p *parser) parseFunc {
	tk := readToken(p)
	if tk.typ == tokenError {
		return p.errorf(tk.value)
	}
	if tk.typ != tokenNumber {
		return p.errorf("expected atom data with type Number, got type %s", tk.typ)
	}

	err := p.theAtom.Value.SetString(tk.value)
	if err != nil {
		return p.errorf(err.Error())
	}
	return parseAtomName
}

// Read empty data section.  Consume one token, which is ignored.
func parseNULL(p *parser) parseFunc {
	readToken(p)
	return parseAtomName
}

func parseFraction(p *parser) parseFunc {
	// Take the next 3 tokens
	var tokens = make([]token, 0, 3)
	tokens = append(tokens, readToken(p)) // fraction numerator
	tokens = append(tokens, readToken(p)) // fraction separator
	tokens = append(tokens, readToken(p)) // fraction denominator
	var values = []string{tokens[0].value, tokens[1].value, tokens[2].value}

	// Verify token types are correct for fractional type data
	if !(tokens[0].typ == tokenNumber && tokens[1].typ == tokenVinculum && tokens[2].typ == tokenNumber) {
		return p.errorf("malformed fraction: %s %s %s", values[0], values[1], values[2])
	}

	// Send tokens for type conversion
	err := p.theAtom.Value.SetString(strings.Join(values, ""))
	if err != nil {
		return p.errorf(err.Error())
	}

	return parseAtomName
}

func parseFC32Value(p *parser) parseFunc {
	tk := readToken(p)
	switch tk.typ {
	case tokenFC32Hex, tokenFC32Quoted:
		p.theAtom.Value.SetString(tk.value)
	case tokenError:
		return p.errorf(tk.value)
	default:
		return p.errorf("expected atom data with type FC32, got type %s", tk.typ)
	}
	if err := p.theAtom.Value.SetString(tk.value); err != nil {
		return p.errorf("failed to set FC32 atom data (%s): %s", tk.value, err.Error())
	}
	return parseAtomName
}

func parseIP32Value(p *parser) parseFunc {
	tk := readToken(p)

	if tk.typ != tokenIP32 {
		return p.errorf("expected atom data with type IP32, got type %s", tk.typ)
	}

	err := p.theAtom.Value.SetString(tk.value)
	if err != nil {
		return p.errorf(err.Error())
	}
	return parseAtomName
}

func parseString(p *parser) parseFunc {
	tk := readToken(p)
	if tk.typ == tokenError {
		return p.errorf(tk.value)
	}

	err := p.theAtom.Value.SetString(tk.value)
	if err != nil {
		return p.errorf(err.Error())
	}
	return parseAtomName
}

func parseStringDelimited(p *parser) parseFunc {
	tk := readToken(p)
	if tk.typ == tokenError {
		return p.errorf(tk.value)
	}

	err := p.theAtom.Value.SetStringDelimited(tk.value)
	if err != nil {
		return p.errorf(err.Error())
	}
	return parseAtomName
}