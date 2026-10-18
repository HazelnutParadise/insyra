package py

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra"
)

// Run returns the value passed to insyra.Return as the type asked for.
func TestRunDecodesIntoTheTypeAskedFor(t *testing.T) {
	useFakePython(t, `result:{"name":"insyra","count":3}`)
	type answer struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	got, err := Run[answer](context.Background(), `insyra.Return({"name": "insyra", "count": 3})`)
	if err != nil {
		t.Fatal(err)
	}
	if got != (answer{Name: "insyra", Count: 3}) {
		t.Errorf("Run returned %+v", got)
	}
}

func TestRunDecodesADataFrame(t *testing.T) {
	useFakePython(t, `result:{"_insyra_type":"datatable","data":[[1,2],[3,4]],"columns":["a","b"],"index":["r1","r2"]}`)
	dt, err := Run[*insyra.DataTable](context.Background(), "insyra.Return(df)")
	if err != nil {
		t.Fatal(err)
	}
	if rows, cols := dt.Size(); rows != 2 || cols != 2 {
		t.Errorf("the table is %d x %d, want 2 x 2", rows, cols)
	}
	if got := dt.ColNames(); !slices.Equal(got, []string{"a", "b"}) {
		t.Errorf("the columns are %q, want a and b", got)
	}
}

func TestRunReturnsTheZeroValueAndTheError(t *testing.T) {
	useFakePython(t, "exit:3")
	got, err := Run[map[string]int](context.Background(), "insyra.Return(1)")
	if err == nil || !strings.Contains(err.Error(), "exit status 3") {
		t.Errorf("a failing run gave %v", err)
	}
	if got != nil {
		t.Errorf("a failed run returned %v, want the zero value", got)
	}
}

func TestRunRefusesATableByValue(t *testing.T) {
	useFakePython(t, `result:1`)
	if _, err := Run[insyra.DataTable](context.Background(), "insyra.Return(1)"); err == nil {
		t.Error("Run[insyra.DataTable] was accepted")
	}
	if _, err := Run[insyra.DataList](context.Background(), "insyra.Return(1)"); err == nil {
		t.Error("Run[insyra.DataList] was accepted")
	}
}

func TestRunStopsWithItsContext(t *testing.T) {
	useFakePython(t, "sleep")
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := Run[int](ctx, "import time\ntime.sleep(60)")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("a run past its deadline gave %v, want context.DeadlineExceeded", err)
	}
	if elapsed := time.Since(start); elapsed > 20*time.Second {
		t.Errorf("the run took %v to stop", elapsed)
	}
}

// A nil context is an error and an ended one returns its error, for every
// form that takes a context, without starting anything.
func TestContextFormsRefuseNilAndEndedContexts(t *testing.T) {
	useFakePython(t, "sleep")
	var nilCtx context.Context
	ended, cancel := context.WithCancel(context.Background())
	cancel()
	for name, call := range map[string]func(context.Context) error{
		"RunCodeContext":      func(c context.Context) error { return RunCodeContext(c, nil, "insyra.Return(1)") },
		"RunCodefContext":     func(c context.Context) error { return RunCodefContext(c, nil, "insyra.Return($v1)", 1) },
		"Run":                 func(c context.Context) error { _, err := Run[int](c, "insyra.Return(1)"); return err },
		"PipInstallContext":   func(c context.Context) error { return PipInstallContext(c, "x") },
		"PipUninstallContext": func(c context.Context) error { return PipUninstallContext(c, "x") },
	} {
		if err := call(nilCtx); err == nil {
			t.Errorf("%s accepted a nil context", name)
		}
		if err := call(ended); !errors.Is(err, context.Canceled) {
			t.Errorf("%s with an ended context gave %v, want context.Canceled", name, err)
		}
	}
}

func TestRunCodeWithTimeoutKeepsItsMeaning(t *testing.T) {
	useFakePython(t, "sleep")
	err := RunCodeWithTimeout(300*time.Millisecond, nil, "import time\ntime.sleep(60)")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("RunCodeWithTimeout past its deadline gave %v", err)
	}
}
