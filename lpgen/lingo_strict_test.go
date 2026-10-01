package lpgen

import (
	"reflect"
	"strings"
	"testing"
)

// The parser used to drop whatever it did not recognise and hand back the rest
// as if it were the whole model: a last statement without its ';' never
// flushed, and a statement it could not read vanished with no word. ParseLingo
// and ParseLingoFile now name the line instead, and the deprecated names keep
// the old loose reading.

// A statement the model never finishes is not a model. The name that returns an
// error has to name the line it starts on and the text, so a caller can go fix
// it; the deprecated name keeps dropping it.
func TestParseLingoRefusesAnUnterminatedStatement(t *testing.T) {
	const text = "MODEL:\nMIN= X1 + X2\nEND"

	m, err := ParseLingo(text)
	if m != nil {
		t.Errorf("an unterminated statement gave a model: %+v", m)
	}
	if err == nil {
		t.Fatal("an unterminated statement gave a nil error")
	}
	for _, want := range []string{"line 2", "MIN= X1 + X2"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the error does not say %q: %v", want, err)
		}
	}

	quiet(t)
	if got := ParseLingoModel_str(text); got == nil {
		t.Fatal("the deprecated name returned nil for text it reads")
	} else if got.Objective != "" {
		t.Errorf("the deprecated name kept the unterminated statement: %q", got.Objective)
	}
}

// A sentence is not a model statement. Silently dropping it is how a wrong
// model looks like a right one, so the new name refuses it and says where.
func TestParseLingoRefusesAStatementItDoesNotRecognise(t *testing.T) {
	m, err := ParseLingo("MODEL:\nMIN= X;\nhello world;\nEND")
	if m != nil {
		t.Errorf("an unreadable statement gave a model: %+v", m)
	}
	if err == nil {
		t.Fatal("an unreadable statement gave a nil error")
	}
	for _, want := range []string{"line 3", "hello world"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the error does not say %q: %v", want, err)
		}
	}
}

// A declaration whose variable cannot be read is the same kind of loss: the
// binary or free restriction is gone while the name is still there.
func TestParseLingoRefusesADeclarationItCannotRead(t *testing.T) {
	tests := []struct {
		name        string
		declaration string
	}{
		{"parentheses reversed", "@BIN)X("},
		{"no variable", "@FREE()"},
		{"two numbers only", "@BND(0, X)"},
		{"a bound that is not a number", "@BND(a, X, 1)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text := "MODEL:\nMIN= X;\n" + tt.declaration + ";\nEND"
			m, err := ParseLingo(text)
			if m != nil {
				t.Errorf("an unreadable declaration gave a model: %+v", m)
			}
			if err == nil {
				t.Fatalf("%q gave a nil error", tt.declaration)
			}
			if !strings.Contains(err.Error(), "line 3") {
				t.Errorf("the error does not name line 3: %v", err)
			}
		})
	}
}

// A comment is not a statement and does not make a model unreadable.
func TestParseLingoSkipsComments(t *testing.T) {
	m, err := ParseLingo("MODEL:\n! the plant has two lines;\nMIN= X;\nX >= 1;\nEND")
	if err != nil {
		t.Fatalf("ParseLingo: %v", err)
	}
	if m.Objective != "X" {
		t.Errorf("objective: got %q, want \"X\"", m.Objective)
	}
	if want := []string{"X >= 1"}; !reflect.DeepEqual(m.Bounds, want) {
		t.Errorf("bounds: got %v, want %v", m.Bounds, want)
	}
}

// LINGO's Display Model writes the integrality and the freedom of a variable
// as declarations rather than as relations. @GIN(Y) is a general integer, so it
// belongs with @INT; @FREE(X) and @BND(l, X, u) are bounds, and reading them
// changes what the solver may do with X.
func TestParseLingoReadsGinFreeAndBnd(t *testing.T) {
	m, err := ParseLingo("MODEL:\nMIN= X + Y + Z;\n@GIN(Y);\n@FREE(X);\n@BND(-5, Z, 2.5);\nX + Y >= -3;\nEND")
	if err != nil {
		t.Fatalf("ParseLingo: %v", err)
	}
	if want := []string{"Y"}; !reflect.DeepEqual(m.IntegerVars, want) {
		t.Errorf("integer vars: got %v, want %v", m.IntegerVars, want)
	}
	if want := []string{"X free", "-5 <= Z <= 2.5"}; !reflect.DeepEqual(m.Bounds, want) {
		t.Errorf("bounds: got %v, want %v", m.Bounds, want)
	}
	if want := []string{"X + Y >= -3"}; !reflect.DeepEqual(m.Constraints, want) {
		t.Errorf("constraints: got %v, want %v", m.Constraints, want)
	}
}

// The deprecated names kept their loose reading for one release: the same text
// that the new name refuses still comes back as a model, minus what was dropped.
func TestDeprecatedLingoReaderStillDropsWhatItDoesNotRead(t *testing.T) {
	quiet(t)

	m := ParseLingoModel_str("MODEL:\nMIN= X;\n@FREE(X);\nhello world;\nEND")
	if m == nil {
		t.Fatal("the deprecated name returned nil")
	}
	if len(m.Bounds) != 0 {
		t.Errorf("the deprecated name read a declaration it does not read: %v", m.Bounds)
	}
	if m.Objective != "X" {
		t.Errorf("objective: got %q, want \"X\"", m.Objective)
	}
}
