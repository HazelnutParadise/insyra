package lpgen

import "testing"

// The LINGO parser reads a file the user supplies, so malformed input is
// expected rather than exceptional. A declaration whose parentheses are the
// wrong way round used to take the program down with a slice-bounds panic,
// because the code took the first "(" and the first ")" without checking which
// came first.
func TestParseLingoModel_MalformedDeclarations(t *testing.T) {
	quiet(t)

	tests := []struct {
		name  string
		model string
	}{
		{name: "reversed parentheses", model: "MODEL:\nMIN= X;\n@BIN)X(;\nEND"},
		{name: "reversed parentheses on INT", model: "MODEL:\nMIN= X;\n@INT)X(;\nEND"},
		{name: "no parentheses", model: "MODEL:\nMIN= X;\n@BIN X;\nEND"},
		{name: "opening only", model: "MODEL:\nMIN= X;\n@BIN(X;\nEND"},
		{name: "closing only", model: "MODEL:\nMIN= X;\n@BIN X);\nEND"},
		{name: "empty parentheses", model: "MODEL:\nMIN= X;\n@BIN();\nEND"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := ParseLingoModel_str(tt.model)
			if m == nil {
				t.Fatal("the parser gave nil")
			}
			// The rest of the model is still read.
			if m.Objective != "X" {
				t.Errorf("objective: got %q, want \"X\"", m.Objective)
			}
		})
	}
}

// A declaration it cannot read is skipped, not guessed at.
func TestParseLingoModel_MalformedDeclarationIsSkipped(t *testing.T) {
	quiet(t)

	m := ParseLingoModel_str("MODEL:\nMIN= X;\n@BIN)X(;\n@BIN(Y);\nEND")
	if m == nil {
		t.Fatal("the parser gave nil")
	}
	if len(m.BinaryVars) != 1 || m.BinaryVars[0] != "Y" {
		t.Errorf("binary vars: got %v, want [Y]", m.BinaryVars)
	}
}
