package wgpu

import (
	"regexp"
	"strings"
	"testing"
)

func TestExactSumLibraryHoldsNoFloatingPoint(t *testing.T) {
	forbidden := regexp.MustCompile(`\b(f16|f32|f64|i32|i64)\b`)
	for lineNumber, line := range strings.Split(exactSumWGSL, "\n") {
		code, _, _ := strings.Cut(line, "//")
		if forbidden.MatchString(code) {
			t.Fatalf("line %d contains a forbidden numeric type: %q", lineNumber+1, line)
		}
	}

	required := []string{
		"struct ExactRegister",
		"fn exact_clear(",
		"fn exact_normalize(",
		"fn exact_add_product(",
		"fn exact_merge(",
		"fn exact_round(",
	}
	for _, fragment := range required {
		if !strings.Contains(exactSumWGSL, fragment) {
			t.Fatalf("exactSumWGSL is missing %q", fragment)
		}
	}
	if strings.Contains(exactSumWGSL, "@compute") {
		t.Fatal("exactSumWGSL must not contain a compute entry point")
	}
}
