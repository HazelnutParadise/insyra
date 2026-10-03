package dsl

import "testing"

func TestTokenize(t *testing.T) {
	input := `filter t1 "A > 10" as t2`
	tokens := Tokenize(input)
	if len(tokens) != 5 {
		t.Fatalf("expected 5 tokens, got %d (%v)", len(tokens), tokens)
	}
	if tokens[0] != "filter" || tokens[1] != "t1" || tokens[2] != "A > 10" || tokens[3] != "as" || tokens[4] != "t2" {
		t.Fatalf("unexpected tokens: %v", tokens)
	}
}
