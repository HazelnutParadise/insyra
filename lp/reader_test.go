package lp

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func mustReadLP(t *testing.T, text string) *problem {
	t.Helper()
	p, err := readLP("test.lp", []byte(text))
	if err != nil {
		t.Fatalf("readLP: %v\n%s", err, text)
	}
	return p
}

func columnNames(p *problem) []string {
	names := make([]string, len(p.cols))
	for i, c := range p.cols {
		names[i] = c.name
	}
	return names
}

func TestReadLPReadsAModel(t *testing.T) {
	p := mustReadLP(t, `\ a comment on its own line
Maximize
 profit: 3 x + 2y - 0 w
Subject To
 c1: x + y <= 4
 x + 3 y =< 6
 c3: - x >= -10
 c4: 2 x \ a comment inside a row, which may span lines
     + 2e0 y = 4
End
`)
	if !p.maximize {
		t.Error("read as a minimisation")
	}
	if got := columnNames(p); !reflect.DeepEqual(got, []string{"x", "y", "w"}) {
		t.Errorf("columns %v, want [x y w] (w is named, so it exists, but its zero term is dropped)", got)
	}
	if want := []term{{0, 3}, {1, 2}}; !reflect.DeepEqual(p.objective, want) {
		t.Errorf("objective %v, want %v", p.objective, want)
	}
	want := []row{
		{terms: []term{{0, 1}, {1, 1}}, rel: lessEq, rhs: 4},
		{terms: []term{{0, 1}, {1, 3}}, rel: lessEq, rhs: 6},
		{terms: []term{{0, -1}}, rel: greaterEq, rhs: -10},
		{terms: []term{{0, 2}, {1, 2}}, rel: equalTo, rhs: 4},
	}
	if !reflect.DeepEqual(p.rows, want) {
		t.Errorf("rows %+v, want %+v", p.rows, want)
	}
	for _, c := range p.cols {
		if c.lower != 0 || !math.IsInf(c.upper, 1) || c.integer {
			t.Errorf("%s: bounds [%v, %v] integer %v, want the default [0, +inf] continuous", c.name, c.lower, c.upper, c.integer)
		}
	}
	if len(p.warnings) != 0 {
		t.Errorf("unexpected warnings %v", p.warnings)
	}
}

func TestReadLPKeywordSpellings(t *testing.T) {
	for i, words := range [][6]string{
		{"min", "st", "bounds", "general", "binary", "end"},
		{"MINIMUM", "s.t.", "bound", "generals", "binaries", "END"},
		{"minimize", "st.", "Bounds", "gen", "bin", "End"},
		{"max", "subject to", "BOUNDS", "integer", "Binary", "end"},
		{"Maximum", "SUCH THAT", "bounds", "int", "bin", "end"},
		{"maximize", "Subject To", "bounds", "integers", "binary", "end"},
	} {
		t.Run(words[0]+" "+words[1], func(t *testing.T) {
			p := mustReadLP(t, fmt.Sprintf("%s\n obj: x + y\n%s\n c: x + y <= 4\n%s\n x <= 3\n%s\n x\n%s\n y\n%s\n",
				words[0], words[1], words[2], words[3], words[4], words[5]))
			if p.maximize != (i >= 3) {
				t.Errorf("maximize %v", p.maximize)
			}
			want := []column{{name: "x", lower: 0, upper: 3, integer: true}, {name: "y", lower: 0, upper: 1, integer: true}}
			if !reflect.DeepEqual(p.cols, want) {
				t.Errorf("columns %+v, want %+v", p.cols, want)
			}
		})
	}
}

// Columns are numbered by where a name first appears, section by section.
func TestReadLPColumnOrder(t *testing.T) {
	p := mustReadLP(t, "min\n obj: y\nst\n c: x + y >= 1\nbounds\n z <= 4\ngeneral\n w\nend\n")
	if got := columnNames(p); !reflect.DeepEqual(got, []string{"y", "x", "z", "w"}) {
		t.Errorf("columns %v, want [y x z w]", got)
	}
}

