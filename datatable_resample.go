package insyra

import (
	"fmt"
	"sort"
	"time"
)

// ResampleFreq identifies the calendar period used by DataTable.Resample.
type ResampleFreq int

const (
	ResampleWeekly ResampleFreq = iota
	ResampleMonthly
	ResampleQuarterly
	ResampleYearly
)

// ResampleAgg describes one column aggregation in DataTable.Resample.
// When As is empty, the output column uses Col as its name.
type ResampleAgg struct {
	Col string
	Op  AggregateOp
	As  string
}

// Resample converts rows keyed by timeCol into calendar-period aggregates.
// Input rows are ordered by time for first/last semantics, and the result is
// labelled by each period's final calendar day. Missing periods are omitted.
func (dt *DataTable) Resample(timeCol string, freq ResampleFreq, aggs ...ResampleAgg) (*DataTable, error) {
	empty := NewDataTable()
	if dt == nil {
		return empty, fmt.Errorf("Resample: DataTable is nil")
	}
	if len(aggs) == 0 {
		return empty, fmt.Errorf("Resample: at least one aggregate is required")
	}
	if !validResampleFreq(freq) {
		return empty, fmt.Errorf("Resample: unknown frequency %d", freq)
	}

	var (
		columns  []*DataList
		times    []time.Time
		timeNum  int
		timeName string
		resErr   error
	)
	dt.AtomicDo(func(t *DataTable) {
		var ok bool
		timeNum, timeName, ok = resolveColForGroup(t, timeCol)
		if !ok {
			resErr = fmt.Errorf("Resample: time column %q not found", timeCol)
			return
		}
		for _, agg := range aggs {
			if agg.Col == "" {
				resErr = fmt.Errorf("Resample: aggregate column is required")
				return
			}
			if _, _, found := resolveColForGroup(t, agg.Col); !found {
				resErr = fmt.Errorf("Resample: aggregate column %q not found", agg.Col)
				return
			}
		}

		nRows := t.getMaxColLength()
		if timeNum < 0 || timeNum >= len(t.columns) {
			resErr = fmt.Errorf("Resample: time column %q not found", timeCol)
			return
		}
		timeValues := t.columns[timeNum].data
		times = make([]time.Time, nRows)
		for row := 0; row < nRows; row++ {
			if row >= len(timeValues) {
				resErr = fmt.Errorf("Resample: time column row %d is not a time.Time", row+1)
				return
			}
			value, ok := timeValues[row].(time.Time)
			if !ok {
				resErr = fmt.Errorf("Resample: time column row %d is not a time.Time: %v", row+1, timeValues[row])
				return
			}
			times[row] = value
		}

		columns = make([]*DataList, len(t.columns))
		for i, column := range t.columns {
			copyColumn := NewDataList()
			copyColumn.data = make([]any, len(column.data))
			copy(copyColumn.data, column.data)
			copyColumn.name = column.name
			columns[i] = copyColumn
		}
	})
	if resErr != nil {
		return empty, resErr
	}

	rowOrder := make([]int, len(times))
	for i := range rowOrder {
		rowOrder[i] = i
	}
	sort.SliceStable(rowOrder, func(i, j int) bool {
		return times[rowOrder[i]].Before(times[rowOrder[j]])
	})

	workColumns := make([]*DataList, len(columns))
	for colNum, column := range columns {
		values := make([]any, len(rowOrder))
		for outputRow, sourceRow := range rowOrder {
			if colNum == timeNum {
				values[outputRow] = resamplePeriodEnd(times[sourceRow], freq)
			} else if sourceRow < len(column.data) {
				values[outputRow] = column.data[sourceRow]
			}
		}
		workColumns[colNum] = NewDataList(values...)
		workColumns[colNum].name = column.name
	}
	work := NewDataTable(workColumns...)

	configs := make([]AggregateConfig, len(aggs))
	for i, agg := range aggs {
		outputName := agg.As
		if outputName == "" {
			outputName = agg.Col
		}
		configs[i] = AggregateConfig{SourceCol: agg.Col, Op: agg.Op, As: outputName}
	}
	result := work.GroupBy(timeName).Aggregate(configs...)
	result.SortBy(DataTableSortConfig{ColumnName: timeName})
	return result, nil
}

func validResampleFreq(freq ResampleFreq) bool {
	return freq >= ResampleWeekly && freq <= ResampleYearly
}

func resamplePeriodEnd(value time.Time, freq ResampleFreq) time.Time {
	dayStart := time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
	switch freq {
	case ResampleWeekly:
		daysUntilSunday := (7 - int(dayStart.Weekday())) % 7
		return dayStart.AddDate(0, 0, daysUntilSunday)
	case ResampleMonthly:
		return time.Date(value.Year(), value.Month()+1, 0, 0, 0, 0, 0, value.Location())
	case ResampleQuarterly:
		endMonth := time.Month(((int(value.Month())-1)/3 + 1) * 3)
		return time.Date(value.Year(), endMonth+1, 0, 0, 0, 0, 0, value.Location())
	case ResampleYearly:
		return time.Date(value.Year(), time.December, 31, 0, 0, 0, 0, value.Location())
	default:
		return time.Time{}
	}
}
