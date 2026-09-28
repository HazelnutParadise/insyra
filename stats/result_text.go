package stats

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/internal/utils"
)

// Layer 0 — the text form of a result.
//
// Every result type's String goes through formatResult, so every result
// prints by the same rules: its title, then one line per exported field
// named as in Go. A list or a table is cut the way the table views cut one:
// whole up to textWholeUpTo entries, otherwise the first textHead and the
// last textTail.
const (
	textWholeUpTo = 60
	textHead      = 20
	textTail      = 5
)

// formatResult renders r, a pointer to a result struct, as the text every
// result's String returns. The first line is title. Then comes one line per
// exported field in declaration order, "  Name: value", with the fields of
// an embedded struct listed in its place. A nil pointer or interface field
// is left out: it does not apply to this result. A table field is written
// as an indented grid under its name. A nil r gives "<nil>". The text has
// no colour codes and does not depend on the terminal.
func formatResult(title string, r any) string {
	if r == nil {
		return "<nil>"
	}

	rv := reflect.ValueOf(r)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return "<nil>"
		}
		rv = rv.Elem()
	}

	if rv.Kind() != reflect.Struct {
		return title + "\n  " + fmt.Sprint(rv.Interface())
	}

	var lines []string
	lines = append(lines, title)
	writeFields(&lines, rv)
	return strings.Join(lines, "\n")
}

// writeFields loops over fields of v: skips unexported unless anonymous;
// an anonymous struct or non-nil pointer to struct is flattened by recursion;
// otherwise writes the field via writeField.
func writeFields(lines *[]string, v reflect.Value) {
	rt := v.Type()
	for i := 0; i < v.NumField(); i++ {
		sf := rt.Field(i)
		fv := v.Field(i)

		if !sf.IsExported() && !sf.Anonymous {
			continue
		}

		if sf.Anonymous {
			if fv.Kind() == reflect.Struct {
				writeFields(lines, fv)
				continue
			}
			if fv.Kind() == reflect.Pointer && fv.Type().Elem().Kind() == reflect.Struct {
				if fv.IsNil() {
					continue
				}
				writeFields(lines, fv.Elem())
				continue
			}
		}

		writeField(lines, sf.Name, fv)
	}
}

// writeField appends "  Name: value" to lines. Omits nil interface/pointer.
// Handles IDataTable and IDataList specially, otherwise uses valueText.
func writeField(lines *[]string, name string, v reflect.Value) {
	if v.Kind() == reflect.Interface || v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return
		}
		if v.Kind() == reflect.Interface {
			v = v.Elem()
			if v.Kind() == reflect.Pointer && v.IsNil() {
				return
			}
		}
	}

	if !v.IsValid() {
		return
	}

	if v.CanInterface() {
		switch x := v.Interface().(type) {
		case insyra.IDataTable:
			writeTable(lines, name, x)
			return
		case insyra.IDataList:
			*lines = append(*lines, "  "+name+": "+listText(reflect.ValueOf(x.Data())))
			return
		}
	}

	*lines = append(*lines, "  "+name+": "+valueText(v))
}

// valueText returns the text for a value v. Unwraps interface/pointer (nil -> "<nil>").
// float64/float32 via utils.FloatText; bool/ints/uints via fmt.Sprint; String -> v.String();
// Slice/Array -> listText; Struct -> "{Name: value, ...}" with exported fields (nil ptr/iface omitted);
// anything else fmt.Sprint when CanInterface, else "?".
func valueText(v reflect.Value) string {
	for v.Kind() == reflect.Interface {
		if v.IsNil() {
			return "<nil>"
		}
		v = v.Elem()
	}

	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return "<nil>"
		}
		v = v.Elem()
	}

	if !v.IsValid() {
		return "<nil>"
	}

	switch v.Kind() {
	case reflect.Float64:
		return utils.FloatText(v.Float(), 64)
	case reflect.Float32:
		return utils.FloatText(float64(v.Float()), 32)
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return fmt.Sprint(v.Interface())
	case reflect.String:
		return v.String()
	case reflect.Slice, reflect.Array:
		return listText(v)
	case reflect.Struct:
		var parts []string
		rt := v.Type()
		for i := 0; i < v.NumField(); i++ {
			sf := rt.Field(i)
			if sf.PkgPath != "" {
				continue
			}
			fv := v.Field(i)
			if fv.Kind() == reflect.Pointer && fv.IsNil() {
				continue
			}
			if fv.Kind() == reflect.Interface && fv.IsNil() {
				continue
			}
			parts = append(parts, sf.Name+": "+valueText(fv))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	default:
		if v.CanInterface() {
			return fmt.Sprint(v.Interface())
		}
		return "?"
	}
}

