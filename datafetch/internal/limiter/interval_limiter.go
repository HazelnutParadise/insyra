package limiter

import (
	"context"
	"sync"
	"time"
)

// IntervalLimiter spaces the scheduled starts of successive calls by interval.
// Each call gets the next free slot, at least interval after the previous one,
// and Wait does not return before that slot. What a caller does after Wait
// returns is outside the guarantee, so two events it timestamps later can be a
// little closer together than interval. Concurrent callers line up and each
// gets its own slot.
type IntervalLimiter struct {
	mu          sync.Mutex
	nextAllowed time.Time
	interval    time.Duration
	now         func() time.Time
	sleep       func(context.Context, time.Duration) error
}

func NewIntervalLimiter(interval time.Duration) *IntervalLimiter {
	return &IntervalLimiter{interval: interval, now: time.Now, sleep: sleep}
}

// NewIntervalLimiterWithClock is NewIntervalLimiter reading the time from now
// and waiting through sleep, which returns ctx's error when ctx ends first.
// Tests use it to run the schedule on a fake clock.
func NewIntervalLimiterWithClock(interval time.Duration, now func() time.Time, sleep func(context.Context, time.Duration) error) *IntervalLimiter {
	return &IntervalLimiter{interval: interval, now: now, sleep: sleep}
}

// Next returns the earliest time the next Wait would let a call start.
func (l *IntervalLimiter) Next() time.Time {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	if l.nextAllowed.Before(now) {
		return now
	}
	return l.nextAllowed
}

func (l *IntervalLimiter) Wait(ctx context.Context) error {
	if l.interval <= 0 {
		return nil
	}

	l.mu.Lock()
	now := l.now()

	waitUntil := l.nextAllowed
	if waitUntil.Before(now) {
		waitUntil = now
	}

	// Reserve the next slot first (important for concurrent callers).
	l.nextAllowed = waitUntil.Add(l.interval)
	l.mu.Unlock()

	wait := waitUntil.Sub(now)
	if wait <= 0 {
		return nil
	}
	if err := l.sleep(ctx, wait); err != nil {
		// Roll back our reservation so a cancelled call does not permanently
		// advance the schedule and starve later callers. Only roll back if no
		// other caller has reserved a slot after us in the meantime.
		l.mu.Lock()
		if l.nextAllowed.Equal(waitUntil.Add(l.interval)) {
			l.nextAllowed = waitUntil
		}
		l.mu.Unlock()
		return err
	}
	return nil
}

func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
