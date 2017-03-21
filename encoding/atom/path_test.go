// Benchmark Marshal / Unmarshal functions
package atom

// TESTS TO ADD:
// * get all data rows in attr container except 0x00000000 index atom
// * get all server names and ip address from NENT (is that in there?)
// * test referring atom by @name with hex syntax

//
// Requirements for Path definition wildcards:
//   - provide a way to select all attribute container data elemnts while
//     omitting the index element. (???)
//   - provide a terse syntax to use form command-line clients to search for
//     an element by name at any position in the tree.  (**/NAME)
//   - provide a way to specify type of the atom to be matched as well was the path
//
// Path definition wildcards to borrow from XPath:
//   * match any single path element of any type
//   ** match any number of nested path elements
//   *[1] return first child of container elt (there's no 0 elt)
//   *[last()] return first child of container elt named "book"
//   *[position()<3] return first 2 child elts of container elt named "book"
//   *[not(position()] return first 2 child elts of container elt named "book"
//   *[@type=XXXX] match any element of type XXXX
//   *[@name=XXXX] match any element with name XXXX
//   XXXX match any element with name XXXX
//   *[@data<35] match any element whose numeric value < 35 (raise error on non-numeric)
//   *[not(@type==UI32) and @data<35] boolean syntax

// TODO:
//   define boolean syntax for operators

// FIXME paths should be resolveable using hex or non-hex FC32 representation.
// Currently, the user-provided path is matched only against what is stored as
// the Name field, which is one or the other.
import (
	"strings"
	"testing"
)

type (
	PathTest struct {
		Atom      *Atom
		Input     string
		WantValue []string
		WantError error
	}
)

const TestAtom1Text = `
ROOT:CONT:
  0001:CONT:
		LEAF:UI32:1
		LEAF:UI32:2
		LEAF:UI32:3
	END
  0002:CONT:
		LEAF:UI32:4
		LEAF:UI32:5
		LEAF:UI32:6
	END
  0003:CONT:
		LEAF:UI32:7
		LEAF:UI32:8
		LEAF:UI32:9
	END
END
`
const TestAtomGINFText = `
GINF:CONT:
	BVER:UI32:4
	BTIM:UI64:1484723582627327
	GIDV:CONT:
		AVER:UI32:2
		ATIM:UI64:1
		AVTP:FC32:'UI32'
		APER:FC32:'READ'
		AVAL:CONT:
			0x00000000:UI32:2
			0x00000001:UI32:908767
		END
	END
	GPVD:CONT:
		AVER:UI32:2
		ATIM:UI64:1
		AVTP:FC32:'UI64'
		APER:FC32:'READ'
		AVAL:CONT:
			0x00000000:UI32:2
			0x00000001:UI64:1484722540084888
		END
	END
	GVND:CONT:
		AVER:UI32:2
		ATIM:UI64:1
		AVTP:FC32:'CSTR'
		APER:FC32:'READ'
		AVAL:CONT:
			0x00000000:UI32:2
			0x00000001:CSTR:"{OID='2.16.124.113590.3.1.3.3.1'}"
		END
	END
	GSIV:CONT:
		AVER:UI32:2
		ATIM:UI64:1
		AVTP:FC32:'CSTR'
		APER:FC32:'READ'
		AVAL:CONT:
			0x00000000:UI32:2
			0x00000001:CSTR:"10.4.0"
		END
	END
END
`

// Tests of atom path matching
var TestAtom1 = new(Atom)
var TestAtomGINF = new(Atom)

func init() {
	err := TestAtom1.UnmarshalText([]byte(TestAtom1Text))
	if err != nil {
		panic(err)
	}
	err = TestAtomGINF.UnmarshalText([]byte(TestAtomGINFText))
	if err != nil {
		panic(err)
	}
}