// elemText returns the text for a slice/array element. Unwraps interface/pointer (nil -> "<nil>").
// A String kind is quoted; anything else uses valueText.
func elemText(v reflect.Value) string {
	for v.Kind() == reflect.Interface {
		if v.IsNil() {
			return "<nil>"
		}
		v = v.Elem()
	}

	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return "<nil>"
		}
		v = v.Elem()
	}

	if !v.IsValid() {
		return "<nil>"
	}

	if v.Kind() == reflect.String {
		return strconv.Quote(v.String())
	}
	return valueText(v)
}

// listText renders a slice or array v. Nil/empty -> "[]".
// Shows all when Len <= textWholeUpTo, else first textHead and last textTail with "..." between.
// Appends " (n values)" when cut.
func listText(v reflect.Value) string {
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return "[]"
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return fmt.Sprint(v.Interface())
	}

	n := v.Len()
	if n == 0 {
		return "[]"
	}

	var elems []string
	if n <= textWholeUpTo {
		for i := 0; i < n; i++ {
			elems = append(elems, elemText(v.Index(i)))
		}
		return "[" + strings.Join(elems, " ") + "]"
	}

	for i := 0; i < textHead; i++ {
		elems = append(elems, elemText(v.Index(i)))
	}
	for i := n - textTail; i < n; i++ {
		elems = append(elems, elemText(v.Index(i)))
	}
	return "[" + strings.Join(elems[:textHead], " ") + " ... " + strings.Join(elems[textHead:], " ") + "] (" + strconv.Itoa(n) + " values)"
}

// writeTable appends an indented grid for table t under name. Checks Flush error.
func writeTable(lines *[]string, name string, t insyra.IDataTable) {
	var colNames, rowNames []string
	var cells [][]any

	t.AtomicDo(func(dt *insyra.DataTable) {
		colNames = dt.ColNames()
		rowNames = dt.RowNames()
		cells = dt.To2DSlice()
	})

	if len(colNames) == 0 || len(cells) == 0 {
		*lines = append(*lines, "  "+name+": (empty table)")
		return
	}

	*lines = append(*lines, "  "+name+":")

	hasRowNames := false
	for _, rn := range rowNames {
		if rn != "" {
			hasRowNames = true
			break
		}
	}

	var gridLines [][]string

	header := make([]string, 0, len(colNames)+1)
	if hasRowNames {
		header = append(header, "")
	}
	header = append(header, colNames...)
	gridLines = append(gridLines, header)

	numRows := len(cells)
	for i := 0; i < numRows; i++ {
		row := make([]string, 0, len(colNames)+1)
		if hasRowNames {
			row = append(row, rowNames[i])
		}
		for j := 0; j < len(colNames); j++ {
			if j < len(cells[i]) {
				row = append(row, utils.ValueText(cells[i][j]))
			} else {
				row = append(row, "<nil>")
			}
		}
		gridLines = append(gridLines, row)
	}

	if numRows > textWholeUpTo {
		headRows := gridLines[1 : 1+textHead]
		tailRows := gridLines[numRows-textTail+1:]

		var truncated [][]string
		truncated = append(truncated, gridLines[0])
		truncated = append(truncated, headRows...)
		var ellipsisRow []string
		if hasRowNames {
			ellipsisRow = append(ellipsisRow, "...")
			for range colNames {
				ellipsisRow = append(ellipsisRow, "")
			}
		} else {
			ellipsisRow = append(ellipsisRow, "...")
			for range colNames[1:] {
				ellipsisRow = append(ellipsisRow, "")
			}
		}
		truncated = append(truncated, ellipsisRow)
		truncated = append(truncated, tailRows...)
		gridLines = truncated
	}

	var buf strings.Builder
	w := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
	for _, row := range gridLines {
		line := strings.Join(row, "\t")
		fmt.Fprintln(w, line)
	}
	if err := w.Flush(); err != nil {
		*lines = append(*lines, "  "+name+": (table could not be written)")
		return
	}

	gridText := strings.TrimRight(buf.String(), " \t\n\r")
	for _, line := range strings.Split(gridText, "\n") {
		*lines = append(*lines, "    "+strings.TrimRight(line, " \t"))
	}

	if numRows > textWholeUpTo {
		*lines = append(*lines, "    ("+strconv.Itoa(numRows)+" rows)")
	}
}
