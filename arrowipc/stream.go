package arrowipc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/HazelnutParadise/insyra"
	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/ipc"
)

type selectedColumn struct {
	index int
	field arrow.Field
}

func startStream(produce func(chan<- *insyra.DataTable) error) (<-chan *insyra.DataTable, <-chan error) {
	tables := make(chan *insyra.DataTable)
	errs := make(chan error, 1)

	go func() {
		var terminalErr error
		defer close(errs)
		defer func() {
			if recovered := recover(); recovered != nil {
				terminalErr = fmt.Errorf("arrowipc: stream failed: %v", recovered)
			}
			if terminalErr != nil {
				errs <- terminalErr
			}
		}()
		defer close(tables)

		terminalErr = produce(tables)
	}()

	return tables, errs
}

func streamPath(ctx context.Context, path string, opt ReadOptions, batchSize int, tables chan<- *insyra.DataTable) (err error) {
	if err := checkContext(ctx); err != nil {
		return err
	}
	if batchSize <= 0 {
		return fmt.Errorf("arrowipc: batchSize must be positive, got %d", batchSize)
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, f.Close())
	}()
	if err := checkContext(ctx); err != nil {
		return err
	}
	stat, err := f.Stat()
	if err != nil {
		return err
	}
	if err := checkContext(ctx); err != nil {
		return err
	}
	return streamSource(ctx, f, stat.Size(), opt, batchSize, tables)
}

func streamSource(ctx context.Context, r io.ReaderAt, size int64, opt ReadOptions, batchSize int, tables chan<- *insyra.DataTable) (err error) {
	if err := checkContext(ctx); err != nil {
		return err
	}
	if batchSize <= 0 {
		return fmt.Errorf("arrowipc: batchSize must be positive, got %d", batchSize)
	}
	if err := checkSource(ctx, r, size); err != nil {
		return err
	}
	maxBytes := opt.MaxBatchBytes
	if maxBytes < 0 {
		return fmt.Errorf("arrowipc: MaxBatchBytes must not be negative")
	}
	if maxBytes == 0 {
		maxBytes = defaultMaxBatchBytes
	}
	info, err := inspectSource(ctx, r, size)
	if err != nil {
		return err
	}
	selected, err := selectColumns(info.Schema, opt.Columns)
	if err != nil {
		return err
	}
	if err := checkContext(ctx); err != nil {
		return err
	}

	source := limitedSource(ctx, r, size, maxBytes)
	reader, err := ipc.NewFileReader(io.NewSectionReader(source, 0, size))
	if err != nil {
		return fmt.Errorf("arrowipc: open file reader: %w", err)
	}
	defer func() {
		err = errors.Join(err, reader.Close())
	}()
	if err := checkContext(ctx); err != nil {
		return err
	}

	if !reader.Schema().Equal(info.Schema) || reader.NumRecords() != info.NumBatches {
		return fmt.Errorf("arrowipc: source schema or batch count changed during opening")
	}
	for i := 0; i < reader.NumRecords(); i++ {
		if err := checkContext(ctx); err != nil {
			return err
		}
		record, err := reader.RecordBatchAt(i)
		if err != nil {
			return fmt.Errorf("arrowipc: read record %d: %w", i, err)
		}
		if err := checkContext(ctx); err != nil {
			record.Release()
			return err
		}
		if err := streamRecord(ctx, record, selected, batchSize, tables); err != nil {
			return err
		}
	}
	return checkContext(ctx)
}

func selectColumns(schema *arrow.Schema, requested []string) ([]selectedColumn, error) {
	fields := schema.Fields()
	var selected []selectedColumn
	if len(requested) == 0 {
		selected = make([]selectedColumn, 0, len(fields))
		seen := make(map[string]struct{}, len(fields))
		for i, field := range fields {
			if _, exists := seen[field.Name]; exists {
				return nil, fmt.Errorf("arrowipc: ambiguous selected column %q", field.Name)
			}
			seen[field.Name] = struct{}{}
			if err := checkExtension(field); err != nil {
				return nil, err
			}
			if !supportedCellType(field.Type) {
				return nil, fmt.Errorf("arrowipc: unsupported selected column %q type %s", field.Name, field.Type)
			}
			selected = append(selected, selectedColumn{index: i, field: field})
		}
		return selected, nil
	}

	selected = make([]selectedColumn, 0, len(requested))
	requestedNames := make(map[string]struct{}, len(requested))
	for _, name := range requested {
		if _, exists := requestedNames[name]; exists {
			return nil, fmt.Errorf("arrowipc: duplicate selected column %q", name)
		}
		requestedNames[name] = struct{}{}

		index := -1
		for i, field := range fields {
			if field.Name != name {
				continue
			}
			if index >= 0 {
				return nil, fmt.Errorf("arrowipc: ambiguous selected column %q", name)
			}
			index = i
		}
		if index < 0 {
			return nil, fmt.Errorf("arrowipc: unknown selected column %q", name)
		}
		field := fields[index]
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

func streamRecord(ctx context.Context, record arrow.RecordBatch, selected []selectedColumn, batchSize int, tables chan<- *insyra.DataTable) error {
	defer record.Release()
	if err := checkContext(ctx); err != nil {
		return err
	}
	if record.NumRows() < 0 {
		return fmt.Errorf("arrowipc: record has invalid row count %d", record.NumRows())
	}
	for _, column := range selected {
		if int64(column.index) >= record.NumCols() {
			return fmt.Errorf("arrowipc: record is missing selected column %q", column.field.Name)
		}
	}

	rows := record.NumRows()
	if rows > 0 && len(selected) == 0 {
		return fmt.Errorf("arrowipc: cannot represent nonempty zero-column record as a DataTable")
	}
	for start := int64(0); start < rows; {
		if err := checkContext(ctx); err != nil {
			return err
		}
		count := int64(batchSize)
		if remaining := rows - start; count > remaining {
			count = remaining
		}
		end := start + count
		columns := make([]*insyra.DataList, 0, len(selected))
		for _, column := range selected {
			if err := checkContext(ctx); err != nil {
				return err
			}
			values := make([]any, int(count))
			array := record.Column(column.index)
			if array == nil || int64(array.Len()) < end {
				return fmt.Errorf("arrowipc: invalid data for selected column %q", column.field.Name)
			}
			for row := start; row < end; row++ {
				if err := checkContext(ctx); err != nil {
					return err
				}
				value, conversionErr := cellValue(array, int(row))
				if err := checkContext(ctx); err != nil {
					return err
				}
				if conversionErr != nil {
					return fmt.Errorf("arrowipc: column %q row %d: %w", column.field.Name, row, conversionErr)
				}
				values[int(row-start)] = value
			}
			columns = append(columns, insyra.NewDataList().Append(values...).SetName(column.field.Name))
		}
		table := insyra.NewDataTable(columns...)
		if err := checkContext(ctx); err != nil {
			return err
		}
		select {
		case tables <- table:
		case <-ctx.Done():
			return ctx.Err()
		}
		if err := checkContext(ctx); err != nil {
			return err
		}
		start = end
	}
	return nil
}
