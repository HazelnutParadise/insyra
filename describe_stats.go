package insyra

import (
	"math"
	"sort"
)

type describeKind int

const (
	describeKindNumeric describeKind = iota
	describeKindCategory
	describeKindEmpty
)

type describeColumnSummary struct {
	kind    describeKind
	values  map[string]any
	hasData bool
}

func describeValues(values []any, percentiles []float64) describeColumnSummary {
	numeric, missing, allNumeric := collectDescribeNumeric(values)
	if allNumeric && len(numeric) > 0 {
		return describeNumericValues(numeric, missing, percentiles)
	}
	if len(values) == 0 {
		return describeColumnSummary{
			kind: describeKindEmpty,
			values: map[string]any{
				"count":   0,
				"missing": 0,
			},
		}
	}
	return describeCategoryValues(values, missing)
}

func collectDescribeNumeric(values []any) ([]float64, int, bool) {
	numeric := make([]float64, 0, len(values))
	missing := 0
	allNumeric := true
	for _, v := range values {
		if describeIsMissing(v) {
			missing++
			continue
		}
		f, ok := ToFloat64Safe(v)
		if !ok || math.IsNaN(f) {
			allNumeric = false
			continue
		}
		numeric = append(numeric, f)
	}
	return numeric, missing, allNumeric
}

func describeIsMissing(v any) bool {
	if v == nil {
		return true
	}
	switch tv := v.(type) {
	case float64:
		return math.IsNaN(tv)
	case float32:
		return math.IsNaN(float64(tv))
	default:
		return false
	}
}

func describeNumericValues(values []float64, missing int, percentiles []float64) describeColumnSummary {
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)

	sum := 0.0
	for _, v := range sorted {
		sum += v
	}
	mean := sum / float64(len(sorted))

	var std any
	if len(sorted) >= 2 {
		ss := 0.0
		for _, v := range sorted {
			diff := v - mean
			ss += diff * diff
		}
		std = math.Sqrt(ss / float64(len(sorted)-1))
	}

	stats := map[string]any{
		"count":   len(sorted),
		"missing": missing,
		"mean":    mean,
		"std":     std,
		"min":     sorted[0],
		"max":     sorted[len(sorted)-1],
	}
	for _, p := range percentiles {
		stats[describePercentileName(p)] = describePercentile(sorted, p)
	}
	return describeColumnSummary{kind: describeKindNumeric, values: stats, hasData: true}
}

func describePercentile(sorted []float64, p float64) float64 {
	n := len(sorted)
	if n == 0 {
		return math.NaN()
	}
	if n == 1 {
		return sorted[0]
	}
	h := p * float64(n-1)
	lower := int(math.Floor(h))
	upper := int(math.Ceil(h))
	if lower == upper {
		return sorted[lower]
	}
	fraction := h - float64(lower)
	return sorted[lower] + fraction*(sorted[upper]-sorted[lower])
}

func describeCategoryValues(values []any, missing int) describeColumnSummary {
	seen := map[string]struct{}{}
	counts := map[string]int{}
	firstValue := map[string]any{}
	orderedKeys := make([]string, 0)
	count := 0
	for _, v := range values {
		if describeIsMissing(v) {
			continue
		}
		key := uniqueKey(v)
		if _, ok := seen[key]; !ok {
			seen[key] = struct{}{}
			firstValue[key] = v
			orderedKeys = append(orderedKeys, key)
		}
		counts[key]++
		count++
	}

	var top any
	var freq any
	best := 0
	for _, key := range orderedKeys {
		if counts[key] > best {
			best = counts[key]
			top = firstValue[key]
			freq = best
		}
	}

	return describeColumnSummary{
		kind: describeKindCategory,
		values: map[string]any{
			"count":   count,
			"missing": missing,
			"unique":  len(seen),
			"top":     top,
			"freq":    freq,
		},
		hasData: count > 0,
	}
}

func describeSummaryColumn(name string, summary describeColumnSummary, statNames []string) *DataList {
	data := make([]any, len(statNames))
	for i, stat := range statNames {
		data[i] = summary.values[stat]
	}
	return NewDataList(data...).SetName(name)
}

func describeColumnLabel(index int, name string) string {
	if name != "" {
		return name
	}
	if label, ok := CalcColIndex(index); ok {
		return label
	}
	return ""
}
