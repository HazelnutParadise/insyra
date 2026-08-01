// Package reftest decides what happens when a verification cannot run because
// the implementation it checks against is not installed.
//
// Several checks in this repository compare Insyra's output against an
// independent implementation — R, Python's scientific stack, scikit-learn, an
// ONNX runtime. None of those can be assumed present, so each check begins by
// looking for its reference and calling t.Skip when it is absent.
//
// The cost of that is that `go test` prints `ok` either way. A skip is a line
// that scrolls past; nothing separates "this was checked and held" from "this
// was never checked". Two changes in this project were nearly archived on
// cross-language tests that had never executed, and the ONNX round trip was
// archived on one that had never executed anywhere — run for the first time it
// failed instantly, on two defects that made every exported model unloadable.
//
// So the decision is made here rather than at each call site, and it can be
// reversed by the environment: on a machine that is supposed to have the
// toolchains, a missing one is a failure.
package reftest

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// StrictEnv, set to "1", turns a missing reference implementation from a skip
// into a failure. Continuous integration sets it on the job that installs the
// toolchains; leaving it unset keeps `go test ./...` passing on a machine that
// has none of them.
const StrictEnv = "INSYRA_REQUIRE_REFERENCE_TOOLCHAINS"

// Strict reports whether missing reference implementations are being treated as
// failures. A check that is opt-in *because* its reference is usually absent
// should run when this is true — the reason it was opt-in no longer holds.
//
// Read that condition narrowly. A flag guarding a check that is known not to
// pass is a different thing, and promoting it here would make the verification
// job red over a difference someone already examined and accepted. The strict
// R factor-analysis parity suite is exactly that case and stays opt-in.
func Strict() bool {
	return strings.TrimSpace(os.Getenv(StrictEnv)) == "1"
}

// Missing is called when a check cannot run because the implementation it
// verifies against was not found. tool names what was missing; verification
// names what consequently went unchecked; cause may be nil.
//
// Both messages name both things on purpose. The skip message is the only
// record that a verification did not happen, so it has to say which one.
func Missing(t *testing.T, tool, verification string, cause error) {
	t.Helper()
	message := fmt.Sprintf("%s is unavailable, so %s did not run", tool, verification)
	if cause != nil {
		message += ": " + strings.TrimSpace(cause.Error())
	}
	if Strict() {
		t.Fatalf("%s (%s=1 requires it)", message, StrictEnv)
	}
	t.Skipf("%s (set %s=1 to make this a failure)", message, StrictEnv)
}

// MissingOutput is Missing for a probe that failed with captured output, which
// is usually more informative than the exit status alone.
func MissingOutput(t *testing.T, tool, verification string, cause error, output []byte) {
	t.Helper()
	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" {
		Missing(t, tool, verification, cause)
		return
	}
	Missing(t, tool, verification, fmt.Errorf("%v: %s", cause, trimmed))
}
