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

	// if none of the matching children were containers, return error
	if !foundCont {
		pathSoFar := strings.Join(pathParts[:index], "/")
		e = fmt.Errorf("atom '%s' has no container child named '%s'", pathSoFar, pathParts[index])
		return
	}
	return
}

func filterOnPathElement(children []*Atom, pathPart string) (nextAtoms []*Atom, e error) {
	name, filter := extractNameAndFilter(pathPart)
	if name == "" {
		e = fmt.Errorf("empty name is not allowed in path specification.  Prepend a name or wildcard ('*','**').")
		return
	}
	fmt.Printf("got scan results: name(%s), filter(%)\n", name, filter)
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
	filter = path[i_start:i_end]
	return
}

// lexer - identifies tokens in the path definition
// holds the state of the scanner
func readPath(path string) {
	// Convert text into Atom values
	var atoms []*Atom
	var lexr = lex(string(input))
	atoms, err = parse(lexr.items)
	if err != nil {
		return
	}

}

func lexPath(input string) *lexer {
	l := &lexer{
		input: input,
		items: make(chan item),
	}
	go l.run(lexFilterExpression)
	return l
}

// lexFilter splits the filter into tokens.
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
			l.emit(itemEOF)
			ok = false
		case r == '@':
			l.lexAtomAttribute()
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

// accept @name, @type or @data.  The @ is already read.
func lexAtomAttribute(l *lexer) stateFn {
	if l.first() != '@' {
		// if this happens it's a code problem, no xpath input should cause this
		panic("lexAtomAttribute called without leading attribute sigil @")
	}
	l.acceptRun("abcdefghijklmnopqrstuvwxyz")
	l.emit(itemAtomAttribute)
}
