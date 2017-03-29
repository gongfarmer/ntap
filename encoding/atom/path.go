package atom

// AtomsAtPath returns a slice of all Atom descendants at the given path.
// If no atom is found, tk returns an error message that describes which path
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
//     Each step is evaluated against the elements in the current node-set.
//
//     A step consists of:
//
//     an axis (defines the tree-relationship between the selected elements and the current node)
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

/// struct stackToken {
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
	//  Selects elements in the document from the current node that match the
      selection no matter where they are
	.   Selects the current node (meaningless for Atom)
	..  Selects the parent of the current node (useful, just not at root)
	@   Selects attributes (adopted for atom attributes)
*/

// parsing objective: a single func which can take in an atom and position, and
// return a bool indicating whether to keep tk.
// future: some XPATH specifiers affect the result by specifying output format.
// https://en.wikipedia.org/wiki/Shunting-yard_algorithm
// this is good because tk handles endless nested parens, and respects explicitly defined order of operations. XPath order of operations is defined somewhere.

import (
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"strconv"
	"strings"
)

const (
	// Operators must end with string "Operator".. tk is how they are identified as
	// Operator tokens by the parser
	tokenLeftParen          = "tknParenL"
	tokenRightParen         = "tknParenR"
	tokenPredicateStart     = "tknPredStart"
	tokenPredicateEnd       = "tknPredEnd"
	tokenPathTest           = "tknPathTest"
	tokenAxisOperator       = "tknAxisOperator" // "/" or "//" which precedes pathTest
	tokenArithmeticOperator = "tknArithmeticOperator"
	tokenBooleanOperator    = "tknBooleanOperator"
	tokenBooleanLiteral     = "tknBooleanLiteral"
	tokenEqualityOperator   = "tknEqualsOperator"
	tokenNodeTest           = "tknNodeTestOperator"
	tokenComparisonOperator = "tknCompareOperator"
	tokenUnionOperator      = "tknUnionOperator"
	tokenStepSeparator      = "tknStepSeparator"
	tokenOperator           = "tknOperator"
	tokenFunctionBool       = "tknFunctionBool"
	tokenFunctionNumeric    = "tknFunctionNum"
	tokenVariable           = "tknVar"
	tokenInteger            = "tknInt"
	tokenFloat              = "tknFloat"
	tokenBareString         = "tknBareString"
	tokenHex                = "tknHex"
)

// // FIXME: maybe this is too many allocations, if we have to convert the entire
// // nodeset into these for each path evaluation
// // Consider changing the Atom type itself to confirm to the interface as much
// // as possible,  which is a long way.
// // Maybe convert to this type only when necessary, eg. to add new capabilities
// // like id() or parent()
// // AtomicElement makes the Atom type implement the Element interface
// type AtomicElement struct {
// 	AtomPtr *Atom
// }
//
// func (ae AtomicElement) Name() string {
// 	return ae.AtomPtr.Name()
// }
// func (ae AtomicElement) Type() string {
// 	return ae.AtomPtr.Type()
// }
// func (ae AtomicElement) DataString() string {
// 	return ae.AtomPtr.DataString()
// }
// func (ae AtomicElement) String() string {
// 	return ae.AtomPtr.String()
// }
// func (ae AtomicElement) Children() []*Element {
// 	return []*Element(ae.AtomPtr.children)
// }

// FIXME define type AtomicElement, which embeds an Atom and adds parent ptr and unique id? Is embedded type a copy or a reference?  Must be a copy since it should take up full width.  Could add atomPtr field instead of embedding.

// A Node is data that can be represented as a string.
// An Atom/Element is a Node, so are Atom.Name(), Atom.Type(), and atom.Data().
// All type system types such as Int64Type, Float64Type, StringType are Nodes.
type Node interface {
	String() string
}

func errInvalidPath(msg string) error {
	if msg == "" {
		return fmt.Errorf("invalid path: <empty>")
	}
	return fmt.Errorf("invalid path: %s", msg)
}
func errInvalidPredicate(msg string) error {
	if msg == "" {
		return fmt.Errorf("invalid predicate")
	}
	return fmt.Errorf("invalid predicate: %s", msg)
}

type tokenList []*token
type PathError error

func (s *tokenList) push(tk *token) {
	*s = append(*s, tk)
}

func (s *tokenList) unshift(tk *token) {
	*s = append(tokenList{tk}, *s...)
}

// remove and return the first list token
func (s *tokenList) shift() (tk *token) {
	if len(*s) == 0 {
		return nil
	}
	tk, *s = (*s)[0], (*s)[1:]
	return
}

// pop an token off the stack and return tk.
// Return ok=false if stack is empty.
func (s *tokenList) pop() (tk *token) {
	size := len(*s)
	ok := size > 0
	if !ok {
		return
	}
	tk = (*s)[size-1]  // get token from stack top
	*s = (*s)[:size-1] // resize stack
	return
}

// pop the stack only if the top token has the specified type.
// Return ok=true if an token is popped.
func (s *tokenList) popType(typ tokenEnum) (tk *token, ok bool) {
	if s.empty() || s.nextType() != typ {
		return
	}
	return s.pop(), true
}

// peek at the top token on the stack, without removing the token.
func (s tokenList) peek() (tk *token) {
	if len(s) == 0 {
		return nil
	}
	return s[len(s)-1]
}

// return the type of the top token on the stack.
func (s tokenList) nextType() tokenEnum {
	if len(s) == 0 {
		return tokenEnum("")
	}
	return s[len(s)-1].typ
}

// empty returns true if the list is empty.
func (s *tokenList) empty() bool {
	return len(*s) == 0
}

// PathParser is a parser for interpreting predicate tokens.
type PathParser struct {
	outputQueue tokenList    // tokens ordered for evaluation
	opStack     tokenList    // holds operators until their operands reach output queue
	tokens      <-chan token // tokens received from lexer
	err         PathError    // indicates parsing succeeded or describes what failed
}

