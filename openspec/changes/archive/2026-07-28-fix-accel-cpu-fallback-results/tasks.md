# Tasks

## 1. Perform the fallback
- [x] 1.1 Add a predicate that separates device-side fallback reasons, where the CPU can still answer, from request-side ones, where it must not
- [x] 1.2 Compute distances on the CPU and return them when `ExecuteDistances` does not reach a device for a device-side reason
- [x] 1.3 Compute nearest query points on the CPU and return them when `ExecuteNearestQuery` does not reach a device for a device-side reason
- [x] 1.4 Apply the same path when a device is reached but fails, so an aborted execution still answers
- [x] 1.5 Leave strict GPU mode returning an error

## 2. Verify
- [x] 2.1 Test that both operations return the correct values on a session with no discovered device, with `Accelerated` false and the reason preserved
- [x] 2.2 Test that the values returned without a device are identical to the values returned with one
- [x] 2.3 Test that a request refused for precision or dtype still returns no result
- [x] 2.4 Test that a backend whose execution fails still yields the CPU answer, with the failure named
- [x] 2.5 `go test ./accel/...` passes with the device enabled and with `INSYRA_ACCEL_DISABLE_WGPU=1`

## 3. Record
- [x] 3.1 Note in `Docs/accel.md` that a result is returned with or without a device, and that `Accelerated` reports where it ran rather than whether it worked
- [x] 3.2 Changelog entry in `CHANGELOG.md` and `CHANGELOG_TW.md`
