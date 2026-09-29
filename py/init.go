// `py` package provides functions for working with Python.
package py

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/HazelnutParadise/insyra"
)

// 用於allpkgs安裝
func init() {}

// applyHideWindow moved to internal utils; use utils.ApplyHideWindow(cmd) instead.

var isPyEnvInit = false

// setupLock serializes the environment setup. Unlike a sync.Mutex, a caller
// waiting for it can give up when its context ends.
type setupLock chan struct{}

// Lock takes the lock, waiting as long as it takes.
func (l setupLock) Lock() { l <- struct{}{} }

// Unlock releases the lock.
func (l setupLock) Unlock() { <-l }

// LockContext takes the lock, or returns ctx.Err() if ctx ends first.
func (l setupLock) LockContext(ctx context.Context) error {
	select {
	case l <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

var pyInitMu = make(setupLock, 1) // serializes pyEnvInit against concurrent RunCode calls

// pyEnvInit prepares the pinned Python environment the first time it is
// needed: it downloads and verifies the pinned uv, then has uv bring the
// environment directory to the pinned versions. ctx bounds the download and
// the sync. A setup that fails is not marked ready, so the next call runs it
// again.
func pyEnvInit(ctx context.Context) error {
	// Serialize the environment-init check/set so concurrent RunCode calls do
	// not race on the shared flags.
	if err := pyInitMu.LockContext(ctx); err != nil {
		return err
	}
	defer pyInitMu.Unlock()
	if isPyEnvInit {
		return nil
	}
	return prepareEnvironmentLocked(ctx)
}

// prepareEnvironmentLocked does pyEnvInit's work. The caller holds pyInitMu.
func prepareEnvironmentLocked(ctx context.Context) error {
	uv, err := ensureUV(ctx, absInstallDir)
	if err != nil {
		return fmt.Errorf("py: failed to install uv: %w", err)
	}
	if !envInSync(absInstallDir) {
		insyra.LogInfo("py", "init", "Preparing the pinned Python environment in %s...", absInstallDir)
		if err := syncEnvironment(ctx, uv, absInstallDir); err != nil {
			return fmt.Errorf("py: failed to prepare the Python environment: %w", err)
		}
		insyra.LogInfo("py", "init", "Python environment ready")
	}
	pyPath = venvPython(filepath.Join(absInstallDir, ".venv"))
	uvPath = uv
	isPyEnvInit = true
	return nil
}
