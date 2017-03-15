package atom

// AtomsAtPath returns a slice of all Atom descendants at the given path.
// If no atom is found, it returns an error message that describes which path
// part doesn't exist.
//
// Requirements for Path definition wildcards:
//   * provide a way to select all attribute container data elemnts while
//     omitting the index element. (???)
//   * provide a terse syntax to use form command-line clients to search for
//     an element by name at any position in the tree.  (**/NAME)
// Path definition wildcards to borrow from XPATH:
//   * match any single path element of any type
//   ** match any number of nested path elements
//   *[1] return first child of container elt (there's no 0 elt)
//   book[last()] return last child of container elt named "book"
//   *[position()<3] return first 2 child elts of container elt named "book"
//   *[not(position()<3)] return first 2 child elts of container elt named "book"
//   *[@type=XXXX] match any element of type XXXX
//   *[@name=XXXX] match any element with name XXXX
//   *[@data<35] match any element whose numeric value < 35 (raise error on non-numeric)
//   *[not(@type!=UI32) and @data<35] boolean syntax // is != a thing??
// brackets too: @type=ui32 and (position > 1 or not(@name = 0x00000000))
// -cannot have bare square brackets, must be preceded by name or keyword.
// -stretch goal: allow wildcards within name/type/data, (eg. @type=UI?? matches types ui01,ui08,ui16,ui32,ui64)
//  [==, position(), 1]   function(2), function(0), param
//  [==, position, last()]
//  [<, position(), 3]
//  [not, <, position(), 3]
//  [not, <, position(), 3]
//  [==, @type, "XXXX"]
//  [==, @name, "XXXX"]
//  [==, @data(numeric), 35]

// TODO:
//   define boolean syntax for operators
//
// FIXME paths should be resolveable using hex or non-hex FC32 representation.
// Currently, the user-provided path is matched only against what is stored as
// the Name field, which is one or the other.

/// struct stackItem {
/// type {func,numeric,string,operator,keyword}
/// goal {position, attribute}
/// value {func, numeric, string, operator, keyword}
/// eval() {
///   func: execute if stack has enough args (can be boolean arg too)
///   numeric: do nothing
///   string: do nothing
///   operator: execute if stack has enough args
///   keyword: evaluate unary (eg. position(), @value,@type)
/// }
/// }

// parsing objective: a single func which can take in an atom and position, and
// return a bool indicating whether to keep it.
// future: some XPATH specifiers affect the result by specifying output format.
// https://en.wikipedia.org/wiki/Shunting-yard_algorithm
// this is good because it handles endless nested parens, and respects explicitly defined order of operations. XPath order of operations is defined somewhere.
import (
	"fmt"
	"strings"
)

const (
	alphabetLowerCase      = "abcdefghijklmnopqrstuvwxyz"
	itemLeftParen          = "iParenL"
	itemRightParen         = "iParenR"
	itemArithmeticOperator = "iArithmeticOp"
	itemBooleanOperator    = "iBooleanOp"
	itemComparisonOperator = "iCompareOp"
	itemOperator           = "iOperator"
	itemFunction           = "iFunction"
)

type itemList []*item

func (s *itemList) push(it *item) {
	*s = append(*s, it)
}

// remove and return the first list item
func (s *itemList) shift() (it *item) {
	if len(*s) == 0 {
		return nil
	}
	it = (*s)[0]
	*s = (*s)[1:]
	return
}

// pop an item off the stack and return it.
// Return ok=false if stack is empty.
func (s *itemList) pop() (it *item) {
	size := len(*s)
	ok := size > 0
	if !ok {
		return
	}
	it = (*s)[size-1]  // get item from stack top
	*s = (*s)[:size-2] // resize stack
	return
}

// pop the stack only if the top item has the specified type.
// Return ok=true if an item is popped.
func (s *itemList) popType(typ itemEnum) (it *item, ok bool) {
	if s.empty() || s.top().typ != typ {
		return
	}
	return s.pop(), true
}

// peek at the top item on the stack without removing it.
func (s *itemList) top() (it *item) {
	if s.empty() {
		return nil
	}
	return (*s)[len(*s)-1]
}

func (s *itemList) empty() bool {
	return len(*s) != 0
}

// filterParser is a parser for translating filter specification tokens
// into a callable boolean filter function.
// This parser's methods construct a stack of operations, then resolve the
// stack as much as possible, leaving placeholders for the atom and position
// which will be passed in.
// The path syntax uses infix notation (operators are between arguments). This
// parser implements Djikstra's shunting-yard algorithm to transform the input
// into an abstract syntax tree which is simpler to evaluate.
type filterParser struct {
	outputQueue itemList    // items ordered for evaluation
	opStack     itemList    // holds operators until their operands reach output queue
	items       <-chan item // items received from lexer
	err         error       // indicates parsing succeeded or describes what failed
}
type filterFunc func(a Atom, i int, next filterFunc) bool

