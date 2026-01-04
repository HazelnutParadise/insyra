//go:build windows
// +build windows

package utils

import (
	"os/exec"
	"syscall"
)

// Local constant for CREATE_NO_WINDOW (winbase.h)
// Use a local constant to avoid relying on syscall name availability in different
// Go toolchain versions/environments.
const CREATE_NO_WINDOW = 0x08000000

// ApplyHideWindow sets process attributes so that executing a command on Windows
// does not create a visible console window. Safe to call with a nil *exec.Cmd
// (no-op).
//
// This implementation uses CREATE_NO_WINDOW so the child process is created
// without a console. It is more robust at preventing a black console window
// from appearing for console applications. If callers rely on the child having
// a console, be aware that CREATE_NO_WINDOW prevents that.
func ApplyHideWindow(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	// Ensure SysProcAttr exists and set the CreationFlags to include
	// CREATE_NO_WINDOW so no console is created for the child process.
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: CREATE_NO_WINDOW}
		return
	}
	// Preserve any existing CreationFlags and add CREATE_NO_WINDOW.
	cmd.SysProcAttr.CreationFlags |= CREATE_NO_WINDOW
}