type AtomPath struct {
	Path      string
	Log       *log.Logger
	Evaluator *PathEvaluator
	err       error
}

// NewAtomPath creates an AtomPath object for the given path string.  It
// performs all lexing and parsing steps, so that evaluating data sets against
// the path will have as little overhead as possible.
func NewAtomPath(path string) (ap *AtomPath, e error) {
	log.Printf("NewAtomPath(%q)", path)

	var pe *PathEvaluator
	pe, e = NewPathEvaluator(path)
	if e != nil {
		return
	}
	for i, tk := range pe.Tokens {
		fmt.Printf("%3d.  %15s  [%s]\n", i, tk.value, tk.typ)
	}

	ap = &AtomPath{
		Path:      strings.TrimSpace(path),
		Log:       log.New(ioutil.Discard, "", log.LstdFlags),
		Evaluator: pe,
		err:       nil,
	}
	return
}

// AtomPath contains an array of 1-N {pathStep, [predicate]...}
// predicates are stored as PredicateEvaluator objects
// also store starting piece: // or / if any
// Union operators are stored as AtomUnion objects which store lists of |, (, ) and atomUnion

// return a set of
/*
(//*[true()] | //*[false()])[data() = 2] | /root/down/two
(
	//*[true()] # AtomPath
	|
	//*[false()] # AtomPath
)
[data() = 2] # AtomPath | /root/down/two # AtomPath

*/
// (//*[@name="0x00000000"] | //*[@name="0x00000001"])[data() = 2]

// step 1: break down path by union operators
// step 2: for each piece:
// step 2a.  consume start of string, / or //, to get starting set for this piece
// step 2b.  split remainder on /, get pathstep and predicates for each

func (ap *AtomPath) SetLogger(l *log.Logger) {
	ap.Log = l
}

func (a *Atom) AtomsAtPath(path string) (atoms []*Atom, e error) {
	atomPath, e := NewAtomPath(path)
	if e != nil {
		return nil, e
	}

	atoms, e = atomPath.GetAtoms(a)
	if e != nil {
		return
	}
	return
}

func (ap *AtomPath) GetAtoms(root *Atom) (atoms []*Atom, e error) {
	return ap.Evaluator.Evaluate(root)
}

// splitLocationStep reads a single path element, and returns two strings
// containing the path test and predicate . The square brackets around the
// filter are stripped.
// Example:
//   "CN1A[@type=UI32]" => "CN1A", ["@type=UI32"]
//   "CN1A[@name=ROOT][@type=UI32][3]" => "CN1A", ["@name=ROOT","@type=UI32","3"]
func splitPathOnUnions(pathRaw string) (pathinfo [][]string, e error) {
	path := strings.Replace(pathRaw, "union", "|", -1) // treat "union" same as "|"
	for _, str := range strings.Split(path, "|") {
		if step, e := splitPathAndPredicate(str); e != nil {
			return nil, e
		} else {
			pathinfo = append(pathinfo, step)
		}
	}
	return
}

// splitPathAndPredicate returns a string slice where the first element is the
// path, and the remaining 0-N elements are predicates to apply to that path.
func splitPathAndPredicate(pathRaw string) (pathinfo []string, e error) {
	// find path, make tk the first element in the slice
	path := strings.TrimSpace(pathRaw)
	i_start := strings.IndexByte(path, '[')
	if i_start == -1 {
		if strings.HasSuffix(path, "]") {
			// predicate terminator without predicate start
			e = errInvalidPredicate("mismatched square brackets")
		}
		return append(pathinfo, path), nil
	}
	pathinfo = append(pathinfo, strings.TrimSpace(path[:i_start]))
	// there's at least one predicate if we get here

	// collect each predicate delimited by [ ... ]
	for _, p := range strings.Split(path[i_start+1:], "[") {
		trimmed := strings.TrimSpace(p)
		if !strings.HasSuffix(trimmed, "]") {
			e = errInvalidPredicate("unterminated square brackets")
			break
		}
		// strip ] and any preceding whitespace
		pathinfo = append(pathinfo, strings.TrimSpace(trimmed[:len(trimmed)-1]))
	}

	// Already verified that there's nothing after the last ], so done
	return
}

// NewPathEvaluator reads the path
func NewPathEvaluator(path string) (pe *PathEvaluator, err error) {
	var lexr = NewPathLexer(path)
	var pp = PathParser{tokens: lexr.tokens}
	pp.receiveTokens()
	pe = &PathEvaluator{
		Path:   path,
		tokens: pp.outputQueue,
		Error:  pp.err}
	return pe, pp.err
}

// lexer - identifies tokens(aka tokens) in the atom path definition.
// Path lexing is done by the same lexer used for Atom Text format lexing.
// They use very different parsers though.

// NewPredicateEvaluator reads the predicate from a string by sending
func NewPredicateEvaluator(predicate string) (pre *PredicateEvaluator, err error) {
	var lexr = NewPredicateLexer(predicate)
	pre = new(PredicateEvaluator)
	pre.tokens, err = parseTokens(lexr.tokens)
	return
}

// Evaluate filters a list of atoms against the predicate conditions, returning
// the atoms that satisfy the predicate.
//
// The candidate atoms must all be made available to the PredicateEvaluator at
// once, because the predicate may refer to individual child atoms by name,
// requiring them to be evaluated against every other candidate.
func (pre *PredicateEvaluator) Evaluate(candidates []*Atom) (atoms []*Atom, e error) {
	log.Print("PredicateEvaluator::Evaluate()  ", pre.tokens, candidates)
	pre.Atoms = candidates
	pre.Count = len(candidates)
	for i, atomPtr := range candidates {
		pre.Position = i + 1 // XPath convention, indexing starts at 1
		pre.AtomPtr = atomPtr
		pre.Tokens = pre.tokens

		// eval candidate atoms against path+predicate(s)
		log.Printf("    eval() with Tokens(%v)", pre.tokens)
		results := pre.eval()
		ok, err := pre.evalResultsToBool(results)
		log.Printf("    eval() returned %t and err %v", ok, pre.Error)
		if err != nil {
			return nil, err
		}
		if ok {
			atoms = append(atoms, atomPtr)
		}
	}
	return
}

