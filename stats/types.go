package stats

type AlternativeHypothesis string

const (
	TwoSided AlternativeHypothesis = "two-sided"
	Greater  AlternativeHypothesis = "greater"
	Less     AlternativeHypothesis = "less"
)
