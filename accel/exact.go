package accel

import (
	"context"
	"fmt"
	"math"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/HazelnutParadise/insyra/accel/internal/wgpu"
)

// ExactNearestResult carries the nearest query points per row, computed to the
// same answer a pure float64 pass would give.
type ExactNearestResult struct {
	ExecutionResult
	// Index and Distance are row-major and hold M entries per row, nearest
	// first: entry r*M+j is row r's j-th nearest query point.
	Index    []uint32
	Distance []float64
	Rows     int
	Queries  int
	M        int
	// Rechecked counts the rows whose device shortlist could not be trusted and
	// were recomputed against every query point. It is the number to watch on
	// real data: a shortlist too narrow for the data shows up here as a rising
	// count long before it shows up as a slowdown.
	Rechecked int
}

// float32Eps is the spacing between consecutive float32 values near 1.
const float32Eps = 1.1920928955078125e-07

// minWorkPerRowForDevice is how much arithmetic a single row must carry before
// narrowing on a device beats doing the whole thing on the host's cores.
//
// It reads dimensions times query points — the distance evaluations one row
// needs — rather than the query count alone, because the two trade off directly.
// Measured on an Apple M3 against a host using all eight cores, over 96 shapes:
// four dimensions need 512 query points before the device is worth it, sixteen
// need 64 to 128, sixty-four need 32. Those cluster around two thousand
// evaluations per row, and no other threshold misclassifies fewer of the 96 —
// this one gets 88 of them right, against 84 at 4096 and 78 at 8192.
//
// The earlier value read the query count alone and was calibrated against a
// single-threaded host, so it was wrong on both counts.
//
// This is one host's number, and a machine with a discrete GPU would put it
// somewhere else entirely. It belongs in a calibrated dispatcher eventually;
// until then it is a value with a measurement attached rather than a guess. It
// is a var so the sweep that calibrates it can look below the floor.
var minWorkPerRowForDevice = 2048

