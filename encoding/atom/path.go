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

// FIXME explain terminology, and lexer/parser/evaluators relationship
// FIXME implement path.Compile
import (
	"fmt"
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
	tokenAxisOperator       = "tknAxisOperator" // "/" or "//" which precedes pathTest
	tokenArithmeticOperator = "tknArithmeticOperator"
	tokenBooleanOperator    = "tknBooleanOperator"
	tokenEqualityOperator   = "tknEqualsOperator"
	tokenNodeTest           = "tknNodeTest"
	tokenComparisonOperator = "tknCompareOperator"
	tokenSetOperator        = "tknSetOperator"
	tokenStepSeparator      = "tknStepSeparatorOperator"
	tokenOperator           = "tknOperator"
	tokenFunctionBool       = "tknFunctionBool"
	tokenFunctionNumeric    = "tknFunctionNum"
	tokenVariable           = "tknVar"
	tokenInteger            = "tknInt"
	tokenFloat              = "tknFloat"
	tokenBareString         = "tknBareString"
	tokenHex                = "tknHex"
)

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

// add a new token to the front of the queue, as the new first lement
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
	Evaluator *PathEvaluator
	err       error
}

// NewAtomPath creates an AtomPath object for the given path string.  It
// performs all lexing and parsing steps, so that evaluating data sets against
// the path will have as little overhead as possible.
func NewAtomPath(path string) (ap *AtomPath, e error) {
	Log.Printf("NewAtomPath(%q)", path)

	var pe *PathEvaluator
	pe, e = NewPathEvaluator(path)
	if e != nil {
		return ap, addPathToError(e, path)
	}

	ap = &AtomPath{
		Path:      strings.TrimSpace(path),
		Evaluator: pe,
		err:       nil,
	}
	return
}

// AtomsAtPath returns the set of descendant atoms in atom a which match the
// given path.
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

// NewPredicateEvaluator consumes a series of predicate tokens from a
// PathEvaluator starting with a PredicateEnd token and ending with a
// PredicateStart token (yes it's supposed to be backwards), and returns a new
// PredicateEvaluator.
func NewPredicateEvaluator(pe *PathEvaluator) (pre PredicateEvaluator, ok bool) {
	// Predicate end comes before pred start, that's the order they're pushed to stack
	// Predicate tokens are in postfix order at this point.
	if pe.Tokens.empty() || pe.Tokens.pop().typ != tokenPredicateEnd {
		pe.errorf("expected predicate end token")
		return pre, false
	}

	// read predicate tokens
	var predicateTokens tokenList
	for pe.NextTokenType() != tokenPredicateStart && !pe.Tokens.empty() {
		predicateTokens.unshift(pe.Tokens.pop())
	}
	pe.Tokens.pop() // discard predicate start token

	// check for predicate with no tokens
	if len(predicateTokens) == 0 {
		pe.Error = addPathToError(errInvalidPredicate("empty predicate"), pe.Path)
		return pre, false
	}

	// evaluate element set by predicate
	return PredicateEvaluator{
		tokens: predicateTokens,
	}, true
}

