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
//   *[last()] return first child of container elt named "book"
//   *[position()<3] return first 2 child elts of container elt named "book"
//   *[not(position()] return first 2 child elts of container elt named "book"
//   *[@type=XXXX] match any element of type XXXX
//   *[@name=XXXX] match any element with name XXXX
//   *[@data<35] match any element whose numeric value < 35 (raise error on non-numeric)
//   *[not(@type!=UI32) and @data<35] boolean syntax
//
// TODO:
//   define boolean syntax for operators
//
// FIXME paths should be resolveable using hex or non-hex FC32 representation.
// Currently, the user-provided path is matched only against what is stored as
// the Name field, which is one or the other.

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
	nextAtoms, e = filterOnPath(a.Children, pathParts[index])

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

func filterOnPath(children []*Atom, pathPart string) (nextAtoms []*Atom, e error) {
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
