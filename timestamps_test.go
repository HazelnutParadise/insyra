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

func TestIsTheSameAsIgnoresTimes(t *testing.T) {
	base := time.Unix(1700000000, 0)
	at := base
	setClock(t, &at)

	dl := NewDataList(1, "a", 2.5).SetName("x")
	at = base.Add(2 * time.Second)
	clone := dl.Clone()
	if !dl.IsTheSameAs(clone) {
		t.Fatal("a list and a clone made two seconds later should be the same")
	}

	built := NewDataList(1, "a").SetName("x")
	at = base.Add(3 * time.Second)
	built.Append(2.5)
	if !dl.IsTheSameAs(built) {
		t.Fatal("equal data and name should be the same whatever the history")
	}

	renamed := dl.Clone().SetName("y")
	if !dl.IsEqualTo(renamed) || dl.IsTheSameAs(renamed) {
		t.Fatal("a different name should be equal but not the same")
	}
	if dl.IsTheSameAs(NewDataList(1, "a", 3.5).SetName("x")) {
		t.Fatal("different data should not be the same")
	}
	if dl.IsTheSameAs(nil) {
		t.Fatal("nil should not be the same")
	}
}
