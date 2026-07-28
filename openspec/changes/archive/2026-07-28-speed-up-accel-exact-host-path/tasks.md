# Tasks

## 1. Parallelise the host
- [x] 1.1 Add a row-range splitter that runs one goroutine per core above a work threshold and stays on one core below it
- [x] 1.2 Split `exactNearestAll`, giving each worker its own scratch so nothing is shared
- [x] 1.3 Split the shortlist verification, and accumulate the recheck count without a race
- [x] 1.4 Confirm the threshold against measurement rather than assuming it

## 2. Stop striding across the shortlist
- [x] 2.1 Have the kernel write the shortlist row-major so one row's candidates are contiguous
- [x] 2.2 Update the CPU-side shortlist producer and the reader to match
- [x] 2.3 Measure what the layout change is worth on its own

## 3. Recalibrate
- [x] 3.1 Replace the query-count floor with a work-per-row threshold
- [x] 3.2 Re-run the shape map against the parallel host and set the threshold from it
- [x] 3.3 Record the new map and the corrected speedups in `delivery-plan.md`

## 4. Verify
- [x] 4.1 Results still equal the reference, with and without a device, on random and adversarial data
- [x] 4.2 The recheck count is unchanged by parallelism
- [x] 4.3 `go test ./accel/... -race` where the device is not reached, to prove the split is clean
- [x] 4.4 `go test ./accel/...` passes with the device enabled and with `INSYRA_ACCEL_DISABLE_WGPU=1`

## 5. Record
- [x] 5.1 Correct the speedup figures in `Docs/accel.md`, `CHANGELOG.md` and `CHANGELOG_TW.md`
- [x] 5.2 Changelog entry for the parallel host path
