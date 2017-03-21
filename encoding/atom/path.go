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

// Terminology:
//     Each step is evaluated against the nodes in the current node-set.
//
//     A step consists of:
//
//     an axis (defines the tree-relationship between the selected nodes and the current node)
//     a node-test (identifies a node within an axis)
//     zero or more predicates (to further refine the selected node-set)
//     The syntax for a location step is:
//
//     axisname::nodetest[predicate]

// TODO:
//   define boolean syntax for operators
//   missing operators:   | (ie. //book | //cd ) div mod
//   xpath axis support ( <something>::  ) https://www.w3schools.com/xml/xpath_axes.asp
//   consider adopting xpath terminology (eg. predicate replaces filter)
//   support for multiple predicates, eg. a[1][@href='help.php']
//   review precedence table and see what else to implement (eg. idiv)
//   distinct-values()  sum()
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
/// } value
/// }

// Path expressions from XPath:
/*
	/   Selects from the root node (meanigless for Atom)
	//  Selects nodes in the document from the current node that match the
      selection no matter where they are
	.   Selects the current node (meaningless for Atom)
	..  Selects the parent of the current node (useful, just not at root)
	@   Selects attributes (adopted for atom attributes)
*/

// parsing objective: a single func which can take in an atom and position, and
// return a bool indicating whether to keep it.
// future: some XPATH specifiers affect the result by specifying output format.
// https://en.wikipedia.org/wiki/Shunting-yard_algorithm
// this is good because it handles endless nested parens, and respects explicitly defined order of operations. XPath order of operations is defined somewhere.
import (
	"fmt"
	"math"
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
	itemFunctionBool       = "iFunctionBool"
	itemFunctionNumeric    = "iFunctionNum"
	itemVariable           = "iVar"
	itemInteger            = "iInt"
	itemFloat              = "iFloat"
	itemHex                = "iHex"
)

func errInvalidPath(msg string) error {
	if msg == "" {
		return fmt.Errorf("invalid path test: <empty>")
	}
	return fmt.Errorf("invalid path test: %s", msg)
}
func errInvalidPredicate(msg string) error {
	if msg == "" {
		return fmt.Errorf("invalid predicate")
	}
	return fmt.Errorf("invalid predicate: %s", msg)
}

type itemList []*item
type PathError error

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
type filterParser struct {
	outputQueue itemList    // items ordered for evaluation
	opStack     itemList    // holds operators until their operands reach output queue
	items       <-chan item // items received from lexer
	err         PathError   // indicates parsing succeeded or describes what failed
}
type filterFunc func(a Atom, i int) bool

func (a *Atom) AtomsAtPath(path_raw string) (atoms []*Atom, e error) {
	path := strings.TrimSpace(path_raw)
	switch {
	case path == "/":
		atoms, e = []*Atom{a}, nil
	case strings.HasPrefix(path, "//"):
		pathParts := strings.Split(path[2:], "/")
		atoms, e = getAtomsAnywhere(a, pathParts, 0)
	case strings.HasPrefix(path, "/"):
		pathParts := strings.Split(path[1:], "/")
		atoms, e = getAtomsAtPath([]*Atom{a}, pathParts, 0)
	case path == "":
		e = errInvalidPath("")
	default:
		pathParts := strings.Split(path, "/")
		atoms, e = getAtomsAtPath([]*Atom{a}, pathParts, 0)
	}
	if e != nil {
		// include path expression in error message
		e = fmt.Errorf(fmt.Sprint(e.Error(), ` in "`, path, `"`))
	}
	return
}

// getAtomsAnywhere searches the entire tree for matches to the given path that
// ppear at any level.
// must return no atoms on error
func getAtomsAnywhere(a *Atom, pathParts []string, index int) (atoms []*Atom, e error) {
	return getAtomsAtPath(a.AtomList(), pathParts, 0)
}

