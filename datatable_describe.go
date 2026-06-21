package insyra

// Describe returns a programmatic per-column statistical summary of the
// DataTable. By default it includes only numeric columns; IncludeAll also
// includes non-numeric and mixed columns.
func (dt *DataTable) Describe(options ...DescribeOptions) *DataTable {
	out := NewDataTable()
	cfg, err := normalizeDescribeOptions(options)
	if err != nil {
		dt.warn("Describe", "%s", err)
		return out
	}

	var columns []*DataList
	dt.AtomicDo(func(dt *DataTable) {
		columns = make([]*DataList, len(dt.columns))
		for i, col := range dt.columns {
			snapshot := NewDataList(col.data...)
			snapshot.name = col.name
			columns[i] = snapshot
		}
	})

	statNames := describeStatNames(cfg.percentiles)
	outCols := make([]*DataList, 0, len(columns))
	for i, col := range columns {
		summary := describeValues(col.data, cfg.percentiles)
		if summary.kind != describeKindNumeric && !cfg.includeAll {
			continue
		}
		outCols = append(outCols, describeSummaryColumn(describeColumnLabel(i, col.name), summary, statNames))
	}
	if len(outCols) > 0 {
		out.AppendCols(outCols...)
		out.SetRowNames(statNames)
	}
	return out
}
