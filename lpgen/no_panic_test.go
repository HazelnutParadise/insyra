package lpgen

import "testing"

// The LINGO parser reads a file the user supplies, so malformed input is
// expected rather than exceptional. A declaration whose parentheses are the
// wrong way round used to take the program down with a slice-bounds panic,
// because the code took the first "(" and the first ")" without checking which
// came first.
func TestParseLingo_MalformedDeclarations(t *testing.T) {
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
			// A declaration it cannot read is an error, never a panic.
			m, err := ParseLingo(tt.model)
			if err == nil {
				t.Fatalf("the parser read %+v", m)
			}
		})
	}
}
