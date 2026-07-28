package accel

import (
	"context"
	"fmt"
)

// DistanceResult carries the squared distances plus the same execution
// reporting every accel operation produces.
type DistanceResult struct {
	ExecutionResult
	// Distances is query-major: entry q*Rows+r is the squared distance from
	// row r to query q.
	Distances []float32
	Rows      int
	Queries   int
}

// ExecuteDistances computes the squared Euclidean distance from every row of the
// dataset to each query point.
//
// Every query must carry one value per column, in the dataset's column order.
// Falling back to the CPU produces the same numbers: the reference is what the
// device result is verified against, bit for bit.
func (s *Session) ExecuteDistances(dataset *Dataset, queries [][]float32, workload WorkloadEstimate) (DistanceResult, error) {
	if s == nil {
		return DistanceResult{}, fmt.Errorf("accel: nil session")
	}
	if dataset == nil {
		return DistanceResult{}, fmt.Errorf("accel: nil dataset")
	}
	if len(queries) == 0 {
		return DistanceResult{}, fmt.Errorf("accel: squared distance needs at least one query point")
	}
	if len(dataset.Buffers) == 0 {
		return DistanceResult{}, fmt.Errorf("accel: squared distance needs at least one column")
	}
	for i, query := range queries {
		if len(query) != len(dataset.Buffers) {
			return DistanceResult{}, fmt.Errorf(
				"accel: query %d has %d dimensions, dataset has %d columns", i, len(query), len(dataset.Buffers))
		}
	}

	workload.Op = OpSquaredDistance
	if workload.Class == "" {
		workload.Class = WorkloadClassColumnar
	}
	if workload.Precision == "" {
		workload.Precision = PrecisionExact
	}
	if workload.Rows <= 0 {
		workload.Rows = dataset.Rows
	}
	if workload.Bytes == 0 {
		workload.Bytes = estimateDatasetResidentBytes(dataset)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	plan := s.planShardableWorkloadLocked(workload)
	result := DistanceResult{
		ExecutionResult: ExecutionResult{
			Accelerated:    plan.Accelerated,
			FallbackReason: plan.FallbackReason,
			MergePolicy:    plan.MergePolicy,
			ExecutorKind:   ExecutorKindUnknown,
			DeviceIDs:      append([]string(nil), plan.DeviceIDs...),
			Op:             OpSquaredDistance,
			Precision:      workload.Precision,
		},
		Queries: len(queries),
	}

	columns, rows, reason := distanceColumns(dataset, workload.Precision)
	if reason != FallbackReasonNone {
		result.Accelerated = false
		result.FallbackReason = reason
		exec, err := s.finishExecution(result.ExecutionResult, fmt.Errorf(
			"accel: dataset is not eligible for device execution (%s)", reason))
		result.ExecutionResult = exec
		return result, err
	}
	result.Rows = rows

	if !plan.Accelerated {
		exec, err := s.finishExecution(result.ExecutionResult, nil)
		result.ExecutionResult = exec
		return result, err
	}

	executor, ok := lookupBackendExecutor(plan.Backend)
	if !ok {
		return s.abortDistance(result, FallbackReasonNoBackendExecutor,
			fmt.Errorf("accel: no execution backend registered for %q", plan.Backend))
	}
	device, ok := s.executionDevice(plan)
	if !ok {
		return s.abortDistance(result, FallbackReasonNoAccelerator,
			fmt.Errorf("accel: plan named no usable device"))
	}
	result.Executor = executor.Name()
	result.ExecutorKind = ExecutorKindRegistered
	result.DeviceIDs = []string{device.ID}
	result.Assignments = assignmentsForDevice(plan, device.ID)

	response, err := executor.Execute(context.Background(), ExecuteRequest{
		Op:        OpSquaredDistance,
		Device:    device,
		Columns:   columns,
		Queries:   queries,
		Precision: workload.Precision,
	})
	if err != nil {
		return s.abortDistance(result, fallbackReasonForExecError(err), err)
	}
	if len(response.Distances) != len(queries)*rows {
		return s.abortDistance(result, FallbackReasonExecutionFailed, fmt.Errorf(
			"accel: backend returned %d distances, expected %d", len(response.Distances), len(queries)*rows))
	}

	result.Distances = response.Distances
	result.Transfer = response.Transfer
	result.Dispatch = response.Dispatch
	result.Readback = response.Readback
	result.BytesUploaded = response.BytesUploaded

	s.ensureDatasetCached(dataset)
	s.applyDeviceResidency(dataset, device.ID)
	exec, err := s.finishExecution(result.ExecutionResult, nil)
	result.ExecutionResult = exec
	return result, err
}

// SquaredDistancesCPU computes the same distances on the CPU. It is the
// reference the device result is checked against, and the path callers land on
// when no device is available.
func SquaredDistancesCPU(dataset *Dataset, queries [][]float32) ([]float32, int, error) {
	if dataset == nil {
		return nil, 0, fmt.Errorf("accel: nil dataset")
	}
	columns, rows, reason := distanceColumns(dataset, PrecisionFloat32)
	if reason != FallbackReasonNone {
		return nil, 0, fmt.Errorf("accel: dataset is not eligible (%s)", reason)
	}
	values := make([][]float32, len(columns))
	for i, column := range columns {
		values[i] = column.Values
	}
	return squaredDistancesReference(values, queries, rows), rows, nil
}

// distanceColumns narrows every column and checks they share a row count.
// Precision is forced to float32 here because a distance kernel has no exact
// path — the caller's opt-in is still what decides whether the device runs.
func distanceColumns(dataset *Dataset, precision Precision) ([]ExecuteColumn, int, FallbackReason) {
	if precision != PrecisionFloat32 {
		return nil, 0, FallbackReasonPrecisionNotAccepted
	}
	columns := make([]ExecuteColumn, 0, len(dataset.Buffers))
	rows := -1
	for _, buffer := range dataset.Buffers {
		values, _, reason := deviceValues(buffer, precision)
		if reason != FallbackReasonNone {
			return nil, 0, reason
		}
		if rows == -1 {
			rows = len(values)
		} else if len(values) != rows {
			return nil, 0, FallbackReasonWorkloadUnsupported
		}
		columns = append(columns, ExecuteColumn{Name: buffer.Name, Values: values})
	}
	if rows <= 0 {
		return nil, 0, FallbackReasonWorkloadUnsupported
	}
	return columns, rows, FallbackReasonNone
}

func (s *Session) abortDistance(result DistanceResult, reason FallbackReason, err error) (DistanceResult, error) {
	result.Accelerated = false
	result.FallbackReason = reason
	result.Distances = nil
	result.Transfer, result.Dispatch, result.Readback, result.BytesUploaded = 0, 0, 0, 0
	exec, ferr := s.finishExecution(result.ExecutionResult, err)
	result.ExecutionResult = exec
	return result, ferr
}

// NearestQueryResult carries the closest query point per row plus the usual
// execution reporting.
type NearestQueryResult struct {
	ExecutionResult
	// Index[r] is the closest query point to row r; Distance[r] is how far.
	Index    []uint32
	Distance []float32
	Rows     int
	Queries  int
}

// ExecuteNearestQuery reports, for every row, which query point is closest and
// its squared distance. Ties go to the lowest query index.
//
// This is OpSquaredDistance with the minimum taken on the device. The full
// matrix never crosses the bus, which is what makes it worth running there.
func (s *Session) ExecuteNearestQuery(dataset *Dataset, queries [][]float32, workload WorkloadEstimate) (NearestQueryResult, error) {
	if s == nil {
		return NearestQueryResult{}, fmt.Errorf("accel: nil session")
	}
	if dataset == nil {
		return NearestQueryResult{}, fmt.Errorf("accel: nil dataset")
	}
	if len(queries) == 0 {
		return NearestQueryResult{}, fmt.Errorf("accel: nearest query needs at least one query point")
	}
	if len(dataset.Buffers) == 0 {
		return NearestQueryResult{}, fmt.Errorf("accel: nearest query needs at least one column")
	}
	for i, query := range queries {
		if len(query) != len(dataset.Buffers) {
			return NearestQueryResult{}, fmt.Errorf(
				"accel: query %d has %d dimensions, dataset has %d columns", i, len(query), len(dataset.Buffers))
		}
	}

	workload.Op = OpNearestQuery
	if workload.Class == "" {
		workload.Class = WorkloadClassColumnar
	}
	if workload.Precision == "" {
		workload.Precision = PrecisionExact
	}
	if workload.Rows <= 0 {
		workload.Rows = dataset.Rows
	}
	if workload.Bytes == 0 {
		workload.Bytes = estimateDatasetResidentBytes(dataset)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	plan := s.planShardableWorkloadLocked(workload)
	result := NearestQueryResult{
		ExecutionResult: ExecutionResult{
			Accelerated:    plan.Accelerated,
			FallbackReason: plan.FallbackReason,
			MergePolicy:    plan.MergePolicy,
			ExecutorKind:   ExecutorKindUnknown,
			DeviceIDs:      append([]string(nil), plan.DeviceIDs...),
			Op:             OpNearestQuery,
			Precision:      workload.Precision,
		},
		Queries: len(queries),
	}

	columns, rows, reason := distanceColumns(dataset, workload.Precision)
	if reason != FallbackReasonNone {
		result.Accelerated = false
		result.FallbackReason = reason
		exec, err := s.finishExecution(result.ExecutionResult, fmt.Errorf(
			"accel: dataset is not eligible for device execution (%s)", reason))
		result.ExecutionResult = exec
		return result, err
	}
	result.Rows = rows

	if !plan.Accelerated {
		exec, err := s.finishExecution(result.ExecutionResult, nil)
		result.ExecutionResult = exec
		return result, err
	}

	executor, ok := lookupBackendExecutor(plan.Backend)
	if !ok {
		return s.abortNearest(result, FallbackReasonNoBackendExecutor,
			fmt.Errorf("accel: no execution backend registered for %q", plan.Backend))
	}
	device, ok := s.executionDevice(plan)
	if !ok {
		return s.abortNearest(result, FallbackReasonNoAccelerator,
			fmt.Errorf("accel: plan named no usable device"))
	}
	result.Executor = executor.Name()
	result.ExecutorKind = ExecutorKindRegistered
	result.DeviceIDs = []string{device.ID}
	result.Assignments = assignmentsForDevice(plan, device.ID)

	response, err := executor.Execute(context.Background(), ExecuteRequest{
		Op:        OpNearestQuery,
		Device:    device,
		Columns:   columns,
		Queries:   queries,
		Precision: workload.Precision,
	})
	if err != nil {
		return s.abortNearest(result, fallbackReasonForExecError(err), err)
	}
	if len(response.NearestIndex) != rows || len(response.Distances) != rows {
		return s.abortNearest(result, FallbackReasonExecutionFailed, fmt.Errorf(
			"accel: backend returned %d indices and %d distances, expected %d of each",
			len(response.NearestIndex), len(response.Distances), rows))
	}

	result.Index = response.NearestIndex
	result.Distance = response.Distances
	result.Transfer = response.Transfer
	result.Dispatch = response.Dispatch
	result.Readback = response.Readback
	result.BytesUploaded = response.BytesUploaded

	s.ensureDatasetCached(dataset)
	s.applyDeviceResidency(dataset, device.ID)
	exec, err := s.finishExecution(result.ExecutionResult, nil)
	result.ExecutionResult = exec
	return result, err
}

// NearestQueryCPU is the reference the device result is checked against, and
// the path callers land on with no device.
func NearestQueryCPU(dataset *Dataset, queries [][]float32) ([]uint32, []float32, int, error) {
	if dataset == nil {
		return nil, nil, 0, fmt.Errorf("accel: nil dataset")
	}
	if len(queries) == 0 {
		return nil, nil, 0, fmt.Errorf("accel: nearest query needs at least one query point")
	}
	columns, rows, reason := distanceColumns(dataset, PrecisionFloat32)
	if reason != FallbackReasonNone {
		return nil, nil, 0, fmt.Errorf("accel: dataset is not eligible (%s)", reason)
	}
	values := make([][]float32, len(columns))
	for i, column := range columns {
		values[i] = column.Values
	}
	indices, distances := nearestQueryReference(values, queries, rows)
	return indices, distances, rows, nil
}

func (s *Session) abortNearest(result NearestQueryResult, reason FallbackReason, err error) (NearestQueryResult, error) {
	result.Accelerated = false
	result.FallbackReason = reason
	result.Index, result.Distance = nil, nil
	result.Transfer, result.Dispatch, result.Readback, result.BytesUploaded = 0, 0, 0, 0
	exec, ferr := s.finishExecution(result.ExecutionResult, err)
	result.ExecutionResult = exec
	return result, ferr
}
