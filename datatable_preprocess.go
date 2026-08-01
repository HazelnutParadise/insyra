package insyra

import "fmt"

// Helpers shared by the preprocessing families (encoding in datatable_encode.go
// and scaling in datatable_scale.go). They live here, rather than in either
// feature file, so neither family owns the other's dependencies.

// resolveEncodingColumn resolves a column reference (name or Excel-style index)
// to its index and a stable label, mirroring the lookup the public Get/Update
// column methods use.
func resolveEncodingColumn(dt *DataTable, ref string) (int, string, bool) {
	if idx, ok := dt.getColNumberByName_notAtomic(ref); ok {
		label := dt.columns[idx].name
		if label == "" {
			label = fallbackEncodingColumnName(idx)
		}
		return idx, label, true
	}
	if idx, ok := ParseColIndex(ref); ok && idx >= 0 && idx < len(dt.columns) {
		label := dt.columns[idx].name
		if label == "" {
			label = fallbackEncodingColumnName(idx)
		}
		return idx, label, true
	}
	return -1, "", false
}

func fallbackEncodingColumnName(idx int) string {
	if name, ok := alphaColIndex(idx); ok {
		return name
	}
	return fmt.Sprintf("col_%d", idx)
}

// copyRowNamesNotAtomic copies row-name metadata from src onto out. Callers must
// already hold src's lock (the *NotAtomic contract).
func copyRowNamesNotAtomic(out, src *DataTable) {
	if src.rowNames != nil {
		out.rowNames = src.rowNames.Clone()
	}
}
