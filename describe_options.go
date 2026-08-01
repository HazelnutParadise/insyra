package insyra

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

// DescribeOptions configures Describe output for DataList, DataTable, and
// GroupedDataTable.
type DescribeOptions struct {
	// Percentiles contains percentile positions in the inclusive range [0, 1].
	// When omitted, Describe uses 0.25, 0.5, and 0.75.
	Percentiles []float64
	// IncludeAll includes non-numeric and mixed columns in DataTable and
	// GroupedDataTable descriptions.
	IncludeAll bool
}

type describeConfig struct {
	percentiles []float64
	includeAll  bool
}

var defaultDescribePercentiles = []float64{0.25, 0.5, 0.75}

func normalizeDescribeOptions(options []DescribeOptions) (describeConfig, error) {
	cfg := describeConfig{
		percentiles: append([]float64(nil), defaultDescribePercentiles...),
	}
	if len(options) > 0 {
		opt := options[0]
		cfg.includeAll = opt.IncludeAll
		if opt.Percentiles != nil {
			cfg.percentiles = append([]float64(nil), opt.Percentiles...)
		}
	}

	for _, p := range cfg.percentiles {
		if math.IsNaN(p) || math.IsInf(p, 0) || p < 0 || p > 1 {
			return describeConfig{}, fmt.Errorf("percentile %v out of range [0, 1]", p)
		}
	}
	sort.Float64s(cfg.percentiles)
	unique := cfg.percentiles[:0]
	for _, p := range cfg.percentiles {
		if len(unique) == 0 || p != unique[len(unique)-1] {
			unique = append(unique, p)
		}
	}
	cfg.percentiles = unique
	return cfg, nil
}

func describeStatNames(percentiles []float64) []string {
	names := []string{"count", "missing", "unique", "top", "freq", "mean", "std", "min"}
	for _, p := range percentiles {
		names = append(names, describePercentileName(p))
	}
	names = append(names, "max")
	return names
}

func describeNumericStatNames(percentiles []float64) []string {
	names := []string{"count", "missing", "mean", "std", "min"}
	for _, p := range percentiles {
		names = append(names, describePercentileName(p))
	}
	names = append(names, "max")
	return names
}

func describeCategoryStatNames() []string {
	return []string{"count", "missing", "unique", "top", "freq"}
}

func describePercentileName(p float64) string {
	percent := p * 100
	if math.Abs(percent-math.Round(percent)) < 1e-9 {
		return fmt.Sprintf("%.0f%%", math.Round(percent))
	}
	s := strconv.FormatFloat(percent, 'f', -1, 64)
	s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	return s + "%"
}
