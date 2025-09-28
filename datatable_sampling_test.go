package insyra

import (
	"testing"
)

func TestDataTable_SimpleRandomSample(t *testing.T) {
	// Create a test DataTable with 5 rows and 3 columns
	dt := NewDataTable()
	dl1 := NewDataList(1, 2, 3, 4, 5)
	dl2 := NewDataList(10, 20, 30, 40, 50)
	dl3 := NewDataList(100, 200, 300, 400, 500)
	dt.AppendCols(dl1, dl2, dl3)

	// Test normal sampling: sampleSize < numRows
	sample := dt.SimpleRandomSample(3)
	rows, cols := sample.Size()
	if rows != 3 || cols != 3 {
		t.Errorf("SimpleRandomSample(3) returned wrong size: got (%d, %d), want (3, 3)", rows, cols)
	}

	// Test boundary: sampleSize == numRows
	sample2 := dt.SimpleRandomSample(5)
	rows2, cols2 := sample2.Size()
	if rows2 != 5 || cols2 != 3 {
		t.Errorf("SimpleRandomSample(5) returned wrong size: got (%d, %d), want (5, 3)", rows2, cols2)
	}

	// Test boundary: sampleSize > numRows
	sample3 := dt.SimpleRandomSample(10)
	rows3, cols3 := sample3.Size()
	if rows3 != 5 || cols3 != 3 {
		t.Errorf("SimpleRandomSample(10) returned wrong size: got (%d, %d), want (5, 3)", rows3, cols3)
	}

	// Test edge case: sampleSize = 0
	sample4 := dt.SimpleRandomSample(0)
	rows4, cols4 := sample4.Size()
	if rows4 != 0 || cols4 != 0 {
		t.Errorf("SimpleRandomSample(0) returned wrong size: got (%d, %d), want (0, 0)", rows4, cols4)
	}

	// Test edge case: sampleSize < 0 (should handle gracefully, return empty)
	sample5 := dt.SimpleRandomSample(-1)
	rows5, cols5 := sample5.Size()
	if rows5 != 0 || cols5 != 0 {
		t.Errorf("SimpleRandomSample(-1) returned wrong size: got (%d, %d), want (0, 0)", rows5, cols5)
	}
}