// ExecuteNearestExact reports the M nearest query points for every row, and
// returns what a float64 computation over every query point would return.
//
// The device is used to narrow the field, never to decide. It ranks the query
// points in single precision and returns a shortlist per row; the host then
// recomputes that shortlist in float64 and picks from it. When the shortlist's
// cut is too close to call in single precision, that row is recomputed against
// every query point instead. So the answer does not depend on the device being
// right, only on it being close, and it does not depend on there being a device
// at all.
//
// Unlike the single-precision operations, this one needs no precision opt-in.
// Narrowing happens inside, where it cannot reach the result.
func (s *Session) ExecuteNearestExact(dataset *Dataset, queries [][]float64, m int, workload WorkloadEstimate) (ExactNearestResult, error) {
	if s == nil {
		return ExactNearestResult{}, fmt.Errorf("accel: nil session")
	}
	if dataset == nil {
		return ExactNearestResult{}, fmt.Errorf("accel: nil dataset")
	}
	if len(dataset.Buffers) == 0 {
		return ExactNearestResult{}, fmt.Errorf("accel: exact nearest needs at least one column")
	}
	if len(queries) == 0 {
		return ExactNearestResult{}, fmt.Errorf("accel: exact nearest needs at least one query point")
	}
	if m < 1 {
		return ExactNearestResult{}, fmt.Errorf("accel: exact nearest needs m >= 1, got %d", m)
	}
	if m > len(queries) {
		return ExactNearestResult{}, fmt.Errorf(
			"accel: asked for the %d nearest of only %d query points", m, len(queries))
	}
	for i, query := range queries {
		if len(query) != len(dataset.Buffers) {
			return ExactNearestResult{}, fmt.Errorf(
				"accel: query %d has %d dimensions, dataset has %d columns", i, len(query), len(dataset.Buffers))
		}
	}

	// The device leg always runs in single precision. That is not the caller's
	// trade to make, because it does not reach the caller's numbers.
	workload.Op = OpNearestShortlist
	workload.Precision = PrecisionFloat32
	if workload.Class == "" {
		workload.Class = WorkloadClassColumnar
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
	result := ExactNearestResult{
		ExecutionResult: ExecutionResult{
			Accelerated:    plan.Accelerated,
			FallbackReason: plan.FallbackReason,
			MergePolicy:    plan.MergePolicy,
			ExecutorKind:   ExecutorKindUnknown,
			DeviceIDs:      append([]string(nil), plan.DeviceIDs...),
			Op:             OpNearestShortlist,
			// The answer is exact whichever path produced it.
			Precision: PrecisionExact,
		},
		Queries: len(queries),
		M:       m,
	}

	host, rows, reason := hostColumns(dataset)
	if reason != FallbackReasonNone {
		result.Accelerated = false
		result.FallbackReason = reason
		exec, err := s.finishExecution(result.ExecutionResult, fmt.Errorf(
			"accel: dataset is not eligible (%s)", reason))
		result.ExecutionResult = exec
		return result, err
	}
	result.Rows = rows

	finishOnHost := func(reason FallbackReason, cause error) (ExactNearestResult, error) {
		result.Accelerated = false
		result.FallbackReason = reason
		result.Transfer, result.Dispatch, result.Readback, result.BytesUploaded = 0, 0, 0, 0
		result.Index, result.Distance = exactNearestAll(host, queries, rows, m)
		result.Rechecked = rows
		exec, err := s.finishExecution(result.ExecutionResult, cause)
		result.ExecutionResult = exec
		return result, err
	}

	if !plan.Accelerated {
		return finishOnHost(result.FallbackReason, nil)
	}
	if len(queries)*len(dataset.Buffers) < minWorkPerRowForDevice {
		return finishOnHost(FallbackReasonWorkloadNotProfitable, nil)
	}
	executor, ok := lookupBackendExecutor(plan.Backend)
	if !ok {
		return finishOnHost(FallbackReasonNoBackendExecutor,
			fmt.Errorf("accel: no execution backend registered for %q", plan.Backend))
	}
	device, ok := s.executionDevice(plan)
	if !ok {
		return finishOnHost(FallbackReasonNoAccelerator, fmt.Errorf("accel: plan named no usable device"))
	}

	// Two spare candidates give the float64 pass room to reorder without
	// falling off the end of the list. A shortlist that cannot even hold m is
	// no shortlist, so the device is skipped rather than clamped — clamping
	// would leave the decision reading past the end of its own scratch.
	if m > wgpu.MaxShortlist-1 {
		return finishOnHost(FallbackReasonWorkloadUnsupported, nil)
	}
	k := m + 2
	if k > len(queries) {
		k = len(queries)
	}
	if k > wgpu.MaxShortlist {
		k = wgpu.MaxShortlist
	}

	columns, _, creason := distanceColumns(dataset, PrecisionFloat32)
	if creason != FallbackReasonNone {
		return finishOnHost(creason, fmt.Errorf("accel: dataset is not eligible (%s)", creason))
	}
	narrowed := narrowQueries(queries)

	result.Executor = executor.Name()
	result.ExecutorKind = ExecutorKindRegistered
	result.DeviceIDs = []string{device.ID}
	result.Assignments = assignmentsForDevice(plan, device.ID)

	response, err := executor.Execute(context.Background(), ExecuteRequest{
		Op:        OpNearestShortlist,
		Device:    device,
		Columns:   columns,
		Queries:   narrowed,
		Precision: PrecisionFloat32,
		Shortlist: k,
	})
	if err != nil {
		return finishOnHost(fallbackReasonForExecError(err), err)
	}
	if len(response.ShortlistIndex) != rows*k ||
		len(response.ShortlistDistance) != rows*k ||
		len(response.ShortlistBoundary) != rows {
		return finishOnHost(FallbackReasonExecutionFailed, fmt.Errorf(
			"accel: backend returned a %d/%d/%d shortlist, expected %d/%d/%d",
			len(response.ShortlistIndex), len(response.ShortlistDistance), len(response.ShortlistBoundary),
			rows*k, rows*k, rows))
	}

	index, distance, rechecked := decideFromShortlist(
		host, queries, rows, m, k,
		response.ShortlistIndex, response.ShortlistDistance, response.ShortlistBoundary)
	result.Index = index
	result.Distance = distance
	result.Rechecked = rechecked
	result.Transfer = response.Transfer
	result.Dispatch = response.Dispatch
	result.Readback = response.Readback
	result.BytesUploaded = response.BytesUploaded

	s.ensureDatasetCached(dataset)
	s.applyDeviceResidency(dataset, device.ID)
	exec, ferr := s.finishExecution(result.ExecutionResult, nil)
	result.ExecutionResult = exec
	return result, ferr
}

// NearestExactCPU is the reference: every distance, every query point, float64
// throughout. It is what the accelerated path is asserted equal to.
func NearestExactCPU(dataset *Dataset, queries [][]float64, m int) ([]uint32, []float64, int, error) {
	if dataset == nil {
		return nil, nil, 0, fmt.Errorf("accel: nil dataset")
	}
	if len(queries) == 0 || m < 1 || m > len(queries) {
		return nil, nil, 0, fmt.Errorf("accel: exact nearest needs 1 <= m <= %d", len(queries))
	}
	host, rows, reason := hostColumns(dataset)
	if reason != FallbackReasonNone {
		return nil, nil, 0, fmt.Errorf("accel: dataset is not eligible (%s)", reason)
	}
	index, distance := exactNearestAll(host, queries, rows, m)
	return index, distance, rows, nil
}

// hostColumns pulls the dataset's values at their own precision. This is the
// source the final arithmetic reads, so it must never be the narrowed copy.
func hostColumns(dataset *Dataset) ([][]float64, int, FallbackReason) {
	columns := make([][]float64, 0, len(dataset.Buffers))
	rows := -1
	for _, buffer := range dataset.Buffers {
		var values []float64
		switch typed := buffer.Values.(type) {
		case []float64:
			values = typed
		case []int64:
			values = make([]float64, len(typed))
			for i, v := range typed {
				values[i] = float64(v)
			}
		default:
			return nil, 0, FallbackReasonDTypeNotEligible
		}
		if rows == -1 {
			rows = len(values)
		} else if len(values) != rows {
			return nil, 0, FallbackReasonWorkloadUnsupported
		}
		columns = append(columns, values)
	}
	if rows <= 0 {
		return nil, 0, FallbackReasonWorkloadUnsupported
	}
	return columns, rows, FallbackReasonNone
}

func narrowQueries(queries [][]float64) [][]float32 {
	out := make([][]float32, len(queries))
	for i, query := range queries {
		point := make([]float32, len(query))
		for j, v := range query {
			point[j] = float32(v)
		}
		out[i] = point
	}
	return out
}

// parallelRowThreshold is how much work a pass must carry before splitting it
// across cores pays for the goroutines. It matches the threshold the clustering
// package uses for the same shape of loop (stats/internal/clustering).
const parallelRowThreshold = 50_000

// runRows splits a row range across the available cores when there is enough
// work to be worth it, and runs it inline when there is not.
//
// body owns a whole range rather than one row so that each worker can allocate
// its scratch once. Ranges never overlap, so writes into a shared output slice
// need no synchronisation.
func runRows(rows, work int, body func(lo, hi int)) {
	workers := runtime.GOMAXPROCS(0)
	if workers <= 1 || rows < 2*workers || work < parallelRowThreshold {
		body(0, rows)
		return
	}
	chunk := (rows + workers - 1) / workers
	var wg sync.WaitGroup
	for lo := 0; lo < rows; lo += chunk {
		hi := lo + chunk
		if hi > rows {
			hi = rows
		}
		wg.Add(1)
		go func(lo, hi int) {
			defer wg.Done()
			body(lo, hi)
		}(lo, hi)
	}
	wg.Wait()
}

// candidate is one query point's exact distance to one row.
type candidate struct {
	dist float64
	idx  uint32
}

// byDistanceThenIndex orders candidates so that an exact tie resolves to the
// lowest query index, matching every other operation in the package.
func byDistanceThenIndex(c []candidate) {
	sort.Slice(c, func(i, j int) bool {
		if c[i].dist != c[j].dist {
			return c[i].dist < c[j].dist
		}
		return c[i].idx < c[j].idx
	})
}

// insertCandidate keeps the best `filled` entries of `best` in order, and
// returns the new count. Insertion is strict and the bubble-up stops on
// equality, so an exact tie keeps the lower query index — the same rule the
// kernel follows.
func insertCandidate(best []candidate, filled int, c candidate) int {
	pos := filled
	if filled == len(best) {
		if !(c.dist < best[filled-1].dist) {
			return filled
		}
		pos = filled - 1
	} else {
		filled++
	}
	best[pos] = c
	for p := pos; p > 0 && best[p-1].dist > best[p].dist; p-- {
		best[p-1], best[p] = best[p], best[p-1]
	}
	return filled
}

// selectNearest fills best[:m] with the m nearest query points to one row.
//
// It selects rather than sorts. m is small and the query count is not, so
// paying O(queries·m) beats sorting every candidate — and this is the path a
// machine with no device takes for every row, which is the one place a lazy
// reference would be felt.
func selectNearest(row []float64, queries [][]float64, m int, best []candidate) {
	filled := 0
	for q, query := range queries {
		filled = insertCandidate(best[:m], filled, candidate{squaredDistanceRow(row, query), uint32(q)})
	}
}

func squaredDistance64(columns [][]float64, queries [][]float64, row, query int) float64 {
	var acc float64
	for c := range columns {
		diff := columns[c][row] - queries[query][c]
		acc += diff * diff
	}
	return acc
}

// gatherRow copies one row out of the column-major buffers.
//
// Reading a row straight from the columns touches one cache line per column,
// each a whole column apart. Doing that once per row instead of once per query
// point turns the inner loop from scattered reads into sequential ones, which is
// most of the difference on a wide dataset.
func gatherRow(columns [][]float64, row int, out []float64) {
	for c := range columns {
		out[c] = columns[c][row]
	}
}

func squaredDistanceRow(row []float64, query []float64) float64 {
	var acc float64
	for c, v := range row {
		diff := v - query[c]
		acc += diff * diff
	}
	return acc
}

// exactNearestAll computes every distance in float64. It is both the reference
// and the path taken when no device narrowed the field.
func exactNearestAll(columns [][]float64, queries [][]float64, rows, m int) ([]uint32, []float64) {
	index := make([]uint32, rows*m)
	distance := make([]float64, rows*m)
	runRows(rows, rows*len(queries)*len(columns), func(lo, hi int) {
		best := make([]candidate, m)
		row := make([]float64, len(columns))
		for r := lo; r < hi; r++ {
			gatherRow(columns, r, row)
			selectNearest(row, queries, m, best)
			for j := 0; j < m; j++ {
				index[r*m+j] = best[j].idx
				distance[r*m+j] = best[j].dist
			}
		}
	})
	return index, distance
}

// decideFromShortlist recomputes the device's candidates in float64 and picks
// from them, falling back to every query point for the rows whose shortlist cut
// is too close to call in single precision.
func decideFromShortlist(
	columns [][]float64, queries [][]float64, rows, m, k int,
	shortIdx []uint32, shortDist []float32, boundary []float32,
) ([]uint32, []float64, int) {
	index := make([]uint32, rows*m)
	distance := make([]float64, rows*m)
	scale := queryScale(queries)

	var rechecked atomic.Int64

	// Verification costs k distances per row, plus every query point again for
	// the rows that cannot be trusted. Estimating it at k is enough to decide
	// whether to split, because a run where most rows are rechecked is heavier
	// still and wants splitting more.
	runRows(rows, rows*k*len(columns), func(lo, hi int) {
		shortlist := make([]candidate, k)
		full := make([]candidate, m)
		row := make([]float64, len(columns))
		local := 0
		for r := lo; r < hi; r++ {
			gatherRow(columns, r, row)
			trustworthy := true
			for j := 0; j < k; j++ {
				q := shortIdx[r*k+j]
				if int(q) >= len(queries) {
					// The device reported an empty slot, which can only happen
					// when it had fewer query points than list slots. Treat it
					// as a row worth redoing rather than reading past the end.
					trustworthy = false
					break
				}
				shortlist[j] = candidate{squaredDistanceRow(row, queries[q]), q}
			}
			if trustworthy && !finiteShortlist(shortDist, boundary, r, k) {
				// Single precision overflowed, so the device compared infinities
				// and its ordering means nothing. The boundary would be +Inf
				// too, which passes the gap test for the wrong reason.
				trustworthy = false
			}
			if trustworthy {
				byDistanceThenIndex(shortlist)
				// Anything the device left out is at least boundary minus the
				// worst single-precision error this row can carry. If that is
				// still above the last candidate being returned, nothing left
				// out can beat it.
				if k < len(queries) &&
					float64(boundary[r])-rowErrorBound(row, scale) <= shortlist[m-1].dist {
					trustworthy = false
				}
			}
			if !trustworthy {
				selectNearest(row, queries, m, full)
				copy(shortlist[:m], full[:m])
				local++
			}
			for j := 0; j < m; j++ {
				index[r*m+j] = shortlist[j].idx
				distance[r*m+j] = shortlist[j].dist
			}
		}
		rechecked.Add(int64(local))
	})
	return index, distance, int(rechecked.Load())
}

// queryScale returns, per column, the largest magnitude any query point holds
// there. It is half of the row error bound and does not depend on the row, so
// it is computed once.
func queryScale(queries [][]float64) []float64 {
	if len(queries) == 0 {
		return nil
	}
	scale := make([]float64, len(queries[0]))
	for _, query := range queries {
		for c, v := range query {
			if a := math.Abs(v); a > scale[c] {
				scale[c] = a
			}
		}
	}
	return scale
}

// rowErrorBound bounds how far a single-precision squared distance for this row
// can sit from the float64 one, for any query point.
//
// Each input is rounded once, so a difference carries an absolute error of at
// most u(|x|+|q|); squaring doubles the relative error, and accumulating d terms
// adds at most (d+1)u of the total. Bounding |q| by the largest magnitude any
// query holds in that column makes the result independent of which query point
// was involved, which is what lets one number cover the candidate the device
// discarded and never reported.
//
// The bound ignores fused multiply-add, which only ever reduces the error, so it
// stays conservative on hardware that fuses.
//
// Two details the obvious derivation gets wrong. The difference costs two
// roundings and not one — rounding each input, then rounding the subtraction —
// so the factor is d+4 rather than d+3. And a relative bound says nothing once
// a squared term underflows below the smallest normal float32, where the error
// stops scaling with the value; the absolute term covers that floor.
func rowErrorBound(row []float64, scale []float64) float64 {
	const (
		u            = float32Eps / 2
		smallestNorm = 1.1754943508222875e-38
	)
	var span float64
	for c, v := range row {
		reach := math.Abs(v) + scale[c]
		span += reach * reach
	}
	terms := float64(len(row) + 4)
	return terms*u*span + terms*smallestNorm
}

// finiteShortlist reports whether the device's single-precision distances for
// one row can be ranked at all. Overflow turns them into infinities that compare
// equal, so an ordering over them carries no information.
func finiteShortlist(shortDist []float32, boundary []float32, row, k int) bool {
	if !isFiniteFloat32(boundary[row]) {
		return false
	}
	if shortDist == nil {
		return true
	}
	for j := 0; j < k; j++ {
		if !isFiniteFloat32(shortDist[row*k+j]) {
			return false
		}
	}
	return true
}

func isFiniteFloat32(v float32) bool {
	return !math.IsInf(float64(v), 0) && !math.IsNaN(float64(v))
}
