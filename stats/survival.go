package stats

import (
	"errors"
	"fmt"
	"math"
	"sort"

	"github.com/HazelnutParadise/insyra"
)

// KaplanMeierResult is the Kaplan-Meier survival curve. Every slice has one
// entry per distinct time at which at least one event happened, in ascending
// order.
type KaplanMeierResult struct {
	Times    []float64 // event times
	AtRisk   []int     // subjects still under observation just before each time
	Events   []int     // events at each time
	Censored []int     // subjects censored at each time (they leave the risk set after it)
	Survival []float64 // estimated probability of lasting beyond each time
	// StdErr is Greenwood's standard error of Survival. It is NaN at a time
	// when every subject at risk had the event.
	StdErr []float64
	// Median is the first time the curve is at or below 0.5. When it equals 0.5
	// on a stretch, the median is the midpoint of that stretch, as R's survival
	// package reports it. It is NaN when the curve never reaches 0.5.
	Median float64
	N      int // subjects
	Total  int // events in all
}

// KaplanMeier estimates the survival curve from how long each subject was
// observed and whether the event happened. observed is true (or 1) when the
// event happened at that duration and false (or 0) when the subject was
// censored, that is, still going at the end of observation. Durations must be
// finite and not negative. A subject censored at the same time as an event is
// counted as still at risk for that event.
//
// For example, to ask whether a bull market is more likely to end the longer
// it has lasted, pass the length of each bull market and false for the one
// still running.
func KaplanMeier(durations, observed insyra.IDataList) (*KaplanMeierResult, error) {
	t, err := testSeries(durations, "durations")
	if err != nil {
		return nil, err
	}
	flags, err := eventFlags(observed, len(t))
	if err != nil {
		return nil, err
	}
	if len(t) == 0 {
		return nil, errors.New("durations is empty")
	}
	for i, v := range t {
		if v < 0 {
			return nil, fmt.Errorf("durations holds %v at row index %d; a duration cannot be negative", v, i)
		}
	}

	order := make([]int, len(t))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool { return t[order[a]] < t[order[b]] })

	res := &KaplanMeierResult{N: len(t), Median: math.NaN()}
	atRisk := len(t)
	survival, greenwood := 1.0, 0.0
	for start := 0; start < len(order); {
		end := start
		events, censored := 0, 0
		for end < len(order) && t[order[end]] == t[order[start]] {
			if flags[order[end]] {
				events++
			} else {
				censored++
			}
			end++
		}
		if events > 0 {
			survival *= 1 - float64(events)/float64(atRisk)
			if atRisk > events {
				greenwood += float64(events) / (float64(atRisk) * float64(atRisk-events))
			} else {
				greenwood = math.Inf(1)
			}
			se := math.NaN()
			if !math.IsInf(greenwood, 1) {
				se = survival * math.Sqrt(greenwood)
			}
			res.Times = append(res.Times, t[order[start]])
			res.AtRisk = append(res.AtRisk, atRisk)
			res.Events = append(res.Events, events)
			res.Censored = append(res.Censored, censored)
			res.Survival = append(res.Survival, survival)
			res.StdErr = append(res.StdErr, se)
			res.Total += events
		}
		atRisk -= events + censored
		start = end
	}

	for i, s := range res.Survival {
		if s <= 0.5+1e-12 {
			res.Median = res.Times[i]
			if math.Abs(s-0.5) <= 1e-12 && i+1 < len(res.Times) {
				res.Median = (res.Times[i] + res.Times[i+1]) / 2
			}
			break
		}
	}
	return res, nil
}

// eventFlags reads the event indicator, which may be bool or 0/1.
func eventFlags(dl insyra.IDataList, want int) ([]bool, error) {
	if insyra.IsNilList(dl) {
		return nil, errors.New("observed is nil")
	}
	var raw []any
	dl.AtomicDo(func(l *insyra.DataList) { raw = l.Data() })
	if len(raw) != want {
		return nil, fmt.Errorf("durations has %d rows and observed has %d", want, len(raw))
	}
	out := make([]bool, len(raw))
	for i, v := range raw {
		switch x := v.(type) {
		case bool:
			out[i] = x
		default:
			f, ok := insyra.ToFloat64Safe(v)
			if !ok || (f != 0 && f != 1) {
				return nil, fmt.Errorf("observed holds %v at row index %d; it must be true, false, 1 or 0", v, i)
			}
			out[i] = f == 1
		}
	}
	return out, nil
}