func (a *Atom) AtomsAtPath(path string) (atoms []*Atom, e error) {
	var pathParts = append([]string{a.Name}, strings.Split(path, "/")...)
	return getAtomsAtPath(a, pathParts, 1)
}

// must return no atoms on error
func getAtomsAtPath(a *Atom, pathParts []string, index int) (atoms []*Atom, e error) {
	if a.Type() != CONT {
		e = fmt.Errorf("atom '%s' is not a container", strings.Join(pathParts[:index], "/"))
		return
	}

	// find all child atoms whose name matches the next path part
	var nextAtoms []*Atom
	nextAtoms, e = filterOnPathElement(a.Children, pathParts[index])

	// if this is the final path part, then return all matched atoms regardless of type
	if index == len(pathParts)-1 { // if last path part
		atoms = append(atoms, nextAtoms...)
		return
	}

	// search child atoms for the rest of the path
	var foundCont bool
	for _, child := range nextAtoms {
		if child.Type() != CONT {
			continue
		}
		foundCont = true
		if moreAtoms, err := getAtomsAtPath(child, pathParts, index+1); err == nil {
			atoms = append(atoms, moreAtoms...)
		} else {
			return atoms, err
		}
	}

	if !foundCont {
		// none of the matching children were containers, return error
		pathSoFar := strings.Join(pathParts[:index], "/")
		e = fmt.Errorf("atom '%s' has no container child named '%s'", pathSoFar, pathParts[index])
		return
	}
	return
}

func filterOnPathElement(children []*Atom, pathPart string) (nextAtoms []*Atom, e error) {
	name, filter := extractNameAndFilter(pathPart)
	fmt.Printf("%20s Extracted name and filter (%s,%s)\n", pathPart, name, filter)
	filterStringToFunc(filter)
	if name == "" {
		e = fmt.Errorf("empty name is not allowed in path specification.  Prepend a name or wildcard ('*','**').")
		return
	}
	//	fmt.Printf("got scan results: name(%s), filter(%)\n", name, filter)
	for _, child := range children {
		if name == "*" || name == "**" || child.Name == name {
			nextAtoms = append(nextAtoms, child)
		}
	}
	return
}

// extractNameAndFilter reads a single path element, and returns two strings
// containing the path element name and filter. The square brackets around the
// filter are stripped.
// Example:
// "CN1A[@name=DOGS and @type=UI32]" => "CN1A", "@name=DOGS and @type=UI32"
func extractNameAndFilter(path string) (name, filter string) {
	i_start := strings.IndexByte(path, '[')
	if i_start == -1 {
		return path, ""
	}
	i_end := strings.LastIndexByte(path, ']')
	if i_end == -1 {
		return path, ""
	}
	name = path[:i_start]
	filter = path[i_start+1 : i_end]
	return
}

// lexer - identifies tokens(aka items) in the atom path definition.
// Path lexing is done by the same lexer used for Atom Text format lexing.
// They use very different parsers though.

// filterStringToFunc converts a filter expression into a func that evaluates
// whether an atom at a given position should be filtered.
func filterStringToFunc(path string) (f filterFunc, e error) {
	var lexr = lexPath(path)
	return parseFilterTokens(lexr.items)
}

func lexPath(input string) *lexer {
	l := &lexer{
		input: input,
		items: make(chan item),
	}
	fmt.Printf(">>> start lexing %s\n", input)
	go l.run(lexFilterExpression)
	return l
}

// lexFilterExpression splits the filter into tokens.
// The filter is everything within the [].
// Example:  for path "CN1A[not(@type=CONT) and not(@name=DOGS)]",
// This function would be extracting tokens from this string:
//     not(@type=CONT) and not(@name=DOGS)
// it should find the following 13 tokens:
//     not ( @type = CONT ) and not ( @name = DOGS )
func lexFilterExpression(l *lexer) stateFn {
	ok := true
	for ok {
		if l.bufferSize() != 0 {
			s := fmt.Sprintf("expecting empty buffer at start of line, got <<<%s>>>", l.buffer())
			panic(s)
		}
		r := l.next()
		switch {
		case isSpace(r):
			l.ignore()
		case r == eof:
			fmt.Println("hit EOF")
			l.emit(itemEOF)
			ok = false
		case r == '@':
			lexAtomAttribute(l)
		case r == '"':
			lexStringInPath(l)
		case r == '(':
			l.emit(itemLeftParen)
		case r == ')':
			l.emit(itemRightParen)
		case r == '+', r == '*': // no division because / is path separator, not needed anyway
			l.emit(itemOperator)
		case strings.ContainsRune(digits, r):
			lexNumberInPath(l)
		case r == '-':
			if l.prevItemType == itemNumber {
				l.emit(itemOperator)
			}
			lexNumberInPath(l)
		case strings.ContainsRune("=<>!", r):
			lexComparisonOperator(l)
		case strings.ContainsRune(alphabetLowerCase, r):
			lexBooleanOperator(l)
		default:
			return l.errorf("invalid filter expression: %s", l.input)
		}
	}
	// correctly reached EOF.
	fmt.Printf("lexFilterExpression is finished with '%s'\n", l.input)
	return nil // stop the run loop
}

