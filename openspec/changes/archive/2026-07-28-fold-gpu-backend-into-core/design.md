## Context
`accel/backend/wgpu` is its own Go module so that gogpu stays out of the core module's requirements. That decision was taken on three grounds; two of them turned out to be wrong when tested, and the remaining one is a maintainer cost the project has chosen to accept in exchange for a one-step install.

The obstacle to simply moving the package is an import cycle. The backend imports `accel` for `Device`, `ExecuteRequest`, and the registries. For the backend to register itself automatically, something a user already imports has to import it — and the only such thing is `accel`, which the backend already imports.

## Goals / Non-Goals
- Goals:
  - `go get github.com/HazelnutParadise/insyra` is the only install step
  - no registration import in user code
  - a consumer who never touches accel still does not compile gogpu
- Non-Goals:
  - removing the public backend registry; third-party backends keep working
  - changing execution behaviour, precision policy, or any measured figure

## Decisions

- Decision: split the backend into a leaf package plus an adapter, rather than merging it into `package accel` wholesale.
  - `accel/internal/wgpu` holds the gogpu mechanics and imports nothing from `accel`. It speaks in its own small vocabulary: an `Info` describing the adapter, a `Cost` describing what an execution took, and sentinel errors for the three failure modes worth distinguishing.
  - `accel/backend_wgpu.go` maps that vocabulary onto `Device`, `ExecuteRequest`, and `ExecuteResponse`, and registers the probe and executor from `init`.
  - Rationale: the cycle disappears because the dependency runs one way. Dropping the files into `package accel` would also work — only `init` collides — but it would put WGSL source, `unsafe`, and driver handling in the same package as the runtime contract, and every `go test ./accel/` would link a GPU stack to test a cache eviction rule.
  - `internal/` is deliberate: with the backend no longer separately installable, its API is not something users should reach for.

- Decision: keep the registry seam even though the builtin backend no longer needs it.
  - Rationale: `RegisterBackendExecutor` is how a third party ships a CUDA or ROCm backend, and how tests inject a fake executor on machines with no GPU. Removing it to save one indirection would take the extension point with it.

## Risks / Trade-offs
- Risk, accepted by decision: a breaking change in gogpu now turns the main build red rather than only the backend module's. gogpu is 0.x and shipped 175 releases in seven and a half months.
  - Mitigation: the dependency is pinned in `go.mod`, so the break arrives when the version is deliberately moved rather than on its own.
- Risk: every insyra consumer's dependency graph now lists six gogpu modules, visible to `go list -m all`, Dependabot, and dependency review.
  - Accepted; this is the cost the one-step install buys.
- Not a risk, having been tested: a consumer who never imports accel neither compiles gogpu nor inherits its minimum Go version. A module that requires a `go 1.29` dependency it does not import builds fine on Go 1.26.
