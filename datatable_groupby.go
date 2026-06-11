package insyra

import (
	"fmt"
	"math"
	"strings"

	"github.com/HazelnutParadise/insyra/internal/utils"
)

// AggregateOp identifies an aggregation operation applied per group by Aggregate.
type AggregateOp int

const (
	// OpSum sums numeric values in the group; non-numeric and nil values are skipped.
	OpSum AggregateOp = iota
	// OpMean computes the arithmetic mean of numeric values in the group.
	OpMean
	// OpMedian computes the median of numeric values in the group.
	OpMedian
	// OpMin returns the minimum numeric value in the group.
	OpMin
	// OpMax returns the maximum numeric value in the group.
	OpMax
	// OpCount counts non-nil values in the group.
	OpCount
	// OpCountAll counts all rows in the group, including those with nil values.
	OpCountAll
	// OpStdev computes the sample standard deviation of numeric values in the group.
	OpStdev
	// OpStdevP computes the population standard deviation of numeric values in the group.
	OpStdevP
	// OpVar computes the sample variance of numeric values in the group.
	OpVar
	// OpVarP computes the population variance of numeric values in the group.
	OpVarP
	// OpFirst returns the first non-nil value in the group (in original row order).
	OpFirst
	// OpLast returns the last non-nil value in the group (in original row order).
	OpLast
	// OpNUnique counts the distinct non-nil values in the group.
	OpNUnique
	// OpCustom invokes Custom func on the group's sub-DataList for the source column.
	OpCustom
)

// String returns the canonical short name of an AggregateOp (e.g. "sum", "mean").
func (o AggregateOp) String() string {
	switch o {
	case OpSum:
		return "sum"
	case OpMean:
		return "mean"
	case OpMedian:
		return "median"
	case OpMin:
		return "min"
	case OpMax:
		return "max"
	case OpCount:
		return "count"
	case OpCountAll:
		return "countall"
	case OpStdev:
		return "stdev"
	case OpStdevP:
		return "stdevp"
	case OpVar:
		return "var"
	case OpVarP:
		return "varp"
	case OpFirst:
		return "first"
	case OpLast:
		return "last"
	case OpNUnique:
		return "nunique"
	case OpCustom:
		return "custom"
	}
	return fmt.Sprintf("op(%d)", int(o))
}

// AggregateConfig describes a single aggregation operation produced by Aggregate.
type AggregateConfig struct {
	// SourceCol is the source column to aggregate. It is matched first by name,
	// then by Excel-style index ("A"/"B"/...). Required for every Op except
	// OpCountAll, where an empty SourceCol is allowed.
	SourceCol string
	// As is the output column name. If empty, the column is auto-named as
	// "<source>_<op>" (e.g., "revenue_sum"). For OpCountAll without a source
	// column, the default is "count_all".
	As string
	// Op selects the aggregation. See AggregateOp constants.
	Op AggregateOp
	// Custom is the user-provided function for OpCustom. It receives a
	// DataList containing the values in the source column for this group
	// (in original row order, including nil entries) and returns any value.
	// Ignored when Op != OpCustom.
	Custom func(group *DataList) any
}

// GroupedDataTable is the lightweight intermediate produced by DataTable.GroupBy.
// It is not safe for concurrent use across goroutines and should be consumed by
// calling Aggregate, AggregateAll, or Count once before reuse.
type GroupedDataTable struct {
	parent *DataTable

	// keyColNumbers holds the numeric column index of every key column,
	// in the order supplied to GroupBy.
	keyColNumbers []int
	// keyColLabels holds a friendly label per key column (column name when
	// available, otherwise the Excel-style index used as the lookup token).
	keyColLabels []string

	// columnsSnapshot is a defensive shallow copy of the parent's columns at
	// the moment GroupBy was called. We do not hold the parent lock during
	// Aggregate, mirroring the existing actor-model conventions.
	columnsSnapshot []*DataList

	// rowsByGroup maps a stable string-encoded composite key to the row
	// indices that belong to the group. Row indices reference rows in
	// columnsSnapshot.
	rowsByGroup map[string][]int

	// groupOrder records the order in which group keys were first seen. This
	// drives the row order of the output DataTable.
	groupOrder []string

	// groupKeyValues stores the original (typed) key values per group, so we
	// can emit them as-is into the result DataTable rather than as their
	// string-encoded form.
	groupKeyValues map[string][]any

	// initErr records errors encountered while building the grouping
	// (missing columns, etc.). Aggregate propagates these to the parent
	// DataTable's instance-level error state.
	initErr string
}

