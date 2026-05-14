package insyra

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// fixtureCase mirrors one entry inside testdata/window_fixtures.json.
type fixtureCase struct {
	Input    []any `json:"input"`
	Periods  *int  `json:"periods,omitempty"`
	Window   *int  `json:"window,omitempty"`
	MinObs   *int  `json:"min_obs,omitempty"`
	Expected []any `json:"expected"`
}

func loadFixtures(t *testing.T) map[string][]fixtureCase {
	t.Helper()
	path := filepath.Join("testdata", "window_fixtures.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var out map[string][]fixtureCase
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return out
}

// runFixtureOp dispatches a single case to the corresponding DataList method.
func runFixtureOp(t *testing.T, op string, c fixtureCase) []any {
	t.Helper()
	dl := NewDataList(c.Input...)
	switch op {
	case "shift":
		return dl.Shift(*c.Periods).Data()
	case "diff":
		return dl.Diff(*c.Periods).Data()
	case "pct_change":
		return dl.PctChange(*c.Periods).Data()
	case "cumsum":
		return dl.CumSum().Data()
	case "cumprod":
		return dl.CumProd().Data()
	case "cummax":
		return dl.CumMax().Data()
	case "cummin":
		return dl.CumMin().Data()
	case "rolling_sum":
		return dl.Rolling(rollingOpts(c)).Sum().Data()
	case "rolling_mean":
		return dl.Rolling(rollingOpts(c)).Mean().Data()
	case "rolling_min":
		return dl.Rolling(rollingOpts(c)).Min().Data()
	case "rolling_max":
		return dl.Rolling(rollingOpts(c)).Max().Data()
	case "rolling_median":
		return dl.Rolling(rollingOpts(c)).Median().Data()
	case "rolling_std":
		return dl.Rolling(rollingOpts(c)).Std().Data()
	case "rolling_var":
		return dl.Rolling(rollingOpts(c)).Var().Data()
	case "expanding_mean":
		return dl.Expanding(*c.MinObs).Mean().Data()
	case "expanding_sum":
		return dl.Expanding(*c.MinObs).Sum().Data()
	case "expanding_std":
		return dl.Expanding(*c.MinObs).Std().Data()
	case "expanding_var":
		return dl.Expanding(*c.MinObs).Var().Data()
	}
	t.Fatalf("unknown fixture op: %s", op)
	return nil
}

func rollingOpts(c fixtureCase) RollingOptions {
	opts := RollingOptions{Window: *c.Window}
	if c.MinObs != nil {
		opts.MinObs = *c.MinObs
	}
	return opts
}

func TestWindowCrossLanguage(t *testing.T) {
	fixtures := loadFixtures(t)
	const tol = 1e-9

	// Iterate fixture operations in alphabetic order so failures are stable.
	ops := make([]string, 0, len(fixtures))
	for k := range fixtures {
		ops = append(ops, k)
	}
	for _, op := range ops {
		for idx, c := range fixtures[op] {
			name := fmt.Sprintf("%s/case%d", op, idx)
			t.Run(name, func(t *testing.T) {
				got := runFixtureOp(t, op, c)
				if len(got) != len(c.Expected) {
					t.Fatalf("length mismatch: got %d (%v), want %d (%v)", len(got), got, len(c.Expected), c.Expected)
				}
				for i := range got {
					if !approxEqual(got[i], c.Expected[i], tol) {
						t.Errorf("[%d] got %v (%T), want %v (%T)", i, got[i], got[i], c.Expected[i], c.Expected[i])
					}
				}
			})
		}
	}
}
