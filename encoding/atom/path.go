package atom

// == Purpose ==
// This code provides the AtomsAtPath method, which returns a slice of Atoms
// that match the given path, from a given root atom.  Path syntax duplicates
// XPath as closely as possible.
//
// An error and an empty set are returned if an invalid path is used.
//

// == Development notes ==

// === Structure ===
// Path evaluation is implemented as a language compiler with 3 steps:
// lexing, parsing and evaluating.
//
// Lexing is performed during AtomPath object creation. The lexer splits the
// given path string into tokens with known types and sends them to the
// pathParser, which runs concurrently.
//
// Parsing is also performed during AtomPath object creation. The pathParser
// receives a stream of tokens from the lexer, and puts them into a stack in
// prefix order so that during evaluation, operator tokens are followed by
// their operands. This follows operator precedence rules.
//
// Evaluation is performed whenever the AtomPath method GetAtoms(a *Atom) is
// called, which provides a root atom to evaluate against the path.
//
// At a low level, there are separate evaluators for the path and predicate
// even though they share the same token stack, because the code is simpler
// this way. Evaluation is almost 100% different within a predicate.  The
// parser does some juggling to delimit predicate tokens with Predicate Start
// and Predicate End tokens, so it is simple to know when to switch evaluators.
//
// === Terminology ===
// Terms used to describe attributes of a path are 100% stolen from the XPath
// documentation.  To make sense of the method / variable names and comments in
// this code, be familiar with these XPath terms:
//
// location step:
//     Like directory paths, you build location paths out of individual steps,
//     called location steps, separated by / or //.
//     Each location step is made up of an axis, a node test, and zero or more
//     predicates, like this (where the * symbol means "zero or more of"):
//         axis node-test [predicate]*
//
// axis:
//     An axis defines a node-set relative to the current node.
//     AtomPath has minimal support for axes, it only supports the child axis
//     which is the default of XPath. The leading / or // in a path expression
//     are the axis.
// node test:
//     A node test is a test applied to the axis which filters out the atoms
//     that don't match the test.  The node test operator "*" matches all atoms
//     in the axis node set. An atom name can be used as a node test to match
//     only nodes with the given name. A node test is required for a valid path,
//     you cannot have a predicate directly after a path
//          //[position() == 1]     illegal: predicate right after path
//          //*[position() == 1]    legal
// predicate:
//     A predicate is another filter, like the node test, which is applied to
//     the node set resulting from the node test.  It performs some tests and
//     removes nodes that don't pass from the node set.
//     A predicate is delimited by []. 0-N predicates may be used -- if a
//     predicate B follows predicate A, then B filters down the node set
//     resulting from A, instead of the nodeset resulting from the node test.

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

