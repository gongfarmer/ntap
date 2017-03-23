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
	itemBareString         = "iBareString"
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

// PredicateParser is a parser for interpreting predicate tokens.
type PredicateParser struct {
	outputQueue itemList    // items ordered for evaluation
	opStack     itemList    // holds operators until their operands reach output queue
	items       <-chan item // items received from lexer
	err         PathError   // indicates parsing succeeded or describes what failed
}

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
	pre, e := NewPredicateEvaluator(predicate)
	fmt.Printf("NEWF \"%s\" => %v\n", predicate, pre.tokens)
	if e != nil {
		return
	}
	atoms, e = pre.Evaluate(candidates)
	return
}

// splitLocationStep reads a single path element, and returns two strings
// containing the path test and predicate . The square brackets around the
// filter are stripped.
// Example:
// "CN1A[@name=DOGS and @type=UI32]" => "CN1A", "@name=DOGS and @type=UI32"
func splitLocationStep(pathRaw string) (pathTest, predicate string, e error) {
	path := strings.TrimSpace(pathRaw)
	i_start := strings.IndexByte(path, '[')
	if i_start == -1 {
		pathTest = path
		if strings.HasSuffix(path, "]") {
			// predicate terminator without predicate start
			e = errInvalidPredicate("mismatched square brackets")
		}
		return
	}
	i_end := strings.LastIndexByte(path, ']')
	if i_end == -1 { // path lacks closing ]
		e = errInvalidPredicate("unterminated square brackets")
		return
	}
	pathTest = path[:i_start]
	predicate = path[i_start+1 : i_end]
	return
}

// lexer - identifies tokens(aka items) in the atom path definition.
// Path lexing is done by the same lexer used for Atom Text format lexing.
// They use very different parsers though.

// NewPredicateEvaluator reads the predicate from a string by sending
func NewPredicateEvaluator(predicate string) (pre *PredicateEvaluator, err error) {
	var lexr = NewPredicateLexer(predicate)
	pre = new(PredicateEvaluator)
	pre.tokens, err = parseTokens(lexr.items)
	return
}

// Evaluate filters a list of atoms against the predicate conditions, returning
// the atoms that satisfy the predicate.
//
// The candidate atoms must all be made available to the PredicaetEvaluator at
// once, because the predicate may refer to individual child atoms by name,
// requiring them to be evaluated against every other candidate.
func (pre *PredicateEvaluator) Evaluate(candidates []*Atom) (atoms []*Atom, e error) {
	fmt.Println("Start Evaluate with ", pre.tokens)
	pre.Atoms = candidates
	pre.Count = len(candidates)
	for i, atomPtr := range candidates {
		pre.Position = i + 1 // XPath convention, indexing starts at 1
		pre.AtomPtr = atomPtr
		pre.Tokens = pre.tokens
		ok := pre.eval()
		if pre.Error != nil {
			return nil, pre.Error
		}
		if ok {
			atoms = append(atoms, atomPtr)
		}
		fmt.Printf(" %t => filter(%2d/%d, %s:%s) on %v\n", ok, i, pre.Count, pre.AtomPtr.Name, pre.AtomPtr.Type(), pre.Tokens)
	}
	return
}

func (pre *PredicateEvaluator) getChildValue(atomName string) (v Comparer, ok bool) {
	for _, a := range pre.AtomPtr.Children {
		if a.Name != atomName {
			continue
		}
		v = atomValueToComparerType(a)
		fmt.Printf("  getChildValue(%s) := %v.(%[2]T)\n", atomName, v)
		ok = true
		break
	}
	return
}

func atomValueToComparerType(a *Atom) (v Comparer) {
	switch {
	case a.Value.IsUint(), a.Value.IsBool():
		x, _ := a.Value.Uint()
		v = Uint64Type(x)
	case a.Value.IsFloat():
		x, _ := a.Value.Float()
		v = Float64Type(x)
	case a.Value.IsInt():
		x, _ := a.Value.Int()
		v = Int64Type(x)
	default:
		x, _ := a.Value.String()
		v = StringType(x)
	}
	return
}

