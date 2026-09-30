package py

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
)

// PY-1 of #254: the environment was prepared only as a side effect of the
// first call. Setup lets a program prepare it when it chooses.
func TestSetupPreparesTheEnvironmentAheadOfTheFirstCall(t *testing.T) {
	envDir, log := useTempEnvironment(t, "ok")
	if err := Setup(context.Background()); err != nil {
		t.Fatal(err)
	}
	if runs := uvRuns(t, log); len(runs) != 1 {
		t.Fatalf("Setup ran uv %d times, want one sync: %q", len(runs), runs)
	}
	if !envInSync(envDir) {
		t.Error("Setup left the environment out of sync")
	}
	if err := pyEnvInit(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := Setup(context.Background()); err != nil {
		t.Fatal(err)
	}
	if runs := uvRuns(t, log); len(runs) != 1 {
		t.Errorf("calls after Setup ran uv again: %q", runs)
	}
}

func TestAFailedSetupIsReturnedAndTriedAgain(t *testing.T) {
	useTempEnvironment(t, "fail")
	if err := Setup(context.Background()); err == nil || !strings.Contains(err.Error(), "fake uv was told to fail") {
		t.Fatalf("a failing sync gave %v, want uv's error output", err)
	}
	pyInitMu.Lock()
	ready := isPyEnvInit
	pyInitMu.Unlock()
	if ready {
		t.Fatal("a failed Setup marked the environment ready")
	}
	t.Setenv(fakeUVEnv, "ok")
	if err := Setup(context.Background()); err != nil {
		t.Errorf("the next Setup failed too: %v", err)
	}
}

func TestSetupRefusesANilOrFinishedContextWithoutTouchingTheEnvironment(t *testing.T) {
	envDir, log := useTempEnvironment(t, "ok")
	var nilCtx context.Context
	if err := Setup(nilCtx); !errors.Is(err, errNilContext) {
		t.Errorf("Setup(nil) gave %v, want errNilContext", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := Setup(ctx); !errors.Is(err, context.Canceled) {
		t.Errorf("Setup with a cancelled context gave %v, want context.Canceled", err)
	}
	if _, err := os.Stat(envDir); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Setup touched the environment directory: %v", err)
	}
	if runs := uvRuns(t, log); len(runs) != 0 {
		t.Errorf("uv ran %d times", len(runs))
	}
}
