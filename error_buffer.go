package insyra

import (
	"slices"
	"sync"
)

type ErrPoppingMode int

// ErrPoppingMode defines the mode for popping errors.
const (
	// ErrPoppingModeFIFO retrieves the first error in the slice.
	ErrPoppingModeFIFO ErrPoppingMode = iota
	// ErrPoppingModeLIFO retrieves the last error in the slice.
	ErrPoppingModeLIFO
)

var (
	errorSlice = make([]errorStruct, 0, 1536)
	errorChan  = make(chan errorStruct, 1024)
	errorMutex = sync.Mutex{}
)

type errorStruct struct {
	packageName string
	fnName      string
	errType     LogLevel
	message     string
}

func init() {
	// Initialize the error channel
	go func() {
		for err := range errorChan {
			if errHandlingFunc := Config.GetDefaultErrHandlingFunc(); errHandlingFunc != nil {
				go errHandlingFunc(err.errType, err.packageName, err.fnName, err.message)
			}
			errorMutex.Lock()
			errorSlice = append(errorSlice, err)
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
	}
	errorChan <- err
}

// PopError retrieves and removes the first error from the errorSlice.
// If the slice is empty, it returns an empty string and LogLevelInfo.
func PopError(mode ErrPoppingMode) (LogLevel, string) {
	errorMutex.Lock()
	defer errorMutex.Unlock()

	if len(errorSlice) == 0 {
		return LogLevelInfo, ""
	}
	var err errorStruct
	switch mode {
	case ErrPoppingModeFIFO:
		err = errorSlice[0]
		errorSlice = errorSlice[1:]
	case ErrPoppingModeLIFO:
		err = errorSlice[len(errorSlice)-1]
		errorSlice = errorSlice[:len(errorSlice)-1]
	}
	return err.errType, err.message
}

func PopErrorByPackageName(packageName string, mode ErrPoppingMode) (LogLevel, string) {
	errorMutex.Lock()
	defer errorMutex.Unlock()

	if len(errorSlice) == 0 {
		return LogLevelInfo, ""
	}

	idxToPop := -1

	switch mode {
	case ErrPoppingModeFIFO:
		// Find the first occurrence
		for i, e := range errorSlice {
			if e.packageName == packageName {
				idxToPop = i
				break
			}
		}
	case ErrPoppingModeLIFO:
		// Find the last occurrence
		for i := len(errorSlice) - 1; i >= 0; i-- {
			if errorSlice[i].packageName == packageName {
				idxToPop = i
				break
			}
		}
	}

	if idxToPop != -1 {
		err := errorSlice[idxToPop]
		errorSlice = slices.Delete(errorSlice, idxToPop, idxToPop+1)
		return err.errType, err.message
	}

	return LogLevelInfo, "" // No error found for the given package name
}

func PopErrorByFuncName(packageName, funcName string, mode ErrPoppingMode) (LogLevel, string) {
	errorMutex.Lock()
	defer errorMutex.Unlock()

	if len(errorSlice) == 0 {
		return LogLevelInfo, ""
	}

	idxToPop := -1

	switch mode {
	case ErrPoppingModeFIFO:
		// Find the first occurrence
		for i, e := range errorSlice {
			if e.packageName == packageName && e.fnName == funcName {
				idxToPop = i
				break
			}
		}
	case ErrPoppingModeLIFO:
		// Find the last occurrence
		for i := len(errorSlice) - 1; i >= 0; i-- {
			if errorSlice[i].packageName == packageName && errorSlice[i].fnName == funcName {
				idxToPop = i
				break
			}
		}
	}

	if idxToPop != -1 {
		err := errorSlice[idxToPop]
		errorSlice = slices.Delete(errorSlice, idxToPop, idxToPop+1)
		return err.errType, err.message
	}

	return LogLevelInfo, "" // No error found for the given package and function name
}

func PopErrorAndCallback(mode ErrPoppingMode, callback func(errType LogLevel, packageName string, funcName string, errMsg string)) {
	errorMutex.Lock()
	defer errorMutex.Unlock()

	if len(errorSlice) == 0 {
		return
	}
	var err errorStruct
	switch mode {
	case ErrPoppingModeFIFO:
		err = errorSlice[0]
		errorSlice = errorSlice[1:]
	case ErrPoppingModeLIFO:
		err = errorSlice[len(errorSlice)-1]
		errorSlice = errorSlice[:len(errorSlice)-1]
	}
	callback(err.errType, err.packageName, err.fnName, err.message)
}

func ClearErrors() {
	errorMutex.Lock()
	defer errorMutex.Unlock()

	errorSlice = make([]errorStruct, 0, 1536)
}

func GetErrorCount() int {
	errorMutex.Lock()
	defer errorMutex.Unlock()

	return len(errorSlice)
}
