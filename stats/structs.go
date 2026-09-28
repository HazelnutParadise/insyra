package stats

type testResultBase struct {
	Statistic   float64 // statistic value (t, z, etc.)
	PValue      float64
	DF          *float64 // degrees of freedom (of the first or only group, nil if not applicable)
	CI          *[2]float64
	EffectSizes []EffectSizeEntry
}

type EffectSizeEntry struct {
	Type  string  // "cohen_d" for the t- and z-tests; the rank-based tests use their own types
	Value float64 // Effect size value
}
