# Tasks

## 1. Shortlist kernel
- [x] 1.1 Add `OpNearestShortlist` and carry a shortlist width plus the shortlist arrays through `ExecuteRequest`/`ExecuteResponse`
- [x] 1.2 Write the WGSL kernel: keep the `k` smallest distances and their indices in registers, and also the best rejected distance
- [x] 1.3 Split the dispatch the same way the other kernels do, so row counts above the workgroup limit still run
- [x] 1.4 Add the CPU reference for the shortlist, ties going to the lowest query index

## 2. Exact decision on the host
- [x] 2.1 Add `Session.ExecuteNearestExact(dataset, queries, m)` returning `m` indices and `float64` distances per row
- [x] 2.2 Recompute the shortlist in `float64` from the dataset's original values, not from the narrowed copy
- [x] 2.3 Derive the single-precision error bound from the dimension count, conservatively, and recompute a row in full when the boundary falls inside it
- [x] 2.4 Report the number of rows recomputed in full
- [x] 2.5 Reject `m` larger than the query count, and choose the shortlist width from `m`
- [x] 2.6 Add `NearestExactCPU` as the reference and route to it when no device takes part

## 3. Verify
- [x] 3.1 Test that the device result equals the `float64` reference exactly, on random data
- [x] 3.2 Test the same on adversarial data — duplicated rows, exactly equidistant query points, values spanning wide magnitudes — where the shortlist boundary is hit deliberately
- [x] 3.3 Test that the answer is unchanged with the device disabled
- [x] 3.4 Test ties resolve to the lowest query index, and that `m` beyond the query count is rejected
- [x] 3.5 Benchmark against the pure `float64` path and record the numbers in `delivery-plan.md`
- [x] 3.6 `go test ./accel/...` passes with the device enabled and with `INSYRA_ACCEL_DISABLE_WGPU=1`

## 4. Record
- [x] 4.1 Document the operation in `Docs/accel.md`, including what the recheck count means
- [x] 4.2 Changelog entry in `CHANGELOG.md` and `CHANGELOG_TW.md`