// GroupBy splits dt into groups defined by the unique combinations of values in
// the given key columns and returns an intermediate object that supports
// Aggregate / AggregateAll / Count.
//
// Each entry in keyCols may be a column name or an Excel-style column index
// ("A", "B", ..., "AA"). Lookups try the name first, then fall back to the
// alphabetic index. Unknown columns are recorded on the parent DataTable's
// Err() and Aggregate will return an empty DataTable.
//
// Group order in the resulting DataTable follows the order in which each key
// combination is first seen during a single linear scan of the input rows.
func (dt *DataTable) GroupBy(keyCols ...string) *GroupedDataTable {
	g := &GroupedDataTable{
		parent:         dt,
		rowsByGroup:    map[string][]int{},
		groupKeyValues: map[string][]any{},
	}
	if len(keyCols) == 0 {
		dt.warn("GroupBy", "no key columns provided")
		g.initErr = "GroupBy requires at least one key column"
		return g
	}
	dt.AtomicDo(func(t *DataTable) {
		// Snapshot column references so subsequent operations work without
		// holding the actor lock. Columns are pointers; the data slices are
		// not mutated by Aggregate, so a shallow copy is safe.
		g.columnsSnapshot = make([]*DataList, len(t.columns))
		copy(g.columnsSnapshot, t.columns)

		g.keyColNumbers = make([]int, 0, len(keyCols))
		g.keyColLabels = make([]string, 0, len(keyCols))
		for _, raw := range keyCols {
			num, label, ok := resolveColForGroup(t, raw)
			if !ok {
				dt.warn("GroupBy", "key column %q not found", raw)
				g.initErr = fmt.Sprintf("GroupBy: key column %q not found", raw)
				return
			}
			g.keyColNumbers = append(g.keyColNumbers, num)
			g.keyColLabels = append(g.keyColLabels, label)
		}

		nRows := 0
		for _, c := range t.columns {
			if l := len(c.data); l > nRows {
				nRows = l
			}
		}

		// Single linear scan to assign rows to groups.
		for row := 0; row < nRows; row++ {
			keyVals := make([]any, len(g.keyColNumbers))
			for i, colNum := range g.keyColNumbers {
				if colNum < 0 || colNum >= len(t.columns) {
					keyVals[i] = nil
					continue
				}
				col := t.columns[colNum]
				if row < len(col.data) {
					keyVals[i] = col.data[row]
				} else {
					keyVals[i] = nil
				}
			}
			encoded := encodeGroupKey(keyVals)
			if _, exists := g.rowsByGroup[encoded]; !exists {
				g.groupOrder = append(g.groupOrder, encoded)
				g.groupKeyValues[encoded] = keyVals
			}
			g.rowsByGroup[encoded] = append(g.rowsByGroup[encoded], row)
		}
	})
	return g
}

// resolveColForGroup matches a token to a DataTable column. It tries the
// column name first, then the Excel-style index, and returns (colNumber,
// displayLabel, ok). The label prefers the column's name when present.
func resolveColForGroup(t *DataTable, token string) (int, string, bool) {
	if num, ok := t.getColNumberByName_notAtomic(token); ok {
		label := token
		if name := t.columns[num].name; name != "" {
			label = name
		}
		return num, label, true
	}
	upper := strings.ToUpper(token)
	if num, ok := utils.ParseColIndex(upper); ok {
		if num >= 0 && num < len(t.columns) {
			label := upper
			if name := t.columns[num].name; name != "" {
				label = name
			}
			return num, label, true
		}
	}
	return 0, token, false
}

// encodeGroupKey produces a stable, collision-resistant string encoding of a
// composite key. We type-tag every component so that the strings "1" and the
// integer 1 fall into distinct groups, and we use a separator that callers
// cannot easily produce in their data.
func encodeGroupKey(values []any) string {
	var b strings.Builder
	for i, v := range values {
		if i > 0 {
			b.WriteString("\x1e") // ASCII record separator
		}
		if v == nil {
			b.WriteString("n:")
			continue
		}
		switch tv := v.(type) {
		case string:
			b.WriteString("s:")
			b.WriteString(tv)
		case bool:
			if tv {
				b.WriteString("b:1")
			} else {
				b.WriteString("b:0")
			}
		case float32, float64:
			f, _ := utils.ToFloat64Safe(v)
			if math.IsNaN(f) {
				b.WriteString("f:NaN")
			} else {
				fmt.Fprintf(&b, "f:%v", f)
			}
		case int, int8, int16, int32, int64,
			uint, uint8, uint16, uint32, uint64:
			fmt.Fprintf(&b, "i:%v", v)
		default:
			fmt.Fprintf(&b, "o:%T:%v", v, v)
		}
	}
	return b.String()
}

