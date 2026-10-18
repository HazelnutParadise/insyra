package insyra

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/HazelnutParadise/insyra/internal/core"
)

// The global error buffer is a bounded, program-wide diagnostic log: every
// warning, error and fatal insyra records lands here so a program can inspect
// what happened. It is NOT the error-handling API. Handle errors through the
// error a function returns, or through the sticky Err()/PopErr() on the
// instance you called — the global buffer mixes records from every goroutine
// and every object, so what you pop may belong to someone else.

type ErrPoppingMode int

// ErrPoppingMode defines the mode for popping errors.
const (
	// ErrPoppingModeFIFO retrieves the first error in the buffer.
	ErrPoppingModeFIFO ErrPoppingMode = iota
	// ErrPoppingModeLIFO retrieves the last error in the buffer.
	ErrPoppingModeLIFO
)

// ErrorInfo represents a structured error with context information.
// It is the public-facing error type returned by error retrieval functions.
type ErrorInfo struct {
	Level       LogLevel
	PackageName string
	FuncName    string
	Message     string
	Timestamp   time.Time
	// Cause is the underlying error, when the failure came from one. It lets
	// errors.Is and errors.As reach a typed error — ccl.CompileError, say —
	// through a recorded Err(), instead of the caller matching on Message.
	Cause error
}

// Error implements the error interface for ErrorInfo.
func (e ErrorInfo) Error() string {
	return fmt.Sprintf("[%s] %s.%s: %s", e.Level.String(), e.PackageName, e.FuncName, e.Message)
}

// Unwrap returns the underlying error, so errors.Is and errors.As see through
// a recorded ErrorInfo to whatever actually failed. It is on the pointer and
// returns nil for a nil *ErrorInfo, which is what Err() returns after a call
// that succeeded: errors.As(dt.Err(), &target) must not panic there.
func (e *ErrorInfo) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// String returns a string representation of the LogLevel.
func (l LogLevel) String() string {
	switch l {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarning:
		return "WARNING"
	case LogLevelError:
		return "ERROR"
	case LogLevelFatal:
		return "FATAL"
	default:
		return fmt.Sprintf("LogLevel(%d)", int(l))
	}
}

type errorStruct struct {
	packageName string
	fnName      string
	errType     LogLevel
	message     string
	timestamp   time.Time
}

// ErrorBufferCapacity is the number of records the global diagnostic buffer
// keeps. When it is full the oldest record is dropped, so a long-running
// program that never reads the buffer cannot grow without bound.
const ErrorBufferCapacity = 1536

var (
	errRing    = core.NewRing[errorStruct](ErrorBufferCapacity)
	errorMutex = sync.Mutex{}
)

func pushError(errType LogLevel, packageName, fnName, errMes string) {
	err := errorStruct{
		errType:     errType,
		packageName: packageName,
		fnName:      fnName,
		message:     errMes,
		timestamp:   time.Now(),
	}
	errorMutex.Lock()
	// Bounded: drop the oldest record rather than letting the ring grow.
	for errRing.Len() >= ErrorBufferCapacity {
		errRing.PopFront()
	}
	errRing.Push(err)
	errorMutex.Unlock()
	if errHandlingFunc := Config.GetDefaultErrHandlingFunc(); errHandlingFunc != nil {
		dispatchToHook(hookCall{fn: errHandlingFunc, err: err})
	}
}

// The error hook runs on one goroutine behind a bounded queue: one goroutine
// per error (the previous design) grew without limit under a slow hook, and
// delivered messages out of order.
type hookCall struct {
	fn  errHandlingFunc
	err errorStruct
}

const hookQueueSize = 1024

var (
	hookQueue     chan hookCall
	hookOnce      sync.Once
	hookDropNoted atomic.Bool
)

func dispatchToHook(call hookCall) {
	hookOnce.Do(func() {
		hookQueue = make(chan hookCall, hookQueueSize)
		go func() {
			for c := range hookQueue {
				c.fn(c.err.errType, c.err.packageName, c.err.fnName, c.err.message)
			}
		}()
	})
	select {
	case hookQueue <- call:
	default:
		if hookDropNoted.CompareAndSwap(false, true) {
			// Not LogWarning: that would record the warning and hand it to
			// the very hook whose queue is full.
			if Config.GetLogLevel() <= LogLevelWarning {
				emit(1, LogLevelWarning, "", "", "error hook queue full (%d); further hook calls are dropped until it drains (errors are still recorded)", hookQueueSize)
			}
		}
	}
}

func ClearErrors() {
	errorMutex.Lock()
	defer errorMutex.Unlock()

	errRing.Clear()
}

func GetErrorCount() int {
	errorMutex.Lock()
	defer errorMutex.Unlock()

	return errRing.Len()
}

// HasError returns true if there are any errors in the buffer.
// This is a non-destructive check that doesn't modify the error buffer.
func HasError() bool {
	errorMutex.Lock()
	defer errorMutex.Unlock()

	return errRing.Len() > 0
}

// GetAllErrors returns a copy of all errors in the buffer without removing them.
// The returned slice is ordered from oldest to newest (FIFO order).
func GetAllErrors() []ErrorInfo {
	errorMutex.Lock()
	defer errorMutex.Unlock()

	errSlice := errRing.ToSlice()
	result := make([]ErrorInfo, len(errSlice))
	for i, err := range errSlice {
		result[i] = ErrorInfo{
			Level:       err.errType,
			PackageName: err.packageName,
			FuncName:    err.fnName,
			Message:     err.message,
			Timestamp:   err.timestamp,
		}
	}
	return result
}

// PopAllErrors retrieves and removes all errors from the buffer.
// The returned slice is ordered from oldest to newest (FIFO order).
func PopAllErrors() []ErrorInfo {
	errorMutex.Lock()
	defer errorMutex.Unlock()

	errSlice := errRing.ToSlice()
	result := make([]ErrorInfo, len(errSlice))
	for i, err := range errSlice {
		result[i] = ErrorInfo{
			Level:       err.errType,
			PackageName: err.packageName,
			FuncName:    err.fnName,
			Message:     err.message,
			Timestamp:   err.timestamp,
		}
	}
	errRing.Clear()
	return result
}

// nilReceiverError is what Err() reports on a nil DataList or DataTable, such
// as the one a lookup returns when it finds nothing. Asking what went wrong
// must not crash; only Err() is guarded, so a nil still fails loudly on every
// other method instead of travelling on unnoticed. Each call builds a new
// value, so one caller changing it cannot change what the next is told.
func nilReceiverError(kind string) *ErrorInfo {
	return &ErrorInfo{
		Level:       LogLevelError,
		PackageName: "insyra",
		FuncName:    "Err",
		Message:     fmt.Sprintf("nil %s: the value does not exist, often because the lookup that produced it found nothing", kind),
		Timestamp:   time.Now(),
	}
}
