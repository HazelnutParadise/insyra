# [ accel ] Package

The `accel` package defines the acceleration runtime surface for Insyra.

The runtime executes on real hardware. A dataset is uploaded to a GPU, ranked by a compute shader, and the host settles the answer in `float64`, so the result matches the CPU path exactly. Nothing extra to install and nothing to import — the GPU backend registers itself when `accel` is initialised. On a host with no usable GPU, every workload falls back to the CPU with an observable reason.

## Current Scope

- Session-scoped runtime entry: `Open(...)` / `NewSession(...)`
- Runtime policy object: `Config`
- Normalized runtime types: `Device`, `Report`, `Buffer`, `Dataset`
- Backend discovery surface:
  - builtin `CUDA`, `Metal`, and `WebGPU` discoverers
  - native probe seams for NVIDIA, Apple, and portable GPU inventory
  - portable probe fallback chains on Windows (`Get-CimInstance`, `Get-WmiObject`, `wmic`) and Linux (`lspci`, `lshw`)
  - discovery timeout handling
- CPU-side typed projection helpers:
  - `ProjectDataList(*insyra.DataList)`
  - `ProjectDataTable(*insyra.DataTable)`
- Columnar transport:
  - numeric and boolean typed buffers
  - validity bitmaps
  - encoded string transport via offsets and values buffer
- Session-local cache accounting:
  - resident buffer index
  - aggregate budget enforcement against normalized accel budgets
  - device usage summaries remain zero until true device residency exists
- Planning and inspection:
  - shardable multi-device planning via `PlanShardable()` / `PlanShardableWorkload(...)`
  - weighted shard assignments and deterministic merge-policy reporting
  - `Config.ShardStrategy`: `single`, `auto` (the default), or `forced`
  - execution via `ExecuteNearestExact(...)`, returning the M nearest query points per row as exact `float64` values
  - `DeviceMatMul(...)`, the device implementation used by `nn`'s default large 2-D float32 MatMul path
  - backend executor registry: `RegisterBackendExecutor(backend, BackendExecutor)`
  - CLI/DSL surfaces: `accel devices`, `accel cache`, `accel plan`, `show accel.devices`, `show accel.cache`, `config accel.mode`

## GPU Execution

