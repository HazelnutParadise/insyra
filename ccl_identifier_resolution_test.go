package insyra

import (
	"strings"
	"testing"

	"github.com/HazelnutParadise/Go-Utils/conv"
)

// The documented rule is that a bare word in CCL is an Excel-style column
// index and a column name is written ['name']. These tests pin that the rule
// holds for every identifier, including the ones that cannot be read as
// letters, and that a reader who writes the name gets told how to write it.

func identifierTestTable() *DataTable {
	price := NewDataList(1, 2, 3)
	price.SetName("price")
	qty := NewDataList(10, 20, 30)
	qty.SetName("qty_1")
	dt := NewDataTable()
	dt.AppendCols(price, qty)
	return dt
}

// first reports the first cell as a number, because a CCL result carries
// whatever numeric type the expression produced.
func first(t *testing.T, dt *DataTable, colName string) float64 {
	t.Helper()
	col := dt.GetColByName(colName)
	if col == nil {
		t.Fatalf("column %q is missing", colName)
	}
	data := col.Data()
	if len(data) == 0 {
		t.Fatalf("column %q is empty", colName)
	}
	return conv.ParseF64(data[0])
}

func cclFailure(t *testing.T, dt *DataTable) string {
	t.Helper()
	err := dt.Err()
	if err == nil {
		t.Fatal("expected a CCL failure, got none")
	}
	dt.ClearErr()
	return err.Error()
}

func TestCCLBareWordThatNamesAColumnSaysHowToWriteIt(t *testing.T) {
	dt := identifierTestTable()
	dt.AddColUsingCCL("out", "price * 2")
	message := cclFailure(t, dt)
	if !strings.Contains(message, "['price']") {
		t.Fatalf("message does not offer the bracketed form: %s", message)
	}
	if !strings.Contains(message, "index") {
		t.Fatalf("message does not say the word was read as an index: %s", message)
	}
	if dt.GetColByName("out") != nil {
		t.Fatal("a failed expression must not add a column")
	}

	dt = identifierTestTable()
	dt.AddColUsingCCL("out", "['price'] * 2")
	if err := dt.Err(); err != nil {
		t.Fatalf("['price'] * 2: %v", err)
	}
	if got := first(t, dt, "out"); got != 2 {
		t.Fatalf("['price'] * 2 first value = %v, want 2", got)
	}
}

func TestCCLBareWordThatCannotBeAnIndexIsNotLookedUpByName(t *testing.T) {
	dt := identifierTestTable()
	dt.AddColUsingCCL("out", "qty_1 * 2")
	message := cclFailure(t, dt)
	if !strings.Contains(message, "['qty_1']") {
		t.Fatalf("message does not offer the bracketed form: %s", message)
	}

	dt = identifierTestTable()
	dt.AddColUsingCCL("out", "['qty_1'] * 2")
	if err := dt.Err(); err != nil {
		t.Fatalf("['qty_1'] * 2: %v", err)
	}
	if got := first(t, dt, "out"); got != 20 {
		t.Fatalf("['qty_1'] * 2 first value = %v, want 20", got)
	}
}

func TestCCLBareWordWithNoMatchingColumnKeepsTheIndexMessage(t *testing.T) {
	dt := identifierTestTable()
	dt.AddColUsingCCL("out", "ZZZ * 2")
	message := cclFailure(t, dt)
	if strings.Contains(message, "['") {
		t.Fatalf("no column is named ZZZ, so the message must not suggest a name form: %s", message)
	}
	if !strings.Contains(message, "ZZZ") {
		t.Fatalf("message does not name the column it looked for: %s", message)
	}
}

func TestCCLAssignmentTargetFollowsTheSameRule(t *testing.T) {
	dt := identifierTestTable()
	dt.ExecuteCCL("price = ['price'] * 10")
	message := cclFailure(t, dt)
	if !strings.Contains(message, "['price']") {
		t.Fatalf("message does not offer the bracketed form: %s", message)
	}
	if got := first(t, dt, "price"); got != 1 {
		t.Fatalf("a failed assignment changed the column, first value = %v", got)
	}

	dt = identifierTestTable()
	dt.ExecuteCCL("['price'] = ['price'] * 10")
	if err := dt.Err(); err != nil {
		t.Fatalf("['price'] = ['price'] * 10: %v", err)
	}
	if got := first(t, dt, "price"); got != 10 {
		t.Fatalf("['price'] assignment first value = %v, want 10", got)
	}

	dt = identifierTestTable()
	dt.ExecuteCCL("A = A * 10")
	if err := dt.Err(); err != nil {
		t.Fatalf("A = A * 10: %v", err)
	}
	if got := first(t, dt, "price"); got != 10 {
		t.Fatalf("index assignment first value = %v, want 10", got)
	}
}

func TestCCLBracketedIndexThatIsNotLettersFailsAtCompileTime(t *testing.T) {
	dt := identifierTestTable()
	dt.AddColUsingCCL("out", "[qty_1] * 2")
	message := cclFailure(t, dt)
	if !strings.Contains(message, "['qty_1']") {
		t.Fatalf("message does not offer the bracketed name form: %s", message)
	}
	if !strings.Contains(message, "cannot compile") {
		t.Fatalf("a reference that cannot resolve is a compile failure, not a row failure: %s", message)
	}
}
