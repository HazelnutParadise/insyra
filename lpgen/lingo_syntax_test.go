package lpgen

import (
	"os"
	"strings"
	"testing"
)

func TestParseLingoSyntax_SimpleModel(t *testing.T) {
	source := `
MODEL:
! Simple linear programming model
MAX = 3*X1 + 4*X2;
2*X1 + 3*X2 <= 12;
4*X1 + 2*X2 <= 16;
X1 >= 0;
X2 >= 0;
END
`
	
	model, err := ParseLingoSyntax(source)
	if err != nil {
		t.Fatalf("Failed to parse LINGO syntax: %v", err)
	}
	
	if model == nil {
		t.Fatal("Model is nil")
	}
	
	if model.ObjectiveType != "Maximize" {
		t.Errorf("Expected Maximize, got %s", model.ObjectiveType)
	}
	
	if len(model.Constraints) < 2 {
		t.Errorf("Expected at least 2 constraints, got %d", len(model.Constraints))
	}
}

func TestParseLingoSyntax_WithSets(t *testing.T) {
	source := `
SETS:
PRODUCTS / P1 P2 P3 /;
ENDSETS

DATA:
COST = 10 20 30;
ENDDATA

MODEL:
MIN = @SUM(PRODUCTS: COST * X_PRODUCTS);
END
`
	
	model, err := ParseLingoSyntax(source)
	if err != nil {
		t.Fatalf("Failed to parse LINGO syntax with sets: %v", err)
	}
	
	if model == nil {
		t.Fatal("Model is nil")
	}
	
	if model.ObjectiveType != "Minimize" {
		t.Errorf("Expected Minimize, got %s", model.ObjectiveType)
	}
	
	// Check that @SUM was expanded
	if !strings.Contains(model.Objective, "X_P1") || !strings.Contains(model.Objective, "X_P2") || !strings.Contains(model.Objective, "X_P3") {
		t.Errorf("Expected X_P1, X_P2, X_P3 in objective, got: %s", model.Objective)
	}
}

func TestExpandSum(t *testing.T) {
	lm := &LingoModel{
		Sets: map[string][]string{
			"I": {"1", "2", "3"},
		},
		Data:        make(map[string][]float64),
		Variables:   make(map[string]string),
		Constraints: make([]string, 0),
	}
	
	// Test with underscore pattern
	expr := "@SUM(I: X_I)"
	expanded := lm.expandSum(expr)
	
	expected := "(X_1 + X_2 + X_3)"
	if expanded != expected {
		t.Errorf("Expected %s, got %s", expected, expanded)
	}
	
	// Test without underscore (simple replacement)
	expr2 := "@SUM(I: 2*I)"
	expanded2 := lm.expandSum(expr2)
	expected2 := "(2*1 + 2*2 + 2*3)"
	if expanded2 != expected2 {
		t.Errorf("Expected %s, got %s", expected2, expanded2)
	}
}

func TestExpandFor(t *testing.T) {
	lm := &LingoModel{
		Sets: map[string][]string{
			"J": {"A", "B", "C"},
		},
		Data:        make(map[string][]float64),
		Variables:   make(map[string]string),
		Constraints: make([]string, 0),
	}
	
	// Test with underscore pattern
	expr := "@FOR(J: X_J <= 10)"
	constraints := lm.expandFor(expr)
	
	if len(constraints) != 3 {
		t.Errorf("Expected 3 constraints, got %d", len(constraints))
	}
	
	expected := []string{"X_A <= 10", "X_B <= 10", "X_C <= 10"}
	for i, c := range constraints {
		if c != expected[i] {
			t.Errorf("Expected constraint %s, got %s", expected[i], c)
		}
	}
	
	// Test without underscore
	expr2 := "@FOR(J: J >= 0)"
	constraints2 := lm.expandFor(expr2)
	expected2 := []string{"A >= 0", "B >= 0", "C >= 0"}
	for i, c := range constraints2 {
		if c != expected2[i] {
			t.Errorf("Expected constraint %s, got %s", expected2[i], c)
		}
	}
}

func TestRemoveComments(t *testing.T) {
	lm := &LingoModel{}
	
	source := `
MAX = X + Y; ! This is a comment
X <= 10;     ! Another comment
! Full line comment
Y >= 0;
`
	
	cleaned := lm.removeComments(source)
	
	if strings.Contains(cleaned, "!") {
		t.Error("Comments were not removed")
	}
	
	if !strings.Contains(cleaned, "MAX = X + Y;") {
		t.Error("Code was incorrectly removed")
	}
}

func TestParseSetLine(t *testing.T) {
	lm := &LingoModel{
		Sets:        make(map[string][]string),
		Data:        make(map[string][]float64),
		Variables:   make(map[string]string),
		Constraints: make([]string, 0),
	}
	
	err := lm.parseSetLine("PRODUCTS / P1 P2 P3 /;")
	if err != nil {
		t.Fatalf("Failed to parse set line: %v", err)
	}
	
	if len(lm.Sets["PRODUCTS"]) != 3 {
		t.Errorf("Expected 3 products, got %d", len(lm.Sets["PRODUCTS"]))
	}
	
	expected := []string{"P1", "P2", "P3"}
	for i, elem := range lm.Sets["PRODUCTS"] {
		if elem != expected[i] {
			t.Errorf("Expected %s, got %s", expected[i], elem)
		}
	}
}