// Expected behaviour is intended to parallel XPath as closely as possible.
// Test expectations are based on behaviour from this XPath test:
//     http://www.freeformatter.com/xpath-tester.html
func TestAtomsAtPath(t *testing.T) {
	// Placeholder error for stuff that should return error but I haven't
	// written the error yet
	zero := []string{}
	allAtoms := "ROOT:CONT: 0001:CONT: LEAF:UI32:1 LEAF:UI32:2 LEAF:UI32:3 0002:CONT: LEAF:UI32:4 LEAF:UI32:5 LEAF:UI32:6 0003:CONT: LEAF:UI32:7 LEAF:UI32:8 LEAF:UI32:9"
	tests := []PathTest{
		// Part 1 -- test paths with no filters

		// Empty path request returns empty result and no error
		PathTest{TestAtom1, "", zero, errInvalidPath(`""`)},

		// Single slash path request returns root element only (which contains entire doc)
		PathTest{TestAtom1, "/", []string{"ROOT:CONT:"}, nil},
		PathTest{TestAtom1, "/ROOT", []string{"ROOT:CONT:"}, nil},

		// predicates ended with / are errors, except a couple of special cases
		PathTest{TestAtom1, "/ROOT/", zero, errInvalidPath("/ROOT/")},

		// Double slash prefix means all matching atoms at any level
		PathTest{TestAtom1, "//", zero, errInvalidPath("//")},
		PathTest{TestAtom1, "//LEAF", []string{
			"LEAF:UI32:1", "LEAF:UI32:2", "LEAF:UI32:3", "LEAF:UI32:4", "LEAF:UI32:5",
			"LEAF:UI32:6", "LEAF:UI32:7", "LEAF:UI32:8", "LEAF:UI32:9"}, nil},
		PathTest{TestAtom1, "//leaf", zero, nil}, // case sensitive
		// FIXME: test that CONT with same name as its leaf can be found

		// "//*" returns every atom in the tree as a separate element.
		// This differs from "/" which returns entire tree as 1 element.
		PathTest{TestAtom1, "//*", strings.Split(allAtoms, " "), nil},

		// Individual atoms can be found
		PathTest{TestAtom1, "ROOT/0001", []string{"0001:CONT:"}, nil},
		PathTest{TestAtom1, "/ROOT/0001", []string{"0001:CONT:"}, nil},
		PathTest{TestAtom1, "/ROOT/0002", []string{"0002:CONT:"}, nil},
		PathTest{TestAtom1, "/ROOT/0003", []string{"0003:CONT:"}, nil},
		PathTest{TestAtom1, "0001", zero, nil},

		// Multiple atoms can be found from same branch
		PathTest{TestAtom1, "ROOT/0001/LEAF", []string{
			"LEAF:UI32:1", "LEAF:UI32:2", "LEAF:UI32:3"}, nil},

		// Multiple atoms can be found from different branches
		PathTest{TestAtom1, "ROOT/*/LEAF", []string{
			"LEAF:UI32:1", "LEAF:UI32:2", "LEAF:UI32:3", "LEAF:UI32:4", "LEAF:UI32:5",
			"LEAF:UI32:6", "LEAF:UI32:7", "LEAF:UI32:8", "LEAF:UI32:9"}, nil},

		PathTest{TestAtomGINF, "GINF/*/AVAL/0x00000001", []string{
			"0x00000001:UI32:908767",
			"0x00000001:UI64:1484722540084888",
			`0x00000001:CSTR:"{OID='2.16.124.113590.3.1.3.3.1'}"`,
			`0x00000001:CSTR:"10.4.0"`}, nil,
		},
		PathTest{TestAtomGINF, "GINF/*/AVAL/*", []string{
			"0x00000000:UI32:2",
			"0x00000001:UI32:908767",
			"0x00000000:UI32:2",
			"0x00000001:UI64:1484722540084888",
			"0x00000000:UI32:2",
			`0x00000001:CSTR:"{OID='2.16.124.113590.3.1.3.3.1'}"`,
			"0x00000000:UI32:2",
			`0x00000001:CSTR:"10.4.0"`}, nil,
		},

		// Test arithmetic operators
		PathTest{TestAtom1, "ROOT/*/LEAF[0]", []string{}, nil}, // there's no 0 index, as per XPath convention.
		PathTest{TestAtom1, "ROOT/*/LEAF[1]", []string{"LEAF:UI32:1"}, nil},
		PathTest{TestAtom1, "ROOT/*/LEAF[2]", []string{"LEAF:UI32:2"}, nil},
		PathTest{TestAtom1, "ROOT/*/LEAF[1+1-1*1]", []string{"LEAF:UI32:1"}, nil},
		PathTest{TestAtom1, "ROOT/*/LEAF[1+((1-1)*1)]", []string{"LEAF:UI32:1"}, nil},
		PathTest{TestAtom1, "ROOT/*/LEAF[2+-1]", []string{"LEAF:UI32:1"}, nil},
		PathTest{TestAtom1, "ROOT/0001/LEAF[0=0]", []string{"LEAF:UI32:1", "LEAF:UI32:2", "LEAF:UI32:3"}, nil},
		PathTest{TestAtom1, "ROOT/0001/LEAF[ 0 = 0 ]", []string{"LEAF:UI32:1", "LEAF:UI32:2", "LEAF:UI32:3"}, nil},
		PathTest{TestAtom1, "ROOT/0001/LEAF[0=1]", []string{}, nil},

		// division is "div" not "/". Div and mod are operators, not functions, so no ()
		PathTest{TestAtom1, "ROOT[64/8-7]", []string{}, errInvalidPredicate("ROOT[64/8-7]")},
		PathTest{TestAtom1, "ROOT[64 div 8-7]", []string{"ROOT:CONT:"}, nil},
		//		PathTest{TestAtom1, "ROOT[-7+64 div 8]", []string{"ROOT:CONT:"}, nil},
		//		PathTest{TestAtom1, "ROOT[0.25 * 4]", []string{"ROOT:CONT:"}, nil},
		//		PathTest{TestAtom1, "ROOT[11 mod 10]", []string{"ROOT:CONT:"}, nil},
		//PathTest{TestAtom1, "ROOT[64 shazbot 8]", []string{}, errInvalidPredicate("ROOT/ROOT/LEAF[64 shazbot 8]")},

		// test that operator precedence follows correct order of operations

		// // test XPath functions
		//		PathTest{TestAtom1, "ROOT/0001/*[position() = 1]", []string{"LEAF:UI32:1"}, nil},
		//		PathTest{TestAtom1, "ROOT/0001/*[count() = 1]", []string{"LEAF:UI32:1"}, nil},
		//		PathTest{TestAtom1, "ROOT/0001/*[count() = position()]", []string{"LEAF:UI32:1"}, nil},
		//		PathTest{TestAtom1, "ROOT/0001/*[last()]", []string{"LEAF:UI32:1"}, nil},
		//		PathTest{TestAtom1, "ROOT/0001/*[not last()]", []string{}, nil},

		// 		// test path specification by index.  start from 1 like xpath.
		// 		// xpath returns no error on request for 0 index, even though it cannot exist.
		// 		// xpath in general favours returning no results over returning an error.
		// 		PathTest{"CN1A/DOGS[0]", []string{}, nil},
		// 		PathTest{"CN1A/DOGS[1]", []string{`DOGS:UI32:1`}, nil},
		// 		PathTest{"CN1A/DOGS[2]", []string{`DOGS:UI32:2`}, nil},
		// 		PathTest{"CN1A/DOGS[3]", []string{`DOGS:UI32:3`}, nil},
		// 		PathTest{"CN1A/DOGS[4]", []string{}, nil},
		// 		PathTest{"*/DOGS[4]", []string{`DOGS:UI32:12`}, nil},
		// 		PathTest{"*/DOGS[5]", []string{`DOGS:UI32:23`}, nil},
		// 		PathTest{"*/DOGS[6]", []string{}, nil},
		//
		// 		// FIXME what if ] is part of the name?  use delimiters? require hex specificiation?  require 4 chars or hex?
		// 		PathTest{"CN1A/*[@name=\"DOGS\"]", []string{`DOGS:UI32:1`, `DOGS:UI32:2`, `DOGS:UI32:3`}, nil},
		// 		PathTest{"CN1A/*[@name='DOGS']", []string{`DOGS:UI32:1`, `DOGS:UI32:2`, `DOGS:UI32:3`}, nil},
		// 		PathTest{"CN1A/*[@name=DOGS]", []string{`DOGS:UI32:1`, `DOGS:UI32:2`, `DOGS:UI32:3`}, nil},
		// 		PathTest{"CN1A/DOGS", []string{`DOGS:UI32:1`, `DOGS:UI32:2`, `DOGS:UI32:3`}, nil},
		// 		PathTest{TestAtomGINF, "//AVAL/@name > 0", []string{`DOGS:UI32:1`, `DOGS:UI32:2`, `DOGS:UI32:3`}, nil},
		//
		// 		// syntactically valid but semantically a contradiction, name CN1A != name DOGS
		// 		PathTest{"CN1A[@name=DOGS]", []string{}, nil},
		//
		// 		PathTest{"CN1A/*[position()>3]", []string{"CATS:UI32:1", "CN2A:CONT:"}, nil},
		// 		PathTest{"*[not(@type=CONT)]", []string{"DOGS:UI32:1", "DOGS:UI32:2", "DOGS:UI32:3", "CATS:UI32:1"}, nil},
		// 		PathTest{"CN1A[not(@type=CONT) and not(@name=DOGS)]", []string{"CATS:UI32:1"}, nil},
		// 		PathTest{"CN1A/DOGS[@data>=2]", []string{
		// 			`DOGS:UI32:2`,
		// 			`DOGS:UI32:3`}, nil,
		// 		},
		// 		PathTest{"CN1A/*[@data<2]", []string{`DOGS:UI32:1`, `CATS:UI32:1`}, nil},
		//
		// 		PathTest{"THER/E IS/NOTH/INGH/ERE.", zero, fmt.Errorf("atom 'ROOT' has no container child named 'THER'")},
		// 		PathTest{"CN1A/CN2A/CN3A/LF4B/LEAF", zero, fmt.Errorf("atom 'ROOT/CN1A/CN2A/CN3A' has no container child named 'LF4B'")},
	}
	runPathTests(t, tests)
}
func runPathTests(t *testing.T, tests []PathTest) {
	for _, test := range tests {
		atoms, gotErr := test.Atom.AtomsAtPath(test.Input)

		// check for expected error result
		switch {
		case gotErr == nil && test.WantError == nil:
		case gotErr != nil && test.WantError == nil:
			t.Errorf("%s: got err {%s}, want err <nil>", test.Input, gotErr)
		case gotErr == nil && test.WantError != nil:
			t.Errorf("%s: got err <nil>, want err {%s}", test.Input, test.WantError)
		case gotErr.Error() != test.WantError.Error():
			t.Errorf("%s: got err {%s}, want err {%s}", test.Input, gotErr, test.WantError)
		}

		// convert result atoms to string representations
		var results []string
		for _, a := range atoms {
			results = append(results, strings.TrimSpace(a.String()))
		}

		// compare each result atom in order
		if len(results) != len(test.WantValue) {
			t.Errorf("%s: got %d elements {%v}, want %d elements {%v}", test.Input, len(results), results, len(test.WantValue), test.WantValue)
			continue
		}
		for i, want := range test.WantValue {
			if want != results[i] {
				t.Errorf("%s: got {%s}, want {%s}", test.Input, results[i], want)
			}
		}
	}
}
