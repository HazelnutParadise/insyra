package insyra

import (
	"testing"
	"time"
)

// setClock makes the package clock return whatever *at holds, until the test
// ends. Root-package tests do not run in parallel, so swapping it is safe.
func setClock(t *testing.T, at *time.Time) {
	t.Helper()
	old := clock
	clock = func() time.Time { return *at }
	t.Cleanup(func() { clock = old })
}

func TestDataListTimesKeepSubSecondChanges(t *testing.T) {
	base := time.Unix(1700000000, 0)
	at := base
	setClock(t, &at)

	dl := NewDataList(1, 2)
	at = base.Add(100 * time.Nanosecond)
	dl.Append(3)
	if got := dl.LastModifiedTime(); !got.Equal(at) {
		t.Fatalf("after first Append LastModifiedTime = %v, want %v", got, at)
	}
	at = base.Add(200 * time.Nanosecond)
	dl.Append(4)
	if got := dl.LastModifiedTime(); !got.Equal(at) {
		t.Fatalf("after second Append LastModifiedTime = %v, want %v", got, at)
	}
	if got := dl.CreationTime(); !got.Equal(base) {
		t.Fatalf("CreationTime = %v, want %v", got, base)
	}
}

func TestDataTableTimesKeepSubSecondChanges(t *testing.T) {
	base := time.Unix(1700000000, 0)
	at := base
	setClock(t, &at)

	dt := NewDataTable(NewDataList(1, 2).SetName("c"))
	at = base.Add(5 * time.Nanosecond)
	dt.AppendRowsByColIndex(map[string]any{"A": 3})
	if got := dt.LastModifiedTime(); !got.Equal(at) {
		t.Fatalf("LastModifiedTime = %v, want %v", got, at)
	}
	if got := dt.CreationTime(); !got.Equal(base) {
		t.Fatalf("CreationTime = %v, want %v", got, base)
	}
}

func TestLastModifiedTimeNeverMovesBack(t *testing.T) {
	base := time.Unix(1700000000, 0)
	at := base
	setClock(t, &at)

	dl := NewDataList(1)
	dt := NewDataTable(NewDataList(1).SetName("c"))
	at = base.Add(time.Second)
	dl.Append(2)
	dt.AppendRowsByColIndex(map[string]any{"A": 2})
	at = base
	dl.Append(3)
	dt.AppendRowsByColIndex(map[string]any{"A": 3})
	want := base.Add(time.Second)
	if got := dl.LastModifiedTime(); !got.Equal(want) {
		t.Fatalf("DataList LastModifiedTime = %v, want %v", got, want)
	}
	if got := dt.LastModifiedTime(); !got.Equal(want) {
		t.Fatalf("DataTable LastModifiedTime = %v, want %v", got, want)
	}
}

func TestDeprecatedTimestampGettersReturnUnixSeconds(t *testing.T) {
	at := time.Unix(1700000000, 500)
	setClock(t, &at)

	dl := NewDataList(1)
	dt := NewDataTable(NewDataList(1).SetName("c"))
	if got := dl.GetCreationTimestamp(); got != 1700000000 {
		t.Errorf("DataList.GetCreationTimestamp = %d", got)
	}
	if got := dl.GetLastModifiedTimestamp(); got != 1700000000 {
		t.Errorf("DataList.GetLastModifiedTimestamp = %d", got)
	}
	if got := dt.GetCreationTimestamp(); got != 1700000000 {
		t.Errorf("DataTable.GetCreationTimestamp = %d", got)
	}
	if got := dt.GetLastModifiedTimestamp(); got != 1700000000 {
		t.Errorf("DataTable.GetLastModifiedTimestamp = %d", got)
	}
	if got := dl.CreationTime(); !got.Equal(at) {
		t.Errorf("CreationTime = %v, want %v", got, at)
	}
}

// IsTheSameAs on this line still compares the two times, in whole Unix
// seconds, as it always has; 0.4 compares data and name only. Storing the times
// at full resolution must not make it stricter than that.
func TestIsTheSameAsStillComparesWholeSeconds(t *testing.T) {
	base := time.Unix(1700000000, 0)
	at := base
	setClock(t, &at)

	dl := NewDataList(1, "a", 2.5).SetName("x")
	at = base.Add(500 * time.Millisecond)
	clone := dl.Clone()
	if !dl.IsTheSameAs(clone) {
		t.Fatal("a list and a clone made within the same second should be the same")
	}

	at = base.Add(2 * time.Second)
	later := dl.Clone()
	if dl.IsTheSameAs(later) {
		t.Fatal("a clone made in a later second should not be the same")
	}

	renamed := clone.Clone().SetName("y")
	if !dl.IsEqualTo(renamed) || dl.IsTheSameAs(renamed) {
		t.Fatal("a different name should be equal but not the same")
	}
	if dl.IsTheSameAs(nil) {
		t.Fatal("nil should not be the same")
	}
}
