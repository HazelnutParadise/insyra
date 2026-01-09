package limiter

import (
	"context"
	"sync"
	"time"
)

// IntervalLimiter ensures each allowed call is spaced by `interval`.
// It serializes concurrent callers: they will line up and each gets its own slot.
type IntervalLimiter struct {
	mu          sync.Mutex
	nextAllowed time.Time
	interval    time.Duration
}

func NewIntervalLimiter(interval time.Duration) *IntervalLimiter {
	return &IntervalLimiter{interval: interval}
}

func (l *IntervalLimiter) Wait(ctx context.Context) error {
	if l.interval <= 0 {
		return nil
	}

	l.mu.Lock()
	now := time.Now()

	waitUntil := l.nextAllowed
	if waitUntil.Before(now) {
		waitUntil = now
	}

	// Reserve the next slot first (important for concurrent callers).
	l.nextAllowed = waitUntil.Add(l.interval)
	l.mu.Unlock()

	wait := time.Until(waitUntil)
	if wait <= 0 {
		return nil
	}

	t := time.NewTimer(wait)
	defer t.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
