package insyra

import (
	"os"
	"testing"
)

// TestMain sets the one configuration every test in this package starts from:
// the defaults, with logging turned down to Fatal so failure-path tests do not
// flood the output. Two test files used to do this from init(), where nobody
// reading a test would find it. A test that changes the configuration puts it
// back with restoreConfig.
func TestMain(m *testing.M) {
	SetDefaultConfig()
	Config.SetLogLevel(LogLevelFatal)
	os.Exit(m.Run())
}