// must return no atoms on error
func getAtomsAtPath(candidates []*Atom, pathParts []string, index int) (atoms []*Atom, e error) {

	// find all atoms whose name matches the next path element
	theCandidates, e := doLocationStep(candidates, pathParts[index])
	if e != nil {
		return
	}

	// on final path element, return all matched atoms regardless of type
	if index == len(pathParts)-1 { // if last path part
		return theCandidates, e
	}

	// search child atoms for the rest of the path
	var nextCandidates [](*Atom)
	for _, a := range theCandidates {
		nextCandidates = append(nextCandidates, a.Children...)
	}
	return getAtomsAtPath(nextCandidates, pathParts, index+1)
}

func doLocationStep(candidates []*Atom, pathPart string) (atoms []*Atom, e error) {
	pathTest, predicate, e := splitLocationStep(pathPart)
	if e != nil {
		return
	}

	// build up a list of possible atoms that match the path test
	atoms, e = doPathTest(candidates, pathTest)
	if e != nil {
		return atoms, e
	}

	// cull those that don't satisfy the predicate
	return doPredicate(atoms, predicate)
}

// pathTest builds a list of atoms whose name matches the path test string.
func doPathTest(candidates []*Atom, pathTest string) (atoms []*Atom, e error) {
	if pathTest == "" {
		return atoms, errInvalidPath("")
	}
	if pathTest == "*" {
		return candidates, nil
	}
	for _, a := range candidates {
		if a.Name == pathTest {
			atoms = append(atoms, a)
		}
	}
	return
}

// doPredicate takes a list of atoms and filters out the ones that do not
// satisfy the predicate.
func doPredicate(candidates []*Atom, predicate string) (atoms []*Atom, e error) {
	if predicate == "" {
		return candidates, nil
	}

	// apply predicate to determine which elements to keep
	filter, e := NewFilter(predicate)
	fmt.Printf("NEWF \"%s\" => %v\n", predicate, filter.tokens)
	if e != nil {
		return atoms, e
	}
	//	for _, t := range filter.tokens {
	//		fmt.Println("      ", t.typ, " : ", t.value)
	//	}

	atoms = candidates[:0] // overwrite nextAtoms in place during filtering
	count := len(candidates)
	for i, atomPtr := range candidates {
		satisfied, e := filter.Satisfied(atomPtr, i, count)
		if e != nil {
			return nil, e
		}
		if satisfied {
			atoms = append(atoms, atomPtr)
		}
		fmt.Printf(" %t => filter(%2d/%d, %s:%s) on %v\n", satisfied, i, count, atomPtr.Name, atomPtr.Type(), filter.tokens)
	}
	return
}

// splitLocationStep reads a single path element, and returns two strings
// containing the path test and predicate . The square brackets around the
// filter are stripped.
// Example:
// "CN1A[@name=DOGS and @type=UI32]" => "CN1A", "@name=DOGS and @type=UI32"
func splitLocationStep(path string) (pathTest, predicate string, e error) {
	i_start := strings.IndexByte(path, '[')
	if i_start == -1 {
		pathTest = path
		return
	}
	i_end := strings.LastIndexByte(path, ']')
	if i_end == -1 { // path lacks closing ]
		e = errInvalidPredicate("")
		return
	}
	pathTest = path[:i_start]
	predicate = path[i_start+1 : i_end]
	return
}

// lexer - identifies tokens(aka items) in the atom path definition.
// Path lexing is done by the same lexer used for Atom Text format lexing.
// They use very different parsers though.

