package insyra

import (
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestConfigSettersRaceFree(t *testing.T) {
	SetDefaultConfig()
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
	Config.SetLogLevel(LogLevelFatal)
}

func TestErrorHookOrdered(t *testing.T) {
	SetDefaultConfig()
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
	defer Config.SetDefaultErrHandlingFunc(nil)
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

func TestIsEqualToNaN(t *testing.T) {
	dl := NewDataList(1.0, math.NaN(), "x")
	if !dl.IsEqualTo(dl.Clone()) {
		t.Fatal("list with NaN should equal its clone")
	}
	if dl.IsEqualTo(NewDataList(1.0, 2.0, "x")) {
		t.Fatal("different lists should not be equal")
	}
}

func TestClearNilsAndNaNsSinglePass(t *testing.T) {
	dl := NewDataList(1.0, math.NaN(), nil, 2.0, nil, math.NaN())
	dl.ClearNilsAndNaNs()
	if !reflect.DeepEqual(dl.Data(), []any{1.0, 2.0}) {
		t.Fatalf("got %v", dl.Data())
	}
}

func TestUpdateErrNamesItself(t *testing.T) {
	SetDefaultConfig()
	Config.SetLogLevel(LogLevelFatal)
	dl := NewDataList(1)
	dl.Update(5, 2)
	if dl.Err() == nil || dl.Err().FuncName != "Update" {
		t.Fatalf("got %v", dl.Err())
	}
}

func TestFindColsIfContainsNoErr(t *testing.T) {
	SetDefaultConfig()
	Config.SetLogLevel(LogLevelFatal)
	dt := NewDataTable(NewDataList(5, 6), NewDataList(7, 8))
	cols := dt.FindColsIfContains(5)
	if !reflect.DeepEqual(cols, []string{"A"}) || dt.Err() != nil {
		t.Fatalf("cols=%v err=%v", cols, dt.Err())
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
