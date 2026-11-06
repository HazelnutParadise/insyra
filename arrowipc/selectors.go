package arrowipc

import (
	"fmt"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/internal/utils"
	"github.com/apache/arrow-go/v18/arrow"
)

func selectColumns(schema *arrow.Schema, requested []any) ([]selectedColumn, error) {
	width := schema.NumFields()
	if len(requested) == 0 {
		requested = make([]any, width)
		for i := 0; i < width; i++ {
			requested[i] = i
		}
	}
	selected := make([]selectedColumn, 0, len(requested))
	taken := make(map[string]struct{}, len(requested))
	chosen := make(map[int]struct{}, len(requested))
	for _, selector := range requested {
		index, err := columnIndex(schema, selector)
		if err != nil {
			return nil, err
		}
		field := schema.Field(index)
		if _, exists := chosen[index]; exists {
			return nil, fmt.Errorf("arrowipc: column %q (position %d) is selected more than once", field.Name, index)
		}
		chosen[index] = struct{}{}
		// Repeated names are renamed name_1, name_2, ..., as DataTable.AppendCols does.
		name := field.Name
		if name != "" {
			for counter := 1; ; counter++ {
				if _, exists := taken[name]; !exists {
					break
				}
				name = fmt.Sprintf("%s_%d", field.Name, counter)
			}
			taken[name] = struct{}{}
		}
		if err := checkExtension(field); err != nil {
			return nil, err
		}
		if !supportedCellType(field.Type) {
			return nil, fmt.Errorf("arrowipc: unsupported selected column %q type %s", field.Name, field.Type)
		}
		selected = append(selected, selectedColumn{index: index, field: field, name: name})
	}
	return selected, nil
}

func columnIndex(schema *arrow.Schema, selector any) (int, error) {
	width := schema.NumFields()
	switch value := selector.(type) {
	case string:
		index, ok := utils.ParseColIndex(value)
		if !ok || index < 0 || index >= width {
			return -1, fmt.Errorf("arrowipc: invalid column index %q; use insyra.Name(%q) for a field name", value, value)
		}
		return index, nil
	case int:
		index := value
		if index < 0 {
			index += width
		}
		if index < 0 || index >= width {
			return -1, fmt.Errorf("arrowipc: column position %d out of range for %d fields", value, width)
		}
		return index, nil
	case insyra.NameSelector:
		indices := schema.FieldIndices(value.Value())
		if len(indices) == 0 {
			return -1, fmt.Errorf("arrowipc: unknown selected column %q", value.Value())
		}
		if len(indices) > 1 {
			return -1, fmt.Errorf("arrowipc: ambiguous selected column %q", value.Value())
		}
		return indices[0], nil
	default:
		return -1, fmt.Errorf("arrowipc: invalid column selector %T; use an Excel-style index, insyra.Name, or int position", selector)
	}
}
