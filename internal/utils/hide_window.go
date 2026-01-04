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
func ApplyHideWindow(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	// If SysProcAttr is not set yet, create it with HideWindow = true.
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		return
	}
	// If already present, set the HideWindow flag.
	cmd.SysProcAttr.HideWindow = true
}