Importing `accel` is all it takes. The backend is built on [gogpu/wgpu](https://github.com/gogpu/wgpu), a pure-Go WebGPU implementation, so it builds with `CGO_ENABLED=0`. One WGSL kernel reaches Metal on macOS, Vulkan on Linux and Windows, and DirectX 12 on Windows.

Programs that never import `accel` do not compile any of it — verified with a consumer that imports only the root package: zero GPU packages in its build. The gogpu modules do appear in `go list -m all` for every consumer, which is the cost of not needing a second install.

Set `INSYRA_ACCEL_DISABLE_WGPU=1` to turn the builtin backend off without changing code.

### Acceleration switch and precedence

The primary programmatic switch is `insyra.Config.SetAcceleration(false)`.
It disables call-time device use for `DeviceMatMul` and the KNN bridge, while
preserving the existing CPU fallback behavior. Acceleration is enabled by
default and can be restored with `insyra.Config.SetAcceleration(true)`.

`INSYRA_ACCEL_DISABLE_WGPU=1` is the operations-level override for the builtin
WebGPU backend. It wins over Config, so setting Config to enabled cannot turn
that backend back on while the environment variable is set. A device runs only
when both layers allow it.

### Device bounds and preference

Device selection has three independent axes. Hard bounds decide eligibility;
`PreferredDevices` only orders devices that remain eligible.

| Mode | Device bounds | Preference |
| --- | --- | --- |
| `cpu` | No discovery or device use | Ignored |
| `auto` | `INSYRA_ACCEL_DEVICES` ∩ `Config.Devices` | Soft ordering within the intersection; empty intersection falls back to CPU with `device-selection-empty` |
| `gpu` | `INSYRA_ACCEL_DEVICES` ∩ `Config.Devices` | Soft ordering within the intersection; empty intersection follows automatic fallback behavior |
| `strict-gpu` or `Strict: true` | `INSYRA_ACCEL_DEVICES` ∩ `Config.Devices` | Soft ordering within the intersection; empty intersection returns an error naming the emptying bound |

`Config.Devices` is a per-session hard allowlist. `INSYRA_ACCEL_DEVICES` is a
process-wide hard mask. An empty value for either leaves that bound open, and
the default is unchanged when both are open. Both accept exact device IDs and
zero-based discovery indices, with comma-separated entries in the environment
variable. IDs are stable and portable across hosts; indices depend on each
host's discovery order and should only be used for host-local configuration.

When an entry matches no discovered device, `Session.Report()` exposes it in
`Report.UnmatchedDeviceSelectors`. Masked devices are absent from
`DiscoveredDeviceIDs`, so planners and executors cannot see or resurrect them.

### Shard strategies

`Config.ShardStrategy` controls how a shardable `ExecuteNearestExact` workload
uses its eligible devices:

- `single` keeps the whole workload on one eligible device.
- `auto` is the default. It uses
  `min(eligible devices, floor(total rows / saturation floor))` assignments,
  collapsing to one assignment when that count is one. The recorded floor is
  32,000 rows for the 32-dimensional class and 8,000 rows for the
  128-dimensional class. Unmeasured shapes use the conservative 32,000-row
  floor.
- `forced` uses every eligible device even below the measured floor. Use it
  when measuring a host whose device topology is different from the recorded
  Apple M3 curve.

Assignments carry their input range and execution report. Inspect
`ExecutionResult.Assignments` for `DeviceID`, `RowStart`, `RowEnd`, `WallTime`,
`Chunks`, and `FallbackReason`. A failed assignment is recomputed on the CPU
for its own rows; successful assignments are kept and merged by input range,
independent of completion order. The final nearest indices and `float64`
distances remain bit-identical to `NearestExactCPU`.

Single-device hardware correctness for the exact-nearest path is verified.
The concurrent-versus-sequential multi-assignment parity test is written as a
gated acceptance test and skips in this sandbox. No multi-GPU wall-clock
measurement exists yet. The multi-GPU speedup is therefore not a claim of this
release and remains pending the standing hardware-coverage follow-up.

### Precision

WGSL has no `f64`, and Apple GPUs have no double-precision hardware at all. A `float64` column therefore cannot run on the device at its own precision, and the runtime will not narrow it behind your back — a data-analysis library that silently changes your numbers is worse than one that declines to accelerate them.

The surviving operation resolves that rather than trading it away: the device ranks in single precision, and the host settles the answer in `float64`. Narrowing happens inside, where it cannot reach the result, so no precision opt-in is needed and none is offered.

## Operations

| Operation | Shape | Notes |
| --- | --- | --- |
| `ExecuteNearestExact` | the M nearest query points per row, in `float64` | The device narrows, the host decides. Exactly the `float64` answer. |
| `DeviceMatMul` | 2-D `float32` matrix product | Returns an error for the caller's exact CPU fallback and records device status in `accel.Default().Report()`. |

Three other operations existed and were removed once measured against a host using every core it has: a column sum at 0.7x, a squared-distance matrix whose readback grew with the answer, and a single-precision nearest query no `float64` caller could use. Nothing is added back without that measurement.

The device operation ships a CPU reference — `NearestExactCPU` — and a test asserting the device result is bit-identical to it on the running platform. Parity depends on both toolchains contracting multiply-add the same way, which is a property of the platform rather than of the kernel, so it is measured where it runs rather than assumed.

### Exact answers from a single-precision device

`ExecuteNearestExact` returns what a `float64` computation over every query point would return — the same indices, the same distances — while still using the device for most of the work.

It gets there by not trusting the device with the decision. The device ranks the query points in single precision and returns a few candidates per row, along with the distance of the best candidate it discarded. The host recomputes those candidates in `float64` and picks from them. If the discarded one is close enough that single precision could not tell them apart, that row is recomputed against every query point instead.

```go
result, err := session.ExecuteNearestExact(ds, centroids, 2, accel.WorkloadEstimate{})
// result.Index[r*2], result.Index[r*2+1] — nearest and second nearest for row r
// result.Distance holds float64 squared distances
// result.Rechecked — how many rows took the full path
// result.Chunks — how many sequential device submissions ran (zero on CPU)
```

No precision opt-in is needed. Narrowing happens inside the operation, where it cannot reach the result.

Device submissions larger than the measured 16,000-row bound are split into sequential row chunks on the same device and merged in input order. Submissions at or below the bound keep the single-submission path. This bounds readback exposure without changing the exact `float64` decision, and `ExecutionResult.Chunks` reports how many submissions ran.

Watch `Rechecked`. It is normally near zero, and a rising count is how data with many near-ties announces itself — long before it shows up as a slowdown.

Measured on an Apple M3 over 200,000 rows, asking for the two nearest, against a host using all eight cores:

| query points | 16 dims, host | 16 dims, device | 64 dims, host | 64 dims, device |
| --- | --- | --- | --- | --- |
| 32 | 14.8 ms | 17.5 ms | 62.9 ms | 66.8 ms |
| 128 | 49.5 ms | 39.4 ms | 222.3 ms | 107.4 ms |
| 512 | 154.4 ms | 74.5 ms | 819.6 ms | 281.7 ms |
| 1024 | 306.0 ms | 123.3 ms | 1.648 s | 480.9 ms |

The device is not always the faster option, and the deciding factor is how much arithmetic one row carries — dimensions times query points — rather than the size of the dataset. Below roughly two thousand distance evaluations per row the host's cores win, so the runtime declines the device and reports `workload-not-profitable`. Across 96 shapes on this host, the device was faster in 42.

On unified-memory hardware like Apple Silicon the gap between a GPU and eight CPU cores is not large. A discrete GPU would move this substantially, and has not been measured.

### No device is not an error

`ExecuteNearestExact` returns the answer whether or not a device ran. On a machine with no GPU, or when a device is present but fails, times out, or exceeds a buffer limit, the runtime computes the result on the CPU and hands it back. You do not have to check anything to get a correct answer.

`Accelerated` and `FallbackReason` are still there to tell you *where* the work ran, which is worth logging, but they are not a correctness gate:

```go
result, err := session.ExecuteNearestExact(ds, queries, 2, accel.WorkloadEstimate{})
// result.Index is populated either way.
if !result.Accelerated {
    log.Printf("ran on the CPU: %s", result.FallbackReason)
}
```

Two cases deliberately return nothing instead. A dataset nothing can read — `dtype-not-eligible`, `workload-unsupported` — has no answer to give, and the reason says so. And strict GPU mode returns an error rather than falling back, which is the whole point of asking for it.

## Still Not Implemented

- One operation only: the exact nearest query points per row
- Multi-GPU wall-clock measurement and speedup claim
- No CUDA-native path — NVIDIA hardware is reached through Vulkan or DirectX 12
- No implicit acceleration of `DataList.Map(func...)` or `DataTable.Map(func...)`
- No full string-kernel execution path beyond transport and eligibility preparation
- Correctness is verified on single-device macOS/Metal hardware; other
  platforms and multi-GPU wall clock are untested

## The shared session

`accel.Default()` returns a session shared by the whole process, created the
first time it is asked for:

```go
session := accel.Default()
result, err := session.ExecuteNearestExact(ds, centroids, 2, accel.WorkloadEstimate{})
```

Discovery runs once, the resident cache is shared so a column stays on the
device across operations, and no caller owns the lifetime — `Close` on it does
nothing. Importing the package opens no device; only the first `Default()` call
does. Use `Open` when you want your own session and your own configuration.

## Installation

`accel` is part of `allpkgs`, so the standard install includes it. Installing only the root module works too — `accel` is a package inside it.

```bash
go get github.com/HazelnutParadise/insyra/accel
```

## Quick Example

```go
package main

import (
    "fmt"

    "github.com/HazelnutParadise/insyra"
    "github.com/HazelnutParadise/insyra/accel"
)

func main() {
    session, err := accel.Open(accel.Config{})
    if err != nil {
        panic(err)
    }
    defer session.Close()

    ds := accel.ProjectDataTable(dt)
    centroids := [][]float64{{0, 0}, {1, 1}, {2, 2}}

    // Find the two nearest centroids per row. A GPU narrows the field when the
    // host has one; the answer is the float64 one either way.
    result, err := session.ExecuteNearestExact(ds, centroids, 2, accel.WorkloadEstimate{})
    if err != nil {
        panic(err)
    }

    fmt.Println(result.Accelerated, result.FallbackReason)
    fmt.Println(result.Index[:4], result.Distance[:4])
    fmt.Println(result.Rechecked, "of", result.Rows, "rows took the full path")
}
```

## Notes

- Default backend preference is `CUDA`, then `Metal`, then `WebGPU`.
- Native discovery is best-effort. Env-driven stubs remain available for deterministic testing and non-GPU development.
- Shared-memory devices can derive working-set budgets from host memory when native budget data is unavailable.
- `accel devices`, `accel cache` and `accel plan` inspect the runtime; none of them executes anything. The command that did was removed with the operations it invoked.
- Execution cost figures are only reported when something actually ran on a device.