func TestReadLPBounds(t *testing.T) {
	inf := math.Inf(1)
	tests := []struct {
		bound        string
		lower, upper float64
	}{
		{"x free", -inf, inf},
		{"x f", -inf, inf}, // GLPK takes any prefix of "free"
		{"-inf <= x <= 5", -inf, 5},
		{"-infinity <= x", -inf, inf},
		{"x >= -inf", -inf, inf},
		{"x >= -3", -3, inf},
		{"x > 1", 1, inf},
		{"x <= +inf", 0, inf},
		{"x < 1", 0, 1},
		{"x <= -2", 0, -2}, // crossed; refused when solving, as GLPK does
		{"x = 3", 3, 3},
		{"x = -3", -3, -3},
		{"2 <= x <= 7", 2, 7},
		{"x => 1.5", 1.5, inf},
	}
	for _, tt := range tests {
		t.Run(tt.bound, func(t *testing.T) {
			p := mustReadLP(t, "min\n obj: x\nst\n c: x >= -100\nbounds\n "+tt.bound+"\nend\n")
			if c := p.cols[0]; c.lower != tt.lower || c.upper != tt.upper {
				t.Errorf("bounds [%v, %v], want [%v, %v]", c.lower, c.upper, tt.lower, tt.upper)
			}
		})
	}
}

// Since GLPK 4.52 a Binary variable keeps a bound from the Bounds section and
// only gains 0 or 1 on an open side. solvemodel_format.lp depends on it.
func TestReadLPBinaryKeepsBounds(t *testing.T) {
	p := mustReadLP(t, "max\n obj: x + y + z\nst\n c: x + y <= 10\nbounds\n 0 <= x <= 4\n y >= -1\nbinary\n x\n y\n z\nend\n")
	want := []column{
		{name: "x", lower: 0, upper: 4, integer: true},
		{name: "y", lower: -1, upper: 1, integer: true},
		{name: "z", lower: 0, upper: 1, integer: true},
	}
	if !reflect.DeepEqual(p.cols, want) {
		t.Errorf("columns %+v, want %+v", p.cols, want)
	}
}

func TestReadLPKeywordsOnlyStartALine(t *testing.T) {
	// Indented, "subject to" is two variable names.
	_, err := readLP("test.lp", []byte("min\n obj: x\n subject to\n c: x <= 1\nend\n"))
	if err == nil || err.Error() != "test.lp:3: constraints section missing" {
		t.Errorf("indented keyword: %v", err)
	}
	// At the start of a line, "e" is a prefix of "end".
	_, err = readLP("test.lp", []byte("min\n obj: x + e\nst\ne: x >= 1\nend\n"))
	if err == nil || err.Error() != "test.lp:4: missing variable name" {
		t.Errorf("prefix keyword: %v", err)
	}
}

