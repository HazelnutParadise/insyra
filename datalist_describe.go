package insyra

// Describe returns a programmatic statistical summary of the DataList.
func (dl *DataList) Describe(options ...DescribeOptions) *DataTable {
	out := NewDataTable()
	cfg, err := normalizeDescribeOptions(options)
	if err != nil {
		dl.warn("Describe", "%s", err)
		return out
	}

	var name string
	var data []any
	dl.AtomicDo(func(dl *DataList) {
		name = dl.name
		data = append([]any(nil), dl.data...)
	})
	if name == "" {
		name = "value"
	}

	summary := describeValues(data, cfg.percentiles)
	statNames := describeStatNames(cfg.percentiles)
	out.AppendCols(describeSummaryColumn(name, summary, statNames))
	out.SetRowNames(statNames)
	return out
}
