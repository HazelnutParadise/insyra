// Package dlbridge plugs the accelerator into dl's optional device MatMul
// socket.
//
// Import it for its side effect:
//
//	import _ "github.com/HazelnutParadise/insyra/accel/dlbridge"
//
// With the import, eligible large 2-D float32 MatMuls try the WebGPU device.
// Device errors return control to dl's exact CPU path, while the shared accel
// report records the fallback reason. Without the import, dl carries no accel
// dependency and behaves exactly as before.
package dlbridge

import (
	"context"
	"errors"
	"fmt"

	"github.com/HazelnutParadise/insyra/accel"
	"github.com/HazelnutParadise/insyra/accel/internal/wgpu"
	"github.com/HazelnutParadise/insyra/dl"
)

func init() {
	dl.RegisterDeviceMatMul(matmul)
}

func matmul(a []float32, aRows, aCols int, b []float32, bRows, bCols int) ([]float32, error) {
	if bRows != aCols {
		return nil, fmt.Errorf("dl matmul shapes [%d,%d]x[%d,%d] are incompatible", aRows, aCols, bRows, bCols)
	}
	session := accel.Default()
	if len(session.Devices()) == 0 {
		reason := session.Report().FallbackReason
		if reason == accel.FallbackReasonNone {
			reason = accel.FallbackReasonNoAccelerator
		}
		return nil, recordFallback(session, reason, fmt.Errorf("dl matmul has no usable accelerator (%s)", reason))
	}

	result, _, err := wgpu.MatMul(context.Background(), a, b, aRows, aCols, bCols)
	if err != nil {
		reason := fallbackReason(err)
		return nil, recordFallback(session, reason, fmt.Errorf("dl matmul device execution: %w", err))
	}
	recordSuccess(session)
	return result, nil
}

func fallbackReason(err error) accel.FallbackReason {
	switch {
	case errors.Is(err, wgpu.ErrShaderCompile):
		return accel.FallbackReasonShaderCompileFailed
	case errors.Is(err, wgpu.ErrBufferTooLarge):
		return accel.FallbackReasonBufferTooLarge
	case errors.Is(err, wgpu.ErrReadbackTimeout):
		return accel.FallbackReasonReadbackTimeout
	case errors.Is(err, wgpu.ErrUnavailable):
		return accel.FallbackReasonNoAccelerator
	default:
		return accel.FallbackReasonExecutionFailed
	}
}

func recordFallback(session *accel.Session, reason accel.FallbackReason, cause error) error {
	report := session.Report()
	report.Accelerated = false
	report.FallbackReason = reason
	if report.Metrics == nil {
		report.Metrics = map[string]float64{}
	}
	report.Metrics["execution.fallback"] = 1
	_ = session.RecordReport(report)
	return cause
}

func recordSuccess(session *accel.Session) {
	report := session.Report()
	report.Accelerated = true
	report.FallbackReason = accel.FallbackReasonNone
	if report.Metrics == nil {
		report.Metrics = map[string]float64{}
	}
	report.Metrics["execution.accelerated"] = 1
	report.Metrics["execution.fallback"] = 0
	_ = session.RecordReport(report)
}
