//go:build !windows
// +build !windows

package utils

import "os/exec"

// ApplyHideWindow is a no-op on non-Windows platforms.
//
// The function exists so callers can unconditionally call it regardless of
// platform; on Unix-like systems there is no console window to hide, so this
// simply accepts the *exec.Cmd and does nothing.
func ApplyHideWindow(cmd *exec.Cmd) {
	// intentionally no-op on non-Windows platforms
	_ = cmd
}
