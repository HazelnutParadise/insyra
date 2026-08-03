# Change: Device MatMul is on by default in dl; the bridge package goes away

## Why

The opt-in blank import shipped in `add-dl-device-matmul` was justified by a
dependency-cycle argument that only holds for the bridge package itself —
`dl → accel → insyra` is acyclic. The real cost of a direct dependency is
compile weight, which ENG.md already measured as affordable (about 1.9 s of
cold build and 200 KB), and the measured 8.9x–52x win now clears the bar the
root package never did. The operator direction is explicit: acceleration
should be on by default with a switch, matching the architecture default
"observable CPU fallback by default, strict GPU-only as opt-in". None of this
has been released, so removing the bridge is not a breaking change.

## What Changes

- `dl` imports `accel` directly and wires the device MatMul itself at init.
  No blank import is needed; loading a model and running it uses the device
  for large 2-D f32 products exactly as the bridge did — same measured 16Mi
  MAC floor, same bit-parity contract, same observable fallback, same strict
  GPU mode behavior.
- The switch: `INSYRA_ACCEL_DISABLE_WGPU=1` (existing) disables the backend
  and dl falls back to the pure CPU path; `dl.RegisterDeviceMatMul(nil)`
  clears the hook programmatically. Both are documented as the opt-out.
- `accel/dlbridge` is deleted. Its registration logic moves behind an
  exported `accel` surface that `dl` can reach (accel/internal/* is not
  importable from dl), with the same fallback-reason reporting.
- Under the `race` build tag the device path is not wired, mirroring accel's
  existing guards for the upstream gogpu Metal checkptr abort; `-race` runs
  are pure CPU and still correct.
- If the direct dependency creates a test-package cycle with
  `accel/internal/wgpu`'s measurement test importing dl, that test moves to
  an external test package (`package wgpu_test`), which Go permits.
- Docs (Docs/dl.md, Docs/dlbridge.md removed or redirected, Docs/README.md
  index), changelogs both languages (amend the existing Unreleased entries
  rather than stacking a correction on them), skills, and delivery-status
  updated in the same change.

## Non-Goals

- No change to the floor, the kernel, the parity contract, or batched/Conv
  scope. This is a wiring-default change only.

## Impact

- Affected specs: `dl-inference`
- Affected code: `dl` (init wiring, go.mod implications for consumers),
  `accel` (exported device-matmul surface), `accel/dlbridge` (removed),
  docs, changelogs, skills.
