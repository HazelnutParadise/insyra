//go:build windows
// +build windows

package utils

import (
	"os/exec"
	"syscall"
)

// ApplyHideWindow sets process attributes so that executing a command on Windows
// does not create a visible console window. Safe to call with a nil *exec.Cmd
// (no-op).
//
// This implementation uses SysProcAttr.HideWindow to hide the child process's
// window instead of setting CreationFlags. This is simpler and avoids use of
// platform-specific numeric flags.
func ApplyHideWindow(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	// If SysProcAttr is nil, create it with HideWindow enabled.
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		return
	}
	// Ensure HideWindow is set regardless of other existing attributes.
	cmd.SysProcAttr.HideWindow = true
}
