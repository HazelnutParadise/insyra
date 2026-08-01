# Tasks

## 1. Remove the kernels
- [x] 1.1 Delete `sumWGSL`, `distanceWGSL` and `nearestWGSL` with their pipeline caches and handle fields
- [x] 1.2 Delete `SumColumns`, `SquaredDistances` and `NearestQuery` from the wgpu package
- [x] 1.3 Delete the corresponding backend dispatch arms and adapter functions

## 2. Remove the runtime surface
- [x] 2.1 Delete `OpSum`, `OpSquaredDistance` and `OpNearestQuery`
- [x] 2.2 Delete `ExecuteDistances`, `ExecuteNearestQuery` and their result types
- [x] 2.3 Delete `SquaredDistancesCPU`, `NearestQueryCPU` and the reference file
- [x] 2.4 Delete `ExecuteDataList`, `ExecuteDataTable` and `ExecuteProjectedDataset`
- [x] 2.5 Move anything the exact operation still needs out of the deleted files rather than leaving it stranded
- [x] 2.6 Remove `Reductions` and `Counts` from the execution result if nothing fills them

## 3. Remove the CLI surface
- [x] 3.1 Delete `accel run <var>` and its help text
- [x] 3.2 Leave `accel devices`, `accel cache` and `accel plan`, which inspect rather than execute

## 4. Verify
- [x] 4.1 `go build ./...` and `go vet ./...` clean
- [x] 4.2 `go test ./...` passes, with the device enabled and with `INSYRA_ACCEL_DISABLE_WGPU=1`
- [x] 4.3 The exact nearest operation still matches its reference on a real device
- [x] 4.4 No test was deleted that covered behaviour still present

## 5. Record
- [x] 5.1 Correct the single-core speedup figures in `Docs/accel.md` and both changelogs
- [x] 5.2 Changelog entries for the removals
- [x] 5.3 Note the removal and its reason in `delivery-plan.md`
