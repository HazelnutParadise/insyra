package insyra

import (
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

// keepConfig restores the global Config fields these tests change, so the
// rest of the package's tests run with the configuration they started with.
func keepConfig(t *testing.T) {
	t.Helper()
	level := Config.GetLogLevel()
	colored := Config.GetDoesUseColoredOutput()
	dontPanic := Config.GetDontPanicStatus()
	hook := Config.GetDefaultErrHandlingFunc()
	t.Cleanup(func() {
		Config.SetLogLevel(level)
		Config.SetUseColoredOutput(colored)
		Config.SetDontPanic(dontPanic)
		Config.SetDefaultErrHandlingFunc(hook)
	})
}

func TestConfigSettersRaceFree(t *testing.T) {
	keepConfig(t)
	Config.SetLogLevel(LogLevelFatal)
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				Config.SetLogLevel(LogLevelWarning)
				Config.SetUseColoredOutput(j%2 == 0)
				Config.SetDontPanic(true)
			}
		}()
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				LogInfo("t", "t", "x")
				_ = Config.GetLogLevel()
				_ = colorText("1", "y")
				_ = Config.GetDontPanicStatus()
			}
		}()
	}
	wg.Wait()
}

func TestErrorHookOrdered(t *testing.T) {
	keepConfig(t)
	Config.SetLogLevel(LogLevelFatal)
	var mu sync.Mutex
	got := []string{}
	done := make(chan struct{}, 200)
	Config.SetDefaultErrHandlingFunc(func(_ LogLevel, _ string, _ string, msg string) {
		mu.Lock()
		got = append(got, msg)
		mu.Unlock()
		done <- struct{}{}
	})
	for i := 0; i < 100; i++ {
		LogWarning("t", "t", "m%03d", i)
	}
	for i := 0; i < 100; i++ {
		<-done
	}
	mu.Lock()
	defer mu.Unlock()
	for i, m := range got {
		if m != "m"+strings.Repeat("0", 3-len(itoa(i)))+itoa(i) {
			t.Fatalf("out of order at %d: %v", i, got[:i+1])
		}
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	s := ""
	for i > 0 {
		s = string(rune('0'+i%10)) + s
		i /= 10
	}
	return s
}

func TestDetectEncodingBoundary(t *testing.T) {
	p := filepath.Join(t.TempDir(), "b.txt")
	body := append([]byte(strings.Repeat("a", 8191)), []byte("中文字再多一些內容")...)
	if err := os.WriteFile(p, body, 0o644); err != nil {
		t.Fatal(err)
	}
	enc, err := DetectEncoding(p)
	if err != nil || enc != "utf-8" {
		t.Fatalf("got %q err=%v", enc, err)
	}
}

// uncomparableCell holds a slice, so Go panics when two of them meet ==.
type uncomparableCell struct{ s []int }

// Such a cell used to be unequal to everything, itself included. Since
// identify-uncomparable-cells it is compared by its type and content, like
// every lookup, so a copy holding the same content is equal and one holding
// different content is not.
func TestIsEqualToUncomparableCellsDoNotPanic(t *testing.T) {
	quietLogs(t)
	dl := NewDataList(1.0, uncomparableCell{s: []int{1}})
	other := NewDataList(1.0, uncomparableCell{s: []int{2}})
	noPanic(t, "IsEqualTo", func() {
		if !dl.IsEqualTo(dl.Clone()) {
			t.Fatal("a copy holding the same uncomparable cell should be equal")
		}
		if dl.IsEqualTo(other) {
			t.Fatal("an uncomparable cell with different content should be unequal")
		}
	})
	noPanic(t, "IsTheSameAs", func() {
		// Not asserted against a clone: Clone stamps a fresh creation time,
		// so the answer would depend on the clock, not on the cells.
		if dl.IsTheSameAs(other) {
			t.Fatal("an uncomparable cell with different content should not be the same")
		}
	})
	if !NewDataList(1.0, "x").IsEqualTo(NewDataList(1.0, "x")) {
		t.Fatal("equal lists should be equal")
	}
	if NewDataList(1.0, 2.0, "x").IsEqualTo(NewDataList(1.0, 3.0, "x")) {
		t.Fatal("different lists should not be equal")
	}
}

// Whether IsEqualTo should treat NaN as equal to NaN is an open decision; this
// line keeps the existing result, where a NaN cell never equals another.
func TestIsEqualToKeepsNaNUnequal(t *testing.T) {
	dl := NewDataList(1.0, math.NaN())
	if dl.IsEqualTo(dl.Clone()) {
		t.Fatal("IsEqualTo should keep NaN != NaN")
	}
}

func TestClearNilsAndNaNsSinglePass(t *testing.T) {
	dl := NewDataList(1.0, math.NaN(), nil, 2.0, nil, math.NaN())
	dl.ClearNilsAndNaNs()
	if !reflect.DeepEqual(dl.Data(), []any{1.0, 2.0}) {
		t.Fatalf("got %v", dl.Data())
	}
}

// ClearNumbers removes the built-in numeric types only; a named numeric type
// such as time.Duration has always been kept.
func TestClearNumbersKeepsNamedNumericTypes(t *testing.T) {
	dl := NewDataList(1, int64(2), 3.5, float32(4), uint8(5), "x", time.Second, nil)
	dl.ClearNumbers()
	if !reflect.DeepEqual(dl.Data(), []any{"x", time.Second, nil}) {
		t.Fatalf("got %v", dl.Data())
	}
}

func TestDropAllMatchesNaNOnlyWhenAsked(t *testing.T) {
	dl := NewDataList(1.0, math.NaN(), "a", nil, 2.0)
	dl.DropAll(math.NaN(), "a")
	if !reflect.DeepEqual(dl.Data(), []any{1.0, nil, 2.0}) {
		t.Fatalf("got %v", dl.Data())
	}
	kept := NewDataList(1.0, math.NaN(), 2.0)
	kept.DropAll(2.0)
	got := kept.Data()
	if len(got) != 2 || got[0] != 1.0 {
		t.Fatalf("got %v", got)
	}
	if f, ok := got[1].(float64); !ok || !math.IsNaN(f) {
		t.Fatalf("NaN should be kept when not asked to drop it, got %v", got)
	}
}

func TestUpdateErrNamesItself(t *testing.T) {
	quietLogs(t)
	dl := NewDataList(1)
	dl.Update(5, 2)
	if dl.Err() == nil || dl.Err().FuncName != "Update" {
		t.Fatalf("got %v", dl.Err())
	}
}

func TestFindColsIfContainsNoErr(t *testing.T) {
	quietLogs(t)
	dt := NewDataTable(NewDataList(5, 6), NewDataList(7, 8))
	ClearErrors()
	cols := dt.FindColsIfContains(5)
	if !reflect.DeepEqual(cols, []string{"A"}) || dt.Err() != nil {
		t.Fatalf("cols=%v err=%v", cols, dt.Err())
	}
	all := dt.FindColsIfContainsAll(5, 6)
	if !reflect.DeepEqual(all, []string{"A"}) {
		t.Fatalf("FindColsIfContainsAll = %v, want [A]", all)
	}
	if n := GetErrorCount(); n != 0 {
		t.Fatalf("a column without the value recorded %d error(s): %v", n, GetAllErrors())
	}
	withNaN := NewDataTable(NewDataList(1.0, math.NaN()), NewDataList(2.0, 3.0))
	if got := withNaN.FindColsIfContains(math.NaN()); !reflect.DeepEqual(got, []string{"A"}) {
		t.Fatalf("FindColsIfContains(NaN) = %v, want [A] as FindFirst finds NaN", got)
	}
}

func TestAppendRowsByColNameDeterministicOrder(t *testing.T) {
	for i := 0; i < 100; i++ {
		dt := NewDataTable().AppendRowsByColName(map[string]any{"b": 1, "a": 2, "c": 3})
		if got := dt.ColNames(); !reflect.DeepEqual(got, []string{"a", "b", "c"}) {
			t.Fatalf("iteration %d: column order %v", i, got)
		}
	}
}
