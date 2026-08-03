# Change: Acceleration joins the Config system; the env var becomes the ops override

## Why

Runtime behavior switches live on `insyra.Config` — log level, panic
behavior, thread safety. Whether the library may use a device is the same
kind of switch, but today it is reachable only through an environment
variable (`INSYRA_ACCEL_DISABLE_WGPU`) and dl's low-level
`RegisterDeviceMatMul(nil)`. That bypasses the project's own configuration
surface. The operator called it out directly.

The env var is not wrong — it is the ops-level kill switch that works
without recompiling, and it predates this change. The fix is layering, not
replacement: the Config switch is the primary programmatic interface, the
env var overrides it for deployment environments, and a device runs only
when both allow it.

## What Changes

- `insyra.Config` gains `SetAcceleration(enabled bool)` and
  `GetAccelerationEnabled()`, default enabled, following the existing
  Set/Get naming and the singleton's thread-safety conventions.
- Every device path consults it at call time: dl's device MatMul hook
  consult, the KNN bridge's device search, and accel session opening. The
  root package cannot import accel, so the flag lives in root and
  accel/dl read it — the dependency direction is already right.
- `INSYRA_ACCEL_DISABLE_WGPU=1` keeps its meaning and precedence: if set,
  devices are off regardless of Config. Docs state the two layers and
  which wins.
- `RegisterDeviceMatMul(nil)` remains as the dl-local low-level escape
  hatch, documented as such.
- Docs (Docs/dl.md, Docs/accel.md, Docs/Config docs if a page exists),
  changelogs both languages, skills — same change.

## Non-Goals

- No per-operation granularity (one global switch; per-op knobs are
  unearned complexity today).
- No change to strict GPU mode semantics.
- No renaming or deprecation of the env var.

## Impact

- Affected specs: `dl-inference`
- Affected code: root `config.go`, dl hook consult site, `accel`
  (session/DeviceMatMul gate), `accel/knnbridge`, docs, changelogs,
  skills.