func NewPredicateLexer(predicate string) *lexer {
	l := &lexer{
		input: predicate,
		items: make(chan item),
	}
	go l.run(lexPredicate)
	return l
}

// lexPredicate splits the filter into tokens.
// The filter is everything within the [].
// Example:  for path "CN1A[not(@type=CONT) and not(@name=DOGS)]",
// This function would be extracting tokens from this string:
//     not(@type=CONT) and not(@name=DOGS)
// it should find the following 13 tokens:
//     not ( @type = CONT ) and not ( @name = DOGS )
// FIXME this is kinda crazy because these return stateFn but don't use it
func lexPredicate(l *lexer) stateFn {
	for {
		if l.bufferSize() != 0 {
			l.errorf(fmt.Sprintf(`could not parse "%s"`, l.buffer()))
			return nil
		}
		r := l.next()
		switch {
		case isSpace(r):
			l.ignore()
		case r == eof:
			l.emit(itemEOF)
			break
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
		if l.prevItemType == itemError {
			return nil
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
	return lexPredicate
}

func lexComparisonOperator(l *lexer) stateFn {
	l.acceptRun("=<>!")
	l.emit(itemComparisonOperator)
	return lexPredicate
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
		return l.errorf(`unrecognized function "%s"`, l.buffer())
	}
	return lexPredicate
}

func lexDelimitedString(l *lexer) stateFn {
	// Find delimiter
	delim := l.first()
	if delim != '"' && delim != '\'' {
		l.backup()
		return l.errorf("expected delimited string, got %s", l.input)
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
				return l.errorf("invalid escape %s", l.input)
			}
		case delim: // accept either delimiter
			done = true
		case '\n':
			l.backup()
			return l.errorf("unterminated string: %s", l.input)
		}
	}

	if r != delim {
		return l.errorf("unterminated string: %s", l.input)
	}

	// discard delimiter and emit string value
	l.backup()
	l.emit(itemString)
	l.next()
	l.ignore()

	return lexPredicate
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
		l.emit(itemBareString)
	}
	return lexPredicate
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
	if (l.buffer() == "0" || l.accept("0")) && l.accept("xX") { // Is it hex?
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
	return lexPredicate
}

// parseTokens translates stream of tokens emitted by the lexer into a
// function that can evaluate whether an atom gets filtered.
func parseTokens(ch <-chan item) (tokens itemList, e error) {
	var pp = PredicateParser{items: ch}
	pp.receiveTokens()
	return pp.outputQueue, pp.err
}

// receiveTokens gets tokens from the lexer and sends them to the parser
// for parsing.
func (pp *PredicateParser) receiveTokens() {
	for {
		it := pp.readItem()
		ok := pp.parseToken(it)
		if it.typ == itemEOF || !ok {
			break
		}
	}
}

// read next time from item channel, and return it.
func (pp *PredicateParser) readItem() (it item) {
	var ok bool
	select {
	case it, ok = <-pp.items:
		if !ok {
			it = item{typ: itemEOF, value: "EOF"}
		}
	}
	return it
}

// errorf sets the error field in the parser, and indicates that parsing should
// stop by returning false.
func (pp *PredicateParser) errorf(format string, args ...interface{}) bool {
	pp.err = errInvalidPredicate(fmt.Sprintf(format, args...))
	return false
}