func (pre *PredicateEvaluator) getChildValue(atomName string) (v Comparer, ok bool) {
	for _, a := range pre.AtomPtr.children {
		if a.name != atomName {
			continue
		}
		v = atomValueToComparerType(a)
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

func NewPathLexer(path string) *lexer {
	l := &lexer{
		input:  path,
		tokens: make(chan token),
	}
	go l.run(lexPath)
	return l
}

func NewPredicateLexer(predicate string) *lexer {
	l := &lexer{
		input:  predicate,
		tokens: make(chan token),
	}
	go l.run(lexPredicate)
	return l
}

func lexPath(l *lexer) stateFn {
	if l.bufferSize() != 0 {
		l.errorf(fmt.Sprintf(`could not parse "%s"`, l.buffer()))
		return nil
	}
	r := l.next()
	switch {
	case isSpace(r):
		l.ignore()
	case r == eof:
		l.emit(tokenEOF)
		break
	case r == '*':
		l.emit(tokenNodeTest)
	case r == '|':
		l.emit(tokenUnionOperator)
	case r == '/':
		lexStepSeparatorOrAxis(l)
	case r == '[':
		l.emit(tokenPredicateStart)
		return lexPredicate
	case r == '(':
		l.emit(tokenLeftParen)
	case r == ')':
		l.emit(tokenRightParen)
	case strings.ContainsRune(alphaNumericChars, r):
		l.acceptRun(alphabetLowerCase)
		if l.peek() == '(' {
			lexFunctionCall(l)
		} else {
			lexBareString(l)
		}
	default:
		return l.errorf("lexPath cannot parse %q", r)
	}
	if l.prevTokenType == tokenError {
		return nil
	}
	return lexPath
}

func lexPredicate(l *lexer) stateFn {
	if l.bufferSize() != 0 {
		l.errorf(fmt.Sprintf(`could not parse "%s"`, l.buffer()))
		return nil
	}
	r := l.next()
	switch {
	case isSpace(r):
		l.ignore()
	case r == eof:
		l.emit(tokenEOF)
		break
	case r == '@':
		lexAtomAttribute(l)
	case r == '"', r == '\'':
		lexDelimitedString(l)
	case r == ']':
		l.emit(tokenPredicateEnd)
		return lexPath
	case r == '(':
		l.emit(tokenLeftParen)
	case r == ')':
		l.emit(tokenRightParen)
	case r == '|':
		return l.errorf("union not permitted within predicate")
	case r == '+', r == '*':
		l.emit(tokenArithmeticOperator)
	case strings.ContainsRune(numericChars, r):
		lexNumberInPath(l)
	case r == '-':
		if strings.ContainsRune(numericChars, rune(l.peek())) && !isNumericToken(l.prevTokenType) {
			lexNumberInPath(l)
		} else {
			l.emit(tokenArithmeticOperator)
		}
	case strings.ContainsRune("=<>!", r):
		lexComparisonOperator(l)
	case strings.ContainsRune(alphaNumericChars, r):
		l.acceptRun(alphabetLowerCase)
		if l.peek() == '(' {
			lexFunctionCall(l)
		} else {
			lexBareString(l)
		}
	default:
		return l.errorf("lexPredicate cannot parse %q", r)
	}

	return lexPredicate
}

func lexStepSeparatorOrAxis(l *lexer) stateFn {
	if l.first() != '/' {
		panic(`lexStepSeparatorOrAxis called without leading "/"`)
	}
	if l.accept("/") {
		l.emit(tokenAxisOperator)
	} else {
		if l.prevTokenType == tokenNodeTest || l.prevTokenType == tokenPredicateEnd {
			l.emit(tokenStepSeparator)
		} else {
			l.emit(tokenAxisOperator)
		}
	}
	return lexNodeTest
}

func lexNodeTest(l *lexer) stateFn {
	l.acceptRun(alphaNumericChars)
	l.emit(tokenNodeTest)
	if l.accept("/") {
		l.emit(tokenStepSeparator)
		return lexNodeTest
	} else {
		return lexPath
	}
}

// lexAtomAttribute accepts @name, @type or @data.  The @ is already read.
func lexAtomAttribute(l *lexer) stateFn {
	if l.first() != '@' {
		panic("lexAtomAttribute called without leading attribute sigil @")
	}
	l.acceptRun(alphaNumericChars)
	l.emit(tokenVariable)
	return lexPredicate
}

func lexComparisonOperator(l *lexer) stateFn {
	l.acceptRun("=<>!")
	if l.buffer() == "=" || l.buffer() == "!=" {
		l.emit(tokenEqualityOperator)
	} else {
		l.emit(tokenComparisonOperator)
	}
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
		l.emit(tokenFunctionBool)
	case "count", "position", "last":
		l.emit(tokenFunctionNumeric)
	case "name", "type", "data":
		l.emit(tokenVariable)
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
	l.emit(tokenString)
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
	case "union":
		l.emit(tokenUnionOperator)
	case "eq", "ne":
		l.emit(tokenEqualityOperator)
	case "lt", "le", "gt", "ge":
		l.emit(tokenComparisonOperator)
	case "div", "idiv", "mod":
		l.emit(tokenArithmeticOperator)
	case "or", "and":
		l.emit(tokenBooleanOperator)
	case "true", "false":
		l.emit(tokenBooleanLiteral)
	default:
		l.emit(tokenBareString)
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
	if (l.buffer() == "0" || l.accept("0")) && l.accept("xX") { // Is tk hex?
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
		l.emit(tokenFloat)
	case isHex:
		l.emit(tokenHex)
	default:
		l.emit(tokenInteger)
	}
	return lexPredicate
}

// parseTokens translates stream of tokens emitted by the lexer into a
// function that can evaluate whether an atom gets filtered.
func parseTokens(ch <-chan token) (tokens tokenList, e error) {
	var pp = PathParser{tokens: ch}
	pp.receiveTokens()
	return pp.outputQueue, pp.err
}

// receiveTokens gets tokens from the lexer and sends them to the parser
// for parsing.
func (pp *PathParser) receiveTokens() {
	for {
		tk := pp.readToken()
		ok := pp.parseToken(tk)

		str := fmt.Sprintf("parseToken(%s, '%v')", tk.typ, tk.value)
		log.Printf("    %-35s %30s %v\n", str, fmt.Sprint(pp.opStack), pp.outputQueue)

		if tk.typ == tokenEOF || !ok {
			break
		}
	}
}

// read next time from token channel, and return tk.
func (pp *PathParser) readToken() (tk token) {
	var ok bool
	select {
	case tk, ok = <-pp.tokens:
		if !ok {
			tk = token{typ: tokenEOF, value: "EOF"}
		}
	}
	return tk
}

// errorf sets the error field in the parser, and indicates that parsing should
// stop by returning false.
func (pp *PathParser) errorf(format string, args ...interface{}) bool {
	pp.err = errInvalidPath(fmt.Sprintf(format, args...))
	return false
}

// parseToken is given tokens from the lexer in the order they are found
// in the path string, and queues them into evaluation order.
// This is based on Djikstra's shunting-yard algorithm.
// https://en.wikipedia.org/wiki/Shunting-yard_algorithm
func (pp *PathParser) parseToken(tk token) bool {
	log.Printf("      parseToken %q {%s} precedence=%d", tk.value, tk.typ, precedence(tk))
	switch tk.typ {
	case tokenError:
		return pp.errorf(tk.value)
	case tokenInteger, tokenHex, tokenFloat, tokenBareString, tokenString, tokenVariable, tokenFunctionNumeric, tokenBooleanLiteral:
		pp.outputQueue.push(&tk)
	case tokenPredicateStart:
		pp.pushOperatorsToOutputUntil(func(t token) bool {
			// want predicate delimiter tokens to precede axis+node test in stack order
			return t.typ != tokenAxisOperator && t.typ != tokenNodeTest
		})
		// push to both.  Only the output queue copy will be kept.
		pp.outputQueue.push(&tk)
		pp.opStack.push(&tk)
	case tokenFunctionBool, tokenNodeTest:
		pp.opStack.push(&tk)
	case tokenComparisonOperator, tokenArithmeticOperator, tokenEqualityOperator, tokenBooleanOperator, tokenAxisOperator:
		p := precedence(tk)
		pp.pushOperatorsToOutputUntil(func(t token) bool {
			return !isOperatorToken(&t) || p > precedence(t)
		})
		pp.opStack.push(&tk)
	case tokenLeftParen:
		pp.opStack.push(&tk)
	case tokenRightParen:
		pp.pushOperatorsToOutputUntil(func(t token) bool { return t.typ == tokenLeftParen })
		pp.opStack.pop() // remove the matching LeftParen from the stack
	case tokenPredicateEnd:
		pp.pushOperatorsToOutputUntil(func(t token) bool { return t.typ == tokenPredicateStart })
		pp.opStack.pop() // discard PredicateStart from opstack. There's already one on the output queue, so no push()
		pp.opStack.push(&tk)
	case tokenUnionOperator:
		pp.pushOperatorsToOutputUntil(func(t token) bool { return t.typ == tokenLeftParen })
		pp.opStack.push(&tk)
	case tokenEOF:
		for !pp.opStack.empty() {
			op := pp.opStack.pop()
			if op.typ == tokenLeftParen || op.typ == tokenRightParen {
				return pp.errorf("mismatched parentheses")
			}
			pp.outputQueue.push(op)
		}
		return false
	default:
		return pp.errorf("unexpected token %q [%s]", tk.value, tk.typ)
	}
	return true
}

// pushOperatorsToOutputUntil() pops tokens from the operator stack, and pushes
// them into the output queue. This continues until the given test function
// fails to satisfy the given test function.
// The failed end token is left on the operator stack.
func (pp *PathParser) pushOperatorsToOutputUntil(test func(t token) bool) {
	for {
		if pp.opStack.empty() || test(*pp.opStack.peek()) {
			break
		}
		pp.outputQueue.push(pp.opStack.pop())
	}
}

// func (pp *PathParser) parsePredicateTokens(tk token) bool {

func isOperatorToken(tk *token) bool {
	return strings.HasSuffix(string(tk.typ), "Operator")
}

type PathEvaluator struct {
	Path           string
	tokens         tokenList // path criteria, does not change after creation
	Tokens         tokenList // path criteria, consumed during each evaluation
	Error          error     // evaluation status, nil on success
	ContextAtomPtr *Atom
}

// errorf sets the error field in the parser, and indicates that parsing should
// stop by returning false.
func (pe *PathEvaluator) errorf(format string, args ...interface{}) bool {
	pe.Error = errInvalidPath(strings.Join([]string{
		fmt.Sprintf(format, args...),
		fmt.Sprintf(" in %q", pe.Path),
	}, ""))
	return false
}
func (pe *PathEvaluator) addPathToError(err error) error {
	return fmt.Errorf(strings.Join([]string{
		err.Error(),
		fmt.Sprintf(" in %q", pe.Path),
	}, ""))
}

func (pe *PathEvaluator) Evaluate(atom *Atom) (result []*Atom, e error) {
	pe.ContextAtomPtr = atom // FIXME does this change over time?
	pe.Tokens = pe.tokens
	fmt.Println("PathEvaluator::Evaluate() ", pe.Tokens)
	result = pe.eval()
	e = pe.Error
	return
}

// eval() is the top-level evaluation function.  It accepts every token that
// can appear as the first token in the evaluation list.
// Return value is a slice of Atoms, but it may contain just a single node.
func (pe *PathEvaluator) eval() (atoms []*Atom) {
	return pe.evalElementSet()
}

// Done returns true if this PathEvaluator is done processing.
// Completion can occur due to normal consumption of all tokens (success case)
// or due to an error state.
func (pe *PathEvaluator) Done() bool {
	return pe.Error != nil || len(pe.Tokens) == 0
}

// NextTokenType returns the tokenType of the next Token in the PathEvalator's Token stack
func (pe *PathEvaluator) NextTokenType() tokenEnum {
	if len(pe.Tokens) == 0 {
		return ""
	}
	return pe.Tokens.nextType()
}

func (pe *PathEvaluator) evalUnionOperator() []*Atom {
	// consume union operator
	op := pe.Tokens.pop()
	if op.typ != tokenUnionOperator {
		pe.errorf("expected union operator, received type %s", op.typ)
		return nil
	}

	// FIXME uniqueness, avoid duplicates
	return append(pe.evalElementSet(), pe.evalElementSet()...)
}
func (pe *PathEvaluator) evalAxisOperator() []*Atom {
	tk := pe.Tokens.pop()
	if tk.typ != tokenAxisOperator && tk.typ != tokenStepSeparator {
		pe.errorf("expected axis operator, got '%v' [%[1]T]", tk.value)
		return nil
	}
	switch tk.value {
	case "//":
		return pe.ContextAtomPtr.Descendants()
	case "/":
		return []*Atom{pe.ContextAtomPtr}
	}

	// same as / for this implementation..
	// because atoms don't know their parent, it's not possible to refer to a
	// higher-level atom in the tree.  This should be corrected, perhaps Element
	// object should add parent awareness.
	return []*Atom{pe.ContextAtomPtr}
}

func (pe *PathEvaluator) evalNodeTest(atoms []*Atom) []*Atom {
	log.Println("evalNodeTest() ", atoms)
	// Get node test token
	tkNodeTest := pe.Tokens.pop()
	if tkNodeTest.typ != tokenNodeTest {
		pe.errorf("expected node test, got '%v' [%[1]T]", tkNodeTest.value)
		return nil
	}

	if tkNodeTest.value == "*" {
		return atoms
	}

	// Filter the ElementPtrSlice by name against the node test
	results := atoms[:0] // overwite elements list while filtering to avoid allocation
	for _, elt := range atoms {
		fmt.Printf("    evalNodeTest[%q == %q\n", (*elt).Name(), tkNodeTest.value)
		if (*elt).Name() == tkNodeTest.value {
			results = append(results, elt)
		}
	}
	return results
}

func (pe *PathEvaluator) evalElementSet() (atoms []*Atom) {
	log.Println("evalElementSet()")
	if pe.Done() {
		return
	}
	if pe.NextTokenType() == tokenPredicateEnd {
		log.Println("Got predicate")
		return pe.evalPredicate()
	}
	if pe.NextTokenType() == tokenUnionOperator {
		log.Println("Got union operator")
		return pe.evalUnionOperator()
	}
	if pe.NextTokenType() == tokenAxisOperator || pe.NextTokenType() == tokenStepSeparator {
		log.Println("Got axis operator")
		atoms = pe.evalAxisOperator()
	} else {
		log.Println("no axis operator. Next token type is ", pe.NextTokenType())
		// No axis operator given, so use context node
		atoms = append(atoms, pe.ContextAtomPtr)
	}
	// this part is mandatory
	if pe.NextTokenType() != tokenNodeTest {
		log.Println("Got node type operator")
		pe.errorf("element set lacks node test")
		return
	}
	atoms = pe.evalNodeTest(atoms) // may be nil in which case returns .

	return
}

// read predicate tokens.
// read nodeset.
// for each Element in the NodeSet, make a predicateEvaluator and apply predicate.
func (pe *PathEvaluator) evalPredicate() []*Atom {
	log.Println("evalPredicate()")
	// Predicate end comes before pred start, that's the order they're pushed to stack
	// Predicate tokens are in postfix order at this point.
	if pe.Tokens.empty() || pe.Tokens.pop().typ != tokenPredicateEnd {
		pe.errorf("expected predicate end token")
		return nil
	}

	// read predicate tokens
	var predicateTokens tokenList
	for pe.NextTokenType() != tokenPredicateStart && !pe.Tokens.empty() {

		log.Println("evalPredicate(): add token to predicate:", pe.NextTokenType())
		predicateTokens.unshift(pe.Tokens.pop())
	}
	pe.Tokens.pop() // discard predicate start token

	// evaluate element set by predicate
	pre := PredicateEvaluator{
		tokens: predicateTokens,
	}

	atoms, err := pre.Evaluate(pe.evalElementSet())
	if err != nil {
		pe.Error = pe.addPathToError(err)
	}
	return atoms
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
	tokens tokenList // predicate criteria, as a list of tokens
	Error  error     // evaluation status, nil on success

	Tokens   tokenList // Copy of tokens to consume during evaluation
	Atoms    []*Atom   // Atoms being evaluated
	AtomPtr  *Atom     // Atom currently being evaluated from the atom list
	Position int       // index of the atom in the atom list, starts from 1
	Count    int       // number of atoms in the atom list
}

func (pre *PredicateEvaluator) errorf(format string, args ...interface{}) PathError {
	msg := fmt.Sprintf(format, args...)
	pre.Error = PathError(errInvalidPredicate(msg))
	return pre.Error
}

// NextTokenType returns the tokenType of the next Token in the PathEvalator's Token stack
func (pre *PredicateEvaluator) NextTokenType() tokenEnum {
	if len(pre.Tokens) == 0 {
		return ""
	}
	return pre.Tokens.nextType()
}

// evaluate the list of operators/values/stuff against the evaluator's atom/pos/count
func (pre *PredicateEvaluator) eval() (results []Equaler) {
Loop:
	for !pre.Tokens.empty() && pre.Error == nil {
		switch pre.NextTokenType() {
		case tokenPredicateStart:
			break Loop
		case tokenBooleanOperator:
			results = append(results, pre.evalBooleanOperator())
		case tokenEqualityOperator:
			results = append(results, pre.evalEqualityOperator())
		case tokenComparisonOperator:
			results = append(results, pre.evalComparisonOperator())
		case tokenArithmeticOperator:
			results = append(results, pre.evalArithmeticOperator())
		case tokenInteger, tokenHex:
			results = append(results, pre.evalNumber())
		case tokenFunctionBool:
			results = append(results, pre.evalFunctionBool())
		case tokenFunctionNumeric:
			results = append(results, pre.evalFunctionNumeric())
		case tokenBooleanLiteral:
			results = append(results, pre.evalBooleanLiteral())
		default:
			t := pre.Tokens.peek()
			pre.errorf("unrecognized token '%v'", t.value)
			break Loop
		}
	}
	return
}
func (pre *PredicateEvaluator) evalBoolean() (result Equaler) {
	if len(pre.Tokens) == 0 {
		pre.errorf("operator '%s' expects boolean argument, got nothing", pre.tokens[len(pre.tokens)-1].value)
		return
	}
	log.Printf("    evalBoolean() %v,%v", pre.NextTokenType(), pre.Tokens.peek().value)
	switch pre.NextTokenType() {
	case tokenBooleanLiteral:
		result = pre.evalBooleanLiteral()
	case tokenEqualityOperator:
		result = pre.evalEqualityOperator()
	case tokenBooleanOperator:
		result = pre.evalBooleanOperator()
	case tokenComparisonOperator:
		result = pre.evalComparisonOperator().(Equaler)
	case tokenFunctionBool:
		result = pre.evalFunctionBool()
	default:
		t := pre.Tokens.peek()
		pre.errorf("expect boolean, got '%s'", t.value)
		return
	}
	// calculate a boolean value from op and vars
	return
}
func (pre *PredicateEvaluator) evalBooleanLiteral() BooleanType {
	t := pre.Tokens.pop()
	if t.typ != tokenBooleanLiteral {
		pre.errorf("expected boolean literal, received type %s", t.typ)
		return BooleanType(false)
	}
	switch t.value {
	case "true":
		return BooleanType(true)
	case "false":
		return BooleanType(false)
	default:
		pre.errorf("unknown boolean literal %s", t.value)
	}
	return BooleanType(false)
}

func (pre *PredicateEvaluator) evalBooleanOperator() BooleanType {
	op := pre.Tokens.pop()
	if op.typ != tokenBooleanOperator {
		pre.errorf("expected boolean operator, received type %s", op.typ)
	}
	results := []Equaler{pre.evalBoolean(), pre.evalBoolean()}
	if len(results) != 2 {
		pre.errorf("boolean '%s' expected 2 results to compare, got %d", op.value, len(results))
		return BooleanType(false)
	}
	tru := BooleanType(true)
	var result bool
	switch op.value {
	case "and":
		result = results[0] == tru && results[1] == tru
	case "or":
		result = results[0] == tru || results[1] == tru
	default:
		pre.errorf("unknown boolean operator: %s", op.value)
	}
	return BooleanType(result)
}

func (pre *PredicateEvaluator) True(v Equaler) bool {
	return v.Equal(BooleanType(true))
}

// Numeric operators. All have arity 2.  Must handle float and int types.  Assumed to be signed.
func (pre *PredicateEvaluator) evalArithmeticOperator() (result Arithmeticker) {
	op := pre.Tokens.pop()
	if op.typ != tokenArithmeticOperator {
		pre.errorf("expected tokenArithmeticOperator, received type %s", op.typ)
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
func (pre *PredicateEvaluator) evalEqualityOperator() Equaler {
	var result bool
	op := pre.Tokens.pop()
	if op.typ != tokenEqualityOperator {
		pre.errorf("expected tokenEqualityOperator, received type %s", op.typ)
		return BooleanType(false)
	}
	rhs := pre.evalEqualer()
	lhs := pre.evalEqualer()
	if pre.Error != nil {
		return BooleanType(false)
	}
	log.Printf("  evalEqualityOperator() %v = %v", lhs, rhs)
	switch op.value {
	case "=", "eq":
		result = lhs.Equal(rhs)
	case "!=", "ne":
		result = !lhs.Equal(rhs)
	default:
		pre.errorf("unknown equality operator: %s", op.value)
		result = false
	}
	return BooleanType(result)
}
func (pre *PredicateEvaluator) evalComparisonOperator() Equaler {
	var result bool
	op := pre.Tokens.pop()
	if op.typ != tokenComparisonOperator {
		pre.errorf("expected tokenComparisonOperator, received type %s", op.typ)
		return BooleanType(false)
	}
	rhs := pre.evalComparable()
	lhs := pre.evalComparable()
	if pre.Error != nil {
		return BooleanType(false)
	}
	switch op.value {
	case "<", "lt":
		result = lhs.LessThan(rhs)
	case ">", "gt":
		result = lhs.GreaterThan(rhs)
	case "<=", "le":
		result = lhs.LessThan(rhs) || lhs.Equal(rhs)
	case ">=", "ge":
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
	switch pre.NextTokenType() {
	case tokenInteger, tokenHex:
		v, err := strconv.ParseInt(pre.Tokens.pop().value, 0, 64)
		if err != nil {
			pre.errorf(err.Error())
			return
		}
		result = Int64Type(v)
	case tokenFloat:
		v, err := strconv.ParseFloat(pre.Tokens.pop().value, 64)
		if err != nil {
			pre.errorf(err.Error())
			return
		}
		result = Float64Type(v)
	case tokenFunctionNumeric:
		result = pre.evalFunctionNumeric()
	case tokenVariable:
		result, ok = pre.evalVariable().(Arithmeticker)
	case tokenArithmeticOperator:
		result = pre.evalArithmeticOperator()
	case tokenBareString:
		t := pre.Tokens.pop()
		if v, ok := pre.getChildValue(t.value); ok {
			result = v.(Arithmeticker)
		} else {
			pre.errorf("expect number, got %s", t.value)
		}
	default:
		pre.errorf("value has invalid numeric type: %s", pre.NextTokenType())
	}
	if err != nil || !ok {
		pre.errorf("expected numeric value")
	}
	return result
}
func (pre *PredicateEvaluator) evalEqualer() (result Equaler) {
	log.Printf("    evalEqualer(), Tokens=%v", pre.Tokens)
	var err error
	switch pre.NextTokenType() {
	case tokenInteger, tokenHex:
		v, e := strconv.ParseInt(pre.Tokens.pop().value, 0, 64)
		err = e
		result = Int64Type(v)
	case tokenFloat:
		v, e := strconv.ParseFloat(pre.Tokens.pop().value, 64)
		err = e
		result = Float64Type(v)
	case tokenBooleanLiteral:
		result = pre.evalBooleanLiteral()
	case tokenBareString:
		t := pre.Tokens.pop()
		if v, ok := pre.getChildValue(t.value); ok {
			result = v // string is Atom name, substitute atom value.
		} else {
			result = StringType(t.value)
		}
	case tokenString:
		result = StringType(pre.Tokens.pop().value)
	case tokenVariable:
		result = pre.evalVariable()
	case tokenFunctionNumeric:
		result = pre.evalFunctionNumeric()
	case tokenArithmeticOperator:
		result = pre.evalArithmeticOperator()
	default:
		t := pre.Tokens.pop()
		pre.errorf("expected Equaler type, got %q [%s])", t.value, t.typ)
		return
	}
	if err != nil {
		pre.errorf("failed to convert '%s' to Equaler value")
		return
	}
	return result
}

// FIXME: this near-duplicates evalEqualer.
// have tk call evalEqualer and then error out on non-Compararer types?
func (pre *PredicateEvaluator) evalComparable() (result Comparer) {
	log.Printf("    evalComparable(), Tokens=%v", pre.Tokens)
	var err error
	switch pre.NextTokenType() {
	case tokenInteger, tokenHex:
		v, e := strconv.ParseInt(pre.Tokens.pop().value, 0, 64)
		err = e
		result = Int64Type(v)
	case tokenFloat:
		v, e := strconv.ParseFloat(pre.Tokens.pop().value, 64)
		err = e
		result = Float64Type(v)
	case tokenBareString:
		t := pre.Tokens.pop()
		if v, ok := pre.getChildValue(t.value); ok {
			result = v // string is Atom name, substitute atom value.
		} else {
			result = StringType(t.value)
		}
	case tokenString:
		result = StringType(pre.Tokens.pop().value)
	case tokenVariable:
		result = pre.evalVariable()
	case tokenFunctionNumeric:
		result = pre.evalFunctionNumeric()
	case tokenArithmeticOperator:
		result = pre.evalArithmeticOperator()
	default:
		t := pre.Tokens.pop()
		pre.errorf("expected comparable type, got %s(%v)", t.typ, t.value)
		return
	}
	if err != nil {
		pre.errorf("failed to convert '%s' to comparable value")
		return
	}
	return result
}
func (pre *PredicateEvaluator) evalVariable() (result Comparer) {
	token := pre.Tokens.pop()
	if token.typ != tokenVariable {
		pre.errorf("expected tokenVariable, received type %s", token.typ)
	}
	switch token.value {
	case "@name", "name":
		return StringType(pre.AtomPtr.Name())
	case "@name_int32":
		name, err := strconv.ParseUint(pre.AtomPtr.name, 0, 32)
		if err != nil {
			pre.errorf("invalid atom @name_int32: %s", pre.AtomPtr.Name)
			return
		}
		return Uint64Type(name)
	case "@type", "type":
		return StringType(pre.AtomPtr.Type())
	case "@data", "data":
	default:
		pre.errorf("unknown variable: %s", token.value)
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
		v, _ := pre.AtomPtr.Value.Uint() // use UINT since tk's represented as 0/1
		result = Uint64Type(v)
	default:
		v, _ := pre.AtomPtr.Value.String()
		result = StringType(v)
	}
	return result
}
func (pre *PredicateEvaluator) evalFunctionBool() Equaler {
	var result bool
	token := pre.Tokens.pop()
	if token.typ != tokenFunctionBool {
		pre.errorf("expected tokenFunctionBool, received type %s", token.typ)
	}
	switch token.value {
	case "not":
		r := pre.evalBoolean()
		if r == nil {
			result = false
		} else {
			result = r.Equal(BooleanType(false))
		}
	default:
		pre.errorf("unknown boolean function: %s", token.value)
	}
	return BooleanType(result)
}

// evalUnionOperator consumes two nodesets and returns their set union.
func (pre *PredicateEvaluator) evalUnionOperator(results []Equaler) (result bool) {
	fmt.Println("evalUnionOperator()")

	// consume union operator
	op := pre.Tokens.pop()
	if op.typ != tokenUnionOperator {
		pre.errorf("expected union operator, received type %s", op.typ)
	}

	// operator requires expressions on both sides, so error if this is the first token
	if pre.tokens[0].typ == tokenUnionOperator {
		pre.errorf("| has no left-hand-side value")
		return false
	}

	// evaluate results so far, same as if predicate was fully evaluated
	var err error
	result, err = pre.evalResultsToBool(results)
	if err != nil {
		// reword error message to include reference to union operator
		errString := err.Error()
		switch {
		case errString == "no result":
			pre.errorf("| has no right-hand-side value")
		case strings.Contains(errString, "unparsed values "):
			pre.errorf("| has multiple right-hand-side values")
		case strings.Contains(errString, "unknown type"):
			pre.errorf(strings.Join([]string{"| expects boolean,", errString}, " "))
		default:
			pre.errorf(errString)
		}
	}
	return
}

func (pre *PredicateEvaluator) evalResultsToBool(results []Equaler) (result bool, err error) {
	if pre.Error != nil {
		return false, pre.Error
	}
	// verify that evaluation resulted in exactly 1 value
	switch len(results) {
	case 0:
		err = fmt.Errorf("no result")
		return
	case 1:
	default:
		err = fmt.Errorf("unparsed values '%v'", results)
		return
	}
	// verify that evaluation resulted in a usable type
	switch r := results[0].(type) {
	case BooleanType:
		result = bool(r)
	case Int64Type:
		result = r.Equal(Int64Type(pre.Position))
	case Uint64Type:
		result = r.Equal(Uint64Type(pre.Position))
	case Float64Type:
		result = r.Equal(Float64Type(pre.Position))
	default:
		err = fmt.Errorf("result '%v' has unknown type %[1]T", results[0])
		return
	}
	return
}
func (pre *PredicateEvaluator) evalFunctionNumeric() (result Arithmeticker) {
	token := pre.Tokens.pop()
	if token.typ != tokenFunctionNumeric {
		pre.errorf("expected tokenFunctionNumeric, received type %s", token.typ)
		return
	}
	switch token.value {
	case "position":
		log.Printf(`    evalFunctionNumeric("%s") = %d`, token.value, pre.Position)
		return Uint64Type(pre.Position)
	case "last", "count":
		log.Printf(`    evalFunctionNumeric("%s") = %d`, token.value, pre.Count)
		return Uint64Type(pre.Count)
	default:
		pre.errorf("unknown numeric function: %s", token.value)
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
	panic(fmt.Sprintf("modulus not supported for type %T, value '%[1]v'", other))
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
	fmt.Println("Uint64Typ::Mod", v, other)
	switch o := other.(type) {
	case Float64Type:
		return Float64Type(math.Mod(float64(v), float64(o)))
	case Int64Type:
		return Int64Type(int64(v) % int64(o))
	case Uint64Type:
		return Uint64Type(uint64(v) % uint64(o))
	}
	panic(fmt.Sprintf("modulus not supported for type %T value'%[1]v'", other))
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
	switch o := other.(type) {
	case Float64Type:
		return Float64Type(math.Mod(float64(v), float64(o)))
	case Int64Type:
		return Int64Type(int64(v) % int64(o))
	case Uint64Type:
		if v < 0 {
			return Int64Type(int64(v) % int64(o))
		} else {
			return Uint64Type(uint64(v) % uint64(o))
		}
	}
	panic(fmt.Sprintf("modulus not supported for type %T value '%[1]v'", other))
}
func (v BooleanType) Equal(other Equaler) bool {
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
func isNumericToken(tk tokenEnum) bool {
	return tk == tokenInteger || tk == tokenFloat
}

// These values are from the XPath 3.1 operator precedence table at
//   https://www.w3.org/TR/xpath-3/#id-precedence-order
// Not all of these operators are actually implemented.
func precedence(tk token) (value int) {
	switch tk.value {
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
	case "div", "idiv", "mod":
		value = 9
	case "*":
		if tk.typ == tokenArithmeticOperator {
			value = 9
		} else {
			value = 16 // node test operator
		}
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
	case "[", "]": // needs lower precedence that node test and axis operators
		value = 20
	case "cast as":
		value = 15
	//		case "-", "+": // unary operators
	//			value = 16
	case "!":
		value = 17
	case "/", "//":
		value = 19
	default:
		value = 0
		//		panic(fmt.Sprintf("unknown operator: %s", op))
	}
	return value
}

/*
(//*[@name="0x00000000"] | //*[@name="0x00000001"])[data() = 2]

|
  //
	*
	[
	  =
		  @name
			"0x00000000"
	]

  //
	*
	[
	  =
	    @name
		  "0x00000001"
	]
[
	=
		data()
		2
]

(//*[@name="0x00000000"] | //*[@name="0x00000001"])[data() = 2]
  0.            @name  [iVar]
  1.       0x00000000  [iString]
  2.                ]  [iPredEnd]
  3.                =  [iEqualsOperator]
  4.                [  [iPredStart]
  5.                *  [iNodeTest]
  6.               //  [iAxisOperator]
  7.            @name  [iVar]
  8.       0x00000001  [iString]
  9.                ]  [iPredEnd]
 10.                =  [iEqualsOperator]
 11.                [  [iPredStart]
 12.                *  [iNodeTest]
 13.               //  [iAxisOperator]
 14.                |  [iUnionOperator]
 15.             data  [iVar]
 16.                2  [iInt]
 17.                ]  [iPredEnd]
 18.                =  [iEqualsOperator]
 19.                [  [iPredStart]



["@name" "0x00000000" "]" "=" "[" "*" "//" "@name" "0x00000001" "]" "=" "[" "*" "//" "|" "data" "2" "]" "=" "["]

*/

/*
type Element struct {
	a ElementMimicker
}
interface ElementMimic {
	Id() // unique id, or ???
	Children()
	Type()
	Name()
	Data()
	Descendants() // flat list of all descendants in hierarchical order
	String()
}
*/
