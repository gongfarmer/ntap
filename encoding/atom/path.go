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
	"strconv"
	"strings"
)

const (
	itemLeftParen          = "iParenL"
	itemRightParen         = "iParenR"
	itemArithmeticOperator = "iArithmeticOp"
	itemBooleanOperator    = "iBooleanOp"
	itemComparisonOperator = "iCompareOp"
	itemOperator           = "iOperator"
	itemFunction           = "iFunction"
	itemVariable           = "iVar"
	itemInteger            = "iInt"
	itemFloat              = "iFloat"
	itemHex                = "iHex"
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
	it, *s = (*s)[0], (*s)[1:]
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
	*s = (*s)[:size-1] // resize stack
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
func (s itemList) top() (it *item) {
	if len(s) == 0 {
		return nil
	}
	return s[len(s)-1]
}

// empty returns true if the list is empty.
func (s *itemList) empty() bool {
	return len(*s) == 0
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
type filterFunc func(a Atom, i int) bool

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

	// find all child atoms whose name matches the next path element
	var nextAtoms []*Atom
	nextAtoms, e = filterOnPathElement(a.Children, pathParts[index])

	// if this is the final path element, then return all matched atoms regardless of type
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

func filterOnPathElement(children []*Atom, pathPart string) (filteredAtoms []*Atom, e error) {
	strName, strFilter := extractNameAndFilter(pathPart)
	// use the name to build up a list of candidate atoms
	if strName == "" {
		e = fmt.Errorf("empty name is not allowed in path specification. Prepend a name or wildcard.")
		return
	}
	var nextAtoms []*Atom
	for _, child := range children {
		if strName == "*" || child.Name == strName {
			nextAtoms = append(nextAtoms, child)
		}
	}

	// apply filter to determine which elements to keep
	filter, e := NewFilter(strFilter)
	fmt.Printf("NEWF \"%s\" => %s/ %v\n", pathPart, strName, filter.tokens)
	if e != nil {
		panic(e)
	}

	filteredAtoms = nextAtoms[:0] // overwrite nextAtoms in-place during filtering
	count := len(nextAtoms)
	for i, atomPtr := range nextAtoms {
		satisfied := false
		if filter.Satisfied(atomPtr, i, count) {
			filteredAtoms = append(filteredAtoms, atomPtr)
			satisfied = true
		}
		fmt.Printf(" %t => filter(%2d/%d, %s:%s) on %v\n", satisfied, i, count, atomPtr.Name, atomPtr.Type(), filter.tokens)
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
func NewFilter(path string) (ev *evaluator, err error) {
	var lexr = lexPath(path)
	ev = new(evaluator)
	ev.tokens, err = parseFilterTokens(lexr.items)
	return
}

func lexPath(input string) *lexer {
	l := &lexer{
		input: input,
		items: make(chan item),
	}
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
			s := fmt.Sprintf("expected to start with empty buffer, got <<<%s>>>", l.buffer())
			panic(s)
		}
		r := l.next()
		switch {
		case isSpace(r):
			l.ignore()
		case r == eof:
			l.emit(itemEOF)
			ok = false
		case r == '@':
			lexAtomAttribute(l)
		case r == '"', r == '\'':
			lexDelimitedString(l)
		case r == '(':
			l.emit(itemLeftParen)
		case r == ')':
			l.emit(itemRightParen)
		case r == '+', r == '*': // no division because / is path separator, not needed anyway
			l.emit(itemOperator)
		case strings.ContainsRune(digits, r):
			lexNumberInPath(l)
		case r == '-':
			if l.prevItemType == itemOperator {
				lexNumberInPath(l)
			}
			l.emit(itemOperator)
		case strings.ContainsRune("=<>!", r):
			lexComparisonOperator(l)

		case r == 'o', r == 'a', r == 'n':
			// FIXME merge this whole o/a/n case with the case below
			l.acceptRun(alphabetLowerCase)
			for _, word := range []string{"or", "and", "not"} {
				if l.buffer() == word {
					l.emit(itemOperator)
					break
				}
			}
			if l.bufferSize() == 0 {
				continue
			}
			if l.peek() == '(' {
				lexFunctionCall(l)
			} else {
				lexBareString(l)
			}
		case strings.ContainsRune(alphaNumericChars, r):
			l.acceptRun(alphabetLowerCase)
			if l.peek() == '(' {
				lexFunctionCall(l)
			} else {
				lexBareString(l)
			}
		default:
			return l.errorf("invalid filter expression: %s", l.input)
		}
	}
	// correctly reached EOF.
	return nil // stop the run loop
}

// lexAtomAttribute accepts @name, @type or @data.  The @ is already read.
func lexAtomAttribute(l *lexer) stateFn {
	if l.first() != '@' {
		panic("lexAtomAttribute called without leading attribute sigil @")
	}
	l.acceptRun(alphaNumericChars)
	l.emit(itemVariable)
	return lexFilterExpression
}

// accept @name, @type or @data.  The @ is already read.
func lexComparisonOperator(l *lexer) stateFn {
	l.acceptRun("=<>!")
	l.emit(itemOperator)
	return lexFilterExpression
}

func lexFunctionCall(l *lexer) stateFn {
	// verify all alphanumeric up to this point
	if strings.TrimLeft(l.buffer(), alphaNumericChars) != "" {
		return l.errorf("invalid function call prefix: %s", l.input)
	}
	// verify parentheses (no functions that support parameters are supported yet!)
	if !(l.accept("(") && l.accept(")")) {
		return l.errorf("invalid function call: %s", l.input)
	}
	l.emit(itemFunction)

	return lexFilterExpression
}

func lexDelimitedString(l *lexer) stateFn {
	// Find delimiter
	delim := l.first()
	if delim != '"' && delim != '\'' {
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
		case delim: // accept either delimiter
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

// lexBareString accepts a non-delimited string of alphanumeric characters.
// This has more restrictions than a delimited string but is simple and fast to
// parse.
// Doesn't handle any escaping, use delimited strings for anything non-trivial.
func lexBareString(l *lexer) stateFn {
	l.acceptRun(alphaNumericChars)
	l.emit(itemString)
	return lexFilterExpression
}

func lexNumberInPath(l *lexer) stateFn {
	var isHex, isFloatingPoint, isSigned, isExponent bool
	if l.accept("+-") { // Optional leading sign.
		isSigned = true
		if l.buffer() == "+" { // discard leading +, keep leading -
			l.ignore()
		}
	}
	digits := "0123456789"
	if l.accept("0") && l.accept("xX") { // Is it hex?
		isHex = true
		digits = hexDigits
	}
	l.acceptRun(digits)
	if l.accept(".") {
		isFloatingPoint = true
		l.acceptRun(digits)
	}
	if l.accept("eE") {
		isExponent = true
		l.accept("+-")
		l.acceptRun("0123456789")
	}

	// Next thing mustn't be alphanumeric.
	if l.accept(alphaNumericChars) {
		return l.errorf("invalid numeric value: %s", l.input)
	}
	if (isHex && isFloatingPoint) || (isHex && isExponent) || (isHex && isSigned) {
		return l.errorf("invalid numeric value: %s", l.input)
	}

	switch {
	case isFloatingPoint:
		l.emit(itemFloat)
	case isHex:
		l.emit(itemHex)
	default:
		l.emit(itemInteger)
	}
	return lexFilterExpression
}

// parseFilterTokens translates stream of tokens emitted by the lexer into a
// function that can evaluate whether an atom gets filtered.
func parseFilterTokens(ch <-chan item) (tokens itemList, e error) {
	var state = filterParser{items: ch}
	state.receiveTokens()
	return state.outputQueue, state.err
}

// receiveTokens gets tokens from the lexer and sends them to the parser
// for parsing.
func (p *filterParser) receiveTokens() {
	//for p.parseFilterToken(p.readItem()) {
	for {
		it := p.readItem()
		p.parseFilterToken(it)
		if it.typ == itemEOF {
			break
		}
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
	case itemInteger, itemHex, itemFloat, itemString, itemVariable:
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
		for !p.opStack.empty() {
			op := p.opStack.pop()
			if op.typ == itemLeftParen || op.typ == itemRightParen {
				return p.errorf("mismatched parentheses in filter expression")
			}
			p.outputQueue.push(op)
		}
		return false
	default:
		panic(fmt.Sprintf("unexpected item type: %s", it.typ))
	}
	return true
}

// evaluate
func (p *filterParser) evaluate() (f filterFunc) {
	fmt.Printf("evaluate: %s\n", p.outputQueue)
	//	for {
	//		it := p.outputQueue.shift()
	//		if it == nil {
	//			break
	//		}
	//		fmt.Printf(" eval %s: %s\n", it.typ, it.value)
	//	}
	return
}

// evaluator is for determining whether a single atom from the list of
// candidate atoms satisfies filter criteria.
//
// A path expression has two parts: /root/name[filter]
// ignoring the /root/ part, evaluation proceeds as follows:
// name:     all child atoms with name "name" are collected in a slice
// [filter]: slice items are checked against [filter] and removed if they don't pass
//
// This is used in the [filter] part. For each atom in the slice, an evaluator
// is created containing the parsed filter, the atom and information about
// where the atom appears in the slice.  Then the eval() method is called to
// parse the filter tokens using the other values in the struct.
type evaluator struct {
	tokens   itemList // filter against which to to evaluate atoms, do not alter
	Tokens   itemList // Copy of .tokens which is modified during evaluation
	AtomPtr  *Atom    // atom currently being evaluated from the atom list
	Position int      // index of the atom in the atom list
	Count    int      // number of atoms in the atom list
}

// evaluate the list of items.
func (e *evaluator) eval() (result bool) {
	for !e.Tokens.empty() {
		switch e.Tokens.top().typ {
		case itemBooleanOperator:
			result = e.evalBooleanOperator()
		case itemComparisonOperator:
			result = e.evalComparisonOperator()
		default:
			panic(fmt.Sprintf("unknown token type %s", e.Tokens.top().typ))
		}
	}
	// calculate a boolean value from op and vars
	return result
}
func (e *evaluator) evalBooleanOperator() (result bool) {
	op := e.Tokens.pop()
	if op.typ != itemBooleanOperator {
		panic(fmt.Sprintf("expected itemBooleanOperator, received type %s", op.typ))
	}
	switch op.value {
	case "not":
		result = !e.eval()
	case "and":
		result = e.eval() && e.eval()
	case "or":
		result = e.eval() || e.eval()
	default:
		panic(fmt.Sprintf("unknown boolean operator: %s", op.value))
	}
	return result
}

// Numeric operators. All have arity 2.  Must handle float and int types.  Assumed to be signed.
func (e *evaluator) evalArithmeticOperator() (result interface{}) {
	op := e.Tokens.pop()
	if op.typ != itemArithmeticOperator {
		panic(fmt.Sprintf("expected itemArithmeticOperator, received type %s", op.typ))
	}
	val1 := e.evalValue().(Arithmeticker)
	val2 := e.evalValue().(Arithmeticker)
	switch op.value {
	case "+":
		result = val1.Plus(val2)
	case "-":
		result = val1.Minus(val2)
	case "*":
		result = val1.Multiply(val2)
	case "/":
		result = val1.Divide(val2)
	default:
		panic(fmt.Sprintf("unknown arithmetic operator: %s", op.value))
	}
	return result
}
func (e *evaluator) evalComparisonOperator() bool {
	op := e.Tokens.pop()
	if op.typ != itemComparisonOperator {
		panic(fmt.Sprintf("expected itemComparisonOperator, received type %s", op.typ))
	}
	val1 := e.evalValue().(Comparer)
	val2 := e.evalValue().(Comparer)
	switch op.value {
	case "=":
		return val1.Equal(val2)
	case "!=":
		return !val1.Equal(val2)
	case "<":
		return val1.LessThan(val2)
	case ">":
		return val1.GreaterThan(val2)
	case "<=":
		return val1.LessThan(val2) || val1.Equal(val2)
	case ">=":
		return val1.GreaterThan(val2) || val1.Equal(val2)
	default:
		panic(fmt.Sprintf("unknown arithmetic operator: %s", op.value))
	}
}
func (e *evaluator) evalValue() (result interface{}) {
	var err error
	switch e.Tokens.top().typ {
	case itemInteger:
		result, err = strconv.ParseInt(e.Tokens.pop().value, 10, 64)
	case itemFloat:
		result, err = strconv.ParseFloat(e.Tokens.pop().value, 64)
	case itemHex:
		result, err = strconv.ParseInt(e.Tokens.pop().value, 16, 64)
	case itemString:
		result = e.Tokens.pop().value
	case itemFunction:
		result = e.evalFunction()
	case itemVariable:
		result = e.evalVariable()
	case itemArithmeticOperator:
		result = e.evalArithmeticOperator()
	default:
		panic(fmt.Sprintf("value has unknown type: %s", e.Tokens.top().typ))
	}
	if err != nil {
		panic("failed to convert '%s' to value")
	}
	return result
}
func (e *evaluator) evalVariable() (result interface{}) {
	item := e.Tokens.pop()
	if item.typ != itemVariable {
		panic(fmt.Sprintf("expected itemVariable, received type %s", item.typ))
	}
	switch item.value {
	case "@name":
		return e.AtomPtr.Name
	case "@type":
		return e.AtomPtr.Type()
	case "@data":
	default:
		panic(fmt.Sprintf("unknown variable: %s", item.value))
	}

	// Must get Atom value. Decide what concrete type to return.
	switch {
	case e.AtomPtr.Value.IsBool():
		result, _ = e.AtomPtr.Value.Bool()
	case e.AtomPtr.Value.IsFloat():
		result, _ = e.AtomPtr.Value.Float()
	case e.AtomPtr.Value.IsUint():
		result, _ = e.AtomPtr.Value.Uint()
	case e.AtomPtr.Value.IsInt(), e.AtomPtr.Value.IsInt():
		result, _ = e.AtomPtr.Value.Int()
	default:
		result, _ = e.AtomPtr.Value.String()
	}
	return result
}
func (e *evaluator) evalFunction() (result interface{}) {
	item := e.Tokens.pop()
	if item.typ != itemFunction {
		panic(fmt.Sprintf("expected itemFunction, received type %s", item.typ))
	}
	switch item.value {
	case "position()":
		return e.Position
	case "count()":
		return e.Count
	case "last()":
		return e.Position == e.Count
	default:
		panic(fmt.Sprintf("unknown variable: %s", item.value))
	}
}

func (e *evaluator) Satisfied(a *Atom, index int, count int) bool {
	e.AtomPtr = a
	e.Position = index
	e.Count = count
	e.Tokens = e.Tokens[:0]
	copy(e.tokens, e.Tokens)
	return e.eval()
}

// Implement explicit type coercion for equality and arithmetic operators
type (
	Float64 float64
	Uint64  uint64
	Int64   int64
	Strang  string

	Comparer interface {
		Equal(other Comparer) bool
		LessThan(other Comparer) bool
		GreaterThan(other Comparer) bool
	}
	Arithmeticker interface {
		Plus(other Arithmeticker) Arithmeticker
		Minus(other Arithmeticker) Arithmeticker
		Multiply(other Arithmeticker) Arithmeticker
		Divide(other Arithmeticker) Arithmeticker
	}
)

func (v Float64) Equal(other Comparer) bool {
	return v == other.(Float64)
}
func (v Float64) LessThan(other Comparer) bool {
	return v < other.(Float64)
}
func (v Float64) GreaterThan(other Comparer) bool {
	return v > other.(Float64)
}
func (v Uint64) Equal(other Comparer) bool {
	return v == other.(Uint64)
}
func (v Uint64) LessThan(other Comparer) bool {
	return v < other.(Uint64)
}
func (v Uint64) GreaterThan(other Comparer) bool {
	return v > other.(Uint64)
}
func (v Int64) Equal(other Comparer) bool {
	return v == other.(Int64)
}
func (v Int64) LessThan(other Comparer) bool {
	return v < other.(Int64)
}
func (v Int64) GreaterThan(other Comparer) bool {
	return v > other.(Int64)
}
func (v Strang) Equal(other Comparer) bool {
	return v == other.(Strang)
}
func (v Strang) LessThan(other Comparer) bool {
	return v < other.(Strang)
}
func (v Strang) GreaterThan(other Comparer) bool {
	return v > other.(Strang)
}
func (v Float64) Plus(other Arithmeticker) Arithmeticker {
	return v + other.(Float64)
}
func (v Float64) Minus(other Arithmeticker) Arithmeticker {
	return v - other.(Float64)
}
func (v Float64) Multiply(other Arithmeticker) Arithmeticker {
	return v * other.(Float64)
}
func (v Float64) Divide(other Arithmeticker) Arithmeticker {
	return v / other.(Float64)
}
func (v Uint64) Plus(other Arithmeticker) Arithmeticker {
	return v + other.(Uint64)
}
func (v Uint64) Minus(other Arithmeticker) Arithmeticker {
	return v - other.(Uint64)
}
func (v Uint64) Multiply(other Arithmeticker) Arithmeticker {
	return v * other.(Uint64)
}
func (v Uint64) Divide(other Arithmeticker) Arithmeticker {
	return v / other.(Uint64)
}
func (v Int64) Plus(other Arithmeticker) Arithmeticker {
	return v + other.(Int64)
}
func (v Int64) Minus(other Arithmeticker) Arithmeticker {
	return v - other.(Int64)
}
func (v Int64) Multiply(other Arithmeticker) Arithmeticker {
	return v * other.(Int64)
}
func (v Int64) Divide(other Arithmeticker) Arithmeticker {
	return v / other.(Int64)
}
