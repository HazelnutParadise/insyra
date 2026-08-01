package accel

import (
	"sync"
	"testing"
)

type countingDiscoverer struct {
	mu    sync.Mutex
	calls int
}

func (d *countingDiscoverer) Name() string { return "counting" }

func (d *countingDiscoverer) Discover(Config) ([]Device, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.calls++
	return nil, nil
}

func (d *countingDiscoverer) count() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.calls
}

func isolateDefault(t *testing.T) *countingDiscoverer {
	t.Helper()
	ResetDiscoverersForTest()
	t.Cleanup(ResetDiscoverersForTest)
	isolateBuiltinProbes(t)
	ResetDefaultForTest()
	t.Cleanup(ResetDefaultForTest)
	counter := &countingDiscoverer{}
	RegisterDiscoverer(counter)
	return counter
}

func TestDefaultReturnsOneSessionAndDiscoversOnce(t *testing.T) {
	counter := isolateDefault(t)

	first := Default()
	second := Default()
	if first == nil {
		t.Fatal("expected a session")
	}
	if first != second {
		t.Fatal("expected the same session on every call")
	}
	if got := counter.count(); got != 1 {
		t.Fatalf("expected discovery to run once, ran %d times", got)
	}
}

func TestDefaultIsUsableWithoutAnyAccelerator(t *testing.T) {
	isolateDefault(t)

	session := Default()
	if session == nil {
		t.Fatal("a host with no device must still get a session")
	}
	if len(session.Devices()) != 0 {
		t.Fatalf("expected no devices, got %d", len(session.Devices()))
	}
	report := session.Report()
	if report.Accelerated {
		t.Fatal("expected the report to say acceleration is unavailable")
	}
	if report.FallbackReason == FallbackReasonNone {
		t.Fatal("expected a fallback reason explaining why")
	}
}

func TestDefaultCloseIsANoOp(t *testing.T) {
	isolateDefault(t)

	session := Default()
	if err := session.Close(); err != nil {
		t.Fatalf("closing the shared session must not error: %v", err)
	}
	if session.Closed() {
		t.Fatal("the shared session must stay open for every other caller")
	}
	// Still usable: a closed session refuses RegisterDevice.
	if err := session.RegisterDevice(Device{ID: "test:0", Backend: BackendCUDA}); err != nil {
		t.Fatalf("the shared session must remain usable after Close: %v", err)
	}
	if Default() != session {
		t.Fatal("Close must not replace the shared session")
	}
}

func TestOwnSessionStillClosesNormally(t *testing.T) {
	session := singleDeviceSession(t, Config{})
	if err := session.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}
	if !session.Closed() {
		t.Fatal("a caller-owned session must actually close")
	}
	if err := session.RegisterDevice(Device{ID: "x", Backend: BackendCUDA}); err == nil {
		t.Fatal("a closed session must refuse mutation")
	}
}

func TestDefaultIsSafeUnderConcurrentFirstUse(t *testing.T) {
	counter := isolateDefault(t)

	const goroutines = 16
	sessions := make([]*Session, goroutines)
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sessions[i] = Default()
		}(i)
	}
	wg.Wait()

	for i, session := range sessions {
		if session != sessions[0] {
			t.Fatalf("goroutine %d got a different session", i)
		}
	}
	if got := counter.count(); got != 1 {
		t.Fatalf("expected exactly one discovery under concurrent first use, got %d", got)
	}
}
