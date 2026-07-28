package commands

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	insyra "github.com/HazelnutParadise/insyra"
	accelpkg "github.com/HazelnutParadise/insyra/accel"
)

func setupCommandHome(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	t.Setenv("HOMEDRIVE", "")
	t.Setenv("HOMEPATH", "")
	t.Setenv("INSYRA_ACCEL_DISABLE_NATIVE_PROBES", "1")
	t.Setenv("INSYRA_ACCEL_DISABLE_NVML_SDK", "1")
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
	if !strings.Contains(output.String(), "device ") {
		t.Fatalf("expected per-device cache usage in output, got %q", output.String())
	}
}

func TestRunAccelCommandRunPrintsReasonAndDeviceCounts(t *testing.T) {
	setupCommandHome(t)

	output := &bytes.Buffer{}
	ctx := &ExecContext{Vars: map[string]any{}, Output: output}

	err := runAccelCommand(ctx, []string{"plan", "--mode", "strict-gpu"})
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
	if !strings.Contains(rendered, "planning_only=true") {
		t.Fatalf("expected planning-only marker in output, got %q", rendered)
	}
}

func TestRunAccelCommandRunPrintsShardPlanSummary(t *testing.T) {
	setupCommandHome(t)
	t.Setenv("INSYRA_ACCEL_STUB_CUDA", "1")
	t.Setenv("INSYRA_ACCEL_STUB_WEBGPU", "1")

	output := &bytes.Buffer{}
	ctx := &ExecContext{Vars: map[string]any{}, Output: output}

	if err := runAccelCommand(ctx, []string{"plan", "--mode", "auto"}); err != nil {
		t.Fatalf("runAccelCommand failed: %v", err)
	}

	rendered := output.String()
	if !strings.Contains(rendered, "planned=2") {
		t.Fatalf("expected planned device count in output, got %q", rendered)
	}
	if !strings.Contains(rendered, "shard_devices=cuda:stub:0,webgpu:stub:0") {
		t.Fatalf("expected shard devices in output, got %q", rendered)
	}
	if !strings.Contains(rendered, "planning_only=true") {
		t.Fatalf("expected planning-only marker in output, got %q", rendered)
	}
	if !strings.Contains(rendered, "merge=cpu") {
		t.Fatalf("expected merge policy in output, got %q", rendered)
	}
	if !strings.Contains(rendered, "assignments=") {
		t.Fatalf("expected assignment summary in output, got %q", rendered)
	}
}

// cliSumExecutor stands in for a real GPU backend so the CLI's execution
// output can be tested on a machine with no device.
type cliSumExecutor struct{}

func (cliSumExecutor) Name() string { return "cli-fake" }

func (cliSumExecutor) Execute(_ context.Context, req accelpkg.ExecuteRequest) (accelpkg.ExecuteResponse, error) {
	var sum float64
	for _, value := range req.Values {
		sum += float64(value)
	}
	return accelpkg.ExecuteResponse{
		Value:         sum,
		Transfer:      time.Millisecond,
		Dispatch:      100 * time.Microsecond,
		Readback:      500 * time.Microsecond,
		BytesUploaded: uint64(len(req.Values) * 4),
	}, nil
}

func accelRunContext(t *testing.T) (*ExecContext, *bytes.Buffer) {
	t.Helper()
	setupCommandHome(t)
	t.Setenv("INSYRA_ACCEL_STUB_CUDA", "1")
	t.Setenv("INSYRA_ACCEL_STUB_WEBGPU", "1")

	values := make([]any, 512)
	for i := range values {
		values[i] = i + 1
	}
	output := &bytes.Buffer{}
	return &ExecContext{
		Vars: map[string]any{
			"numbers": insyra.NewDataList(values...).SetName("numbers"),
		},
		Output: output,
	}, output
}

func TestRunAccelCommandRunExecutesDataListVariable(t *testing.T) {
	ctx, output := accelRunContext(t)
	accelpkg.ResetBackendExecutorsForTest()
	t.Cleanup(accelpkg.ResetBackendExecutorsForTest)
	if err := accelpkg.RegisterBackendExecutor(accelpkg.BackendCUDA, cliSumExecutor{}); err != nil {
		t.Fatalf("register executor failed: %v", err)
	}

	if err := runAccelCommand(ctx, []string{"run", "numbers", "--mode", "auto", "--precision", "float32"}); err != nil {
		t.Fatalf("runAccelCommand failed: %v", err)
	}

	rendered := output.String()
	if !strings.Contains(rendered, "executed=true") {
		t.Fatalf("expected executed marker in output, got %q", rendered)
	}
	if !strings.Contains(rendered, "var=numbers") {
		t.Fatalf("expected variable name in output, got %q", rendered)
	}
	if !strings.Contains(rendered, "executor=cli-fake") {
		t.Fatalf("expected the executor name in output, got %q", rendered)
	}
	if !strings.Contains(rendered, "participants=1") {
		t.Fatalf("this change executes on one device; got %q", rendered)
	}
	if !strings.Contains(rendered, "precision=float32") {
		t.Fatalf("expected the precision in output, got %q", rendered)
	}
	if !strings.Contains(rendered, "numbers=131328") {
		t.Fatalf("expected the computed sum in output, got %q", rendered)
	}
	if !strings.Contains(rendered, "transfer=") {
		t.Fatalf("expected measured cost in output, got %q", rendered)
	}
	if strings.Contains(rendered, "planning_only=true") {
		t.Fatalf("did not expect planning-only marker in execution output, got %q", rendered)
	}
}

