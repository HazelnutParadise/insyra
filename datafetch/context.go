package datafetch

import (
	"context"
	"errors"
	"time"
)

// errNilContext is what a Context method returns for a nil context, where the
// standard library would panic.
var errNilContext = errors.New("datafetch: nil context")

// contextErr is checked before a Context method does anything: it returns
// errNilContext for a nil context, ctx.Err() once ctx is done, and nil
// otherwise.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errNilContext
	}
	return ctx.Err()
}

// sleepContext waits for d, or until ctx is done if that comes first, in which
// case it returns ctx.Err().
func sleepContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