// parseToken is given tokens from the lexer in the order they are found
// in the path string, and queues them into evaluation order.
// This is based on Djikstra's shunting-yard algorithm.
// https://en.wikipedia.org/wiki/Shunting-yard_algorithm
func (pp *PredicateParser) parseToken(it item) bool {
	fmt.Println("parse item ", it.value, it.typ)
	switch it.typ {
	case itemError:
		return pp.errorf(it.value)
	case itemInteger, itemHex, itemFloat, itemBareString, itemString, itemVariable, itemFunctionNumeric:
		pp.outputQueue.push(&it)
	case itemFunctionBool:
		pp.opStack.push(&it)
	case itemComparisonOperator, itemArithmeticOperator, itemBooleanOperator:
		itemPrec := precedence(it.value)
		for {
			if pp.opStack.empty() || !isOperatorItem(pp.opStack.top()) {
				break
			}
			if itemPrec > precedence(pp.opStack.top().value) {
				break
			}
			pp.outputQueue.push(pp.opStack.pop())
		}
		pp.opStack.push(&it)
	case itemLeftParen:
		pp.opStack.push(&it)
	case itemRightParen:
		for {
			if pp.opStack.empty() {
				return pp.errorf("mismatched parentheses")
			}
			if pp.opStack.top().typ == itemLeftParen {
				pp.opStack.pop()
				break
			}
			op := pp.opStack.pop()
			pp.outputQueue.push(op)
		}
	case itemEOF:
		for !pp.opStack.empty() {
			op := pp.opStack.pop()
			if op.typ == itemLeftParen || op.typ == itemRightParen {
				return pp.errorf("mismatched parentheses")
			}
			pp.outputQueue.push(op)
		}
		return false
	default:
		return pp.errorf("unexpected item %s", it.value)
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

// PredicateEvaluator determines which candidate atoms satisfy the
// predicate criteria.
//
// The predicate is the part of the path within the [].
// Examples:
//    /ROOT[1]
//    /ROOT[@name=NONE]
//		/ROOT/UI_1[@data < 2]
type PredicateEvaluator struct {
	tokens itemList // predicate criteria, as a list of tokens
	Error  error    // evaluation status, nil on success

	Tokens   itemList // Copy of tokens to consume during evaluation
	Atoms    []*Atom  // Atoms being evaluated
	AtomPtr  *Atom    // Atom currently being evaluated from the atom list
	Position int      // index of the atom in the atom list, starts from 1
	Count    int      // number of atoms in the atom list
}

func (pre *PredicateEvaluator) errorf(format string, args ...interface{}) PathError {
	msg := fmt.Sprintf(format, args...)
	pre.Error = PathError(errInvalidPredicate(msg))
	return pre.Error
}

// evaluate the list of operators/values/stuff against the evaluator's atom/pos/count
func (pre *PredicateEvaluator) eval() (result bool) {
	var results []Equaler
	for !pre.Tokens.empty() && pre.Error == nil {
		fmt.Println("EVAL TOKEN TYPE: ", pre.Tokens.top().typ)
		switch pre.Tokens.top().typ {
		case itemBooleanOperator:
			results = append(results, pre.evalBooleanOperator())
		case itemComparisonOperator:
			results = append(results, pre.evalComparisonOperator())
		case itemArithmeticOperator:
			results = append(results, pre.evalArithmeticOperator())
		case itemInteger, itemHex:
			results = append(results, pre.evalNumber())
		case itemFunctionBool:
			results = append(results, pre.evalFunctionBool())
		case itemFunctionNumeric:
			results = append(results, pre.evalFunctionNumeric())
		default:
			t := pre.Tokens.top()
			pre.errorf("unrecognized token '%v'", t.value)
			break // FIXME want break for, but this is just break out of switch
		}
	}
	fmt.Println("EVAL() results: ", results, len(results), pre.Error)
	if pre.Error != nil {
		return
	}
	// verify that evaluation resulted in exactly 1 value
	switch len(results) {
	case 0:
		pre.errorf("no result")
		return
	case 1:
	default:
		pre.errorf("unparsed values '%v'", results)
		return
	}
	// verify that evaluation resulted in a usable type
	switch r := results[0].(type) {
	case BooleanType:
		return bool(r)
	case Int64Type:
		return r.Equal(Int64Type(pre.Position))
	case Uint64Type:
		return r.Equal(Uint64Type(pre.Position))
	case Float64Type:
		return r.Equal(Float64Type(pre.Position))
	default:
		pre.errorf("result '%v' has unknown type %[1]T", results[0])
		return
	}
	fmt.Println("EVAL() what am I doing over here???")
	// calculate a boolean value from op and vars
	return
}
func (pre *PredicateEvaluator) evalBoolean() (result Equaler) {
	switch pre.Tokens.top().typ {
	case itemBooleanOperator:
		result = pre.evalBooleanOperator()
	case itemComparisonOperator:
		result = pre.evalComparisonOperator().(Equaler)
	case itemFunctionBool:
		result = pre.evalFunctionBool()
	default:
		t := pre.Tokens.top()
		pre.errorf("expect boolean, got '%s'", t.value)
		return
	}
	// calculate a boolean value from op and vars
	return
}
func (pre *PredicateEvaluator) evalBooleanOperator() Equaler {
	op := pre.Tokens.pop()
	if op.typ != itemBooleanOperator {
		pre.errorf("expected boolean operator, received type %s", op.typ)
	}
	True := BooleanType(true)
	var result bool
	switch op.value {
	case "and":
		result = pre.evalBoolean().Equal(True) && pre.evalBoolean().Equal(True)
	case "or":
		result = pre.evalBoolean().Equal(True) || pre.evalBoolean().Equal(True)
	default:
		pre.errorf("unknown boolean operator: %s", op.value)
	}
	return BooleanType(result)
}

// Numeric operators. All have arity 2.  Must handle float and int types.  Assumed to be signed.
func (pre *PredicateEvaluator) evalArithmeticOperator() (result Arithmeticker) {
	op := pre.Tokens.pop()
	if op.typ != itemArithmeticOperator {
		pre.errorf("expected itemArithmeticOperator, received type %s", op.typ)
	}
	rhs := pre.evalNumber()
	lhs := pre.evalNumber()
	if pre.Error != nil {
		return
	}
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
		pre.errorf("unknown arithmetic operator: %s", op.value)
		return
	}
	return result
}
func (pre *PredicateEvaluator) evalComparisonOperator() Equaler {
	var result bool
	op := pre.Tokens.pop()
	if op.typ != itemComparisonOperator {
		pre.errorf("expected itemComparisonOperator, received type %s", op.typ)
		return BooleanType(false)
	}
	rhs := pre.evalComparable()
	lhs := pre.evalComparable()
	if pre.Error != nil {
		return BooleanType(false)
	}
	switch op.value {
	case "=":
		fmt.Println(lhs, rhs)
		result = lhs.Equal(rhs)
	case "!=":
		result = !lhs.Equal(rhs)
	case "<":
		result = lhs.LessThan(rhs)
	case ">":
		result = lhs.GreaterThan(rhs)
	case "<=":
		result = lhs.LessThan(rhs) || lhs.Equal(rhs)
	case ">=":
		result = lhs.GreaterThan(rhs) || lhs.Equal(rhs)
	default:
		pre.errorf("unknown comparison operator: %s", op.value)
		result = false
	}
	return BooleanType(result)
}
func (pre *PredicateEvaluator) evalNumber() (result Arithmeticker) {
	var err error
	ok := true
	switch pre.Tokens.top().typ {
	case itemInteger, itemHex:
		v, err := strconv.ParseInt(pre.Tokens.pop().value, 0, 64)
		if err != nil {
			pre.errorf(err.Error())
			return
		}
		result = Int64Type(v)
	case itemFloat:
		v, err := strconv.ParseFloat(pre.Tokens.pop().value, 64)
		if err != nil {
			pre.errorf(err.Error())
			return
		}
		result = Float64Type(v)
	case itemFunctionNumeric:
		result = pre.evalFunctionNumeric()
	case itemVariable:
		result, ok = pre.evalVariable().(Arithmeticker)
	case itemArithmeticOperator:
		result = pre.evalArithmeticOperator()
	case itemBareString:
		t := pre.Tokens.pop()
		if v, ok := pre.getChildValue(t.value); ok {
			result = v.(Arithmeticker)
		} else {
			pre.errorf("expect number, got %s", t.value)
		}
	default:
		pre.errorf("value has invalid numeric type: %s", pre.Tokens.top().typ)
	}
	if err != nil || !ok {
		pre.errorf("expected numeric value")
	}
	return result
}
func (pre *PredicateEvaluator) evalComparable() (result Comparer) {
	var err error
	switch pre.Tokens.top().typ {
	case itemInteger, itemHex:
		v, errr := strconv.ParseInt(pre.Tokens.pop().value, 0, 64)
		err = errr
		result = Int64Type(v)
	case itemFloat:
		v, errr := strconv.ParseFloat(pre.Tokens.pop().value, 64)
		err = errr
		result = Float64Type(v)
	case itemBareString:
		t := pre.Tokens.pop()
		if v, ok := pre.getChildValue(t.value); ok {
			fmt.Println("IBARESTRING converted to atom value ", v)
			// string was an Atom name.  Substitute the atom value.
			result = v
		} else {
			fmt.Println("IBARESTRING failed to convert to atom value ", t.value)
			result = StringType(t.value)
		}
	case itemString:
		result = StringType(pre.Tokens.pop().value)
	case itemVariable:
		result = pre.evalVariable()
	case itemFunctionNumeric:
		result = pre.evalFunctionNumeric()
	case itemArithmeticOperator:
		result = pre.evalArithmeticOperator()
	default:
		pre.errorf("expected comparable type, got %s", pre.Tokens.top().typ)
		return
	}
	if err != nil {
		fmt.Println("got error ", err)
		pre.errorf("failed to convert '%s' to comparable value")
		return
	}
	return result
}
func (pre *PredicateEvaluator) evalVariable() (result Comparer) {
	item := pre.Tokens.pop()
	if item.typ != itemVariable {
		pre.errorf("expected itemVariable, received type %s", item.typ)
	}
	switch item.value {
	case "@name":
		return StringType(pre.AtomPtr.Name)
	case "@type":
		return StringType(pre.AtomPtr.Type())
	case "@data":
	default:
		pre.errorf("unknown variable: %s", item.value)
		return
	}

	// Must get Atom value. Choose concrete type to return.
	switch {
	case pre.AtomPtr.Value.IsFloat():
		v, _ := pre.AtomPtr.Value.Float()
		result = Float64Type(v)
	case pre.AtomPtr.Value.IsInt():
		v, _ := pre.AtomPtr.Value.Int()
		result = Int64Type(v)
	case pre.AtomPtr.Value.IsUint():
		v, _ := pre.AtomPtr.Value.Uint()
		result = Uint64Type(v)
	case pre.AtomPtr.Value.IsBool():
		v, _ := pre.AtomPtr.Value.Uint() // use UINT since it's represented as 0/1
		result = Uint64Type(v)
	default:
		v, _ := pre.AtomPtr.Value.String()
		result = StringType(v)
	}
	return result
}
func (pre *PredicateEvaluator) evalFunctionBool() Equaler {
	var result bool
	item := pre.Tokens.pop()
	if item.typ != itemFunctionBool {
		pre.errorf("expected itemFunctionBool, received type %s", item.typ)
	}
	switch item.value {
	case "not":
		r := pre.evalBoolean()
		if r == nil {
			result = false
		} else {
			result = r.Equal(BooleanType(false))
		}
	default:
		pre.errorf("unknown boolean function: %s", item.value)
	}
	return BooleanType(result)
}
func (pre *PredicateEvaluator) evalFunctionNumeric() (result Arithmeticker) {
	item := pre.Tokens.pop()
	if item.typ != itemFunctionNumeric {
		pre.errorf("expected itemFunctionNumeric, received type %s", item.typ)
		return
	}
	switch item.value {
	case "position":
		return Uint64Type(pre.Position)
	case "last", "count":
		return Uint64Type(pre.Count)
	default:
		pre.errorf("unknown numeric function: %s", item.value)
	}
	return
}

// Implement a small type system with type coercion for operators
type (
	Int64Type   int64
	Uint64Type  uint64
	Float64Type float64
	StringType  string
	BooleanType bool

	Equaler interface {
		Equal(other Equaler) bool
	}
	Comparer interface {
		Equaler
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

func (v Float64Type) Equal(other Equaler) bool {
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
func (v Uint64Type) Equal(other Equaler) bool {
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
	fmt.Println("UINT64 LessThan", v, other)
	switch o := other.(type) {
	case Float64Type:
		return Float64Type(v) < o
	case Int64Type:
		return Int64Type(v) < o
	case StringType:
		if x, e := strconv.ParseFloat(string(o), 64); e == nil {
			return float64(v) < x
		}
		if x, e := strconv.ParseUint(string(o), 10, 64); e == nil {
			return uint64(v) < x
		}
		if x, e := strconv.ParseInt(string(o), 0, 64); e == nil {
			return int64(v) < x
		}
	default:
		return v < o.(Uint64Type)
	}
	return false
}
func (v Uint64Type) GreaterThan(other Comparer) bool {
	fmt.Println("UINT64 GreaterThan", v, other)
	switch o := other.(type) {
	case Float64Type:
		return Float64Type(v) > o
	case Int64Type:
		return Int64Type(v) > o
	case StringType:
		return !(o.LessThan(v) || o.Equal(v))
	default:
		return v > other.(Uint64Type)
	}
	return false
}
func (v Int64Type) Equal(other Equaler) bool {
	switch o := other.(type) {
	case Float64Type:
		return Float64Type(v) == o
	case Uint64Type:
		return v == Int64Type(o)
	case StringType:
		if x, e := strconv.ParseFloat(string(o), 64); e == nil {
			return float64(v) == x
		}
		if x, e := strconv.ParseUint(string(o), 10, 64); e == nil {
			return uint64(v) == x
		}
		if x, e := strconv.ParseInt(string(o), 0, 64); e == nil {
			return int64(v) == x
		}
	default:
		return v == other.(Int64Type)
	}
	return false
}
func (v Int64Type) LessThan(other Comparer) bool {
	switch o := other.(type) {
	case Float64Type:
		return Float64Type(v) < o
	case StringType:
		return !(o.GreaterThan(v) || o.Equal(v))
	case Uint64Type:
		return v < Int64Type(o)
	default:
		return v < o.(Int64Type)
	}
}
func (v Int64Type) GreaterThan(other Comparer) bool {
	switch o := other.(type) {
	case Float64Type:
		return Float64Type(v) > o
	case StringType:
		return !(o.LessThan(v) || o.Equal(v))
	case Uint64Type:
		return v > Int64Type(o)
	default:
		return v > o.(Int64Type)
	}
}
func (v StringType) Equal(other Equaler) bool {
	switch o := other.(type) {
	case StringType:
		// case insensitive comparison
		return strings.EqualFold(string(v), string(o))
	case Int64Type:
		return string(v) == strconv.Itoa(int(o))
	case Uint64Type:
		return string(v) == strconv.Itoa(int(o))
	case Float64Type:
		return string(v) == fmt.Sprintf("%G", o)
	}
	return false
}
func (v StringType) LessThan(other Comparer) bool {
	fmt.Println("StringType LessThan", v, other)
	str := string(v)
	if x, e := strconv.ParseFloat(str, 64); e == nil {
		return Float64Type(x).LessThan(other)
	}
	if x, e := strconv.ParseUint(str, 10, 64); e == nil {
		return Uint64Type(x).LessThan(other)
	}
	if x, e := strconv.ParseInt(str, 0, 64); e == nil {
		// this case handles hex strings too, based on prefix
		return Int64Type(x).LessThan(other)
	}
	if o, ok := other.(StringType); ok {
		return str > string(o)
	}
	return false
}
func (v StringType) GreaterThan(other Comparer) bool {
	str := string(v)
	if x, e := strconv.ParseFloat(str, 64); e == nil {
		return Float64Type(x).GreaterThan(other)
	}
	if x, e := strconv.ParseUint(str, 10, 64); e == nil {
		return Uint64Type(x).GreaterThan(other)
	}
	if x, e := strconv.ParseInt(str, 0, 64); e == nil {
		// this case handles hex strings too, based on prefix
		return Int64Type(x).GreaterThan(other)
	}
	if o, ok := other.(StringType); ok {
		return str > string(o)
	}
	return false
}
func (v Float64Type) Plus(other Arithmeticker) Arithmeticker {
	switch o := other.(type) {
	case Float64Type:
		return Float64Type(float64(v) + float64(o))
	case Int64Type:
		return Float64Type(float64(v) + float64(o))
	case Uint64Type:
		return Float64Type(float64(v) + float64(o))
	}
	panic(fmt.Sprintf("addition not supported for type %T, value '%[1]v'", other))
}
func (v Float64Type) Minus(other Arithmeticker) Arithmeticker {
	switch o := other.(type) {
	case Float64Type:
		return Float64Type(float64(v) - float64(o))
	case Int64Type:
		return Float64Type(float64(v) - float64(o))
	case Uint64Type:
		return Float64Type(float64(v) - float64(o))
	}
	panic(fmt.Sprintf("multiplication not supported for type %T, value '%[1]v'", other))
}
func (v Float64Type) Multiply(other Arithmeticker) Arithmeticker {
	switch o := other.(type) {
	case Float64Type:
		return Float64Type(float64(v) * float64(o))
	case Int64Type:
		return Float64Type(float64(v) * float64(o))
	case Uint64Type:
		return Float64Type(float64(v) * float64(o))
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
	fmt.Println("UI64::Divide", v, other)
	switch o := other.(type) {
	case Float64Type:
		return Float64Type(float64(v) / float64(o))
	case Int64Type:
		return Float64Type(float64(v) / float64(o))
	case Uint64Type:
		return Float64Type(float64(v) / float64(o))
	}
	panic(fmt.Sprintf("division not supported for type %T value '%[1]v'", other))
}
func (v Uint64Type) IntegerDivide(other Arithmeticker) Arithmeticker {
	switch o := other.(type) {
	case Float64Type:
		return Int64Type(int64(v) / int64(o))
	case Int64Type:
		return Int64Type(int64(v) / int64(o))
	case Uint64Type:
		return Uint64Type(uint64(v) / uint64(o))
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
	switch o := other.(type) {
	case Float64Type:
		return Float64Type(float64(v) + float64(o))
	case Int64Type:
		return Int64Type(int64(v) + int64(o))
	case Uint64Type:
		if int64(v) < 0 {
			return Int64Type(int64(v) + int64(o))
		} else {
			return Uint64Type(uint64(v) + uint64(o))
		}
	}
	panic(fmt.Sprintf("integer addition not supported for type %T value '%[1]v'", other))
}
func (v Int64Type) Minus(other Arithmeticker) Arithmeticker {
	switch other := other.(type) {
	case Float64Type:
		return Float64Type(float64(v) - float64(other))
	case Int64Type:
		return Int64Type(int64(v) - int64(other))
	case Uint64Type:
		if v < 0 {
			return Int64Type(int64(v) - int64(other))
		} else {
			return Uint64Type(uint64(v) - uint64(other))
		}
	}
	panic(fmt.Sprintf("subtraction not supported for type %T value '%[1]v'", other))
}
func (v Int64Type) Multiply(other Arithmeticker) Arithmeticker {
	switch other := other.(type) {
	case Float64Type:
		return Float64Type(float64(v) * float64(other))
	case Int64Type:
		return Int64Type(int64(v) * int64(other))
	case Uint64Type:
		if v < 0 {
			return Int64Type(int64(v) * int64(other))
		} else {
			return Uint64Type(uint64(v) * uint64(other))
		}
	}
	panic(fmt.Sprintf("subtraction not supported for type %T value '%[1]v'", other))
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
		return Float64Type(float64(v) / float64(other))
	case Int64Type:
		return Float64Type(float64(v) / float64(other))
	case Uint64Type:
		return Float64Type(float64(v) / float64(other))
	}
	panic(fmt.Sprintf("division not supported for type %T value '%[1]v'", other))
}
func (v Int64Type) Mod(other Arithmeticker) Arithmeticker {
	switch other := other.(type) {
	case Float64Type:
		return Float64Type(math.Mod(float64(v), float64(other)))
	default:
		return v % other.(Int64Type)
	}
}
func (v BooleanType) Equal(other Equaler) bool {
	fmt.Println("EQUALERBOOL")
	switch o := other.(type) {
	case BooleanType:
		return bool(v) == bool(o)
	case Int64Type:
		if bool(v) == false {
			return int64(o) == 0
		} else {
			return int64(o) != 0
		}
	case Uint64Type:
		if bool(v) == false {
			return uint64(o) == 0
		} else {
			return uint64(o) != 0
		}
	default:
		return false
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
