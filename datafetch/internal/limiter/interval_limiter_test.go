package limiter

import (
	"context"
	"sort"
	"sync"
	"testing"
	"time"
)

// This package had no test at all. It is what keeps datafetch from hammering
// an exchange's endpoint, so the spacing and the cancellation rollback are
// worth pinning.

// A zero or negative interval means "no limit": Wait returns at once and never
// reserves a slot.
func TestWait_NoInterval(t *testing.T) {
	for _, interval := range []time.Duration{0, -time.Second} {
		l := NewIntervalLimiter(interval)
		start := time.Now()
		for i := 0; i < 5; i++ {
			if err := l.Wait(context.Background()); err != nil {
				t.Fatalf("interval %v: %v", interval, err)
			}
		}
		if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
			t.Errorf("interval %v: five calls took %v, want no waiting", interval, elapsed)
		}
	}
}

// The first call goes through immediately; the second waits out the interval.
func TestWait_SpacesConsecutiveCalls(t *testing.T) {
	const interval = 60 * time.Millisecond
	l := NewIntervalLimiter(interval)

	start := time.Now()
	if err := l.Wait(context.Background()); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if elapsed := time.Since(start); elapsed > interval/2 {
		t.Errorf("the first call waited %v, want no wait", elapsed)
	}

	if err := l.Wait(context.Background()); err != nil {
		t.Fatalf("second call: %v", err)
	}
	elapsed := time.Since(start)
	if elapsed < interval {
		t.Errorf("two calls took %v, want at least one interval (%v)", elapsed, interval)
	}
}

// Concurrent callers line up rather than all going at once: the slot is
// reserved before the wait, so n callers take at least (n-1) intervals.
func TestWait_SerialisesConcurrentCallers(t *testing.T) {
	const interval = 40 * time.Millisecond
	const callers = 4
	l := NewIntervalLimiter(interval)

	start := time.Now()
	var wg sync.WaitGroup
	wg.Add(callers)
	errs := make([]error, callers)
	for i := 0; i < callers; i++ {
		go func(i int) {
			defer wg.Done()
			errs[i] = l.Wait(context.Background())
		}(i)
	}
	wg.Wait()
	elapsed := time.Since(start)

	for i, err := range errs {
		if err != nil {
			t.Errorf("caller %d: %v", i, err)
		}
	}
	if want := time.Duration(callers-1) * interval; elapsed < want {
		t.Errorf("%d callers took %v, want at least %v — they were not spaced", callers, elapsed, want)
	}
}

// A caller whose context is cancelled while waiting reports the cancellation.
func TestWait_CancelledContext(t *testing.T) {
	const interval = time.Second
	l := NewIntervalLimiter(interval)

	if err := l.Wait(context.Background()); err != nil {
		t.Fatalf("first call: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	start := time.Now()
	err := l.Wait(ctx)
	if err == nil {
		t.Fatal("a cancelled wait returned no error")
	}
	if elapsed := time.Since(start); elapsed > interval/2 {
		t.Errorf("the cancelled call took %v, want it to give up at the deadline", elapsed)
	}
}

// A cancelled call gives its reservation back, so the caller after it is not
// pushed a whole extra interval into the future. Without the rollback the
// third call below would wait two intervals instead of one.
func TestWait_CancelledCallRollsBackItsReservation(t *testing.T) {
	const interval = 120 * time.Millisecond
	l := NewIntervalLimiter(interval)

	if err := l.Wait(context.Background()); err != nil { // takes slot 0, returns now
		t.Fatalf("first call: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), interval/6)
	defer cancel()
	if err := l.Wait(ctx); err == nil { // reserves slot 1, then gives up
		t.Fatal("the second call should have been cancelled")
	}
	cancel()

	start := time.Now()
	if err := l.Wait(context.Background()); err != nil {
		t.Fatalf("third call: %v", err)
	}
	// Slot 1 was released, so this call takes it: at most one interval from the
	// first call, which started roughly one sixth of an interval ago.
	if elapsed := time.Since(start); elapsed > interval*3/2 {
		t.Errorf("the call after a cancelled one waited %v, want about one interval (%v) — the reservation was not released", elapsed, interval)
	}
}

// The whole point of the limiter is that two allowed calls are never closer
// together than the interval — including around a cancelled caller, whose
// rollback must not rewind past a slot someone else has already taken. This
// asserts that invariant directly over a mix of cancelled and normal callers.
func TestWait_AllowedCallsAreNeverCloserThanTheInterval(t *testing.T) {
	const interval = 40 * time.Millisecond
	l := NewIntervalLimiter(interval)

	var mu sync.Mutex
	var allowed []time.Time

	cancelled, cancelAll := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		ctx := context.Background()
		if i%3 == 1 { // every third caller gives up
			ctx = cancelled
		}
		wg.Add(1)
		go func(ctx context.Context) {
			defer wg.Done()
			if err := l.Wait(ctx); err == nil {
				mu.Lock()
				allowed = append(allowed, time.Now())
				mu.Unlock()
			}
		}(ctx)
	}
	time.Sleep(interval / 2)
	cancelAll()
	wg.Wait()

	if len(allowed) < 4 {
		t.Fatalf("only %d callers were allowed through, want at least 4", len(allowed))
	}
	sort.Slice(allowed, func(i, j int) bool { return allowed[i].Before(allowed[j]) })
	// The first allowed call sets the clock; each one after it must be a full
	// interval later. Half the interval of slack absorbs scheduling jitter
	// between the timer firing and the timestamp being taken.
	for i := 1; i < len(allowed); i++ {
		if gap := allowed[i].Sub(allowed[i-1]); gap < interval/2 {
			t.Errorf("calls %d and %d were %v apart, want about %v", i-1, i, gap, interval)
		}
	}
}
