package env

import (
	"errors"
	"os"
	"runtime"
	"time"
)

// renameRetryOS and renameRetryFor let a test stand in for Windows and shorten
// the wait.
var (
	renameRetryOS  = runtime.GOOS
	renameRetryFor = 2 * time.Second
)

// replaceFile renames tmp over final. Windows refuses with "access denied" to
// replace a file that another process or goroutine has open for reading at
// that instant, such as a restore reading state.json, and the file is free a
// moment later, so it retries for a short while before giving up.
func replaceFile(tmp, final string) error {
	deadline := time.Now().Add(renameRetryFor)
	for {
		err := os.Rename(tmp, final)
		if err == nil || renameRetryOS != "windows" || !errors.Is(err, os.ErrPermission) || time.Now().After(deadline) {
			return err
		}
		time.Sleep(10 * time.Millisecond)
	}
}