// Aggregate produces a new DataTable with one row per group. The first columns
// are the keys passed to GroupBy (in the original order); the remaining
// columns are produced by configs (in the order passed). Custom funcs receive
// a DataList containing the source-column values for that group, including
// any nil entries, in the original row order.
func (g *GroupedDataTable) Aggregate(configs ...AggregateConfig) *DataTable {
	out := NewDataTable()
	if g == nil {
		return out
	}
	if g.initErr != "" {
		if g.parent != nil {
			g.parent.warn("Aggregate", "%s", g.initErr)
		}
		return out
	}
	if len(configs) == 0 {
		if g.parent != nil {
			g.parent.warn("Aggregate", "no aggregate configs provided")
		}
		return out
	}

	// Validate configs ahead of time and resolve source columns.
	resolved := make([]aggregateResolved, len(configs))
	for i, cfg := range configs {
		resolved[i] = g.resolveConfig(cfg)
		if resolved[i].err != "" && g.parent != nil {
			g.parent.warn("Aggregate", "%s", resolved[i].err)
		}
	}

	// Build the key columns first.
	keyCols := make([]*DataList, len(g.keyColLabels))
	for i, label := range g.keyColLabels {
		keyCols[i] = NewDataList()
		keyCols[i].SetName(label)
	}

	// Create empty result columns for each aggregate config.
	resultCols := make([]*DataList, len(configs))
	for i, r := range resolved {
		col := NewDataList()
		col.SetName(r.outputName)
		resultCols[i] = col
	}

	// Walk groups in first-seen order and compute aggregates.
	for _, encoded := range g.groupOrder {
		keyVals := g.groupKeyValues[encoded]
		for i, v := range keyVals {
			keyCols[i].Append(v)
		}
		rowIdxs := g.rowsByGroup[encoded]
		for i, r := range resolved {
			value := g.computeAggregate(r, rowIdxs)
			resultCols[i].Append(value)
		}
	}

	// Append in deterministic order: keys first, then aggregates.
	out.AppendCols(keyCols...)
	out.AppendCols(resultCols...)
	return out
}

// AggregateAll applies op to every column that is not a group key. The output
// names are auto-derived (e.g., "revenue_sum"). Columns that produce no valid
// numeric values for any group simply emit NaN, matching DataList.Sum etc.
func (g *GroupedDataTable) AggregateAll(op AggregateOp) *DataTable {
	if g == nil {
		return NewDataTable()
	}
	if g.initErr != "" {
		if g.parent != nil {
			g.parent.warn("AggregateAll", "%s", g.initErr)
		}
		return NewDataTable()
	}
	if op == OpCustom {
		if g.parent != nil {
			g.parent.warn("AggregateAll", "OpCustom is not supported by AggregateAll")
		}
		return NewDataTable()
	}
	keySet := map[int]struct{}{}
	for _, n := range g.keyColNumbers {
		keySet[n] = struct{}{}
	}
	configs := make([]AggregateConfig, 0, len(g.columnsSnapshot))
	for i, col := range g.columnsSnapshot {
		if _, isKey := keySet[i]; isKey {
			continue
		}
		ref := col.name
		if ref == "" {
			idx, ok := utils.CalcColIndex(i)
			if !ok {
				continue
			}
			ref = idx
		}
		configs = append(configs, AggregateConfig{SourceCol: ref, Op: op})
	}
	if len(configs) == 0 {
		return NewDataTable()
	}
	return g.Aggregate(configs...)
}

// Count returns a DataTable with a single aggregate column ("count") that
// holds the size of each group (including rows where every value is nil).
func (g *GroupedDataTable) Count() *DataTable {
	return g.Aggregate(AggregateConfig{Op: OpCountAll, As: "count"})
}

// aggregateResolved is the internal, validated form of an AggregateConfig.
type aggregateResolved struct {
	cfg        AggregateConfig
	sourceCol  *DataList
	sourceNum  int
	hasSource  bool
	outputName string
	err        string
}