// readErrorCases are LP texts GLPK refuses, with the message it prints. The
// GLPK comparison below checks every one that is marked as matching.
var readErrorCases = []struct {
	name, text, want string
	glpkDiffers      bool
}{
	{"missing right-hand side", "min\n obj: x\nst\n c: x >=\nend\n", "test.lp:5: missing right-hand side", false},
	{"number out of range", "min\n obj: x\nst\n c: x >= 1e400\nend\n", "test.lp:4: numeric constant '1e400' out of range", false},
	{"incomplete exponent", "min\n obj: 3ex\nst\n c: x >= 1\nend\n", "test.lp:2: numeric constant '3e' incomplete", false},
	{"lone decimal point", "min\n obj: . x\nst\n c: x >= 1\nend\n", "test.lp:2: invalid use of decimal point", false},
	{"an unnamed row's name taken later", "min\n obj: x\nst\n x >= 1\n r.4: x >= 2\nend\n", "test.lp:5: constraint 'r.4' multiply defined", false},
	{"a variable twice in one form", "min\n obj: x + x\nst\n c: x >= 1\nend\n", "test.lp:2: multiple use of variable 'x' not allowed", false},
	{"a comment after the right-hand side", "min\n obj: x\nst\n c: x >= 1 \\ note\nend\n", "test.lp:4: invalid symbol(s) beyond right-hand side", false},
	{"a control character", "min\n obj: x\x01\nst\n c: x >= 1\nend\n", "test.lp:2: invalid control character 0x01", false},
	{"no objective sense", "st\n c: x >= 1\nend\n", "test.lp:1: 'minimize' or 'maximize' keyword missing", false},
	{"no constraints", "min\n obj: x\nend\n", "test.lp:3: constraints section missing", false},
	{"no constraint sense", "min\n obj: x\nst\n c: x 1\nend\n", "test.lp:4: missing constraint sense", false},
	{"text after end", "min\n obj: x\nst\n c: x >= 1\nend\n x\n", "test.lp:6: extra symbol(s) detected beyond 'end'", false},
	{"bounds after general", "min\n obj: x\nst\n c: x >= 1\ngeneral\n x\nbounds\n x <= 3\nend\n", "test.lp:7: symbol 'bounds' in wrong position", false},
	{"+inf as a lower bound", "min\n obj: x\nst\n c: x >= 1\nbounds\n +inf <= x\nend\n", "test.lp:6: invalid use of '+inf' as lower bound", false},
	{"-inf as an upper bound", "min\n obj: x\nst\n c: x >= 1\nbounds\n x <= -inf\nend\n", "test.lp:6: invalid use of '-inf' as upper bound", false},
	{"an unsigned inf", "min\n obj: x\nst\n c: x >= 1\nbounds\n x <= inf\nend\n", "test.lp:6: missing upper bound", false},
	{"a bound with no variable", "min\n obj: x\nst\n c: x >= 1\nbounds\n 3 <= <= 4\nend\n", "test.lp:6: missing variable name", false},
	{"two lower bounds", "min\n obj: x\nst\n c: x >= 1\nbounds\n 1 <= x >= 2\nend\n", "test.lp:6: invalid bound definition", false},
	{"a name alone in bounds", "min\n obj: x\nst\n c: x >= 1\nbounds\n x\nend\n", "test.lp:7: invalid bound definition", false},
	{"subject to cut short", "min\n obj: x\nsubject tx\n c: x >= 1\nend\n", "test.lp:3: keyword 'subject to' incomplete", false},
	// GLPK prints the raw byte.
	{"a non-ASCII byte", "min\n obj: x\xc3\xa9\nst\n c: x >= 1\nend\n", "test.lp:2: character 0xC3 not recognized", true},
	// GLPK reads this as "no lower bound": a test in its reader is inverted.
	{"a name as a signed lower bound", "min\n obj: x\nst\n c: x >= 1\nbounds\n x >= -foo\nend\n", "test.lp:6: missing lower bound", true},
}

func TestReadLPErrors(t *testing.T) {
	for _, tt := range readErrorCases {
		t.Run(tt.name, func(t *testing.T) {
			p, err := readLP("test.lp", []byte(tt.text))
			if p != nil || err == nil || err.Error() != tt.want {
				t.Errorf("got %v, want %q", err, tt.want)
			}
		})
	}
}

func TestReadLPWarnings(t *testing.T) {
	tests := []struct{ name, text, want string }{
		{"no end", "min\n obj: x\nst\n c: x >= 1\n", "test.lp:4: warning: keyword 'end' missing"},
		{"no final newline", "min\n obj: x\nst\n c: x >= 1\nend", "test.lp:5: warning: missing final end of line"},
		// GLPK warns once per side, however many bounds are redefined.
		{"a redefined bound", "min\n obj: x + y\nst\n c: x >= 1\nbounds\n x <= 5\n x <= 6\n y <= 2\n y <= 3\nend\n",
			"test.lp:7: warning: upper bound of variable 'x' redefined"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := mustReadLP(t, tt.text)
			if !reflect.DeepEqual(p.warnings, []string{tt.want}) {
				t.Errorf("warnings %q, want [%q]", p.warnings, tt.want)
			}
		})
	}
}

