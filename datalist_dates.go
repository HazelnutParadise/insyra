package insyra

import "time"

// dateParseLayouts are the time layouts DataList.ParseDates,
// DataTable.ParseDatesCols and ReadSQLOptions.ParseDates try in order when the
// caller supplies no layouts of its own.
var dateParseLayouts = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02 15:04:05.999999999",
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05",
	"2006-01-02",
}

// ParseDates converts string cells to time.Time in place and returns the
// DataList so calls can be chained.
//
// Each string cell is tried against layouts in order and, on the first layout
// that matches, becomes the parsed instant expressed in UTC. Cells that are
// already time.Time are kept exactly as they are, including their location.
// Every other cell — a number, a bool, a nil, or a string no layout matched —
// becomes nil, so a column is either usable as a time column or visibly empty
// rather than half converted.
//
// With no layouts given, the defaults are the same ISO-style list
// ReadSQLOptions.ParseDates uses. Passing layouts replaces that list entirely.
func (dl *DataList) ParseDates(layouts ...string) *DataList {
	use := effectiveDateLayouts(layouts)
	dl.AtomicDo(func(dl *DataList) {
		parseDatesInPlace(dl.data, use)
		dl.updateTimestamp()
	})
	return dl
}

// ParseDatesCols applies ParseDates to the named columns in place and returns
// the DataTable so calls can be chained. Columns may be named or given as
// Excel-style indices ("A", "B", …); a column that does not exist records a
// warning and is skipped, leaving the rest of the table converted.
//
// This is the conversion `load sql … parsedates` performs, exposed for tables
// that came from anywhere else — a CSV date column, for instance, which
// inference leaves as strings and Resample therefore refuses.
func (dt *DataTable) ParseDatesCols(cols []string, layouts ...string) *DataTable {
	use := effectiveDateLayouts(layouts)
	dt.AtomicDo(func(t *DataTable) {
		for _, col := range cols {
			num, _, found := resolveColForGroup(t, col)
			if !found {
				t.warn("ParseDatesCols", "column %q not found", col)
				continue
			}
			parseDatesInPlace(t.columns[num].data, use)
		}
		t.updateTimestamp()
	})
	return dt
}

// effectiveDateLayouts returns the caller's layouts, or the package defaults
// when none were given.
func effectiveDateLayouts(layouts []string) []string {
	if len(layouts) == 0 {
		return dateParseLayouts
	}
	return layouts
}

// parseDatesInPlace rewrites data according to the ParseDates rules.
func parseDatesInPlace(data []any, layouts []string) {
	for i, v := range data {
		switch value := v.(type) {
		case time.Time:
			continue
		case string:
			if t, ok := parseTimeWithLayouts(value, layouts); ok {
				data[i] = t
				continue
			}
		case []byte:
			if t, ok := parseTimeWithLayouts(string(value), layouts); ok {
				data[i] = t
				continue
			}
		}
		data[i] = nil
	}
}

// parseTimeWithLayouts tries each layout in order, returning the first match
// as a UTC instant.
func parseTimeWithLayouts(s string, layouts []string) (time.Time, bool) {
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), true
		}
	}
	return time.Time{}, false
}
