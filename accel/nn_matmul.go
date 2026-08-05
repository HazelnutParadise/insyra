package accel

import (
	"context"
	"errors"
	"fmt"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/accel/internal/wgpu"
)

// DeviceMatMul computes one 2-D float32 matrix product on the discovered
// device. A device failure is reported through Default and returned as an
// error so the caller can preserve its exact CPU fallback.
func DeviceMatMul(a []float32, aRows, aCols int, b []float32, bRows, bCols int) ([]float32, error) {
	if bRows != aCols {
		return nil, fmt.Errorf("dl matmul shapes [%d,%d]x[%d,%d] are incompatible", aRows, aCols, bRows, bCols)
	}
	if !insyra.Config.GetAccelerationEnabled() {
		return nil, fmt.Errorf("dl matmul acceleration is disabled by Config")
	}

	session := Default()
	if wgpuDisabled() {
		return nil, recordDeviceMatMulFallback(session, FallbackReasonNoAccelerator, aRows,
			fmt.Errorf("dl matmul device backend is disabled"))
	}
	if len(session.Devices()) == 0 {
		reason := session.Report().FallbackReason
		if reason == FallbackReasonNone {
			reason = FallbackReasonNoAccelerator
		}
		return nil, recordDeviceMatMulFallback(session, reason, aRows,
			fmt.Errorf("dl matmul has no usable accelerator (%s)", reason))
	}

	result, _, err := wgpu.MatMul(context.Background(), a, b, aRows, aCols, bCols)
	if err != nil {
		reason := deviceMatMulFallbackReason(err)
		return nil, recordDeviceMatMulFallback(session, reason, aRows,
			fmt.Errorf("dl matmul device execution: %w", err))
	}
	recordDeviceMatMulSuccess(session, aRows)
	return result, nil
}

func deviceMatMulFallbackReason(err error) FallbackReason {
	switch {
	case errors.Is(err, wgpu.ErrShaderCompile):
		return FallbackReasonShaderCompileFailed
	case errors.Is(err, wgpu.ErrBufferTooLarge):
		return FallbackReasonBufferTooLarge
	case errors.Is(err, wgpu.ErrReadbackTimeout):
		return FallbackReasonReadbackTimeout
	case errors.Is(err, wgpu.ErrUnavailable):
		return FallbackReasonNoAccelerator
	default:
		return FallbackReasonExecutionFailed
	}
}

func recordDeviceMatMulFallback(session *Session, reason FallbackReason, rows int, cause error) error {
	report := session.Report()
	report.Accelerated = false
	report.FallbackReason = reason
	if report.Metrics == nil {
		report.Metrics = map[string]float64{}
	}
	report.Metrics["execution.fallback"] = 1
	_ = session.RecordReport(report)
	session.logDeviceMatMul(false, reason, rows)
	return cause
}

func recordDeviceMatMulSuccess(session *Session, rows int) {
	report := session.Report()
	report.Accelerated = true
	report.FallbackReason = FallbackReasonNone
	if report.Metrics == nil {
		report.Metrics = map[string]float64{}
	}
	report.Metrics["execution.accelerated"] = 1
	report.Metrics["execution.fallback"] = 0
	_ = session.RecordReport(report)
	session.logDeviceMatMul(true, FallbackReasonNone, rows)
}

func (s *Session) logDeviceMatMul(accelerated bool, reason FallbackReason, rows int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	report := s.reportLocked()
	assignments := make([]ShardAssignment, 0, len(report.SelectedDeviceIDs))
	for _, id := range report.SelectedDeviceIDs {
		assignment := ShardAssignment{
			DeviceID:       id,
			Backend:        report.SelectedBackend,
			Rows:           rows,
			Chunks:         1,
			FallbackReason: reason,
		}
		assignments = append(assignments, assignment)
	}
	s.logExecutionLocked("matmul", ExecutionResult{
		Accelerated:    accelerated,
		FallbackReason: reason,
		Assignments:    assignments,
		DeviceIDs:      append([]string(nil), report.SelectedDeviceIDs...),
		Chunks:         1,
	}, rows)
}
