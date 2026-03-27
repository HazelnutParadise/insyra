package commands

import (
	"bytes"
	"strings"
	"testing"

	insyra "github.com/HazelnutParadise/insyra"
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
	if !strings.Contains(rendered, "probe=env-stub") {
		t.Fatalf("expected probe source in output, got %q", rendered)
	}
	if !strings.Contains(rendered, "caps=encoded_strings,env_stub,heterogeneous_planning,portable,shardable,shared_memory,validity_bitmap") {
		t.Fatalf("expected normalized capability list in output, got %q", rendered)
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
	t.Setenv("INSYRA_ACCEL_STUB_WEBGPU", "1")

	output := &bytes.Buffer{}
	ctx := &ExecContext{
		Vars: map[string]any{
			"numbers": insyra.NewDataList(1, 2, nil, 4).SetName("numbers"),
		},
		Output: output,
	}

	if err := runShowCommand(ctx, []string{"accel.cache"}); err != nil {
		t.Fatalf("runShowCommand failed: %v", err)
	}

	if !strings.Contains(output.String(), "resident_buffers=1") {
		t.Fatalf("expected resident buffer count in output, got %q", output.String())
	}
	if !strings.Contains(output.String(), "numbers") {
		t.Fatalf("expected buffer name in cache output, got %q", output.String())
	}
	if !strings.Contains(output.String(), "device webgpu:stub:0") {
		t.Fatalf("expected per-device cache usage in output, got %q", output.String())
	}
}

func TestRunAccelCommandRunPrintsReasonAndDeviceCounts(t *testing.T) {
	setupCommandHome(t)

	output := &bytes.Buffer{}
	ctx := &ExecContext{Vars: map[string]any{}, Output: output}

	err := runAccelCommand(ctx, []string{"run", "--mode", "strict-gpu"})
	if err == nil {
		t.Fatal("expected strict-gpu run to fail without accelerators")
	}

	rendered := output.String()
	if !strings.Contains(rendered, "reason=strict-gpu-unavailable") {
		t.Fatalf("expected strict-gpu reason in output, got %q", rendered)
	}
	if !strings.Contains(rendered, "discovered=0") {
		t.Fatalf("expected discovered count in output, got %q", rendered)
	}
}

func TestRunAccelCommandRunPrintsShardPlanSummary(t *testing.T) {
	setupCommandHome(t)
	t.Setenv("INSYRA_ACCEL_STUB_CUDA", "1")
	t.Setenv("INSYRA_ACCEL_STUB_WEBGPU", "1")

	output := &bytes.Buffer{}
	ctx := &ExecContext{Vars: map[string]any{}, Output: output}

	if err := runAccelCommand(ctx, []string{"run", "--mode", "auto"}); err != nil {
		t.Fatalf("runAccelCommand failed: %v", err)
	}

	rendered := output.String()
	if !strings.Contains(rendered, "planned=2") {
		t.Fatalf("expected planned device count in output, got %q", rendered)
	}
	if !strings.Contains(rendered, "shard_devices=cuda:stub:0,webgpu:stub:0") {
		t.Fatalf("expected shard devices in output, got %q", rendered)
	}
}
