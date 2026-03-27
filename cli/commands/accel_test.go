package commands

import (
	"bytes"
	"strings"
	"testing"
)

func setupCommandHome(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	t.Setenv("HOMEDRIVE", "")
	t.Setenv("HOMEPATH", "")
}

func TestRunAccelCommandDevicesPrintsBuiltinStubDevices(t *testing.T) {
	setupCommandHome(t)
	t.Setenv("INSYRA_ACCEL_STUB_WEBGPU", "1")

	output := &bytes.Buffer{}
	ctx := &ExecContext{Vars: map[string]any{}, Output: output}

	if err := runAccelCommand(ctx, []string{"devices", "--mode", "auto"}); err != nil {
		t.Fatalf("runAccelCommand failed: %v", err)
	}

	rendered := output.String()
	if !strings.Contains(rendered, "webgpu:stub:0") {
		t.Fatalf("expected stub device id in output, got %q", rendered)
	}
	if !strings.Contains(rendered, "backend=webgpu") {
		t.Fatalf("expected backend in output, got %q", rendered)
	}
}

func TestShowCommandSupportsAccelDevices(t *testing.T) {
	setupCommandHome(t)
	t.Setenv("INSYRA_ACCEL_STUB_WEBGPU", "1")

	output := &bytes.Buffer{}
	ctx := &ExecContext{Vars: map[string]any{}, Output: output}

	if err := runShowCommand(ctx, []string{"accel.devices"}); err != nil {
		t.Fatalf("runShowCommand failed: %v", err)
	}

	rendered := output.String()
	if !strings.Contains(rendered, "webgpu:stub:0") {
		t.Fatalf("expected show accel.devices to print stub device, got %q", rendered)
	}
}

func TestShowCommandSupportsAccelCache(t *testing.T) {
	setupCommandHome(t)

	output := &bytes.Buffer{}
	ctx := &ExecContext{Vars: map[string]any{}, Output: output}

	if err := runShowCommand(ctx, []string{"accel.cache"}); err != nil {
		t.Fatalf("runShowCommand failed: %v", err)
	}

	if !strings.Contains(output.String(), "cache not implemented") {
		t.Fatalf("expected cache placeholder output, got %q", output.String())
	}
}
