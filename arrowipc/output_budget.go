package arrowipc

import (
	"context"
	"fmt"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
)

const defaultMaxOutputBytes int64 = 256 << 20

// Charge interface slots conservatively on both 32-bit and 64-bit hosts.
// Runtime objects and temporary copies are not a process RSS guarantee.
const outputCellBytes int64 = 16

// Check the full table before allocating its cell slices or cloning payloads.
// Dictionary values are charged once per occurrence, as they are copied then.
func checkOutputBudget(ctx context.Context, record arrow.RecordBatch, selected []selectedColumn, start, end, limit int64) error {
	count := end - start
	if count > limit/outputCellBytes/int64(len(selected)) {
		return fmt.Errorf("arrowipc: output cell slots exceed MaxOutputBytes %d", limit)
	}
	remaining := limit - count*int64(len(selected))*outputCellBytes
	for _, column := range selected {
		values := record.Column(column.index)
		if values == nil || int64(values.Len()) < end {
			return fmt.Errorf("arrowipc: invalid data for selected column %q", column.field.Name)
		}
		for row := start; row < end; row++ {
			if err := checkContext(ctx); err != nil {
				return err
			}
			size, err := cellPayloadBytes(values, int(row))
			if err != nil {
				return fmt.Errorf("arrowipc: column %q row %d: %w", column.field.Name, row, err)
			}
			if size > remaining {
				return fmt.Errorf("arrowipc: expanded output exceeds MaxOutputBytes %d", limit)
			}
			remaining -= size
		}
	}
	return nil
}

func cellPayloadBytes(values arrow.Array, row int) (int64, error) {
	if values.IsNull(row) {
		return 0, nil
	}
	switch a := values.(type) {
	case *array.String:
		return int64(len(a.Value(row))), nil
	case *array.LargeString:
		return int64(len(a.Value(row))), nil
	case *array.Binary:
		return int64(len(a.Value(row))), nil
	case *array.LargeBinary:
		return int64(len(a.Value(row))), nil
	case *array.FixedSizeBinary:
		return int64(len(a.Value(row))), nil
	case *array.Dictionary:
		index := a.GetValueIndex(row)
		if index < 0 || index >= a.Dictionary().Len() {
			return 0, fmt.Errorf("arrowipc: dictionary index %d out of range", index)
		}
		return cellPayloadBytes(a.Dictionary(), index)
	default:
		return 0, nil
	}
}