// Evaluate filters a list of atoms against the predicate conditions, returning
// the atoms that satisfy the predicate.
//
// The candidate atoms must all be made available to the PredicateEvaluator at
// once, because the predicate may refer to individual child atoms by name,
// requiring them to be evaluated against every other candidate.
func (pre *PredicateEvaluator) Evaluate(candidates []*Atom) (atoms []*Atom, e error) {
	Log.Print("PredicateEvaluator::Evaluate()  ", pre.tokens, candidates)
	pre.Atoms = candidates
	pre.Count = len(candidates)
	for i, atomPtr := range candidates {
		pre.Position = i + 1 // XPath convention, indexing starts at 1
		pre.AtomPtr = atomPtr
		pre.Tokens = pre.tokens

		// eval candidate atoms against path+predicate(s)
		Log.Printf("    eval() with Tokens(%v)", pre.tokens)
		results := pre.eval()
		ok, err := pre.evalResultsToBool(results)
		Log.Printf("    eval() returned %t and err %v", ok, pre.Error)
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
		if a.Name() != atomName {
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
		l.emit(tokenSetOperator)
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
		lexBareStringInPath(l)
	default:
		return l.errorf("operator %q is not valid within path", r)
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
		lexBareStringInPredicate(l)
	default:
		return l.errorf("operator %q is not valid within predicate", r)
	}

	return lexPredicate
}

func lexStepSeparatorOrAxis(l *lexer) stateFn {
	if l.first() != '/' {
		return l.errorf(`lexStepSeparatorOrAxis called without leading "/"`)
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
	if l.acceptRun(alphaNumericChars) > 0 {
		l.emit(tokenNodeTest)
	} else if l.accept("*") {
		l.emit(tokenNodeTest)
	} else {
		l.errorf("expected node test, found none")
		return nil
	}
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
		l.errorf("lexAtomAttribute called without leading attribute sigil @")
		return nil
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

	// Warning: if any functions are added with names containing something besides lower
	// case chars, then update lexBareString to accept those chars as well
	switch l.buffer() { // determine function return type
	case "not":
		l.emit(tokenFunctionBool)
	case "true", "false":
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

func lexBareStringInPath(l *lexer) stateFn {
	l.acceptRun(alphaNumericChars)
	if l.peek() == '(' {
		return lexFunctionCall(l)
	}
	switch l.buffer() {
	case "union", "intersect":
		l.emit(tokenSetOperator)
	default:
		l.emit(tokenNodeTest)
	}
	return lexPath
}

// lexBareString accepts a non-delimited string of alphanumeric characters.
// This has more restrictions than a delimited string but is simple and fast to
// parse.
// Doesn't handle any escaping, use delimited strings for anything non-trivial.
func lexBareStringInPredicate(l *lexer) stateFn {
	l.acceptRun(alphaNumericChars)
	if l.peek() == '(' {
		return lexFunctionCall(l)
	}
	switch l.buffer() {
	case "eq", "ne":
		l.emit(tokenEqualityOperator)
	case "lt", "le", "gt", "ge":
		l.emit(tokenComparisonOperator)
	case "div", "idiv", "mod":
		l.emit(tokenArithmeticOperator)
	case "or", "and":
		l.emit(tokenBooleanOperator)

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

// receiveTokens gets tokens from the lexer and sends them to the parser
// for parsing.
func (pp *PathParser) receiveTokens() {
	for {
		tk := pp.readToken()
		ok := pp.parseToken(tk)

		str := fmt.Sprintf("parseToken(%s, '%v')", tk.typ, tk.value)
		Log.Printf("    %-35s %35v | %v\n", str, fmt.Sprint(pp.opStack), pp.outputQueue)

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
	Log.Printf("      parseToken %q {%s} ", tk.value, tk.typ)
	switch tk.typ {
	case tokenError:
		return pp.errorf(tk.value)
	case tokenInteger, tokenHex, tokenFloat, tokenBareString, tokenString, tokenVariable:
		pp.outputQueue.push(&tk)
	case tokenPredicateStart:
		pp.moveOperatorsToOutputUntil(func(t token) bool {
			// want predicate delimiter tokens to precede axis+node test in stack order
			return (t.typ != tokenAxisOperator) && (t.typ != tokenNodeTest) && (t.typ != tokenStepSeparator)
		})
		// push to both.  Only the output queue copy will be kept.
		pp.outputQueue.push(&tk)
		pp.opStack.push(&tk)
	case tokenFunctionBool, tokenFunctionNumeric:
		pp.opStack.push(&tk)
	case tokenNodeTest: // act like an operator with same precedence as //, /
		pp.moveOperatorsToOutput(token{tokenStepSeparator, "/", 0})
		pp.outputQueue.push(&tk)
	case tokenComparisonOperator, tokenArithmeticOperator, tokenEqualityOperator, tokenBooleanOperator, tokenAxisOperator, tokenStepSeparator, tokenSetOperator:
		pp.moveOperatorsToOutput(tk)
		pp.opStack.push(&tk)
	case tokenLeftParen:
		pp.opStack.push(&tk)
	case tokenRightParen:
		pp.moveOperatorsToOutputUntil(func(t token) bool { return t.typ == tokenLeftParen })
		pp.opStack.pop() // remove the matching LeftParen from the stack
		if isFunctionToken(pp.opStack.peek()) {
			pp.outputQueue.push(pp.opStack.pop()) // move completed function call to output
		}
	case tokenPredicateEnd:
		pp.moveOperatorsToOutputUntil(func(t token) bool { return t.typ == tokenPredicateStart })
		//		pp.moveOperatorsToOutput(tk)
		//		pp.opStack.pop() // discard PredicateStart from opstack. There's already one on the output queue, so no push()
		pp.opStack.pop() // remove predicate start from operator stack
		pp.outputQueue.push(&tk)
		//	case tokenSetOperator:
		//		pp.moveOperatorsToOutputUntil(func(t token) bool { return t.typ == tokenLeftParen })
		//		pp.opStack.push(&tk)
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

// moveOperatorsToOutputUntil() pops tokens from the operator stack, and pushes
// them into the output queue. This continues until the given test function
// fails to satisfy the given test function.
// The failed end token is left on the operator stack.
func (pp *PathParser) moveOperatorsToOutputUntil(test func(t token) bool) {
	for {
		if pp.opStack.empty() {
			break
		}
		nextToken := (*pp).opStack.peek()
		if test(*nextToken) {
			break
		}
		pp.outputQueue.push(pp.opStack.pop())
	}
}

// moveOperatorsToOutput implements this part of the Shunting-yard algorithm:

//  while there is an operator token o2, at the top of the operator stack and either
//    o1 is left-associative and its precedence is less than or equal to that of o2, or
//    o1 is right associative, and has precedence less than that of o2,
//        pop o2 off the operator stack, onto the output queue;
func (pp *PathParser) moveOperatorsToOutput(tk token) {
	tkPrec, tAssoc := operatorOrder(tk)
	for {
		if pp.opStack.empty() {
			break
		}

		nextToken := pp.opStack.peek()
		if !isOperatorToken(nextToken) {
			break
		}
		p, _ := operatorOrder(*nextToken)
		Log.Println("        opStack before: ", pp.opStack)
		//if (tAssoc == assocLeft && tkPrec <= p) || (tAssoc == assocRight && tkPrec < p) {
		if tkPrec < p || (tAssoc == assocLeft && tkPrec == p) {
			pp.outputQueue.push(pp.opStack.pop())
		} else {
			break
		}
	}
	Log.Println("        opStack after: ", pp.opStack)
}

// func (pp *PathParser) parsePredicateTokens(tk token) bool {

func isOperatorToken(tk *token) bool {
	if tk == nil {
		return false
	}
	return strings.HasSuffix(string(tk.typ), "Operator")
}
func isFunctionToken(tk *token) bool {
	if tk == nil {
		return false
	}
	return strings.Contains(string(tk.typ), "Function")
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
func addPathToError(err error, path string) error {
	return fmt.Errorf(strings.Join([]string{
		err.Error(),
		fmt.Sprintf(" in %q", path),
	}, ""))
}

func (pe *PathEvaluator) Evaluate(atom *Atom) (result []*Atom, e error) {
	pe.ContextAtomPtr = atom // FIXME does this change over time?
	if pe.tokens.empty() {
		e = errInvalidPath("<empty>")
		return
	}

	// Special case, otherwise path specifiers may not end with /
	if len(pe.tokens) == 1 && pe.tokens[0].value == "/" {
		return []*Atom{pe.ContextAtomPtr}, nil
	}

	pe.Tokens = pe.tokens
	Log.Println("PathEvaluator::Evaluate() ", pe.Tokens)
	result = pe.evalElementSet()
	e = pe.Error
	return
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

func (pe *PathEvaluator) evalSetOperator() (atoms []*Atom) {
	Log.Println("evalSetOperator()")
	op := pe.Tokens.pop()
	if op.typ != tokenSetOperator {
		pe.errorf("expected union or intersect operator, received type %s", op.typ)
		return nil
	}

	switch op.value {
	case "union", "|":
		atoms = append(pe.evalElementSet(), pe.evalElementSet()...)
	case "intersect":
		// hash elements in first set
		var zero struct{}
		var eltMap = make(map[string]struct{})
		for _, a := range pe.evalElementSet() {
			eltMap[a.String()] = zero
		}
		// find elements in second set that are in first set
		for _, a := range pe.evalElementSet() {
			if _, ok := eltMap[a.String()]; ok {
				atoms = append(atoms, a)
			}
		}
	}
	return
}
func (pe *PathEvaluator) evalAxisOperator() (atoms []*Atom) {
	tk := pe.Tokens.pop()
	Log.Printf("evalAxisOperator(%q)", tk.value)
	if tk.typ != tokenAxisOperator && tk.typ != tokenStepSeparator {
		pe.errorf("expected axis operator, got '%v' [%[1]T]", tk.value)
		return nil
	}

	//	if pe.NextTokenType() == tokenNodeTest {
	//		pe.errorf("operator '%s' must be followed element name or *", tk.value)
	//		return nil
	//	}

	if tk.value == "//" {
		atoms = pe.ContextAtomPtr.Descendants()
		return
	} else if tk.value == "/" {
		atoms = []*Atom{pe.ContextAtomPtr}
		return
	}

	// The empty case is the same as / for this implementation..
	// because atoms don't know their parent, it's not possible to refer to a
	// higher-level atom in the tree.
	return []*Atom{pe.ContextAtomPtr}
}

// evaluate path expression starting with a node test.
// There's no preceding axis operator.
func (pe *PathEvaluator) evalNodeTest() (atoms []*Atom) {
	// Get node test token
	tkNodeTest := pe.Tokens.pop()
	if tkNodeTest.typ != tokenNodeTest {
		pe.errorf("expected node test, got '%v' [%[1]T]", tkNodeTest.value)
		return nil
	}

	// Get element set to filter
	if pe.NextTokenType() == tokenStepSeparator {
		// New path step, so apply nodeTest to the children of whatever atoms are
		// returned by the path  expression following the step separator operator
		pe.Tokens.pop()
		if pe.Tokens.empty() {
			pe.errorf("expected path elements after /")
			return nil
		}
		for _, a := range pe.evalElementSet() {
			atoms = append(atoms, a.Children()...)
		}
	} else if pe.NextTokenType() == tokenAxisOperator {
		atoms = pe.evalAxisOperator()
	} else {
		atoms = append(atoms, pe.ContextAtomPtr)
	}
	Log.Printf("evalNodeTest(%q) %v", tkNodeTest.value, atoms)

	// Filter the ElementPtrSlice by name against the node test
	if tkNodeTest.value == "*" {
		return atoms
	}
	results := atoms[:0] // overwite elements list while filtering to avoid allocation
	for _, elt := range atoms {
		if (*elt).Name() == tkNodeTest.value {
			results = append(results, elt)
		}
	}
	return results
}

func (pe *PathEvaluator) evalElementSet() (atoms []*Atom) {
	Log.Printf("evalElementSet() [%s]'", pe.NextTokenType())
	if pe.Done() {
		return
	}
	switch pe.NextTokenType() {
	case tokenPredicateEnd:
		return pe.evalPredicate()
	case tokenSetOperator:
		return pe.evalSetOperator()
	case tokenAxisOperator, tokenStepSeparator:
		tk := pe.Tokens.pop()
		pe.errorf("operator '%s' must be followed by element name or *", tk.value)
		return
		//		return pe.evalAxisOperator()
	case tokenNodeTest:
		return pe.evalNodeTest() // may be nil in which case returns .
	}

	Log.Println("evalElementSet: Returning context atom. Next Token Type is ", pe.NextTokenType())
	// No axis operator given, so use context node
	atoms = append(atoms, pe.ContextAtomPtr)
	return
}

// read predicate tokens.
// read nodeset.
// for each Element in the NodeSet, make a predicateEvaluator and apply predicate.
func (pe *PathEvaluator) evalPredicate() []*Atom {
	Log.Println("evalPredicate()")
	// evaluate element set by predicate
	pre, ok := NewPredicateEvaluator(pe)
	if ok != true {
		return nil // error is already set by NewPredicateEvaluator
	}
	atoms, err := pre.Evaluate(pe.evalElementSet())
	if err != nil {
		pe.Error = addPathToError(err, pe.Path)
		return nil
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
		default:
			t := pre.Tokens.peek()
			pre.errorf("unrecognized token '%v'", t.value)
			break Loop
		}
	}
	return
}
func (pre *PredicateEvaluator) evalBoolean() (result Equaler) {
	if pre.Tokens.empty() {
		pre.errorf("expect boolean value, got nothing")
		return
	}
	Log.Printf("    evalBoolean() %v,%v", pre.NextTokenType(), pre.Tokens.peek().value)
	switch pre.NextTokenType() {
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

func (pre *PredicateEvaluator) evalBooleanOperator() BooleanType {
	op := pre.Tokens.pop()
	if op.typ != tokenBooleanOperator {
		pre.errorf("expected boolean operator, received type %s", op.typ)
	}
	results := []Equaler{pre.evalBoolean(), pre.evalBoolean()}
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
	var err error
	switch op.value {
	case "+":
		result, err = lhs.Plus(rhs)
	case "-":
		result, err = lhs.Minus(rhs)
	case "*":
		result, err = lhs.Multiply(rhs)
	case "div":
		result, err = lhs.Divide(rhs)
	case "idiv":
		result, err = lhs.IntegerDivide(rhs)
	case "mod":
		result, err = lhs.Mod(rhs)
	default:
		pre.errorf("unknown arithmetic operator: %s", op.value)
		return
	}
	if err != nil {
		pre.errorf(err.Error())
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
	Log.Printf("  evalEqualityOperator() %v = %v", lhs, rhs)
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
	Log.Printf("    evalEqualer(), Tokens=%v", pre.Tokens)
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
	case tokenEqualityOperator:
		result = pre.evalEqualityOperator()
	case tokenVariable:
		result = pre.evalVariable()
	case tokenFunctionNumeric:
		result = pre.evalFunctionNumeric()
	case tokenArithmeticOperator:
		result = pre.evalArithmeticOperator()
	case tokenFunctionBool:
		result = pre.evalFunctionBool()
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
	Log.Printf("    evalComparable(), Tokens=%v", pre.Tokens)
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
	case "@name_hex":
		return StringType(fmt.Sprint("0x%08X", pre.AtomPtr.NameAsUint32()))
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
	case "true":
		result = true
	case "false":
		result = false
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
		Log.Printf(`    evalFunctionNumeric("%s") = %d`, token.value, pre.Position)
		return Uint64Type(pre.Position)
	case "last", "count":
		Log.Printf(`    evalFunctionNumeric("%s") = %d`, token.value, pre.Count)
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
		Plus(other Arithmeticker) (Arithmeticker, error)
		Minus(other Arithmeticker) (Arithmeticker, error)
		Multiply(other Arithmeticker) (Arithmeticker, error)
		Divide(other Arithmeticker) (Arithmeticker, error)
		IntegerDivide(other Arithmeticker) (Arithmeticker, error)
		Mod(other Arithmeticker) (Arithmeticker, error)
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
	case StringType:
		if fp, err := strconv.ParseFloat(string(o), 64); err != nil {
			return false
		} else {
			return float64(v) == fp
		}
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
	case StringType:
		if o_uint, err := strconv.ParseUint(string(o), 0, 64); err != nil {
			return false
		} else {
			return uint64(v) == o_uint
		}
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
		if x, e := strconv.ParseFloat(string(o), 64); e == nil {
			return float64(v) > x
		}
		if x, e := strconv.ParseUint(string(o), 10, 64); e == nil {
			return uint64(v) > x
		}
		if x, e := strconv.ParseInt(string(o), 0, 64); e == nil {
			return int64(v) > x
		}
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
		if x, e := strconv.ParseFloat(string(o), 64); e == nil {
			return float64(v) < x
		}
		if x, e := strconv.ParseUint(string(o), 10, 64); e == nil {
			return uint64(v) < x
		}
		if x, e := strconv.ParseInt(string(o), 0, 64); e == nil {
			return int64(v) < x
		}
	case Uint64Type:
		return v < Int64Type(o)
	default:
		return v < o.(Int64Type)
	}
	return false
}
func (v Int64Type) GreaterThan(other Comparer) bool {
	switch o := other.(type) {
	case Float64Type:
		return Float64Type(v) > o
	case Uint64Type:
		return v > Int64Type(o)
	case StringType:
		if x, e := strconv.ParseFloat(string(o), 64); e == nil {
			return float64(v) > x
		}
		if x, e := strconv.ParseUint(string(o), 10, 64); e == nil {
			return uint64(v) > x
		}
		if x, e := strconv.ParseInt(string(o), 0, 64); e == nil {
			return int64(v) > x
		}
	default:
		return v > o.(Int64Type)
	}
	return false
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
func (v Float64Type) Plus(other Arithmeticker) (result Arithmeticker, err error) {
	switch o := other.(type) {
	case Float64Type:
		result = Float64Type(float64(v) + float64(o))
	case Int64Type:
		result = Float64Type(float64(v) + float64(o))
	case Uint64Type:
		result = Float64Type(float64(v) + float64(o))
	default:
		err = fmt.Errorf("addition not supported for type %T, value '%[1]v'", other)
	}
	return
}
func (v Float64Type) Minus(other Arithmeticker) (result Arithmeticker, err error) {
	switch o := other.(type) {
	case Float64Type:
		result = Float64Type(float64(v) - float64(o))
	case Int64Type:
		result = Float64Type(float64(v) - float64(o))
	case Uint64Type:
		result = Float64Type(float64(v) - float64(o))
	default:
		err = fmt.Errorf("multiplication not supported for type %T, value '%[1]v'", other)
	}
	return
}
func (v Float64Type) Multiply(other Arithmeticker) (result Arithmeticker, err error) {
	switch o := other.(type) {
	case Float64Type:
		result = Float64Type(float64(v) * float64(o))
	case Int64Type:
		result = Float64Type(float64(v) * float64(o))
	case Uint64Type:
		result = Float64Type(float64(v) * float64(o))
	default:
		err = fmt.Errorf("multiplication not supported for type %T, value '%[1]v'", other)
	}
	return
}
func (v Float64Type) Divide(other Arithmeticker) (result Arithmeticker, err error) {
	switch other := other.(type) {
	case Float64Type:
		result = Float64Type(float64(v) / float64(other))
	case Int64Type:
		result = Float64Type(float64(v) / float64(other))
	case Uint64Type:
		result = Float64Type(float64(v) / float64(other))
	default:
		err = fmt.Errorf("division not supported for type %T, value '%[1]v'", other)
	}
	return
}
func (v Float64Type) IntegerDivide(other Arithmeticker) (result Arithmeticker, err error) {
	switch other := other.(type) {
	case Float64Type:
		result = Int64Type(int64(v) / int64(other))
	case Int64Type:
		result = Int64Type(int64(v) / int64(other))
	case Uint64Type:
		result = Int64Type(int64(v) / int64(other))
	default:
		err = fmt.Errorf("integer division not supported for type %T, value '%[1]v'", other)
	}
	return
}
func (v Float64Type) Mod(other Arithmeticker) (result Arithmeticker, err error) {
	switch other := other.(type) {
	case Float64Type:
		result = Float64Type(math.Mod(float64(v), float64(other)))
	case Int64Type:
		result = Float64Type(math.Mod(float64(v), float64(other)))
	case Uint64Type:
		result = Float64Type(math.Mod(float64(v), float64(other)))
	default:
		err = fmt.Errorf("modulus not supported for type %T, value '%[1]v'", other)
	}
	return
}
func (v Uint64Type) Plus(other Arithmeticker) (result Arithmeticker, err error) {
	switch other := other.(type) {
	case Float64Type:
		result = Float64Type(v) + other
	case Int64Type:
		result = Int64Type(v) + other
	default:
		result = v + other.(Uint64Type)
	}
	return
}
func (v Uint64Type) Minus(other Arithmeticker) (result Arithmeticker, err error) {
	switch o := other.(type) {
	case Float64Type:
		result = Float64Type(v) - o
	case Int64Type:
		result = Int64Type(v) - o
	default:
		result = v - o.(Uint64Type)
	}
	return
}
func (v Uint64Type) Multiply(other Arithmeticker) (result Arithmeticker, err error) {
	switch o := other.(type) {
	case Float64Type:
		result = Float64Type(v) * o
	case Int64Type:
		result = Int64Type(v) * o
	default:
		result = v * o.(Uint64Type)
	}
	return
}
func (v Uint64Type) Divide(other Arithmeticker) (result Arithmeticker, err error) {
	switch o := other.(type) {
	case Float64Type:
		result = Float64Type(float64(v) / float64(o))
	case Int64Type:
		result = Float64Type(float64(v) / float64(o))
	case Uint64Type:
		result = Float64Type(float64(v) / float64(o))
	default:
		err = fmt.Errorf("division not supported for type %T value '%[1]v'", other)
	}
	return
}
func (v Uint64Type) IntegerDivide(other Arithmeticker) (result Arithmeticker, err error) {
	switch o := other.(type) {
	case Float64Type:
		result = Int64Type(int64(v) / int64(o))
	case Int64Type:
		result = Int64Type(int64(v) / int64(o))
	case Uint64Type:
		result = Uint64Type(uint64(v) / uint64(o))
	default:
		err = fmt.Errorf("integer division not supported for type %T value'%[1]v'", other)
	}
	return
}
func (v Uint64Type) Mod(other Arithmeticker) (result Arithmeticker, err error) {
	switch o := other.(type) {
	case Float64Type:
		result = Float64Type(math.Mod(float64(v), float64(o)))
	case Int64Type:
		result = Int64Type(int64(v) % int64(o))
	case Uint64Type:
		result = Uint64Type(uint64(v) % uint64(o))
	default:
		err = fmt.Errorf("modulus not supported for type %T value'%[1]v'", other)
	}
	return
}
func (v Int64Type) Plus(other Arithmeticker) (result Arithmeticker, err error) {
	switch o := other.(type) {
	case Float64Type:
		result = Float64Type(float64(v) + float64(o))
	case Int64Type:
		result = Int64Type(int64(v) + int64(o))
	case Uint64Type:
		if int64(v) < 0 {
			result = Int64Type(int64(v) + int64(o))
		} else {
			result = Uint64Type(uint64(v) + uint64(o))
		}
	default:
		err = fmt.Errorf("integer addition not supported for type %T value '%[1]v'", other)
	}
	return
}
func (v Int64Type) Minus(other Arithmeticker) (result Arithmeticker, err error) {
	switch other := other.(type) {
	case Float64Type:
		result = Float64Type(float64(v) - float64(other))
	case Int64Type:
		result = Int64Type(int64(v) - int64(other))
	case Uint64Type:
		if v < 0 {
			result = Int64Type(int64(v) - int64(other))
		} else {
			result = Uint64Type(uint64(v) - uint64(other))
		}
	default:
		err = fmt.Errorf("subtraction not supported for type %T value '%[1]v'", other)
	}
	return
}
func (v Int64Type) Multiply(other Arithmeticker) (result Arithmeticker, err error) {
	switch other := other.(type) {
	case Float64Type:
		result = Float64Type(float64(v) * float64(other))
	case Int64Type:
		result = Int64Type(int64(v) * int64(other))
	case Uint64Type:
		if v < 0 {
			result = Int64Type(int64(v) * int64(other))
		} else {
			result = Uint64Type(uint64(v) * uint64(other))
		}
	default:
		err = fmt.Errorf("subtraction not supported for type %T value '%[1]v'", other)
	}
	return
}
func (v Int64Type) IntegerDivide(other Arithmeticker) (result Arithmeticker, err error) {
	switch other := other.(type) {
	case Float64Type:
		result = Int64Type(int64(v) / int64(other))
	case Int64Type:
		result = Int64Type(int64(v) / int64(other))
	case Uint64Type:
		result = Int64Type(int64(v) / int64(other))
	default:
		err = fmt.Errorf("integer division not supported for type %T value '%[1]v'", other)
	}
	return
}
func (v Int64Type) Divide(other Arithmeticker) (result Arithmeticker, err error) {
	switch other := other.(type) {
	case Float64Type:
		result = Float64Type(float64(v) / float64(other))
	case Int64Type:
		result = Float64Type(float64(v) / float64(other))
	case Uint64Type:
		result = Float64Type(float64(v) / float64(other))
	default:
		err = fmt.Errorf("division not supported for type %T value '%[1]v'", other)
	}
	return
}
func (v Int64Type) Mod(other Arithmeticker) (result Arithmeticker, err error) {
	switch o := other.(type) {
	case Float64Type:
		result = Float64Type(math.Mod(float64(v), float64(o)))
	case Int64Type:
		result = Int64Type(int64(v) % int64(o))
	case Uint64Type:
		if v < 0 {
			result = Int64Type(int64(v) % int64(o))
		} else {
			result = Uint64Type(uint64(v) % uint64(o))
		}
	default:
		err = fmt.Errorf("modulus not supported for type %T value '%[1]v'", other)
	}
	return
}
func (v BooleanType) Equal(other Equaler) bool {
	switch o := other.(type) {
	case BooleanType:
		return bool(v) == bool(o)
	case Int64Type:
		if bool(v) == false {
			return int64(o) == 0
		}
		return int64(o) != 0
	case Uint64Type:
		if bool(v) == false {
			return uint64(o) == 0
		}
		return uint64(o) != 0
	default:
		return false
	}
}
func isNumericToken(tk tokenEnum) bool {
	return tk == tokenInteger || tk == tokenFloat
}

type associativity int

const (
	assocNone associativity = iota
	assocLeft
	assocRight
)

// These values are from the XPath 3.1 operator precedence table at
//   https://www.w3.org/TR/xpath-3/#id-precedence-order
// Not all of these operators are actually implemented here.
func operatorOrder(tk token) (int, associativity) {
	switch tk.value {
	case ",":
		return 1, assocNone
	case "for", "some", "let", "every", "if":
		return 2, assocNone
	case "or":
		return 3, assocNone
	case "and":
		return 4, assocNone
	case "eq", "ne", "lt", "le", "gt", "ge", "=", "!=", "<", "<=", ">", ">=", "is", "<<", ">>":
		return 5, assocNone
	case "||": // string concatenate
		return 6, assocLeft
	case "to":
		return 7, assocNone
	case "+", "-": // binary operators
		return 8, assocLeft
	case "*", "div", "idiv", "mod":
		return 9, assocLeft
	case "union", "|":
		return 10, assocNone
	case "intersect", "except":
		return 11, assocLeft
	case "instance of":
		return 12, assocNone
	case "treat as":
		return 13, assocNone
	case "castable as":
		return 14, assocNone
	case "cast as":
		return 15, assocNone
	case "=>":
		return 16, assocLeft
	case "!":
		return 18, assocLeft
	case "/", "//":
		return 19, assocLeft
	case "[", "]":
		return 20, assocLeft
	}
	return -1, assocNone
}
