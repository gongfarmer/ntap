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
	"fmt"
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

const TestAtom2Text = `
ROOT:CONT:
		UI_1:UI64:1
		UIMX:UI64:0xFFFFFFFF
		SI_N:SI64:-10
		SI_P:SI64:15
		FP_P:FP64:15.5
		FP_N:FP64:-15.5
END
`

// Tests of atom path matching
var TestAtom1 = new(Atom)
var TestAtom2 = new(Atom)
var TestAtomGINF = new(Atom)

func init() {
	err := TestAtom1.UnmarshalText([]byte(TestAtom1Text))
	if err != nil {
		panic(err)
	}
	err = TestAtom2.UnmarshalText([]byte(TestAtom2Text))
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
		PathTest{TestAtom1, "", zero, errInvalidPath(`<empty> in ""`)},

		// Single slash path request returns root element only (which contains entire doc)
		PathTest{TestAtom1, "/", []string{"ROOT:CONT:"}, nil},
		PathTest{TestAtom1, "/ROOT", []string{"ROOT:CONT:"}, nil},

		// predicates ended with / are errors, except a couple of special cases
		PathTest{TestAtom1, "/ROOT/", zero, errInvalidPath(`<empty> in "/ROOT/"`)},

		// Double slash prefix means all matching atoms at any level
		PathTest{TestAtom1, "//", zero, errInvalidPath(`<empty> in "//"`)},
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

		// test path indexing.  indexing starts from 1 not 0 (this is an XPath convention.)
		// xpath returns no error on request for 0 index, even though it cannot exist.
		PathTest{TestAtom1, "ROOT/0001/LEAF[0]", []string{}, nil},
		PathTest{TestAtom1, "ROOT/0001/LEAF[1]", []string{"LEAF:UI32:1"}, nil},
		PathTest{TestAtom1, "ROOT/0001/LEAF[2]", []string{"LEAF:UI32:2"}, nil},
		PathTest{TestAtom1, "ROOT/0001/LEAF[3]", []string{"LEAF:UI32:3"}, nil},
		PathTest{TestAtom1, "ROOT/0001/LEAF[4]", []string{}, nil},
		PathTest{TestAtom1, "/ROOT/*/LEAF[4]", []string{"LEAF:UI32:4"}, nil},
		PathTest{TestAtom1, "/ROOT/*/LEAF[5]", []string{"LEAF:UI32:5"}, nil},
		PathTest{TestAtom1, "/ROOT/*/LEAF[6]", []string{"LEAF:UI32:6"}, nil},
		PathTest{TestAtom1, "/ROOT/*/LEAF[100]", []string{}, nil},

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

		// test that operator precedence follows correct order of operations
		PathTest{TestAtom1, "ROOT[ 3 + 4 * 2 div ( 1 - 5 )]", []string{"ROOT:CONT:"}, nil},
		PathTest{TestAtom1, "ROOT[64 div 8-7]", []string{"ROOT:CONT:"}, nil},
		PathTest{TestAtom1, "ROOT[-7+64 div 8]", []string{"ROOT:CONT:"}, nil},
		PathTest{TestAtom1, "ROOT[0.25 * 4]", []string{"ROOT:CONT:"}, nil},
		PathTest{TestAtom1, "ROOT[11 mod 10]", []string{"ROOT:CONT:"}, nil},

		// division is "div" not "/".
		PathTest{TestAtom1, "ROOT[64/8-7]", []string{}, fmt.Errorf(`invalid predicate: unterminated square brackets in "ROOT[64/8-7]"`)},

		// handle gibberish operators gracefully
		PathTest{TestAtom1, "ROOT[64 shazbot 8]", []string{}, fmt.Errorf(`invalid predicate: unrecognized token 'shazbot' in "ROOT[64 shazbot 8]"`)},

		// test XPath functions
		PathTest{TestAtom1, "ROOT[position() = 1]", []string{"ROOT:CONT:"}, nil},
		PathTest{TestAtom1, "ROOT[count() = 1]", []string{"ROOT:CONT:"}, nil},
		PathTest{TestAtom1, "ROOT[count() = position()]", []string{"ROOT:CONT:"}, nil},
		PathTest{TestAtom1, "ROOT[last()]", []string{"ROOT:CONT:"}, nil},
		PathTest{TestAtom1, "ROOT[not(last())]", zero, errInvalidPredicate(`expect boolean, got 'last' in "ROOT[not(last())]"`)},
		PathTest{TestAtom1, "ROOT[not(not(last()))]", zero, errInvalidPredicate(`expect boolean, got 'last' in "ROOT[not(not(last()))]"`)},
		PathTest{TestAtom1, "ROOT[shazbot()]", zero, errInvalidPredicate(`unrecognized function "shazbot" in "ROOT[shazbot()]"`)},
		PathTest{TestAtom1, "ROOT[shazbot(5)]", zero, errInvalidPredicate(`unrecognized function "shazbot" in "ROOT[shazbot(5)]"`)},
		PathTest{TestAtom1, "ROOT[not(shazbot())]", zero, errInvalidPredicate(`unrecognized function "shazbot" in "ROOT[not(shazbot())]"`)},

		// test usage of attributes which retrieve atom data

		// multiple delimiters (or no delimiters) are accepted
		PathTest{TestAtom1, `ROOT/0001/LEAF[@name="LEAF"]`, []string{`LEAF:UI32:1`, `LEAF:UI32:2`, `LEAF:UI32:3`}, nil},
		PathTest{TestAtom1, "ROOT/0001/LEAF[@name='LEAF']", []string{`LEAF:UI32:1`, `LEAF:UI32:2`, `LEAF:UI32:3`}, nil},

		PathTest{TestAtom1, "//LEAF",
			strings.Split("LEAF:UI32:1 LEAF:UI32:2 LEAF:UI32:3 LEAF:UI32:4 LEAF:UI32:5 LEAF:UI32:6 LEAF:UI32:7 LEAF:UI32:8 LEAF:UI32:9", " "), nil},
		PathTest{TestAtom1, "//LEAF[ @data = 2 ] ", []string{"LEAF:UI32:2"}, nil},
		PathTest{TestAtom1, "//*[@data=3]", []string{"LEAF:UI32:3"}, nil},
		PathTest{TestAtomGINF, "//AVAL/*", []string{
			"0x00000000:UI32:2",
			"0x00000001:UI32:908767",
			"0x00000000:UI32:2",
			"0x00000001:UI64:1484722540084888",
			"0x00000000:UI32:2",
			`0x00000001:CSTR:"{OID='2.16.124.113590.3.1.3.3.1'}"`,
			"0x00000000:UI32:2",
			`0x00000001:CSTR:"10.4.0"`,
		}, nil},
		PathTest{TestAtomGINF, "//AVAL/[@name > 0]", []string{}, errInvalidPath(`<empty> in "//AVAL/[@name > 0]"`)},
		PathTest{TestAtomGINF, "//AVAL/*[@name > 0]", []string{
			"0x00000001:UI32:908767",
			"0x00000001:UI64:1484722540084888",
			`0x00000001:CSTR:"{OID='2.16.124.113590.3.1.3.3.1'}"`,
			`0x00000001:CSTR:"10.4.0"`,
		}, nil},
		PathTest{TestAtom1, "/ROOT[@name=NONE]", []string{}, nil},

		// Test less-than operator and its type conversions
		PathTest{TestAtom2, "/ROOT/UI_1[@data < 2]", []string{"UI_1:UI64:1"}, nil},
		PathTest{TestAtom2, "/ROOT[UI_1 < 2]", strings.Split("ROOT:CONT:", " "), nil},
		PathTest{TestAtom2, "/ROOT[UI_1 < 2.0]", []string{"ROOT:CONT:"}, nil},
		PathTest{TestAtom2, "/ROOT[2 < UI_1]", []string{}, nil},
		PathTest{TestAtom2, "/ROOT[UI_1 < UIMX]", []string{"ROOT:CONT:"}, nil},
		PathTest{TestAtom2, "/ROOT[SI_N < UIMX]", []string{"ROOT:CONT:"}, nil},
		PathTest{TestAtom2, "/ROOT[SI_N < UI_1]", []string{"ROOT:CONT:"}, nil},
		PathTest{TestAtom2, "/ROOT[UI_1 < SI_P]", []string{"ROOT:CONT:"}, nil},
		PathTest{TestAtom2, "/ROOT[UI_1 < FP_P]", []string{"ROOT:CONT:"}, nil},
		PathTest{TestAtom2, "/ROOT[FP_N < UI_1]", []string{"ROOT:CONT:"}, nil},
		PathTest{TestAtom2, "/ROOT[FP_N < UIMX]", []string{"ROOT:CONT:"}, nil},
		PathTest{TestAtom2, "/ROOT[FP_N < SI_N]", []string{"ROOT:CONT:"}, nil},
		PathTest{TestAtom2, "/ROOT[FP_N < SI_P]", []string{"ROOT:CONT:"}, nil},
		PathTest{TestAtom2, "/ROOT[FP_N < FP_P]", []string{"ROOT:CONT:"}, nil},

		// Test greater-than operator and its type conversions
		PathTest{TestAtom2, "/ROOT/UI_1[2 > @data]", []string{"UI_1:UI64:1"}, nil},
		PathTest{TestAtom2, "/ROOT[2 > UI_1]", strings.Split("ROOT:CONT:", " "), nil},
		PathTest{TestAtom2, "/ROOT[2.0 > UI_1]", []string{"ROOT:CONT:"}, nil},
		PathTest{TestAtom2, "/ROOT[UI_1 > 2]", []string{}, nil},
		PathTest{TestAtom2, "/ROOT[UIMX > UI_1]", []string{"ROOT:CONT:"}, nil},
		PathTest{TestAtom2, "/ROOT[UIMX > SI_N]", []string{"ROOT:CONT:"}, nil},
		PathTest{TestAtom2, "/ROOT[UI_1 > SI_N]", []string{"ROOT:CONT:"}, nil},
		PathTest{TestAtom2, "/ROOT[SI_P > UI_1]", []string{"ROOT:CONT:"}, nil},
		PathTest{TestAtom2, "/ROOT[FP_P > UI_1]", []string{"ROOT:CONT:"}, nil},
		PathTest{TestAtom2, "/ROOT[UI_1 > FP_N]", []string{"ROOT:CONT:"}, nil},
		PathTest{TestAtom2, "/ROOT[UIMX > FP_N]", []string{"ROOT:CONT:"}, nil},
		PathTest{TestAtom2, "/ROOT[SI_N > FP_N]", []string{"ROOT:CONT:"}, nil},
		PathTest{TestAtom2, "/ROOT[SI_P > FP_N]", []string{"ROOT:CONT:"}, nil},
		PathTest{TestAtom2, "/ROOT[FP_P > FP_N]", []string{"ROOT:CONT:"}, nil},

		// Test equals operator and its type conversions
		PathTest{TestAtom2, "/ROOT/UI_1[2 = @data]", zero, nil},
		PathTest{TestAtom2, "/ROOT[2 = UI_1]", zero, nil},
		PathTest{TestAtom2, "/ROOT[2.0 = UI_1]", zero, nil},
		PathTest{TestAtom2, "/ROOT[UI_1 = 2]", zero, nil},
		PathTest{TestAtom2, "/ROOT[UIMX = UI_1]", zero, nil},
		PathTest{TestAtom2, "/ROOT[UIMX = SI_N]", zero, nil},
		PathTest{TestAtom2, "/ROOT[UI_1 = SI_N]", zero, nil},
		PathTest{TestAtom2, "/ROOT[SI_P = UI_1]", zero, nil},
		PathTest{TestAtom2, "/ROOT[FP_P = UI_1]", zero, nil},
		PathTest{TestAtom2, "/ROOT[UI_1 = FP_N]", zero, nil},
		PathTest{TestAtom2, "/ROOT[UIMX = FP_N]", zero, nil},
		PathTest{TestAtom2, "/ROOT[SI_N = FP_N]", zero, nil},
		PathTest{TestAtom2, "/ROOT[SI_P = FP_N]", zero, nil},
		PathTest{TestAtom2, "/ROOT[FP_P = FP_N]", zero, nil},
		PathTest{TestAtom2, "/ROOT[UI_1 = UI_1]", []string{"ROOT:CONT:"}, nil},
		PathTest{TestAtom2, "/ROOT[2.0 = 2]", []string{"ROOT:CONT:"}, nil},
		PathTest{TestAtom2, "/ROOT[UIMX = UIMX]", []string{"ROOT:CONT:"}, nil},
		PathTest{TestAtom2, "/ROOT[SI_N = SI_N]", []string{"ROOT:CONT:"}, nil},
		PathTest{TestAtom2, "/ROOT[SI_P = SI_P]", []string{"ROOT:CONT:"}, nil},
		PathTest{TestAtom2, "/ROOT[FP_P = FP_P]", []string{"ROOT:CONT:"}, nil},
		PathTest{TestAtom2, "/ROOT[FP_N = FP_N]", []string{"ROOT:CONT:"}, nil},

		// Test plus operator and its type conversions
		PathTest{TestAtom2, "/ROOT[UI_1 + 2 = 3]", strings.Split("ROOT:CONT:", " "), nil},
		PathTest{TestAtom2, "/ROOT[UI_1 + 2.0 = 0x00000003]", []string{"ROOT:CONT:"}, nil},
		PathTest{TestAtom2, "/ROOT[SI_N + UI_1 * 11]", []string{"ROOT:CONT:"}, nil},
		PathTest{TestAtom2, "/ROOT[5 = SI_N + SI_P]", []string{"ROOT:CONT:"}, nil},
		PathTest{TestAtom2, "/ROOT[SI_N + UI_1 = -9]", []string{"ROOT:CONT:"}, nil},
		PathTest{TestAtom2, "/ROOT[SI_P + UI_1 = 16]", []string{"ROOT:CONT:"}, nil},
		PathTest{TestAtom2, "/ROOT[SI_N + FP_P = 5.5]", []string{"ROOT:CONT:"}, nil},
		PathTest{TestAtom2, "/ROOT[SI_N + FP_N = -25.5]", []string{"ROOT:CONT:"}, nil},
		//		PathTest{TestAtom2, "/ROOT[UI_1 < SI_P]", []string{"ROOT:CONT:"}, nil},
		//		PathTest{TestAtom2, "/ROOT[UI_1 < FP_P]", []string{"ROOT:CONT:"}, nil},
		//		PathTest{TestAtom2, "/ROOT[FP_N < UI_1]", []string{"ROOT:CONT:"}, nil},
		//		PathTest{TestAtom2, "/ROOT[FP_N < UIMX]", []string{"ROOT:CONT:"}, nil},
		//		PathTest{TestAtom2, "/ROOT[FP_N < SI_N]", []string{"ROOT:CONT:"}, nil},
		//		PathTest{TestAtom2, "/ROOT[FP_N < SI_P]", []string{"ROOT:CONT:"}, nil},
		//		PathTest{TestAtom2, "/ROOT[FP_N < FP_P]", []string{"ROOT:CONT:"}, nil},

		// Test minus operator and its type conversions
		PathTest{TestAtom2, "/ROOT[0xFFFFFFFE = UIMX - UI_1]", []string{"ROOT:CONT:"}, nil},

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

		// 		// FIXME what if ] is part of the name?  use delimiters? require hex specificiation?  require 4 chars or hex?
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
