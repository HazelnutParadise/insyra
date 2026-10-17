## 1. Tests first
- [x] 1.1 Add `TestBackendAuthorNamesAreDeprecated` (go/ast over the package) and `TestDeprecatedRegisterDeviceStillRegisters`, and record that the first fails on the old code
- [x] 1.2 Add `internal/acceltest` and move `nn/device_matmul_test.go` onto `acceltest.ResetDefault`

## 2. Implementation
- [x] 2.1 Rename the listed names to unexported equivalents inside `accel` and its tests
- [x] 2.2 Add Deprecated wrappers and type aliases under the old names
- [x] 2.3 Wire `internal/acceltest.ResetDefault` from `accel`'s init

## 3. Docs and bookkeeping
- [x] 3.1 Update `Docs/accel.md` to drop the backend registry and the CPU reference from the user surface
- [x] 3.2 Add CHANGELOG.md / CHANGELOG_TW.md entries under `## Unreleased` → `accel`
- [x] 3.3 Mark AC-1 fixed in `api-review.md`, add a `delivery-status.md` milestone, and add an `AGENTS.md` removal follow-up
- [x] 3.4 gofmt, build, vet, test and golangci-lint
