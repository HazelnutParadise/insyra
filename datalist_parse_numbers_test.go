package insyra

import (
	"math"
	"reflect"
	"testing"
)

// assertParseNumbersCells compares cell by cell so that a wrong Go type is a
// failure even when the value prints the same, and so that a NaN written by
// ParseNumbers matches the NaN the CSV reader writes.
func assertParseNumbersCells(t *testing.T, got, want []any, context string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: expected %d cells %v, got %d cells %v", context, len(want), want, len(got), got)
	}
	for i := range want {
		g, w := got[i], want[i]
		if reflect.TypeOf(g) != reflect.TypeOf(w) {
			t.Errorf("%s: cell %d: expected type %v (%v), got %v (%v)",
				context, i+1, reflect.TypeOf(w), w, reflect.TypeOf(g), g)
			continue
		}
		gf, gIsFloat := g.(float64)
		wf, wIsFloat := w.(float64)
		if gIsFloat && wIsFloat {
			if math.IsNaN(wf) {
				if !math.IsNaN(gf) {
					t.Errorf("%s: cell %d: expected NaN, got %v", context, i+1, gf)
				}
				continue
			}
		}
		if !reflect.DeepEqual(g, w) {
			t.Errorf("%s: cell %d: expected %v (%v), got %v (%v)",
				context, i+1, w, reflect.TypeOf(w), g, reflect.TypeOf(g))
		}
	}
}

// A 19-digit id survives as an int64, the way the CSV reader types the same
// column, instead of being rounded to the nearest float64 above 2^53.
func TestParseNumbersKeepsLargeIntegers(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)

	dl := NewDataList("9007199254740993", "1").ParseNumbers()

	assertParseNumbersCells(t, dl.Data(), []any{int64(9007199254740993), int64(1)}, "large integers")
	if dl.Err() != nil {
		t.Errorf("Expected no error, got %v", dl.Err())
	}
}

// One decimal anywhere in the list makes every readable cell a float64,
// matching a CSV column that holds at least one non-integer.
func TestParseNumbersOneDecimalMakesFloats(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)

	dl := NewDataList("1", "2.5", "3").ParseNumbers()

	assertParseNumbersCells(t, dl.Data(), []any{1.0, 2.5, 3.0}, "one decimal")
}

// Values that are already numbers are read too, and surrounding spaces do not
// stop a string from being read. A uint above math.MaxInt64 has no int64 to
// become, so it reads as a decimal.
func TestParseNumbersCountsExistingNumbers(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)

	dl := NewDataList("1", 2, "3", int8(8)).ParseNumbers()
	assertParseNumbersCells(t, dl.Data(), []any{int64(1), int64(2), int64(3), int64(8)}, "existing numbers")
	if dl.Err() != nil {
		t.Errorf("Expected no error, got %v", dl.Err())
	}

	mixed := NewDataList(" 7 ", uint64(math.MaxUint64)).ParseNumbers()
	assertParseNumbersCells(t, mixed.Data(), []any{float64(7), float64(math.MaxUint64)}, "spaced and large uint")
	if mixed.Err() != nil {
		t.Errorf("Expected no error, got %v", mixed.Err())
	}
}

// An empty cell is a gap, not a number, so it reads as NaN and the rest of the
// list reads as float64, the way the CSV reader types a column with a blank.
func TestParseNumbersEmptyStringIsNaN(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)

	dl := NewDataList("1", "", "3").ParseNumbers()

	assertParseNumbersCells(t, dl.Data(), []any{1.0, math.NaN(), 3.0}, "empty string")
	if dl.Err() != nil {
		t.Errorf("Expected no error, got %v", dl.Err())
	}
}

// A value ParseNumbers cannot read stays exactly as it was, and the list
// reports every one of them in a single error.
func TestParseNumbersLeavesUnreadableAndReportsOnce(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)

	dl := NewDataList("1", "hello", "3", true, nil).ParseNumbers()
	assertParseNumbersCells(t, dl.Data(), []any{int64(1), "hello", int64(3), true, nil}, "unreadable")
	if dl.Err() == nil {
		t.Fatal("Expected an error, got nil")
	}
	wantMsg := `2 value(s) could not be read as numbers and were left unchanged; the first is "hello" at row 2`
	if dl.Err().Message != wantMsg {
		t.Errorf("Expected error message %q, got %q", wantMsg, dl.Err().Message)
	}

	nonString := NewDataList(true, "x").ParseNumbers()
	if nonString.Err() == nil {
		t.Fatal("Expected an error, got nil")
	}
	wantNonString := "2 value(s) could not be read as numbers and were left unchanged; the first is a bool at row 1"
	if nonString.Err().Message != wantNonString {
		t.Errorf("Expected error message %q, got %q", wantNonString, nonString.Err().Message)
	}
}

// A nil cell is not a value that failed to read; it is left alone silently.
func TestParseNumbersNilIsNotAnError(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)

	dl := NewDataList(nil, "5").ParseNumbers()

	assertParseNumbersCells(t, dl.Data(), []any{nil, int64(5)}, "nil cell")
	if dl.Err() != nil {
		t.Errorf("Expected no error, got %v", dl.Err())
	}
}

// ParseNumbers and the CSV reader apply one rule, so a column read both ways
// holds the same values in the same Go types.
func TestParseNumbersMatchesCSVInference(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)

	csvText := "id,price,note,blank\n9007199254740993,1.5,a,\n2,2,b,\n3,,c,\n"

	raw, err := ReadCSVString(csvText, CSVReadOptions{RawStrings: true})
	if err != nil {
		t.Fatalf("Reading raw CSV failed: %v", err)
	}
	inferred, err := ReadCSVString(csvText)
	if err != nil {
		t.Fatalf("Reading CSV failed: %v", err)
	}

	colNames := raw.ColNames()
	for _, i := range []int{0, 1, 3} { // column 2 is the text column
		colName := colNames[i]
		parsed := raw.GetColByNumber(i).ParseNumbers()
		if parsed.Err() != nil {
			t.Errorf("Column %q: expected no error, got %v", colName, parsed.Err())
		}
		assertParseNumbersCells(t, parsed.Data(), inferred.GetColByNumber(i).Data(), "column "+colName)
	}
}

// Capitalize follows the language-neutral root rules, so words that a
// language-specific rule would treat differently (Dutch "ij", Turkish dotted i)
// are cased the root way. The test passes before and after that change, which is
// what makes it a check of the behaviour rather than of the implementation.
func TestCapitalizeUsesRootCasingRules(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)

	got := NewDataList("ijssel", "istanbul", "hello world", 3).Capitalize().Data()
	want := []any{"Ijssel", "Istanbul", "Hello World", 3}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Expected %v, got %v", want, got)
	}
	if err := NewDataList("ijssel", "istanbul", "hello world", 3).Capitalize().Err(); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}