// lexAtomAttribute accepts @name, @type or @data.  The @ is already read.
func lexAtomAttribute(l *lexer) stateFn {
	if l.first() != '@' {
		panic("lexAtomAttribute called without leading attribute sigil @")
	}
	l.acceptRun(alphabetLowerCase)
	l.emit(itemFunction)
	return lexFilterExpression
}

// accept @name, @type or @data.  The @ is already read.
func lexComparisonOperator(l *lexer) stateFn {
	l.acceptRun("=<>!")
	l.emit(itemOperator)
	return lexFilterExpression
}

// accept "and", "or".
func lexBooleanOperator(l *lexer) stateFn {
	l.acceptRun(alphabetLowerCase)
	l.emit(itemOperator)
	return lexFilterExpression
}

// accept a delimited string
func lexStringInPath(l *lexer) stateFn {
	// expect double quotes
	if l.first() != '"' {
		l.backup()
		return l.errorf("strings should be delimited with double-quotes, got %s", l.input)
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
				return l.errorf("invalid escape atom path: %s", l.input)
			}
		case '"':
			done = true
		case '\n':
			l.backup()
			return l.errorf("unterminated string in atom path: %s", l.input)
		}
	}

	if r != '"' {
		return l.errorf("unterminated string in atom path: %s", l.input)
	}

	l.emit(itemString)
	return lexFilterExpression
}

func lexNumberInPath(l *lexer) stateFn {
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
		return l.errorf("invalid numeric value: %s", l.input)
	}
	l.emit(itemNumber)
	return lexFilterExpression
}

// parseFilterTokens translates stream of tokens emitted by the lexer into a
// function that can evaluate whether an atom gets filtered.
func parseFilterTokens(ch <-chan item) (f filterFunc, e error) {
	var state = filterParser{items: ch}
	state.receiveTokens()
	return state.evaluate(), state.err
}

// receiveTokens gets tokens from the lexer and sends them to the parser
// for parsing.
func (p *filterParser) receiveTokens() {
	for p.parseFilterToken(p.readItem()) {
	}
}

// read next time from item channel, and return it.
func (p *filterParser) readItem() (it item) {
	var ok bool
	select {
	case it, ok = <-p.items:
		if !ok {
			it = item{typ: itemEOF, value: "EOF"}
		}
	}
	return it
}

// errorf sets the error field in the parser, and returns false to indicate that
// parsing should stop.
func (p *filterParser) errorf(format string, args ...interface{}) bool {
	p.err = fmt.Errorf(
		strings.Join([]string{
			"parse error in path filter: ",
			fmt.Sprintf(format, args...),
		}, ""))
	return false
}

// parseFilterTokens receives tokens from the lexer in the order they are found
// in the path string, and queues them into evaluation order.
// This is an implementation of Djikstra's shunting-yard algorithm.
// https://en.wikipedia.org/wiki/Shunting-yard_algorithm
func (p *filterParser) parseFilterToken(it item) bool {
	switch it.typ {
	case itemNumber:
		p.outputQueue.push(&it)
	case itemFunction:
		p.opStack.push(&it)
	case itemOperator:
		for {
			op, ok := p.opStack.popType(itemOperator)
			if !ok {
				break
			}
			p.outputQueue.push(op)
		}
		p.opStack.push(&it)
	case itemLeftParen:
		p.opStack.push(&it)
	case itemRightParen:
		for {
			if p.opStack.empty() {
				return p.errorf("mismatched parentheses in filter expression")
			}
			if p.opStack.top().typ == itemLeftParen {
				p.opStack.pop()
				break
			}
			op := p.opStack.pop()
			p.outputQueue.push(op)
		}
		if p.opStack.top().typ == itemFunction {
			op := p.opStack.pop()
			p.outputQueue.push(op)
		}
	case itemEOF:
		for {
			op := p.opStack.pop()
			if op == nil {
				break
			}
			if op.typ == itemLeftParen || op.typ == itemRightParen {
				return p.errorf("mismatched parentheses in filter expression")
			}
			p.outputQueue.push(op)
		}
		return false
	default:
		return p.errorf("unexpected item type: %s", it.typ)
	}
	return true
}

// evaluate
func (p *filterParser) evaluate() (f filterFunc) {
	for {
		it := p.outputQueue.shift()
		if it == nil {
			break
		}
		fmt.Printf(" eval %s: %s\n", it.typ, it.value)
	}
	return
}
