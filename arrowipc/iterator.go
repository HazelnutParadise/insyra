package arrowipc

import (
	"context"
	"iter"

	"github.com/HazelnutParadise/insyra"
)

func streamSequence(ctx context.Context, start func(context.Context) (<-chan *insyra.DataTable, <-chan error)) iter.Seq2[*insyra.DataTable, error] {
	return func(yield func(*insyra.DataTable, error) bool) {
		if err := checkContext(ctx); err != nil {
			yield(nil, err)
			return
		}
		inner, cancel := context.WithCancel(ctx)
		tables, errs := start(inner)
		// Also runs when the caller's loop panics. Never recover the caller's
		// panic or yield again after it; only finish the internal producer.
		defer func() {
			cancel()
			for range tables {
			}
			<-errs
		}()
		for table := range tables {
			if !yield(table, nil) {
				return
			}
		}
		if err := <-errs; err != nil {
			yield(nil, err)
		}
	}
}