const (
	// Operators must end with string "Operator", that is how they are identified as
	// operator-type tokens by the parser
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

type (
	// AtomPath is for evaluating paths.
	// To evaluate paths, construct one by providing a path string to NewPath().
	// Then send it a root atom to evaluate against
	AtomPath struct {
		Path      string
		evaluator *pathEvaluator
		err       error
	}
	tokenList []*token

	// pathParser is a parser for interpreting atom path tokens.
	pathParser struct {
		outputQueue tokenList    // tokens ordered for evaluation
		opStack     tokenList    // holds operators until their operands reach output queue
		tokens      <-chan token // tokens received from lexer
		err         error        // indicates parsing succeeded or describes what failed
	}

	pathEvaluator struct {
		Path           string
		tokens         tokenList // path criteria, does not change after creation
		Tokens         tokenList // path criteria, consumed during each evaluation
		Error          error     // evaluation status, nil on success
		ContextAtomPtr *Atom
	}

	// predicateEvaluator determines which candidate atoms satisfy the
	// predicate criteria.
	//
	// The predicate is the part of the path within the [].
	// Examples:
	//    /ROOT[1]
	//    /ROOT[@name=NONE]
	//		/ROOT/UI_1[@data < 2]
	predicateEvaluator struct {
		tokens tokenList // predicate criteria, as a list of tokens
		Error  error     // evaluation status, nil on success

		Tokens   tokenList // Copy of tokens to consume during evaluation
		Atoms    []*Atom   // Atoms being evaluated
		AtomPtr  *Atom     // Atom currently being evaluated from the atom list
		Position int       // index of the atom in the atom list, starts from 1
		Count    int       // number of atoms in the atom list
	}
)

// NewAtomPath creates an AtomPath object for the given path string.  It
// performs all lexing and parsing steps, so that evaluating data sets against
// the path will have as little overhead as possible.
func NewAtomPath(path string) (ap *AtomPath, e error) {
	Log.Printf("NewAtomPath(%q)", path)

	var pe *pathEvaluator
	pe, e = newPathEvaluator(path)
	if e != nil {
		return ap, addPathToError(e, path)
	}

	ap = &AtomPath{
		Path:      strings.TrimSpace(path),
		evaluator: pe,
		err:       nil,
	}
	return
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

// push a new token onto the end of the stack, as the new top element
func (s *tokenList) push(tk *token) {
	*s = append(*s, tk)
}

// add a new token to the front of the queue, as the new first element
func (s *tokenList) unshift(tk *token) {
	*s = append(tokenList{tk}, *s...)
}

// remove and return the first token token in the queue (zeroth element)
func (s *tokenList) shift() (tk *token) {
	if len(*s) == 0 {
		return nil
	}
	tk, *s = (*s)[0], (*s)[1:]
	return
}

// pop one token off the stack and return it.
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

// AtomsAtPath returns the set of descendant atoms in which match the given
// path.
//
// This is shorthand for creating an AtomPath object and calling
// AtomPath.GetAtoms().  Do it the long way if you plan to perform the path
// evaluation multiple times, because keeping the compiled AtomPath object
// instead of repeating the lexing and parsing steps saves allocations.
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

// GetAtoms returns the result atoms from evaluating the given atom as the root
// of the path expression.
func (ap *AtomPath) GetAtoms(root *Atom) (atoms []*Atom, e error) {
	return ap.evaluator.evaluate(root)
}

// newPathEvaluator reads a path string and returns a pathEvaluator object
// representing the path.
func newPathEvaluator(path string) (pe *pathEvaluator, err error) {
	var lexr = newPathLexer(path)
	var pp = pathParser{tokens: lexr.tokens}
	pp.receiveTokens()
	pe = &pathEvaluator{
		Path:   path,
		tokens: pp.outputQueue,
		Error:  pp.err}
	return pe, pp.err
}

// newPredicateEvaluator consumes a series of predicate tokens from a
// pathEvaluator starting with a PredicateEnd token and ending with a
// PredicateStart token (yes it's supposed to be backwards), and returns a new
// predicateEvaluator.
func newPredicateEvaluator(pe *pathEvaluator) (pre predicateEvaluator, ok bool) {
	// Predicate end comes before pred start, that's the order they're pushed to stack
	// Predicate tokens are in postfix order at this point.
	if pe.Tokens.empty() || pe.Tokens.pop().typ != tokenPredicateEnd {
		pe.errorf("expected predicate end token")
		return pre, false
	}

	// read predicate tokens
	var predicateTokens tokenList
	for pe.nextTokenType() != tokenPredicateStart && !pe.Tokens.empty() {
		predicateTokens.unshift(pe.Tokens.pop())
	}
	pe.Tokens.pop() // discard predicate start token

	// check for predicate with no tokens
	if len(predicateTokens) == 0 {
		pe.Error = addPathToError(errInvalidPredicate("empty predicate"), pe.Path)
		return pre, false
	}

	// evaluate element set by predicate
	return predicateEvaluator{
		tokens: predicateTokens,
	}, true
}

// Evaluate filters a list of atoms against the predicate conditions, returning
// the atoms that satisfy the predicate.
//
// The candidate atoms must all be made available to the predicateEvaluator at
// once, because the predicate may refer to individual child atoms by name,
// requiring them to be evaluated against every other candidate.
func (pre *predicateEvaluator) Evaluate(candidates []*Atom) (atoms []*Atom, e error) {
	Log.Print("predicateEvaluator::Evaluate()  ", pre.tokens, candidates)
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

func (pre *predicateEvaluator) getChildValue(atomName string) (v iComparer, ok bool) {
	for _, a := range pre.AtomPtr.children {
		if a.Name() != atomName {
			continue
		}
		v = atomValueToiComparerType(a)
		ok = true
		break
	}
	return
}

func atomValueToiComparerType(a *Atom) (v iComparer) {
	switch {
	case a.Value.IsUint(), a.Value.IsBool():
		x, _ := a.Value.Uint()
		v = typeUint64(x)
	case a.Value.IsFloat():
		x, _ := a.Value.Float()
		v = typeFloat64(x)
	case a.Value.IsInt():
		x, _ := a.Value.Int()
		v = typeInt64(x)
	default:
		x, _ := a.Value.String()
		v = typeString(x)
	}
	return
}

func newPathLexer(path string) *lexer {
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
	}
	return lexPath
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
func (pp *pathParser) receiveTokens() {
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
func (pp *pathParser) readToken() (tk token) {
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
func (pp *pathParser) errorf(format string, args ...interface{}) bool {
	pp.err = errInvalidPath(fmt.Sprintf(format, args...))
	return false
}

// parseToken is given tokens from the lexer in the order they are found
// in the path string, and queues them into evaluation order.
// This is based on Djikstra's shunting-yard algorithm.
// https://en.wikipedia.org/wiki/Shunting-yard_algorithm
func (pp *pathParser) parseToken(tk token) bool {
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
func (pp *pathParser) moveOperatorsToOutputUntil(test func(t token) bool) {
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
//
//  while there is an operator token o2, at the top of the operator stack and either
//    o1 is left-associative and its precedence is less than or equal to that of o2, or
//    o1 is right associative, and has precedence less than that of o2,
//        pop o2 off the operator stack, onto the output queue;
func (pp *pathParser) moveOperatorsToOutput(tk token) {
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

// errorf sets the error field in the parser, and indicates that parsing should
// stop by returning false.
func (pe *pathEvaluator) errorf(format string, args ...interface{}) bool {
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

func (pe *pathEvaluator) evaluate(atom *Atom) (result []*Atom, e error) {
	pe.ContextAtomPtr = atom
	if pe.tokens.empty() {
		e = errInvalidPath("<empty>")
		return
	}

	// Special case, otherwise path specifiers may not end with /
	if len(pe.tokens) == 1 && pe.tokens[0].value == "/" {
		return []*Atom{pe.ContextAtomPtr}, nil
	}

	pe.Tokens = pe.tokens
	Log.Println("pathEvaluator::evaluate() ", pe.Tokens)
	result = pe.evalElementSet()
	e = pe.Error
	return
}

// Done returns true if this pathEvaluator is done processing.
// Completion can occur due to normal consumption of all tokens (success case)
// or due to an error state.
func (pe *pathEvaluator) Done() bool {
	return pe.Error != nil || len(pe.Tokens) == 0
}

// nextTokenType returns the tokenType of the next Token in the PathEvalator's Token stack
func (pe *pathEvaluator) nextTokenType() tokenEnum {
	if len(pe.Tokens) == 0 {
		return ""
	}
	return pe.Tokens.nextType()
}

func (pe *pathEvaluator) evalSetOperator() (atoms []*Atom) {
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
func (pe *pathEvaluator) evalAxisOperator() (atoms []*Atom) {
	tk := pe.Tokens.pop()
	Log.Printf("evalAxisOperator(%q)", tk.value)
	if tk.typ != tokenAxisOperator && tk.typ != tokenStepSeparator {
		pe.errorf("expected axis operator, got '%v' [%[1]T]", tk.value)
		return nil
	}

	//	if pe.nextTokenType() == tokenNodeTest {
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
func (pe *pathEvaluator) evalNodeTest() (atoms []*Atom) {
	// Get node test token
	tkNodeTest := pe.Tokens.pop()
	if tkNodeTest.typ != tokenNodeTest {
		pe.errorf("expected node test, got '%v' [%[1]T]", tkNodeTest.value)
		return nil
	}

	// Get element set to filter
	if pe.nextTokenType() == tokenStepSeparator {
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
	} else if pe.nextTokenType() == tokenAxisOperator {
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

func (pe *pathEvaluator) evalElementSet() (atoms []*Atom) {
	Log.Printf("evalElementSet() [%s]'", pe.nextTokenType())
	if pe.Done() {
		return
	}
	switch pe.nextTokenType() {
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

	Log.Println("evalElementSet: Returning context atom. Next Token Type is ", pe.nextTokenType())
	// No axis operator given, so use context node
	atoms = append(atoms, pe.ContextAtomPtr)
	return
}

// evalPredicate returns the set of Atoms from the input set which match the
// predicate.
func (pe *pathEvaluator) evalPredicate() []*Atom {
	Log.Println("evalPredicate()")
	// evaluate element set by predicate
	pre, ok := newPredicateEvaluator(pe)
	if ok != true {
		return nil // error is already set by newPredicateEvaluator
	}
	atoms, err := pre.Evaluate(pe.evalElementSet())
	if err != nil {
		pe.Error = addPathToError(err, pe.Path)
		return nil
	}
	return atoms
}

// errorf is called to record an error state resulting from predicate evaluation.
func (pre *predicateEvaluator) errorf(format string, args ...interface{}) error {
	msg := fmt.Sprintf(format, args...)
	pre.Error = errInvalidPredicate(msg)
	return pre.Error
}

// nextTokenType returns the tokenType of the next Token in the PathEvalator's
// Token stack.
func (pre *predicateEvaluator) nextTokenType() tokenEnum {
	if len(pre.Tokens) == 0 {
		return ""
	}
	return pre.Tokens.nextType()
}

// eval evaluates the list of operators/values/stuff against the evaluator's
// atom/pos/count.
func (pre *predicateEvaluator) eval() (results []iEqualer) {
Loop:
	for !pre.Tokens.empty() && pre.Error == nil {
		switch pre.nextTokenType() {
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

// evalBoolean evaluates some of the next predicate tokens and returns a
// boolean result.
func (pre *predicateEvaluator) evalBoolean() (result iEqualer) {
	if pre.Tokens.empty() {
		pre.errorf("expect boolean value, got nothing")
		return
	}
	Log.Printf("    evalBoolean() %v,%v", pre.nextTokenType(), pre.Tokens.peek().value)
	switch pre.nextTokenType() {
	case tokenEqualityOperator:
		result = pre.evalEqualityOperator()
	case tokenBooleanOperator:
		result = pre.evalBooleanOperator()
	case tokenComparisonOperator:
		result = pre.evalComparisonOperator().(iEqualer)
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

// evalBooleanOperator evaluates a boolean operator token, followed by some operands, and
// returns a boolean result.
func (pre *predicateEvaluator) evalBooleanOperator() typeBoolean {
	op := pre.Tokens.pop()
	if op.typ != tokenBooleanOperator {
		pre.errorf("expected boolean operator, received type %s", op.typ)
	}
	results := []iEqualer{pre.evalBoolean(), pre.evalBoolean()}
	tru := typeBoolean(true)
	var result bool
	switch op.value {
	case "and":
		result = results[0] == tru && results[1] == tru
	case "or":
		result = results[0] == tru || results[1] == tru
	default:
		pre.errorf("unknown boolean operator: %s", op.value)
	}
	return typeBoolean(result)
}

// evalArithmeticOperator evaluates an arithmetic operator token, followed by some
// operands, and returns a numeric result.
func (pre *predicateEvaluator) evalArithmeticOperator() (result iArithmeticker) {
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

// evalEqualityOperator evaluates an equality operator token, followed by some
// operands, and returns a result that can be compared with =.
func (pre *predicateEvaluator) evalEqualityOperator() iEqualer {
	var result bool
	op := pre.Tokens.pop()
	if op.typ != tokenEqualityOperator {
		pre.errorf("expected tokenEqualityOperator, received type %s", op.typ)
		return typeBoolean(false)
	}
	rhs := pre.evaliEqualer()
	lhs := pre.evaliEqualer()
	if pre.Error != nil {
		return typeBoolean(false)
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
	return typeBoolean(result)
}

// evalComparisonOperator evaluates an comparison operator token and some following
// operands, and returns a boolean result.
func (pre *predicateEvaluator) evalComparisonOperator() iEqualer {
	var result bool
	op := pre.Tokens.pop()
	if op.typ != tokenComparisonOperator {
		pre.errorf("expected tokenComparisonOperator, received type %s", op.typ)
		return typeBoolean(false)
	}
	rhs := pre.evalComparable()
	lhs := pre.evalComparable()
	if pre.Error != nil {
		return typeBoolean(false)
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
	return typeBoolean(result)
}

// evalNumber evaluates a numeric token and returns an Arithmeticker result.
func (pre *predicateEvaluator) evalNumber() (result iArithmeticker) {
	var err error
	ok := true
	switch pre.nextTokenType() {
	case tokenInteger, tokenHex:
		v, err := strconv.ParseInt(pre.Tokens.pop().value, 0, 64)
		if err != nil {
			pre.errorf(err.Error())
			return
		}
		result = typeInt64(v)
	case tokenFloat:
		v, err := strconv.ParseFloat(pre.Tokens.pop().value, 64)
		if err != nil {
			pre.errorf(err.Error())
			return
		}
		result = typeFloat64(v)
	case tokenFunctionNumeric:
		result = pre.evalFunctionNumeric()
	case tokenVariable:
		result, ok = pre.evalVariable().(iArithmeticker)
	case tokenArithmeticOperator:
		result = pre.evalArithmeticOperator()
	case tokenBareString:
		t := pre.Tokens.pop()
		if v, ok := pre.getChildValue(t.value); ok {
			result = v.(iArithmeticker)
		} else {
			pre.errorf("expect number, got %s", t.value)
		}
	default:
		pre.errorf("value has invalid numeric type: %s", pre.nextTokenType())
	}
	if err != nil || !ok {
		pre.errorf("expected numeric value")
	}
	return result
}

// evalNumber evaluates a token that con be compared with =, and returns an iEqualer result.
func (pre *predicateEvaluator) evaliEqualer() (result iEqualer) {
	Log.Printf("    evaliEqualer(), Tokens=%v", pre.Tokens)
	var err error
	switch pre.nextTokenType() {
	case tokenInteger, tokenHex:
		v, e := strconv.ParseInt(pre.Tokens.pop().value, 0, 64)
		err = e
		result = typeInt64(v)
	case tokenFloat:
		v, e := strconv.ParseFloat(pre.Tokens.pop().value, 64)
		err = e
		result = typeFloat64(v)
	case tokenBareString:
		t := pre.Tokens.pop()
		if v, ok := pre.getChildValue(t.value); ok {
			result = v // string is Atom name, substitute atom value.
		} else {
			result = typeString(t.value)
		}
	case tokenString:
		result = typeString(pre.Tokens.pop().value)
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
		pre.errorf("expected iEqualer type, got %q [%s])", t.value, t.typ)
		return
	}
	if err != nil {
		pre.errorf("failed to convert '%s' to iEqualer value")
		return
	}
	return result
}

// FIXME: this near-duplicates evaliEqualer.
// have tk call evaliEqualer and then error out on non-Compararer types?
func (pre *predicateEvaluator) evalComparable() (result iComparer) {
	Log.Printf("    evalComparable(), Tokens=%v", pre.Tokens)
	var err error
	switch pre.nextTokenType() {
	case tokenInteger, tokenHex:
		v, e := strconv.ParseInt(pre.Tokens.pop().value, 0, 64)
		err = e
		result = typeInt64(v)
	case tokenFloat:
		v, e := strconv.ParseFloat(pre.Tokens.pop().value, 64)
		err = e
		result = typeFloat64(v)
	case tokenBareString:
		t := pre.Tokens.pop()
		if v, ok := pre.getChildValue(t.value); ok {
			result = v // string is Atom name, substitute atom value.
		} else {
			result = typeString(t.value)
		}
	case tokenString:
		result = typeString(pre.Tokens.pop().value)
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
func (pre *predicateEvaluator) evalVariable() (result iComparer) {
	token := pre.Tokens.pop()
	if token.typ != tokenVariable {
		pre.errorf("expected tokenVariable, received type %s", token.typ)
	}
	switch token.value {
	case "@name", "name":
		return typeString(pre.AtomPtr.Name())
	case "@name_hex":
		return typeString(fmt.Sprint("0x%08X", pre.AtomPtr.NameAsUint32()))
	case "@type", "type":
		return typeString(pre.AtomPtr.Type())
	case "@data", "data":
	default:
		pre.errorf("unknown variable: %s", token.value)
		return
	}

	// Must get Atom value. Choose concrete type to return.
	switch {
	case pre.AtomPtr.Value.IsFloat():
		v, _ := pre.AtomPtr.Value.Float()
		result = typeFloat64(v)
	case pre.AtomPtr.Value.IsInt():
		v, _ := pre.AtomPtr.Value.Int()
		result = typeInt64(v)
	case pre.AtomPtr.Value.IsUint():
		v, _ := pre.AtomPtr.Value.Uint()
		result = typeUint64(v)
	case pre.AtomPtr.Value.IsBool():
		v, _ := pre.AtomPtr.Value.Uint() // use UINT since tk's represented as 0/1
		result = typeUint64(v)
	default:
		v, _ := pre.AtomPtr.Value.String()
		result = typeString(v)
	}
	return result
}
func (pre *predicateEvaluator) evalFunctionBool() iEqualer {
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
			result = r.Equal(typeBoolean(false))
		}
	default:
		pre.errorf("unknown boolean function: %s", token.value)
	}
	return typeBoolean(result)
}

func (pre *predicateEvaluator) evalResultsToBool(results []iEqualer) (result bool, err error) {
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
	case typeBoolean:
		result = bool(r)
	case typeInt64:
		result = r.Equal(typeInt64(pre.Position))
	case typeUint64:
		result = r.Equal(typeUint64(pre.Position))
	case typeFloat64:
		result = r.Equal(typeFloat64(pre.Position))
	default:
		err = fmt.Errorf("result '%v' has unknown type %[1]T", results[0])
		return
	}
	return
}
func (pre *predicateEvaluator) evalFunctionNumeric() (result iArithmeticker) {
	token := pre.Tokens.pop()
	if token.typ != tokenFunctionNumeric {
		pre.errorf("expected tokenFunctionNumeric, received type %s", token.typ)
		return
	}
	switch token.value {
	case "position":
		Log.Printf(`    evalFunctionNumeric("%s") = %d`, token.value, pre.Position)
		return typeUint64(pre.Position)
	case "last", "count":
		Log.Printf(`    evalFunctionNumeric("%s") = %d`, token.value, pre.Count)
		return typeUint64(pre.Count)
	default:
		pre.errorf("unknown numeric function: %s", token.value)
	}
	return
}

// Implement a small type system with type coercion for operators
type (
	typeInt64   int64
	typeUint64  uint64
	typeFloat64 float64
	typeString  string
	typeBoolean bool

	iEqualer interface {
		Equal(other iEqualer) bool
	}
	iComparer interface {
		iEqualer
		LessThan(other iComparer) bool
		GreaterThan(other iComparer) bool
	}
	iArithmeticker interface {
		iComparer
		Plus(other iArithmeticker) (iArithmeticker, error)
		Minus(other iArithmeticker) (iArithmeticker, error)
		Multiply(other iArithmeticker) (iArithmeticker, error)
		Divide(other iArithmeticker) (iArithmeticker, error)
		IntegerDivide(other iArithmeticker) (iArithmeticker, error)
		Mod(other iArithmeticker) (iArithmeticker, error)
	}
)

// Define explicitly how to do type conversion when performing arithmetic on
// pairs of heterogenous types.

func (v typeFloat64) Equal(other iEqualer) bool {
	switch o := other.(type) {
	case typeFloat64:
		return v == o
	case typeUint64:
		return float64(v) == float64(o)
	case typeInt64:
		return float64(v) == float64(o)
	case typeString:
		fp, err := strconv.ParseFloat(string(o), 64)
		if err != nil {
			return false
		}
		return float64(v) == fp
	default:
		return false
	}
}
func (v typeFloat64) LessThan(other iComparer) bool {
	switch o := other.(type) {
	case typeFloat64:
		return v < o
	case typeUint64:
		return float64(v) < float64(o)
	case typeInt64:
		return float64(v) < float64(o)
	default:
		return false
	}
}
func (v typeFloat64) GreaterThan(other iComparer) bool {
	switch o := other.(type) {
	case typeFloat64:
		return v > o
	case typeUint64:
		return float64(v) > float64(o)
	case typeInt64:
		return float64(v) > float64(o)
	default:
		return false
	}
}
func (v typeUint64) Equal(other iEqualer) bool {
	switch o := other.(type) {
	case typeFloat64:
		return typeFloat64(v) == o
	case typeInt64:
		return typeInt64(v) == o
	case typeString:
		oUint, err := strconv.ParseUint(string(o), 0, 64)
		if err != nil {
			return false
		}
		return uint64(v) == oUint
	default:
		return v == o.(typeUint64)
	}
}
func (v typeUint64) LessThan(other iComparer) bool {
	switch o := other.(type) {
	case typeFloat64:
		return typeFloat64(v) < o
	case typeInt64:
		return typeInt64(v) < o
	case typeString:
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
		return v < o.(typeUint64)
	}
	return false
}
func (v typeUint64) GreaterThan(other iComparer) bool {
	switch o := other.(type) {
	case typeFloat64:
		return typeFloat64(v) > o
	case typeInt64:
		return typeInt64(v) > o
	case typeString:
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
		return v > other.(typeUint64)
	}
	return false
}
func (v typeInt64) Equal(other iEqualer) bool {
	switch o := other.(type) {
	case typeFloat64:
		return typeFloat64(v) == o
	case typeUint64:
		return v == typeInt64(o)
	case typeString:
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
		return v == other.(typeInt64)
	}
	return false
}
func (v typeInt64) LessThan(other iComparer) bool {
	switch o := other.(type) {
	case typeFloat64:
		return typeFloat64(v) < o
	case typeString:
		if x, e := strconv.ParseFloat(string(o), 64); e == nil {
			return float64(v) < x
		}
		if x, e := strconv.ParseUint(string(o), 10, 64); e == nil {
			return uint64(v) < x
		}
		if x, e := strconv.ParseInt(string(o), 0, 64); e == nil {
			return int64(v) < x
		}
	case typeUint64:
		return v < typeInt64(o)
	default:
		return v < o.(typeInt64)
	}
	return false
}
func (v typeInt64) GreaterThan(other iComparer) bool {
	switch o := other.(type) {
	case typeFloat64:
		return typeFloat64(v) > o
	case typeUint64:
		return v > typeInt64(o)
	case typeString:
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
		return v > o.(typeInt64)
	}
	return false
}
func (v typeString) Equal(other iEqualer) bool {
	switch o := other.(type) {
	case typeString:
		// case insensitive comparison
		return strings.EqualFold(string(v), string(o))
	case typeInt64:
		return string(v) == strconv.Itoa(int(o))
	case typeUint64:
		return string(v) == strconv.Itoa(int(o))
	case typeFloat64:
		return string(v) == fmt.Sprintf("%G", o)
	}
	return false
}
func (v typeString) LessThan(other iComparer) bool {
	str := string(v)
	if x, e := strconv.ParseFloat(str, 64); e == nil {
		return typeFloat64(x).LessThan(other)
	}
	if x, e := strconv.ParseUint(str, 10, 64); e == nil {
		return typeUint64(x).LessThan(other)
	}
	if x, e := strconv.ParseInt(str, 0, 64); e == nil {
		// this case handles hex strings too, based on prefix
		return typeInt64(x).LessThan(other)
	}
	if o, ok := other.(typeString); ok {
		return str > string(o)
	}
	return false
}
func (v typeString) GreaterThan(other iComparer) bool {
	str := string(v)
	if x, e := strconv.ParseFloat(str, 64); e == nil {
		return typeFloat64(x).GreaterThan(other)
	}
	if x, e := strconv.ParseUint(str, 10, 64); e == nil {
		return typeUint64(x).GreaterThan(other)
	}
	if x, e := strconv.ParseInt(str, 0, 64); e == nil {
		// this case handles hex strings too, based on prefix
		return typeInt64(x).GreaterThan(other)
	}
	if o, ok := other.(typeString); ok {
		return str > string(o)
	}
	return false
}
func (v typeFloat64) Plus(other iArithmeticker) (result iArithmeticker, err error) {
	switch o := other.(type) {
	case typeFloat64:
		result = typeFloat64(float64(v) + float64(o))
	case typeInt64:
		result = typeFloat64(float64(v) + float64(o))
	case typeUint64:
		result = typeFloat64(float64(v) + float64(o))
	default:
		err = fmt.Errorf("addition not supported for type %T, value '%[1]v'", other)
	}
	return
}
func (v typeFloat64) Minus(other iArithmeticker) (result iArithmeticker, err error) {
	switch o := other.(type) {
	case typeFloat64:
		result = typeFloat64(float64(v) - float64(o))
	case typeInt64:
		result = typeFloat64(float64(v) - float64(o))
	case typeUint64:
		result = typeFloat64(float64(v) - float64(o))
	default:
		err = fmt.Errorf("multiplication not supported for type %T, value '%[1]v'", other)
	}
	return
}
func (v typeFloat64) Multiply(other iArithmeticker) (result iArithmeticker, err error) {
	switch o := other.(type) {
	case typeFloat64:
		result = typeFloat64(float64(v) * float64(o))
	case typeInt64:
		result = typeFloat64(float64(v) * float64(o))
	case typeUint64:
		result = typeFloat64(float64(v) * float64(o))
	default:
		err = fmt.Errorf("multiplication not supported for type %T, value '%[1]v'", other)
	}
	return
}
func (v typeFloat64) Divide(other iArithmeticker) (result iArithmeticker, err error) {
	switch other := other.(type) {
	case typeFloat64:
		result = typeFloat64(float64(v) / float64(other))
	case typeInt64:
		result = typeFloat64(float64(v) / float64(other))
	case typeUint64:
		result = typeFloat64(float64(v) / float64(other))
	default:
		err = fmt.Errorf("division not supported for type %T, value '%[1]v'", other)
	}
	return
}
func (v typeFloat64) IntegerDivide(other iArithmeticker) (result iArithmeticker, err error) {
	switch other := other.(type) {
	case typeFloat64:
		result = typeInt64(int64(v) / int64(other))
	case typeInt64:
		result = typeInt64(int64(v) / int64(other))
	case typeUint64:
		result = typeInt64(int64(v) / int64(other))
	default:
		err = fmt.Errorf("integer division not supported for type %T, value '%[1]v'", other)
	}
	return
}
func (v typeFloat64) Mod(other iArithmeticker) (result iArithmeticker, err error) {
	switch other := other.(type) {
	case typeFloat64:
		result = typeFloat64(math.Mod(float64(v), float64(other)))
	case typeInt64:
		result = typeFloat64(math.Mod(float64(v), float64(other)))
	case typeUint64:
		result = typeFloat64(math.Mod(float64(v), float64(other)))
	default:
		err = fmt.Errorf("modulus not supported for type %T, value '%[1]v'", other)
	}
	return
}
func (v typeUint64) Plus(other iArithmeticker) (result iArithmeticker, err error) {
	switch other := other.(type) {
	case typeFloat64:
		result = typeFloat64(v) + other
	case typeInt64:
		result = typeInt64(v) + other
	default:
		result = v + other.(typeUint64)
	}
	return
}
func (v typeUint64) Minus(other iArithmeticker) (result iArithmeticker, err error) {
	switch o := other.(type) {
	case typeFloat64:
		result = typeFloat64(v) - o
	case typeInt64:
		result = typeInt64(v) - o
	default:
		result = v - o.(typeUint64)
	}
	return
}
func (v typeUint64) Multiply(other iArithmeticker) (result iArithmeticker, err error) {
	switch o := other.(type) {
	case typeFloat64:
		result = typeFloat64(v) * o
	case typeInt64:
		result = typeInt64(v) * o
	default:
		result = v * o.(typeUint64)
	}
	return
}
func (v typeUint64) Divide(other iArithmeticker) (result iArithmeticker, err error) {
	switch o := other.(type) {
	case typeFloat64:
		result = typeFloat64(float64(v) / float64(o))
	case typeInt64:
		result = typeFloat64(float64(v) / float64(o))
	case typeUint64:
		result = typeFloat64(float64(v) / float64(o))
	default:
		err = fmt.Errorf("division not supported for type %T value '%[1]v'", other)
	}
	return
}
func (v typeUint64) IntegerDivide(other iArithmeticker) (result iArithmeticker, err error) {
	switch o := other.(type) {
	case typeFloat64:
		result = typeInt64(int64(v) / int64(o))
	case typeInt64:
		result = typeInt64(int64(v) / int64(o))
	case typeUint64:
		result = typeUint64(uint64(v) / uint64(o))
	default:
		err = fmt.Errorf("integer division not supported for type %T value'%[1]v'", other)
	}
	return
}
func (v typeUint64) Mod(other iArithmeticker) (result iArithmeticker, err error) {
	switch o := other.(type) {
	case typeFloat64:
		result = typeFloat64(math.Mod(float64(v), float64(o)))
	case typeInt64:
		result = typeInt64(int64(v) % int64(o))
	case typeUint64:
		result = typeUint64(uint64(v) % uint64(o))
	default:
		err = fmt.Errorf("modulus not supported for type %T value'%[1]v'", other)
	}
	return
}
func (v typeInt64) Plus(other iArithmeticker) (result iArithmeticker, err error) {
	switch o := other.(type) {
	case typeFloat64:
		result = typeFloat64(float64(v) + float64(o))
	case typeInt64:
		result = typeInt64(int64(v) + int64(o))
	case typeUint64:
		if int64(v) < 0 {
			result = typeInt64(int64(v) + int64(o))
		} else {
			result = typeUint64(uint64(v) + uint64(o))
		}
	default:
		err = fmt.Errorf("integer addition not supported for type %T value '%[1]v'", other)
	}
	return
}
func (v typeInt64) Minus(other iArithmeticker) (result iArithmeticker, err error) {
	switch other := other.(type) {
	case typeFloat64:
		result = typeFloat64(float64(v) - float64(other))
	case typeInt64:
		result = typeInt64(int64(v) - int64(other))
	case typeUint64:
		if v < 0 {
			result = typeInt64(int64(v) - int64(other))
		} else {
			result = typeUint64(uint64(v) - uint64(other))
		}
	default:
		err = fmt.Errorf("subtraction not supported for type %T value '%[1]v'", other)
	}
	return
}
func (v typeInt64) Multiply(other iArithmeticker) (result iArithmeticker, err error) {
	switch other := other.(type) {
	case typeFloat64:
		result = typeFloat64(float64(v) * float64(other))
	case typeInt64:
		result = typeInt64(int64(v) * int64(other))
	case typeUint64:
		if v < 0 {
			result = typeInt64(int64(v) * int64(other))
		} else {
			result = typeUint64(uint64(v) * uint64(other))
		}
	default:
		err = fmt.Errorf("subtraction not supported for type %T value '%[1]v'", other)
	}
	return
}
func (v typeInt64) IntegerDivide(other iArithmeticker) (result iArithmeticker, err error) {
	switch other := other.(type) {
	case typeFloat64:
		result = typeInt64(int64(v) / int64(other))
	case typeInt64:
		result = typeInt64(int64(v) / int64(other))
	case typeUint64:
		result = typeInt64(int64(v) / int64(other))
	default:
		err = fmt.Errorf("integer division not supported for type %T value '%[1]v'", other)
	}
	return
}
func (v typeInt64) Divide(other iArithmeticker) (result iArithmeticker, err error) {
	switch other := other.(type) {
	case typeFloat64:
		result = typeFloat64(float64(v) / float64(other))
	case typeInt64:
		result = typeFloat64(float64(v) / float64(other))
	case typeUint64:
		result = typeFloat64(float64(v) / float64(other))
	default:
		err = fmt.Errorf("division not supported for type %T value '%[1]v'", other)
	}
	return
}
func (v typeInt64) Mod(other iArithmeticker) (result iArithmeticker, err error) {
	switch o := other.(type) {
	case typeFloat64:
		result = typeFloat64(math.Mod(float64(v), float64(o)))
	case typeInt64:
		result = typeInt64(int64(v) % int64(o))
	case typeUint64:
		if v < 0 {
			result = typeInt64(int64(v) % int64(o))
		} else {
			result = typeUint64(uint64(v) % uint64(o))
		}
	default:
		err = fmt.Errorf("modulus not supported for type %T value '%[1]v'", other)
	}
	return
}
func (v typeBoolean) Equal(other iEqualer) bool {
	switch o := other.(type) {
	case typeBoolean:
		return bool(v) == bool(o)
	case typeInt64:
		if bool(v) == false {
			return int64(o) == 0
		}
		return int64(o) != 0
	case typeUint64:
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