// filterStringToFunc converts a predicate into a func that evaluates
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
			str := "expected to start with empty buffer, got '%s'"
			panic(fmt.Sprintf(str, l.buffer()))
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
		case r == '+', r == '*':
			l.emit(itemArithmeticOperator)
		case strings.ContainsRune(digits, r):
			lexNumberInPath(l)
		case r == '-':
			if strings.ContainsRune(digits, rune(l.peek())) && !isNumericItem(l.prevItemType) {
				lexNumberInPath(l)
			} else {
				l.emit(itemArithmeticOperator)
			}
		case strings.ContainsRune("=<>!", r):
			lexComparisonOperator(l)
		case strings.ContainsRune(alphaNumericChars, r):
			l.acceptRun(alphabetLowerCase)
			if l.buffer() == "or" || l.buffer() == "and" {
				l.emit(itemBooleanOperator)
			} else if l.peek() == '(' {
				lexFunctionCall(l)
			} else {
				lexBareString(l)
			}
		default:
			return l.errorf("invalid predicate: %s", l.input)
		}
	}

	return nil
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

func lexComparisonOperator(l *lexer) stateFn {
	l.acceptRun("=<>!")
	l.emit(itemComparisonOperator)
	return lexFilterExpression
}

func lexFunctionCall(l *lexer) stateFn {
	// verify all alphanumeric up to this point
	if strings.TrimLeft(l.buffer(), alphaNumericChars) != "" {
		return l.errorf("invalid function call prefix: %s", l.input)
	}

	// determine function return type
	switch l.buffer() {
	case "not":
		l.emit(itemFunctionBool)
	case "count", "position", "last":
		l.emit(itemFunctionNumeric)
	default:
		return l.errorf("unrecognized function '%s'", l.buffer())
	}
	return lexFilterExpression
}

func lexDelimitedString(l *lexer) stateFn {
	// Find delimiter
	delim := l.first()
	if delim != '"' && delim != '\'' {
		l.backup()
		return l.errorf("string should be delimited with quotes, got %s", l.input)
	}
	l.ignore() // discard delimiter

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

	if r != delim {
		return l.errorf("unterminated string in atom path: %s", l.input)
	}

	// discard delimiter and emit string value
	l.backup()
	l.emit(itemString)
	l.next()
	l.ignore()

	return lexFilterExpression
}

// lexBareString accepts a non-delimited string of alphanumeric characters.
// This has more restrictions than a delimited string but is simple and fast to
// parse.
// Doesn't handle any escaping, use delimited strings for anything non-trivial.
func lexBareString(l *lexer) stateFn {
	l.acceptRun(alphaNumericChars)
	switch l.buffer() {
	case "div", "idiv", "mod":
		l.emit(itemArithmeticOperator)
	default:
		l.emit(itemString)
	}
	return lexFilterExpression
}

