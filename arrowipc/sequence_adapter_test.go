package arrowipc_test

import (
	"context"
	"io"
	"iter"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/arrowipc"
)

func streamFrom(ctx context.Context, r io.ReaderAt, size int64, opt arrowipc.ReadOptions, batchSize int) (<-chan *insyra.DataTable, <-chan error) {
	return sequenceChannels(ctx, arrowipc.StreamFrom(ctx, r, size, opt, batchSize))
}
func streamPath(ctx context.Context, path string, opt arrowipc.ReadOptions, batchSize int) (<-chan *insyra.DataTable, <-chan error) {
	return sequenceChannels(ctx, arrowipc.Stream(ctx, path, opt, batchSize))
}

// Adapt the public sequence only in tests so both maintained lines use the
// same independent value/ownership/error assertions and generated fixtures.
func sequenceChannels(ctx context.Context, seq iter.Seq2[*insyra.DataTable, error]) (<-chan *insyra.DataTable, <-chan error) {
	tables := make(chan *insyra.DataTable)
	errs := make(chan error, 1)
	go func() {
		defer close(tables)
		defer close(errs)
		for table, err := range seq {
			if err != nil {
				errs <- err
				return
			}
			select {
			case tables <- table:
			case <-ctx.Done():
				errs <- ctx.Err()
				return
			}
		}
	}()
	return tables, errs
}
