package insyra

import (
	"fmt"
	"sync"
	"time"

	"github.com/HazelnutParadise/insyra/internal/core"
)

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
}

// Error implements the error interface for ErrorInfo.
func (e ErrorInfo) Error() string {
	return fmt.Sprintf("[%s] %s.%s: %s", e.Level.String(), e.PackageName, e.FuncName, e.Message)
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
	case LogLevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

type errorStruct struct {
	packageName string
	fnName      string
	errType     LogLevel
	message     string
	timestamp   time.Time
}

var (
	errRing    = core.NewRing[errorStruct](1536)
	errorChan  = make(chan errorStruct, 1024)
	errorMutex = sync.Mutex{}
)

func init() {
	// Initialize the error channel
	go func() {
		for err := range errorChan {
			if errHandlingFunc := Config.GetDefaultErrHandlingFunc(); errHandlingFunc != nil {
				go errHandlingFunc(err.errType, err.packageName, err.fnName, err.message)
			}
			errorMutex.Lock()
			errRing.Push(err)
			errorMutex.Unlock()
		}
	}()
}

func pushError(errType LogLevel, packageName, fnName, errMes string) {
	err := errorStruct{
		errType:     errType,
		packageName: packageName,
		fnName:      fnName,
		message:     errMes,
		timestamp:   time.Now(),
	}
	errorChan <- err
}

// PopError retrieves and removes the first error from the buffer.
// If the buffer is empty, it returns an empty string and LogLevelInfo.
func PopError(mode ErrPoppingMode) (LogLevel, string) {
	errorMutex.Lock()
	defer errorMutex.Unlock()

	if errRing.Len() == 0 {
		return LogLevelInfo, ""
	}
	var err errorStruct
	switch mode {
	case ErrPoppingModeFIFO:
		err, _ = errRing.PopFront()
	case ErrPoppingModeLIFO:
		err, _ = errRing.PopBack()
	}
	return err.errType, err.message
}

func PopErrorByPackageName(packageName string, mode ErrPoppingMode) (LogLevel, string) {
	errorMutex.Lock()
	defer errorMutex.Unlock()

	if errRing.Len() == 0 {
		return LogLevelInfo, ""
	}

	idxToPop := -1

	switch mode {
	case ErrPoppingModeFIFO:
		// Find the first occurrence
		for i := 0; i < errRing.Len(); i++ {
			entry, ok := errRing.Get(i)
			if ok && entry.packageName == packageName {
				idxToPop = i
				break
			}
		}
	case ErrPoppingModeLIFO:
		// Find the last occurrence
		for i := errRing.Len() - 1; i >= 0; i-- {
			entry, ok := errRing.Get(i)
			if ok && entry.packageName == packageName {
				idxToPop = i
				break
			}
		}
	}

	if idxToPop != -1 {
		err, _ := errRing.DeleteAt(idxToPop)
		return err.errType, err.message
	}

	return LogLevelInfo, "" // No error found for the given package name
}

func PopErrorByFuncName(packageName, funcName string, mode ErrPoppingMode) (LogLevel, string) {
	errorMutex.Lock()
	defer errorMutex.Unlock()

	if errRing.Len() == 0 {
		return LogLevelInfo, ""
	}

	idxToPop := -1

	switch mode {
	case ErrPoppingModeFIFO:
		// Find the first occurrence
		for i := 0; i < errRing.Len(); i++ {
			entry, ok := errRing.Get(i)
			if ok && entry.packageName == packageName && entry.fnName == funcName {
				idxToPop = i
				break
			}
		}
	case ErrPoppingModeLIFO:
		// Find the last occurrence
		for i := errRing.Len() - 1; i >= 0; i-- {
			entry, ok := errRing.Get(i)
			if ok && entry.packageName == packageName && entry.fnName == funcName {
				idxToPop = i
				break
			}
		}
	}

	if idxToPop != -1 {
		err, _ := errRing.DeleteAt(idxToPop)
		return err.errType, err.message
	}

	return LogLevelInfo, "" // No error found for the given package and function name
}

func PopErrorAndCallback(mode ErrPoppingMode, callback func(errType LogLevel, packageName string, funcName string, errMsg string)) {
	errorMutex.Lock()
	defer errorMutex.Unlock()

	if errRing.Len() == 0 {
		return
	}
	var err errorStruct
	switch mode {
	case ErrPoppingModeFIFO:
		err, _ = errRing.PopFront()
	case ErrPoppingModeLIFO:
		err, _ = errRing.PopBack()
	}
	callback(err.errType, err.packageName, err.fnName, err.message)
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

// HasErrorAboveLevel returns true if there are any errors at or above the specified level.
// This is a non-destructive check that doesn't modify the error buffer.
func HasErrorAboveLevel(level LogLevel) bool {
	errorMutex.Lock()
	defer errorMutex.Unlock()

	for i := 0; i < errRing.Len(); i++ {
		entry, ok := errRing.Get(i)
		if ok && entry.errType >= level {
			return true
		}
	}
	return false
}

// PeekError returns the error at the specified position without removing it.
// Returns nil if the buffer is empty or index is out of bounds.
// Mode determines whether to peek from the front (FIFO) or back (LIFO).
func PeekError(mode ErrPoppingMode) *ErrorInfo {
	errorMutex.Lock()
	defer errorMutex.Unlock()

	if errRing.Len() == 0 {
		return nil
	}

	var e errorStruct
	switch mode {
	case ErrPoppingModeFIFO:
		e, _ = errRing.Get(0)
	case ErrPoppingModeLIFO:
		e, _ = errRing.Get(errRing.Len() - 1)
	}

	return &ErrorInfo{
		Level:       e.errType,
		PackageName: e.packageName,
		FuncName:    e.fnName,
		Message:     e.message,
		Timestamp:   e.timestamp,
	}
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

// GetErrorsByLevel returns all errors at the specified level without removing them.
func GetErrorsByLevel(level LogLevel) []ErrorInfo {
	errorMutex.Lock()
	defer errorMutex.Unlock()

	var result []ErrorInfo
	for i := 0; i < errRing.Len(); i++ {
		entry, ok := errRing.Get(i)
		if ok && entry.errType == level {
			result = append(result, ErrorInfo{
				Level:       entry.errType,
				PackageName: entry.packageName,
				FuncName:    entry.fnName,
				Message:     entry.message,
				Timestamp:   entry.timestamp,
			})
		}
	}
	return result
}

// GetErrorsByPackage returns all errors from the specified package without removing them.
func GetErrorsByPackage(packageName string) []ErrorInfo {
	errorMutex.Lock()
	defer errorMutex.Unlock()

	var result []ErrorInfo
	for i := 0; i < errRing.Len(); i++ {
		entry, ok := errRing.Get(i)
		if ok && entry.packageName == packageName {
			result = append(result, ErrorInfo{
				Level:       entry.errType,
				PackageName: entry.packageName,
				FuncName:    entry.fnName,
				Message:     entry.message,
				Timestamp:   entry.timestamp,
			})
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

// PopErrorInfo retrieves and removes an error with full context information.
// Returns nil if the buffer is empty.
func PopErrorInfo(mode ErrPoppingMode) *ErrorInfo {
	errorMutex.Lock()
	defer errorMutex.Unlock()

	if errRing.Len() == 0 {
		return nil
	}

	var e errorStruct
	switch mode {
	case ErrPoppingModeFIFO:
		e, _ = errRing.PopFront()
	case ErrPoppingModeLIFO:
		e, _ = errRing.PopBack()
	}

	return &ErrorInfo{
		Level:       e.errType,
		PackageName: e.packageName,
		FuncName:    e.fnName,
		Message:     e.message,
		Timestamp:   e.timestamp,
	}
}