// resolveConfig validates a single AggregateConfig against the snapshot.
func (g *GroupedDataTable) resolveConfig(cfg AggregateConfig) aggregateResolved {
	r := aggregateResolved{cfg: cfg, sourceNum: -1}
	// Source column is optional only for OpCountAll.
	if cfg.SourceCol == "" {
		if cfg.Op != OpCountAll {
			r.err = fmt.Sprintf("Aggregate: SourceCol required for op %s", cfg.Op)
			r.outputName = nonEmptyOr(cfg.As, "")
			return r
		}
		r.outputName = nonEmptyOr(cfg.As, "count_all")
		return r
	}
	num, label, ok := g.lookupSnapshotCol(cfg.SourceCol)
	if !ok {
		r.err = fmt.Sprintf("Aggregate: source column %q not found", cfg.SourceCol)
		r.outputName = nonEmptyOr(cfg.As, fmt.Sprintf("%s_%s", cfg.SourceCol, cfg.Op))
		return r
	}
	r.sourceNum = num
	r.sourceCol = g.columnsSnapshot[num]
	r.hasSource = true
	r.outputName = nonEmptyOr(cfg.As, fmt.Sprintf("%s_%s", label, cfg.Op))
	if cfg.Op == OpCustom && cfg.Custom == nil {
		r.err = fmt.Sprintf("Aggregate: OpCustom requires Custom func for column %q", cfg.SourceCol)
		return r
	}
	return r
}

// lookupSnapshotCol resolves a token against the snapshot taken at GroupBy
// time, returning (colNumber, displayLabel, ok). Order: name first, then
// Excel-style index.
func (g *GroupedDataTable) lookupSnapshotCol(token string) (int, string, bool) {
	for i, col := range g.columnsSnapshot {
		if col.name == token {
			return i, token, true
		}
	}
	upper := strings.ToUpper(token)
	if num, ok := utils.ParseColIndex(upper); ok {
		if num >= 0 && num < len(g.columnsSnapshot) {
			label := upper
			if name := g.columnsSnapshot[num].name; name != "" {
				label = name
			}
			return num, label, true
		}
	}
	return 0, token, false
}

// computeAggregate runs a single aggregate over the rows belonging to a group.
func (g *GroupedDataTable) computeAggregate(r aggregateResolved, rowIdxs []int) any {
	if r.err != "" {
		return nil
	}
	switch r.cfg.Op {
	case OpCountAll:
		return len(rowIdxs)
	}
	if !r.hasSource {
		return nil
	}
	// Build a sub-DataList for the group from the source column. Preserve
	// row order so OpFirst / OpLast and custom funcs see what the user
	// expects.
	sub := NewDataList()
	for _, idx := range rowIdxs {
		if idx < len(r.sourceCol.data) {
			sub.Append(r.sourceCol.data[idx])
		} else {
			sub.Append(nil)
		}
	}
	switch r.cfg.Op {
	case OpSum:
		return sub.Sum()
	case OpMean:
		return sub.Mean()
	case OpMedian:
		return sub.Median()
	case OpMin:
		return sub.Min()
	case OpMax:
		return sub.Max()
	case OpStdev:
		return sub.Stdev()
	case OpStdevP:
		return sub.StdevP()
	case OpVar:
		return sub.Var()
	case OpVarP:
		return sub.VarP()
	case OpCount:
		count := 0
		for _, v := range sub.data {
			if v != nil {
				count++
			}
		}
		return count
	case OpFirst:
		for _, v := range sub.data {
			if v != nil {
				return v
			}
		}
		return nil
	case OpLast:
		for i := len(sub.data) - 1; i >= 0; i-- {
			if sub.data[i] != nil {
				return sub.data[i]
			}
		}
		return nil
	case OpNUnique:
		return countUniqueNonNil(sub.data)
	case OpCustom:
		return r.cfg.Custom(sub)
	}
	return nil
}

// countUniqueNonNil counts distinct non-nil values. For values that are not
// usable as map keys, falls back to a stringified representation.
func countUniqueNonNil(values []any) int {
	seen := map[string]struct{}{}
	for _, v := range values {
		if v == nil {
			continue
		}
		key := uniqueKey(v)
		seen[key] = struct{}{}
	}
	return len(seen)
}

func uniqueKey(v any) string {
	switch tv := v.(type) {
	case string:
		return "s:" + tv
	case bool:
		if tv {
			return "b:1"
		}
		return "b:0"
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("i:%v", v)
	case float32, float64:
		f, _ := utils.ToFloat64Safe(v)
		if math.IsNaN(f) {
			return "f:NaN"
		}
		return fmt.Sprintf("f:%v", f)
	}
	return fmt.Sprintf("o:%T:%v", v, v)
}

func nonEmptyOr(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

