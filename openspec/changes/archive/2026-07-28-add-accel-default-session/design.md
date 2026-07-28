## Context
`Open` runs discovery synchronously, and discovery probes the host — on a machine with a GPU it opens an adapter and a device. Doing that once per call site would be wasteful; doing it in `init` would make importing `accel` open a GPU device even for a program that never accelerates anything, which is exactly what the lazy registration in `backend_wgpu.go` was careful to avoid.

## Goals / Non-Goals
- Goals: one session per process, created on first use, safe to call from anywhere
- Non-Goals: a global mode switch, per-goroutine sessions, or any change to how `Open` behaves for callers who want their own session

## Decisions

- Decision: lazily construct on first `Default()` call via `sync.Once`, never in `init`.
  - Rationale: importing a package should not open a GPU device. A program that imports `accel` transitively — which, now that `accel` is in `allpkgs`, is most of them — must not pay device acquisition unless it asks for acceleration.

- Decision: a discovery error does not stop `Default()` from returning a session.
  - Rationale: `Open` already returns a usable session alongside its error so the report stays inspectable, and the runtime's whole contract is observable CPU fallback. A `Default()` that returned nil on a driverless machine would push nil-checking into every call site to express "no GPU here", which the fallback reason already expresses.

- Decision: `Close` on the shared session is a no-op rather than an error.
  - Rationale: library code that receives the shared session cannot know it is shared. Making `Close` fail would turn a reasonable defensive call into a spurious error; making it actually close would let one caller disable acceleration for the whole process. Ignoring it is the only behaviour that is safe from every call site.
  - `Closed()` therefore always reports false for the shared session, which is honest: it never closes.

- Decision: configuration comes from `DefaultConfig()`, with no environment override of its own.
  - Rationale: the knobs that matter already have env switches read during discovery (`INSYRA_ACCEL_DISABLE_WGPU`, `INSYRA_ACCEL_DISABLE_NATIVE_PROBES`). Adding a second configuration channel for the default session would create two sources of truth for the same behaviour. A caller who needs different settings should use `Open`.

## Risks / Trade-offs
- Risk: the shared session's cache lives for the life of the process.
  - Mitigation: it is bounded — the resident cache enforces the device budget and evicts by least-recent access. That machinery predates this change and is already tested.
- Trade-off: tests touching the shared session are order-sensitive unless they reset it.
  - Mitigation: `ResetDefaultForTest` exists for exactly that, matching `ResetDiscoverersForTest` and `ResetBackendExecutorsForTest`.