func lexNumberInPath(l *lexer) stateFn {
	var isHex, isFloatingPoint, isSigned, isExponent bool
	if l.bufferSize() == 0 && l.accept("+-") { // Optional leading sign.
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
// This is based on Djikstra's shunting-yard algorithm.
// https://en.wikipedia.org/wiki/Shunting-yard_algorithm
func (p *filterParser) parseFilterToken(it item) bool {
	switch it.typ {
	case itemInteger, itemHex, itemFloat, itemString, itemVariable, itemFunctionNumeric:
		p.outputQueue.push(&it)
	case itemFunctionBool:
		p.opStack.push(&it)
	case itemComparisonOperator, itemArithmeticOperator, itemBooleanOperator:
		itemPrec := precedence(it.value)
		for {
			if p.opStack.empty() || !isOperatorItem(p.opStack.top()) {
				break
			}
			if itemPrec > precedence(p.opStack.top().value) {
				break
			}
			p.outputQueue.push(p.opStack.pop())
		}
		p.opStack.push(&it)
	case itemLeftParen:
		p.opStack.push(&it)
	case itemRightParen:
		for {
			if p.opStack.empty() {
				return p.errorf("mismatched parentheses in predicate")
			}
			if p.opStack.top().typ == itemLeftParen {
				p.opStack.pop()
				break
			}
			op := p.opStack.pop()
			p.outputQueue.push(op)
		}
	case itemEOF:
		for !p.opStack.empty() {
			op := p.opStack.pop()
			if op.typ == itemLeftParen || op.typ == itemRightParen {
				return p.errorf("mismatched parentheses in predicate")
			}
			p.outputQueue.push(op)
		}
		return false
	default:
		panic(fmt.Sprintf("unexpected item type: %s", it.typ))
	}
	return true
}

func isOperatorItem(it *item) bool {
	switch it.typ {
	case itemComparisonOperator, itemArithmeticOperator, itemBooleanOperator:
		return true
	}
	return false
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
	Tokens   itemList // Copy of .tokens to consume during evaluation
	AtomPtr  *Atom    // atom currently being evaluated from the atom list
	Position int      // index of the atom in the atom list, starts from 1
	Count    int      // number of atoms in the atom list
	Error    error
}

func (e *evaluator) errorf(format string, args ...interface{}) PathError {
	msg := fmt.Sprintf(format, args...)
	e.Error = PathError(errInvalidPredicate(msg))
	return e.Error
}

// evaluate the list of operators/values/stuff against the evaluator's atom/pos/count
func (e *evaluator) eval() (result bool) {
	for !e.Tokens.empty() && e.Error == nil {
		switch e.Tokens.top().typ {
		case itemBooleanOperator:
			result = e.evalBooleanOperator()
		case itemComparisonOperator:
			result = e.evalComparisonOperator()
		case itemArithmeticOperator:
			number := e.evalArithmeticOperator()
			result = number.Equal(Int64Type(e.Position))
		case itemInteger, itemHex:
			number := e.evalNumber()
			result = number.Equal(Int64Type(e.Position))
		case itemFunctionBool:
			result = e.evalFunctionBool()
		case itemFunctionNumeric:
			number := e.evalFunctionNumeric()
			result = number.Equal(Int64Type(e.Position))
		default:
			t := e.Tokens.top()
			e.errorf("unrecognized token '%s'", t.value)
			return
		}
	}
	// calculate a boolean value from op and vars
	return
}
func (e *evaluator) evalBooleanOperator() (result bool) {
	op := e.Tokens.pop()
	if op.typ != itemBooleanOperator {
		e.errorf("expected itemBooleanOperator, received type %s", op.typ)
	}
	switch op.value {
	case "and":
		result = e.eval() && e.eval()
	case "or":
		result = e.eval() || e.eval()
	default:
		e.errorf("unknown boolean operator: %s", op.value)
	}
	return result
}

// Numeric operators. All have arity 2.  Must handle float and int types.  Assumed to be signed.
func (e *evaluator) evalArithmeticOperator() (result Arithmeticker) {
	op := e.Tokens.pop()
	if op.typ != itemArithmeticOperator {
		e.errorf("expected itemArithmeticOperator, received type %s", op.typ)
	}
	rhs := e.evalNumber()
	lhs := e.evalNumber()
	switch op.value {
	case "+":
		result = lhs.Plus(rhs)
	case "-":
		result = lhs.Minus(rhs)
	case "*":
		result = lhs.Multiply(rhs)
	case "div":
		result = lhs.Divide(rhs)
	case "idiv":
		result = lhs.IntegerDivide(rhs)
	case "mod":
		result = lhs.Mod(rhs)
	default:
		e.errorf("unknown arithmetic operator: %s", op.value)
		return
	}
	return result
}
func (e *evaluator) evalComparisonOperator() bool {
	op := e.Tokens.pop()
	if op.typ != itemComparisonOperator {
		e.errorf("expected itemComparisonOperator, received type %s", op.typ)
		return false
	}
	rhs := e.evalComparable()
	lhs := e.evalComparable()
	if e.Error != nil {
		return false
	}
	switch op.value {
	case "=":
		fmt.Println(lhs, rhs)
		return lhs.Equal(rhs)
	case "!=":
		return !lhs.Equal(rhs)
	case "<":
		return lhs.LessThan(rhs)
	case ">":
		return lhs.GreaterThan(rhs)
	case "<=":
		return lhs.LessThan(rhs) || lhs.Equal(rhs)
	case ">=":
		return lhs.GreaterThan(rhs) || lhs.Equal(rhs)
	default:
		e.errorf("unknown comparison operator: %s", op.value)
		return false
	}
}
func (e *evaluator) evalNumber() (result Arithmeticker) {
	var err error
	ok := true
	switch e.Tokens.top().typ {
	case itemInteger:
		v, err := strconv.ParseInt(e.Tokens.pop().value, 10, 64)
		if err != nil {
			e.errorf(err.Error())
			return
		}
		result = Int64Type(v)
	case itemFloat:
		v, err := strconv.ParseFloat(e.Tokens.pop().value, 64)
		if err != nil {
			e.errorf(err.Error())
			return
		}
		result = Float64Type(v)
	case itemHex:
		v, err := strconv.ParseInt(e.Tokens.pop().value, 16, 64)
		if err != nil {
			e.errorf(err.Error())
			return
		}
		result = Int64Type(v)
	case itemFunctionNumeric:
		result = e.evalFunctionNumeric()
	case itemVariable:
		result, ok = e.evalVariable().(Arithmeticker)
	case itemArithmeticOperator:
		result = e.evalArithmeticOperator()
	default:
		e.errorf("value has invalid numeric type: %s", e.Tokens.top().typ)
	}
	if err != nil || !ok {
		e.errorf("failed to convert '%s' to numeric value")
	}
	return result
}
func (e *evaluator) evalComparable() (result Comparer) {
	var err error
	switch e.Tokens.top().typ {
	case itemInteger:
		v, errr := strconv.ParseInt(e.Tokens.pop().value, 10, 64)
		err = errr
		result = Int64Type(v)
	case itemFloat:
		v, errr := strconv.ParseFloat(e.Tokens.pop().value, 64)
		err = errr
		result = Float64Type(v)
	case itemHex:
		v, errr := strconv.ParseInt(e.Tokens.pop().value, 16, 64)
		err = errr
		result = Int64Type(v)
	case itemString:
		result = StringType(e.Tokens.pop().value)
	case itemVariable:
		result = e.evalVariable()
	case itemFunctionNumeric:
		result = e.evalFunctionNumeric()
	case itemArithmeticOperator:
		result = e.evalArithmeticOperator()
	default:
		e.errorf("expected comparable type, got %s", e.Tokens.top().typ)
		return
	}
	if err != nil {
		fmt.Println("got error ", err)
		e.errorf("failed to convert '%s' to comparable value")
		return
	}
	return result
}
func (e *evaluator) evalVariable() (result Comparer) {
	item := e.Tokens.pop()
	if item.typ != itemVariable {
		e.errorf("expected itemVariable, received type %s", item.typ)
	}
	switch item.value {
	case "@name":
		return StringType(e.AtomPtr.Name)
	case "@type":
		return StringType(e.AtomPtr.Type())
	case "@data":
	default:
		e.errorf("unknown variable: %s", item.value)
		return
	}

	// Must get Atom value. Choose concrete type to return.
	switch {
	case e.AtomPtr.Value.IsFloat():
		v, _ := e.AtomPtr.Value.Float()
		result = Float64Type(v)
	case e.AtomPtr.Value.IsInt():
		v, _ := e.AtomPtr.Value.Int()
		result = Int64Type(v)
	case e.AtomPtr.Value.IsUint():
		v, _ := e.AtomPtr.Value.Uint()
		result = Uint64Type(v)
	case e.AtomPtr.Value.IsBool():
		v, _ := e.AtomPtr.Value.Uint() // use UINT since it's represented as 0/1
		result = Uint64Type(v)
	default:
		v, _ := e.AtomPtr.Value.String()
		result = StringType(v)
	}
	return result
}
func (e *evaluator) evalFunctionBool() (result bool) {
	item := e.Tokens.pop()
	if item.typ != itemFunctionBool {
		e.errorf("expected itemFunctionBool, received type %s", item.typ)
	}
	switch item.value {
	case "not":
		return !e.eval()
	default:
		e.errorf("unknown boolean function: %s", item.value)
	}
	return
}
func (e *evaluator) evalFunctionNumeric() (result Arithmeticker) {
	item := e.Tokens.pop()
	if item.typ != itemFunctionNumeric {
		e.errorf("expected itemFunctionNumeric, received type %s", item.typ)
		return
	}
	switch item.value {
	case "position":
		return Uint64Type(e.Position)
	case "last", "count":
		return Uint64Type(e.Count)
	default:
		e.errorf("unknown numeric function: %s", item.value)
	}
	return
}

// Satisfied returns true if the atom/index/count combination satisfies
// the predicate.
func (e *evaluator) Satisfied(a *Atom, index int, count int) (result bool, err error) {
	e.AtomPtr = a
	e.Position = index + 1 // XPath convention: position starts at 1
	e.Count = count
	e.Tokens = e.tokens // copy will be consumed during evaluation
	// FIXME: can I make this use an indexing system rather than popping, to avoid allocation?
	result = e.eval()
	err = e.Error
	return
}

// Implement explicit type coercion for equality and arithmetic operators
type (
	Int64Type   int64
	Uint64Type  uint64
	Float64Type float64
	StringType  string

	Comparer interface {
		Equal(other Comparer) bool
		LessThan(other Comparer) bool
		GreaterThan(other Comparer) bool
	}
	Arithmeticker interface {
		Comparer
		Plus(other Arithmeticker) Arithmeticker
		Minus(other Arithmeticker) Arithmeticker
		Multiply(other Arithmeticker) Arithmeticker
		Divide(other Arithmeticker) Arithmeticker
		IntegerDivide(other Arithmeticker) Arithmeticker
		Mod(other Arithmeticker) Arithmeticker
	}
)

// Define explicitly how to do type conversion when performing arithmetic on
// pairs of heterogenous types.

func (v Float64Type) Equal(other Comparer) bool {
	switch o := other.(type) {
	case Float64Type:
		return v == o
	case Uint64Type:
		return float64(v) == float64(o)
	case Int64Type:
		return float64(v) == float64(o)
	default:
		return false
	}
}
func (v Float64Type) LessThan(other Comparer) bool {
	switch o := other.(type) {
	case Float64Type:
		return v < o
	case Uint64Type:
		return float64(v) < float64(o)
	case Int64Type:
		return float64(v) < float64(o)
	default:
		return false
	}
}
func (v Float64Type) GreaterThan(other Comparer) bool {
	switch o := other.(type) {
	case Float64Type:
		return v > o
	case Uint64Type:
		return float64(v) > float64(o)
	case Int64Type:
		return float64(v) > float64(o)
	default:
		return false
	}
}
func (v Uint64Type) Equal(other Comparer) bool {
	switch o := other.(type) {
	case Float64Type:
		return Float64Type(v) == o
	case Int64Type:
		return Int64Type(v) == o
	default:
		return v == o.(Uint64Type)
	}
}
func (v Uint64Type) LessThan(other Comparer) bool {
	switch o := other.(type) {
	case Float64Type:
		return Float64Type(v) < o
	case Int64Type:
		return Int64Type(v) < o
	default:
		return v < o.(Uint64Type)
	}
}
func (v Uint64Type) GreaterThan(other Comparer) bool {
	switch other := other.(type) {
	case Float64Type:
		return Float64Type(v) > other
	case Int64Type:
		return Int64Type(v) > other
	default:
		return v > other.(Uint64Type)
	}
}
func (v Int64Type) Equal(other Comparer) bool {
	switch other := other.(type) {
	case Float64Type:
		return Float64Type(v) == other
	default:
		return v == other.(Int64Type)
	}
}
func (v Int64Type) LessThan(other Comparer) bool {
	switch other := other.(type) {
	case Float64Type:
		return Float64Type(v) < other
	default:
		return v < other.(Int64Type)
	}
}
func (v Int64Type) GreaterThan(other Comparer) bool {
	switch other := other.(type) {
	case Float64Type:
		return Float64Type(v) > other
	default:
		return v > other.(Int64Type)
	}
}
func (v StringType) Equal(other Comparer) bool {
	otherString, ok := other.(StringType)
	if !ok {
		panic(fmt.Sprintf("cannot type assert to string: %v", other))
	}
	return strings.EqualFold(string(v), string(otherString)) // case insensitive comparison
}
func (v StringType) LessThan(other Comparer) bool {
	if other, ok := other.(StringType); ok {
		return v < other
	}
	return false
}
func (v StringType) GreaterThan(other Comparer) bool {
	if other, ok := other.(StringType); ok {
		return v > other
	}
	return false
}
func (v Float64Type) Plus(other Arithmeticker) Arithmeticker {
	return v + other.(Float64Type)
}
func (v Float64Type) Minus(other Arithmeticker) Arithmeticker {
	return v - other.(Float64Type)
}
func (v Float64Type) Multiply(other Arithmeticker) Arithmeticker {
	switch other := other.(type) {
	case Float64Type:
		return Float64Type(float64(v) * float64(other))
	case Int64Type:
		return Float64Type(float64(v) * float64(other))
	case Uint64Type:
		return Float64Type(float64(v) * float64(other))
	}
	panic(fmt.Sprintf("multiplication not supported for type %T, value '%[1]v'", other))
}
func (v Float64Type) Divide(other Arithmeticker) Arithmeticker {
	switch other := other.(type) {
	case Float64Type:
		return Float64Type(float64(v) / float64(other))
	case Int64Type:
		return Float64Type(float64(v) / float64(other))
	case Uint64Type:
		return Float64Type(float64(v) / float64(other))
	}
	panic(fmt.Sprintf("division not supported for type %T, value '%[1]v'", other))
}
func (v Float64Type) IntegerDivide(other Arithmeticker) Arithmeticker {
	switch other := other.(type) {
	case Float64Type:
		return Int64Type(int64(v) / int64(other))
	case Int64Type:
		return Int64Type(int64(v) / int64(other))
	case Uint64Type:
		return Int64Type(int64(v) / int64(other))
	}
	panic(fmt.Sprintf("integer division not supported for type %T, value '%[1]v'", other))
}
func (v Float64Type) Mod(other Arithmeticker) Arithmeticker {
	switch other := other.(type) {
	case Float64Type:
		return Float64Type(math.Mod(float64(v), float64(other)))
	case Int64Type:
		return Float64Type(math.Mod(float64(v), float64(other)))
	case Uint64Type:
		return Float64Type(math.Mod(float64(v), float64(other)))
	}
	panic(fmt.Sprintf("arithmetic modulus not supported for type %T, value '%[1]v'", other))
}
func (v Uint64Type) Plus(other Arithmeticker) Arithmeticker {
	switch other := other.(type) {
	case Float64Type:
		return Float64Type(v) + other
	case Int64Type:
		return Int64Type(v) + other
	default:
		return v + other.(Uint64Type)
	}
}
func (v Uint64Type) Minus(other Arithmeticker) Arithmeticker {
	switch o := other.(type) {
	case Float64Type:
		return Float64Type(v) - o
	case Int64Type:
		return Int64Type(v) - o
	default:
		return v - o.(Uint64Type)
	}
}
func (v Uint64Type) Multiply(other Arithmeticker) Arithmeticker {
	switch o := other.(type) {
	case Float64Type:
		return Float64Type(v) * o
	case Int64Type:
		return Int64Type(v) * o
	default:
		return v * o.(Uint64Type)
	}
}
func (v Uint64Type) Divide(other Arithmeticker) Arithmeticker {
	switch o := other.(type) {
	case Float64Type:
		return Float64Type(v) / o
	case Int64Type:
		return Int64Type(v) / o
	default:
		return v / o.(Uint64Type)
	}
}
func (v Uint64Type) IntegerDivide(other Arithmeticker) Arithmeticker {
	switch other := other.(type) {
	case Float64Type:
		return Int64Type(int64(v) / int64(other))
	case Int64Type:
		return Int64Type(int64(v) / int64(other))
	case Uint64Type:
		return Uint64Type(uint64(v) / uint64(other))
	}
	panic(fmt.Sprintf("integer division not supported for type %T value'%[1]v'", other))
}
func (v Uint64Type) Mod(other Arithmeticker) Arithmeticker {
	switch o := other.(type) {
	case Float64Type:
		return Float64Type(math.Mod(float64(v), float64(o)))
	case Int64Type:
		return Int64Type(v) % o
	default:
		return v % o.(Uint64Type)
	}
}
func (v Int64Type) Plus(other Arithmeticker) Arithmeticker {
	switch other := other.(type) {
	case Float64Type:
		return Float64Type(v) + other
	default:
		return v + other.(Int64Type)
	}
}
func (v Int64Type) Minus(other Arithmeticker) Arithmeticker {
	switch other := other.(type) {
	case Float64Type:
		return Float64Type(v) - other
	default:
		return v - other.(Int64Type)
	}
}
func (v Int64Type) Multiply(other Arithmeticker) Arithmeticker {
	switch other := other.(type) {
	case Float64Type:
		return Float64Type(v) * other
	default:
		return v * other.(Int64Type)
	}
}
func (v Int64Type) IntegerDivide(other Arithmeticker) Arithmeticker {
	switch other := other.(type) {
	case Float64Type:
		return Int64Type(int64(v) / int64(other))
	case Int64Type:
		return Int64Type(int64(v) / int64(other))
	case Uint64Type:
		return Int64Type(int64(v) / int64(other))
	}
	panic(fmt.Sprintf("integer division not supported for type %T value '%[1]v'", other))
}
func (v Int64Type) Divide(other Arithmeticker) Arithmeticker {
	switch other := other.(type) {
	case Float64Type:
		return Float64Type(v) / other
	default:
		return v / other.(Int64Type)
	}
}
func (v Int64Type) Mod(other Arithmeticker) Arithmeticker {
	switch other := other.(type) {
	case Float64Type:
		return Float64Type(math.Mod(float64(v), float64(other)))
	default:
		return v % other.(Int64Type)
	}
}
func isNumericItem(it itemEnum) bool {
	return it == itemInteger || it == itemFloat
}

// These values are from the XPath 3.1 operator precdence table at
//   https://www.w3.org/TR/xpath-3/#id-precedence-order
// Not all of these operators are implemented here.
func precedence(op string) (value int) {
	switch op {
	case ",":
		value = 1
	case "for", "some", "let", "every", "if":
		value = 2
	case "or":
		value = 3
	case "and":
		value = 4
	case "eq", "ne", "lt", "le", "gt", "ge", "=", "!=", "<", "<=", ">", ">=", "is", "<<", ">>":
		value = 5
	case "||":
		value = 6
	case "to":
		value = 7
	case "+", "-": // binary operators
		value = 8
	case "*", "div", "idiv", "mod":
		value = 9
	case "union", "|":
		value = 10
	case "intersect", "except":
		value = 11
	case "instance of":
		value = 12
	case "treat as":
		value = 13
	case "castable as":
		value = 14
	case "cast as":
		value = 15
	case "=>":
		value = 16
	//		case "-", "+": // unary operators
	//			value = 17
	case "!":
		value = 18
	case "/", "//":
		value = 19
	default:
		panic(fmt.Sprintf("unknown operator: %s", op))
	}
	return value
}
