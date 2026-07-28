## 1. Implementation
- [x] 1.1 Validate the change with `openspec validate add-accel-default-session --strict`
- [x] 1.2 Write the tests first: same instance twice, discovery runs once, Close is a no-op and leaves the session usable, and a session is returned even when discovery finds nothing
- [x] 1.3 Add `accel/default.go` with `Default()` behind `sync.Once` and `ResetDefaultForTest()`
- [x] 1.4 Make `Close` a no-op on the shared session and keep `Closed()` reporting false for it
- [x] 1.5 Assert that importing the package opens no device until `Default()` is called
- [x] 1.6 Run `go test ./...`, `go test -race ./...`, every accel test individually, and the GPU test with `INSYRA_ACCEL_GPU_TESTS=1`
- [x] 1.7 Update `Docs/accel.md`, CHANGELOG and CHANGELOG_TW, and `delivery-plan.md`
