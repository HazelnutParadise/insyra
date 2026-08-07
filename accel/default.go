package accel

import "sync"

// The process-shared session. Library code that accelerates on a user's behalf
// cannot open a session per call — discovery would run every time, nothing
// would stay resident across operations, and no caller would own Close.
var (
	defaultOnce    sync.Once
	defaultSession *Session
	defaultMu      sync.Mutex
)

// Default returns the session shared by this process, creating it on first use.
//
// Construction is lazy on purpose: importing accel must not open a GPU device,
// because most programs reach this package transitively through allpkgs without
// ever asking for acceleration.
//
// A host with no usable device still gets a session. Discovery failure is
// reported through the session's report and fallback reason, which is the
// runtime's normal way of saying "no acceleration here" — returning nil would
// push a nil check into every call site to express the same thing.
//
// The returned session is shared, so Close on it does nothing. Callers that
// need their own lifetime should use Open.
func Default() *Session {
	defaultMu.Lock()
	once := &defaultOnce
	defaultMu.Unlock()

	once.Do(func() {
		session := NewSession()
		session.shared = true
		// Discovery errors are already carried in the report; a shared session
		// that refused to exist would be worse than one that explains itself.
		_ = session.Discover()
		defaultMu.Lock()
		defaultSession = session
		defaultMu.Unlock()
	})

	defaultMu.Lock()
	defer defaultMu.Unlock()
	return defaultSession
}

// ResetDefaultForTest drops the shared session so the next Default call
// rediscovers. Tests use this the way they use ResetDiscoverersForTest.
func ResetDefaultForTest() {
	defaultMu.Lock()
	defer defaultMu.Unlock()
	defaultOnce = sync.Once{}
	defaultSession = nil
}