func TestParseDataLine(t *testing.T) {
	lm := &LingoModel{
		Sets:        make(map[string][]string),
		Data:        make(map[string][]float64),
		Variables:   make(map[string]string),
		Constraints: make([]string, 0),
	}
	
	err := lm.parseDataLine("COST = 10 20 30;")
	if err != nil {
		t.Fatalf("Failed to parse data line: %v", err)
	}
	
	if len(lm.Data["COST"]) != 3 {
		t.Errorf("Expected 3 cost values, got %d", len(lm.Data["COST"]))
	}
	
	expected := []float64{10, 20, 30}
	for i, val := range lm.Data["COST"] {
		if val != expected[i] {
			t.Errorf("Expected %f, got %f", expected[i], val)
		}
	}
}

func TestParseLingoSyntax_ComplexModel(t *testing.T) {
	source := `
! Production Planning Model
SETS:
PRODUCTS / P1 P2 P3 /;
RESOURCES / R1 R2 /;
ENDSETS

DATA:
PROFIT = 10 15 20;
CAPACITY = 100 80;
ENDDATA

MODEL:
! Maximize total profit
MAX = @SUM(PRODUCTS: PROFIT * X_PRODUCTS);

! Non-negativity
@FOR(PRODUCTS: X_PRODUCTS >= 0);
END
`
	
	model, err := ParseLingoSyntax(source)
	if err != nil {
		t.Fatalf("Failed to parse complex LINGO model: %v", err)
	}
	
	if model == nil {
		t.Fatal("Model is nil")
	}
	
	if model.ObjectiveType != "Maximize" {
		t.Errorf("Expected Maximize, got %s", model.ObjectiveType)
	}
	
	// Check that @SUM was expanded
	if !strings.Contains(model.Objective, "X_P1") {
		t.Errorf("Expected X_P1 in objective, got: %s", model.Objective)
	}
	
	// Check that @FOR was expanded to 3 constraints
	if len(model.Constraints) != 3 {
		t.Errorf("Expected 3 constraints, got %d", len(model.Constraints))
	}
}

func TestParseLingo_BinaryAndInteger(t *testing.T) {
	source := `
MODEL:
MAX = X + Y;
@BIN(X);
@GIN(Y);
X + Y <= 10;
END
`
	
	model, err := ParseLingoSyntax(source)
	if err != nil {
		t.Fatalf("Failed to parse LINGO with @BIN/@GIN: %v", err)
	}
	
	if model == nil {
		t.Fatal("Model is nil")
	}
}

func TestParseLingoModel_BackwardCompatibility(t *testing.T) {
	// Test that existing ParseLingoModel_str still works
	modelStr := `MODEL:
MIN= X1 + X2;
[_1] X1 + X2 >= 10;
[_2] X1 >= 0;
[_3] X2 >= 0;
END`
	
	model := ParseLingoModel_str(modelStr)
	if model == nil {
		t.Fatal("ParseLingoModel_str returned nil")
	}
	
	if model.ObjectiveType != "Minimize" {
		t.Errorf("Expected Minimize, got %s", model.ObjectiveType)
	}
	
	if len(model.Constraints) < 1 {
		t.Errorf("Expected at least 1 constraint, got %d", len(model.Constraints))
	}
}

func TestParseLingoFile_SimpleModel(t *testing.T) {
	// Test parsing a simple LINGO file
	model, err := ParseLingoFile("examples/simple_model.lng")
	if err != nil {
		t.Skipf("Skipping test: example file not found: %v", err)
		return
	}
	
	if model == nil {
		t.Fatal("Model is nil")
	}
	
	if model.ObjectiveType != "Maximize" {
		t.Errorf("Expected Maximize, got %s", model.ObjectiveType)
	}
	
	if !strings.Contains(model.Objective, "X1") || !strings.Contains(model.Objective, "X2") {
		t.Errorf("Objective function should contain X1 and X2, got: %s", model.Objective)
	}
}

func TestParseLingoFile_BinaryInteger(t *testing.T) {
	// Test parsing a LINGO file with @BIN and @GIN
	model, err := ParseLingoFile("examples/binary_integer_model.lng")
	if err != nil {
		t.Skipf("Skipping test: example file not found: %v", err)
		return
	}
	
	if model == nil {
		t.Fatal("Model is nil")
	}
	
	// Check if binary variables are declared
	foundBinary := false
	for _, binVar := range model.BinaryVars {
		if binVar == "X" {
			foundBinary = true
			break
		}
	}
	if !foundBinary {
		t.Error("Expected X to be declared as binary variable")
	}
	
	// Check if integer variables are declared
	foundInteger := false
	for _, intVar := range model.IntegerVars {
		if intVar == "Y" {
			foundInteger = true
			break
		}
	}
	if !foundInteger {
		t.Error("Expected Y to be declared as integer variable")
	}
}

func TestGenerateLPFile_FromLINGO(t *testing.T) {
	// Test full pipeline: parse LINGO -> generate LP file
	source := `
MODEL:
MAX = 5*X + 3*Y;
X + Y <= 10;
@BIN(X);
X >= 0;
Y >= 0;
END
`
	
	model, err := ParseLingoSyntax(source)
	if err != nil {
		t.Fatalf("Failed to parse LINGO syntax: %v", err)
	}
	
	// Generate LP file to temp location
	tmpFile := "/tmp/test_lingo_output.lp"
	model.GenerateLPFile(tmpFile)
	
	// Check if file was created
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Error("LP file was not created")
	} else {
		// Clean up
		_ = os.Remove(tmpFile)
	}
}
