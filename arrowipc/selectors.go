package arrowipc

import (
	"fmt"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/internal/utils"
	"github.com/apache/arrow-go/v18/arrow"
)

func selectColumns(schema *arrow.Schema, requested []any) ([]selectedColumn, error) {
	fields := schema.Fields()
	if len(requested) == 0 {
		requested = make([]any, len(fields))
		for i := range fields {
			requested[i] = i
		}
	}
	selected := make([]selectedColumn, 0, len(requested))
	seen := make(map[string]struct{}, len(requested))
	for _, selector := range requested {
		index, err := columnIndex(fields, selector)
		if err != nil {
			return nil, err
		}
		field := fields[index]
		if _, exists := seen[field.Name]; exists {
			return nil, fmt.Errorf("arrowipc: duplicate or ambiguous selected column %q", field.Name)
		}
		seen[field.Name] = struct{}{}
		if err := checkExtension(field); err != nil {
			return nil, err
		}
		if !supportedCellType(field.Type) {
			return nil, fmt.Errorf("arrowipc: unsupported selected column %q type %s", field.Name, field.Type)
		}
		selected = append(selected, selectedColumn{index: index, field: field})
	}
	return selected, nil
}

func columnIndex(fields []arrow.Field, selector any) (int, error) {
	switch value := selector.(type) {
	case string:
		index, ok := utils.ParseColIndex(value)
		if !ok || index >= len(fields) {
			return -1, fmt.Errorf("arrowipc: invalid column index %q; use insyra.Name(%q) for a field name", value, value)
		}
		return index, nil
	case int:
		index := value
		if index < 0 {
			index += len(fields)
		}
		if index < 0 || index >= len(fields) {
			return -1, fmt.Errorf("arrowipc: column position %d out of range for %d fields", value, len(fields))
		}
		return index, nil
	case insyra.NameSelector:
		index := -1
		for i, field := range fields {
			if field.Name != value.Value() {
				continue
			}
			if index >= 0 {
				return -1, fmt.Errorf("arrowipc: ambiguous selected column %q", value.Value())
			}
			index = i
		}
		if index < 0 {
			return -1, fmt.Errorf("arrowipc: unknown selected column %q", value.Value())
		}
		return index, nil
	default:
		return -1, fmt.Errorf("arrowipc: invalid column selector %T; use an Excel-style index, insyra.Name, or int position", selector)
	}
}
