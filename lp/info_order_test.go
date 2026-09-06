package lp

import (
	"reflect"
	"testing"
)

func TestAdditionalInfoRowOrder(t *testing.T) {
	want := []string{"Status", "Execution Time", "Warnings", "Full Output", "Iterations", "Nodes"}
	for i := 0; i < 5; i++ {
		dt := createAdditionalInfoDataTable("Error", 0.1, "w", "out", "1", "2")
		if got := dt.RowNames(); !reflect.DeepEqual(got, want) {
			t.Fatalf("row order %v, want %v", got, want)
		}
	}
}
