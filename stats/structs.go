package stats

// TestResult holds what every hypothesis test reports. Each test's result
// type embeds it, so its fields read directly on the result (r.PValue), and
// &r.TestResult is the part a function over any test can take.
type TestResult struct {
	Statistic   float64           // the test statistic (t, z, F, χ², U, W+, H, Q, …)
	PValue      float64           // p-value under the test's alternative hypothesis
	DF          *float64          // degrees of freedom (of the first or only group); nil if the test has none
	CI          *[2]float64       // confidence interval; nil if the test reports none
	EffectSizes []EffectSizeEntry // effect sizes; see each result type for which ones
}

// HypothesisTestResult is satisfied by the result of every hypothesis test,
// through the TestResult it embeds, so one function or one slice can take a
// t-test, a Mann-Whitney U test and a chi-square test alike.
type HypothesisTestResult interface {
	// Base returns the part every hypothesis test reports.
	Base() *TestResult
}

// Base returns r itself, not a copy. Every result type that embeds
// TestResult gets this method, which makes it a HypothesisTestResult.
func (r *TestResult) Base() *TestResult { return r }

type EffectSizeEntry struct {
	Type  string  // "cohen_d" for the t- and z-tests; the rank-based tests use their own types
	Value float64 // Effect size value
}
