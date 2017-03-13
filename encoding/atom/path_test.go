// Benchmark Marshal / Unmarshal functions
package atom

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
		Input     string
		WantValue []string
		WantError error
	}
)

// Tests of atom path matching
// TODO:
// test case where target name appears earlier in the path too
var TestAtom = new(Atom)

func init() {
	TestAtom.UnmarshalText([]byte(`
ROOT:CONT:
  CN1A:CONT:
		DOGS:UI32:1
    CN2A:CONT:
      CN3A:CONT:
        CN4A:CONT:
          LF5A:UI32:1
          LF5B:CSTR:"hello from depth 5"
        END
        LF4B:CSTR:"hello from depth 4"
      END
    END
  END
  CN1B:CONT:
		DOGS:UI32:2
    NODE:CONT:
      NODE:CONT:
        NODE:CONT:
          NODE:CONT:
            NODE:CONT:
              NODE:CONT:
                NODE:USTR:"branch1 result"
              END
            END
          END
          NODE:CONT:
            NODE:CONT:
              NODE:CONT:
                NODE:USTR:"branch2 result"
              END
            END
          END
          NODE:CONT:
            NODE:CONT:
              NODE:CONT:
                NODE:USTR:"branch3 result"
              END
            END
          END
          NODE:USTR:"too much NODE"
        END
      END
    END
  END
  CN1C:CONT:
    DOGS:UI32:3
  END
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
END
`))
}

func TestAtomsAtPath(t *testing.T) {

	zero := []string{}
	tests := []PathTest{
		PathTest{"CN1A/CN2A/CN3A/CN4A/LF5A", []string{"LF5A:UI32:1"}, nil},
		PathTest{"CN1A/CN2A/CN3A/LF4B", []string{`LF4B:CSTR:"hello from depth 4"`}, nil},
		PathTest{"CN1A/CN2A/CN3A/CN4A/LF5B", []string{`LF5B:CSTR:"hello from depth 5"`}, nil},
		PathTest{"CN1B/NODE/NODE/NODE/NODE/NODE/NODE/NODE", []string{
			`NODE:USTR:"branch1 result"`,
			`NODE:USTR:"branch2 result"`,
			`NODE:USTR:"branch3 result"`}, nil,
		},
		PathTest{"*/DOGS", []string{
			`DOGS:UI32:1`,
			`DOGS:UI32:2`,
			`DOGS:UI32:3`}, nil,
		},
		PathTest{"GINF/*/AVAL/0x00000001", []string{
			`0x00000001:UI32:908767`,
			`0x00000001:UI64:1484722540084888`,
			`0x00000001:CSTR:"{OID='2.16.124.113590.3.1.3.3.1'}"`,
			`0x00000001:CSTR:"10.4.0"`}, nil,
		},
		PathTest{"GINF/*/AVAL/*", []string{
			`0x00000000:UI32:2`,
			`0x00000001:UI32:908767`,
			`0x00000000:UI32:2`,
			`0x00000001:UI64:1484722540084888`,
			`0x00000000:UI32:2`,
			`0x00000001:CSTR:"{OID='2.16.124.113590.3.1.3.3.1'}"`,
			`0x00000000:UI32:2`,
			`0x00000001:CSTR:"10.4.0"`}, nil,
		},
		PathTest{"CN1A/NONE", zero, nil},

		PathTest{"THER/E IS/NOTH/INGH/ERE.", zero, fmt.Errorf("atom 'ROOT' has no container child named 'THER'")},
		PathTest{"CN1A/CN2A/CN3A/LF4B/LEAF", zero, fmt.Errorf("atom 'ROOT/CN1A/CN2A/CN3A' has no container child named 'LF4B'")},
	}
	runPathTests(t, tests)
}
func runPathTests(t *testing.T, tests []PathTest) {
	for _, test := range tests {
		atoms, gotErr := TestAtom.AtomsAtPath(test.Input)

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
		for i, want := range test.WantValue {
			if want != results[i] {
				t.Errorf("%s: got %s, want %s", test.Input, results[i], want)
			}
		}
	}
}