// Both engines read the same text, so they must refuse the same files at the
// same line, warn at the same lines, and find the same answer for the odd
// corners of the format.
func TestReaderAgreesWithGLPK(t *testing.T) {
	glpsolOrSkip(t)

	t.Run("errors", func(t *testing.T) {
		for _, tt := range readErrorCases {
			if tt.glpkDiffers {
				continue
			}
			t.Run(tt.name, func(t *testing.T) {
				sol, err := solveText("test.lp", []byte(tt.text), Options{Engine: EngineGLPK})
				if sol != nil || !errors.Is(err, ErrInvalidModel) || !strings.HasSuffix(err.Error(), tt.want) {
					t.Errorf("GLPK: solution %+v, error %v; want ErrInvalidModel ending %q", sol, err, tt.want)
				}
			})
		}
	})

	for _, tt := range []struct{ name, text string }{
		{"an empty row", "min\n obj: x\nst\n c: 0 x >= 1\nend\n"},
		{"an all-zero objective", "min\n obj: 0 x\nst\n c: x >= 1\nend\n"},
		{"tabs and carriage returns", "min\n\tobj:\tx\nst\n c:\tx >= 1\r\nend\n"},
		{"no end", "min\n obj: x\nst\n c: x >= 1\n"},
		{"no final newline", "min\n obj: x\nst\n c: x >= 1\nend"},
		{"a redefined bound", "min\n obj: -x - y\nst\n c: x >= 1\nbounds\n x <= 5\n x <= 6\n y <= 2\n y <= 3\nend\n"},
		{"an explicit row name reused by an unnamed row", "min\n obj: x\nst\n r.5: x >= 1\n x >= 2\nend\n"},
		{"an underflowing number", "min\n obj: x\nst\n c: x >= 1e-400\nend\n"},
		{"free by prefix", "min\n obj: x\nst\n c: x >= -5\nbounds\n x f\nend\n"},
		{"binary keeps its bounds", "max\n obj: x + y\nst\n c: x + y <= 10\nbounds\n 0 <= x <= 4\n y >= -1\nbinary\n x\n y\nend\n"},
		{"a keyword spelled by prefix", "ma\n obj: x\nsu to\n c: x <= 3\ne\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			milp, err := solveText("test.lp", []byte(tt.text), Options{})
			if err != nil {
				t.Fatalf("go-milp: %v", err)
			}
			glpk, err := solveText("test.lp", []byte(tt.text), Options{Engine: EngineGLPK})
			if err != nil {
				t.Fatalf("GLPK: %v", err)
			}
			if milp.Status != glpk.Status {
				t.Fatalf("go-milp says %v, GLPK says %v", milp.Status, glpk.Status)
			}
			if milp.Status == StatusOptimal && math.Abs(milp.Objective-glpk.Objective) > 1e-6 {
				t.Errorf("go-milp objective %v, GLPK objective %v", milp.Objective, glpk.Objective)
			}
			for _, warning := range strings.Split(milp.Log, "\n") {
				if warning != "" && !strings.Contains(glpk.Log, warning) {
					t.Errorf("GLPK did not print the warning %q:\n%s", warning, glpk.Log)
				}
			}
		})
	}

	for _, name := range []string{"bounds_crossed", "integer_bound_fraction"} {
		t.Run(name, func(t *testing.T) {
			text, err := os.ReadFile(filepath.Join("testdata", "glpk", name+".lp"))
			if err != nil {
				t.Fatal(err)
			}
			for _, engine := range []Engine{EngineMILP, EngineGLPK} {
				sol, err := solveText(name, text, Options{Engine: engine})
				if sol != nil || !errors.Is(err, ErrInvalidModel) {
					t.Errorf("%s: solution %+v, error %v; want ErrInvalidModel", engine, sol, err)
				}
			}
		})
	}
}