func TestRunAccelCommandRunWithoutExecutorReportsFallback(t *testing.T) {
	ctx, output := accelRunContext(t)
	accelpkg.ResetBackendExecutorsForTest()
	t.Cleanup(accelpkg.ResetBackendExecutorsForTest)

	if err := runAccelCommand(ctx, []string{"run", "numbers", "--mode", "auto", "--precision", "float32"}); err != nil {
		t.Fatalf("runAccelCommand failed: %v", err)
	}

	rendered := output.String()
	if !strings.Contains(rendered, "executed=false") {
		t.Fatalf("expected no execution without a registered backend, got %q", rendered)
	}
	if !strings.Contains(rendered, "reason=no-backend-executor") {
		t.Fatalf("expected the missing-backend reason, got %q", rendered)
	}
	if strings.Contains(rendered, "transfer=") {
		t.Fatalf("expected no cost figures when nothing ran, got %q", rendered)
	}
}

func TestRunAccelCommandRunRefusesFloat64WithoutPrecisionOptIn(t *testing.T) {
	ctx, output := accelRunContext(t)
	accelpkg.ResetBackendExecutorsForTest()
	t.Cleanup(accelpkg.ResetBackendExecutorsForTest)
	if err := accelpkg.RegisterBackendExecutor(accelpkg.BackendCUDA, cliSumExecutor{}); err != nil {
		t.Fatalf("register executor failed: %v", err)
	}
	// Large enough to clear the planner's profitability floor, so the refusal
	// under test is the precision one and not the size one.
	scores := make([]any, 512)
	for i := range scores {
		scores[i] = float64(i) + 0.5
	}
	ctx.Vars["scores"] = insyra.NewDataList(scores...).SetName("scores")

	if err := runAccelCommand(ctx, []string{"run", "scores", "--mode", "auto"}); err != nil {
		t.Fatalf("runAccelCommand failed: %v", err)
	}

	rendered := output.String()
	if !strings.Contains(rendered, "reason=precision-not-accepted") {
		t.Fatalf("expected a float64 column to be refused without --precision float32, got %q", rendered)
	}
}

func TestRunAccelCommandRejectsInvalidPrecision(t *testing.T) {
	ctx, _ := accelRunContext(t)
	err := runAccelCommand(ctx, []string{"run", "numbers", "--precision", "float16"})
	if err == nil {
		t.Fatal("expected an unknown precision to be rejected")
	}
	if !strings.Contains(err.Error(), "invalid accel precision") {
		t.Fatalf("expected a precision error, got %v", err)
	}
}

func TestRunAccelCommandRunRequiresVariableName(t *testing.T) {
	setupCommandHome(t)

	output := &bytes.Buffer{}
	ctx := &ExecContext{Vars: map[string]any{}, Output: output}

	err := runAccelCommand(ctx, []string{"run", "--mode", "auto"})
	if err == nil {
		t.Fatal("expected accel run without variable name to fail")
	}
}

func TestAccelCobraCommandAcceptsModeFlag(t *testing.T) {
	setupCommandHome(t)
	t.Setenv("INSYRA_ACCEL_STUB_WEBGPU", "1")

	output := &bytes.Buffer{}
	ctx := &ExecContext{Vars: map[string]any{}, Output: output}
	commands := BuildCobraCommands(ctx)

	var accelCmd any
	for _, cmd := range commands {
		if cmd.Name() == "accel" {
			accelCmd = cmd
			break
		}
	}
	if accelCmd == nil {
		t.Fatal("expected accel cobra command to be registered")
	}

	command := accelCmd.(interface {
		SetArgs([]string)
		Execute() error
	})
	command.SetArgs([]string{"devices", "--mode", "auto"})
	if err := command.Execute(); err != nil {
		t.Fatalf("expected cobra accel command to accept --mode, got %v", err)
	}
	if !strings.Contains(output.String(), "webgpu:stub:0") {
		t.Fatalf("expected cobra accel command to render devices, got %q", output.String())
	}
}
