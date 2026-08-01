package insyra

// Describe returns one summary row per group. Key columns are emitted first,
// followed by flattened summary columns named "<source>_<stat>".
func (g *GroupedDataTable) Describe(options ...DescribeOptions) *DataTable {
	out := NewDataTable()
	if g == nil {
		return out
	}
	cfg, err := normalizeDescribeOptions(options)
	if err != nil {
		if g.parent != nil {
			g.parent.warn("Describe", "%s", err)
		}
		return out
	}
	if g.initErr != "" {
		if g.parent != nil {
			g.parent.warn("Describe", "%s", g.initErr)
		}
		return out
	}

	keySet := map[int]struct{}{}
	for _, n := range g.keyColNumbers {
		keySet[n] = struct{}{}
	}

	type describeSource struct {
		index   int
		label   string
		stats   []string
		summary describeColumnSummary
	}
	sources := make([]describeSource, 0, len(g.columnsSnapshot))
	for i, col := range g.columnsSnapshot {
		if _, isKey := keySet[i]; isKey {
			continue
		}
		summary := describeValues(col.data, cfg.percentiles)
		if summary.kind != describeKindNumeric && !cfg.includeAll {
			continue
		}
		stats := describeNumericStatNames(cfg.percentiles)
		if summary.kind != describeKindNumeric {
			stats = describeCategoryStatNames()
		}
		sources = append(sources, describeSource{
			index:   i,
			label:   describeColumnLabel(i, col.name),
			stats:   stats,
			summary: summary,
		})
	}

	resultCols := make([]*DataList, 0, len(g.keyColLabels)+len(sources)*len(describeStatNames(cfg.percentiles)))
	for _, label := range g.keyColLabels {
		resultCols = append(resultCols, NewDataList().SetName(label))
	}
	for _, src := range sources {
		for _, stat := range src.stats {
			resultCols = append(resultCols, NewDataList().SetName(src.label+"_"+stat))
		}
	}

	for _, encoded := range g.groupOrder {
		keyVals := g.groupKeyValues[encoded]
		for i, v := range keyVals {
			resultCols[i].Append(v)
		}
		offset := len(g.keyColLabels)
		rowIdxs := g.rowsByGroup[encoded]
		for _, src := range sources {
			values := make([]any, 0, len(rowIdxs))
			col := g.columnsSnapshot[src.index]
			for _, row := range rowIdxs {
				if row < len(col.data) {
					values = append(values, col.data[row])
				} else {
					values = append(values, nil)
				}
			}
			summary := describeValues(values, cfg.percentiles)
			for _, stat := range src.stats {
				resultCols[offset].Append(summary.values[stat])
				offset++
			}
		}
	}

	if len(resultCols) > 0 {
		out.AppendCols(resultCols...)
	}
	return out
}
