package stats

import (
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

func TestNonNumericCompositeCellIsNamedByTypeNotContents(t *testing.T) {
	cyclic := make([]any, 1)
	cyclic[0] = cyclic
	dl := insyra.NewDataList(1.0)
	dl.Append(cyclic)
	_, err := numericValues(dl.Data(), "x")
	if err == nil || !strings.Contains(err.Error(), "[]interface {}") {
		t.Fatalf("want an error naming the cell's type, got %v", err)
	}
}

func TestNonNumericScalarCellStillShowsItsValue(t *testing.T) {
	_, err := numericValues([]any{1.0, "abc"}, "x")
	if err == nil || !strings.Contains(err.Error(), "abc") {
		t.Fatalf("got %v", err)
	}
}
