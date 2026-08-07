package ml

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/HazelnutParadise/insyra"
)

// GridSearchResult is every candidate's cross-validation result plus the
// winner, refitted on the full training data.
type GridSearchResult struct {
	// BestIndex and BestName identify the winning candidate. Ties keep the
	// earliest candidate, so grid order is a deliberate input: put the
	// simplest configuration first and a tie resolves toward simplicity.
	BestIndex int
	BestName  string

	// BestModel is the winning estimator refitted on all supplied rows. The
	// per-fold models are fitted on k−1 folds each; evaluating any of them as
	// "the" model quietly hands over a model trained on less data than the
	// caller has.
	BestModel Model

	// Results is parallel to the candidate list.
	Results []*CrossValidationResult

	// Seed is the sampling seed every candidate shared. When the caller
	// supplied one it is echoed here; when they did not, it is the one drawn
	// for this run — rerunning with it reproduces the identical search.
	Seed uint64
}

// GridSearch cross-validates every candidate on identical folds and returns
// the winner by the metric's declared direction, refitted on the full data.
//
// scikit-learn's GridSearchCV expands a parameter grid into candidates by
// reflecting over constructor parameters — the same machinery clone() needed,
// which is this protocol's recorded departure. Configuration lives in closures
// here, so the grid arrives already expanded: a slice of named estimators.
// What this function centralises is the part that is easy to get silently
// wrong by hand — identical folds for every candidate, direction-aware
// ranking, and the final refit.
func GridSearch(x *insyra.DataTable, y *insyra.DataList, candidates []Estimator, k int, metric Metric, options ...insyra.SamplingOptions) (*GridSearchResult, error) {
	if len(candidates) == 0 {
		return nil, fmt.Errorf("ml: grid search needs at least one candidate")
	}
	seen := make(map[string]struct{}, len(candidates))
	for i, candidate := range candidates {
		if candidate.Name == "" {
			return nil, fmt.Errorf("ml: candidate %d has no name; a score nobody can attribute to a configuration is unusable", i)
		}
		if _, duplicate := seen[candidate.Name]; duplicate {
			return nil, fmt.Errorf("ml: candidate name %q is not unique", candidate.Name)
		}
		seen[candidate.Name] = struct{}{}
		if candidate.Fit == nil {
			return nil, fmt.Errorf("ml: candidate %q has no Fit function", candidate.Name)
		}
	}
	if metric == nil || isNilPointer(metric) {
		return nil, fmt.Errorf("ml: metric is nil")
	}
	if metric.Direction() == NoDirection {
		return nil, fmt.Errorf("ml: metric %q declares no direction, so a grid searched with it has no winner", metric.Name())
	}
	opts, err := oneSamplingOption(options)
	if err != nil {
		return nil, err
	}

	// Every candidate must see the same folds, or their means are numbers
	// about different data and comparing them is meaningless. CrossValidate
	// draws folds per call, so the guarantee is made here: an unseeded request
	// gets one seed drawn once and applied to all candidates.
	if !opts.UseSeed {
		opts.UseSeed = true
		opts.Seed = rand.Uint64()
	}

	result := &GridSearchResult{BestIndex: -1, Seed: opts.Seed, Results: make([]*CrossValidationResult, len(candidates))}
	for i, candidate := range candidates {
		cv, err := CrossValidate(x, y, candidate, k, metric, opts)
		if err != nil {
			return nil, fmt.Errorf("ml: candidate %q: %w", candidate.Name, err)
		}
		result.Results[i] = cv
		if math.IsNaN(cv.Mean) {
			continue
		}
		if result.BestIndex < 0 {
			result.BestIndex = i
			continue
		}
		better, err := Better(cv, result.Results[result.BestIndex])
		if err != nil {
			return nil, fmt.Errorf("ml: candidate %q: %w", candidate.Name, err)
		}
		if better {
			result.BestIndex = i
		}
	}
	if result.BestIndex < 0 {
		return nil, fmt.Errorf("ml: no candidate produced a comparable score")
	}
	result.BestName = candidates[result.BestIndex].Name

	best, err := candidates[result.BestIndex].Fit(x, y)
	if err != nil {
		return nil, fmt.Errorf("ml: refitting winner %q on the full data: %w", result.BestName, err)
	}
	if best == nil || isNilPointer(best) {
		return nil, fmt.Errorf("ml: winner %q returned a nil model on refit", result.BestName)
	}
	result.BestModel = best
	return result, nil
}
