# [ accel ] Package

The `accel` package defines the opt-in acceleration runtime surface for Insyra.

The runtime executes on real hardware. A numeric column is uploaded to a GPU, reduced by a compute shader, and read back, and the returned value matches the CPU path. Nothing extra to install and nothing to import — the GPU backend registers itself when `accel` is initialised. On a host with no usable GPU, every workload falls back to the CPU with an observable reason.

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
  - execution via `ExecuteProjectedDataset(...)`, `ExecuteDataList(...)`, and `ExecuteDataTable(...)`, returning the computed value per column
  - backend executor registry: `RegisterBackendExecutor(backend, BackendExecutor)`
  - CLI/DSL surfaces: `accel devices`, `accel cache`, `accel plan`, `accel run <var>`, `show accel.devices`, `show accel.cache`, `config accel.mode`

## GPU Execution

Importing `accel` is all it takes. The backend is built on [gogpu/wgpu](https://github.com/gogpu/wgpu), a pure-Go WebGPU implementation, so it builds with `CGO_ENABLED=0`. One WGSL kernel reaches Metal on macOS, Vulkan on Linux and Windows, and DirectX 12 on Windows.

Programs that never import `accel` do not compile any of it — verified with a consumer that imports only the root package: zero GPU packages in its build. The gogpu modules do appear in `go list -m all` for every consumer, which is the cost of not needing a second install.

Set `INSYRA_ACCEL_DISABLE_WGPU=1` to turn the builtin backend off without changing code.

### Precision is opt-in

WGSL has no `f64`, and Apple GPUs have no double-precision hardware at all. A `float64` column therefore cannot run on the device at its own precision, and the runtime will not narrow it behind your back — a data-analysis library that silently changes your numbers is worse than one that declines to accelerate them.

Ask for single precision explicitly when that trade is acceptable:

```go
result, err := session.ExecuteDataList(dl, accel.WorkloadEstimate{
    Precision: accel.PrecisionFloat32,
})
```

Without it the workload falls back to the CPU and `result.FallbackReason` is `precision-not-accepted`. Only the reduction inside one workgroup runs at single precision; the host folds the per-workgroup partials in `float64`.

### What it costs

Measured on an Apple M3 over a `float64` column narrowed to `float32`:

| Column | transfer | dispatch | readback |
| --- | --- | --- | --- |
| 64 Ki | 0.021 ms | 0.034 ms | 0.38 ms |
| 1 Mi | 0.47 ms | 0.061 ms | 0.93 ms |
| 4 Mi | 2.43 ms | 0.289 ms | 2.91 ms |

A single column sum is memory-bound, so moving the data costs far more than the arithmetic. Do not expect one reduction to beat a CPU loop on a unified-memory machine. `result.Transfer`, `result.Dispatch`, and `result.Readback` are measured per execution, so you can check rather than assume.

## Operations

| Operation | Shape | Notes |
| --- | --- | --- |
| `ExecuteDataList` / `ExecuteDataTable` | sum per column | Memory-bound; the CPU wins. Kept as the proving path, not a recommendation. |
| `ExecuteDistances` | squared distance, row by query | Returns the whole matrix, so readback grows with it. |
| `ExecuteNearestQuery` | closest query point per row | The matrix is reduced on the device. 13.6x over the CPU at 64 query points on an M3. |

Every device operation ships a CPU reference — `SquaredDistancesCPU`, `NearestQueryCPU` — and a test asserting the device result is bit-identical to it on the running platform. Parity depends on both toolchains contracting multiply-add the same way, which is a property of the platform rather than of the kernel, so it is measured where it runs rather than assumed.

### No device is not an error

`ExecuteDistances` and `ExecuteNearestQuery` return the answer whether or not a device ran. On a machine with no GPU, or when a device is present but fails, times out, or exceeds a buffer limit, the runtime computes the result on the CPU and hands it back. You do not have to check anything to get a correct answer.

`Accelerated` and `FallbackReason` are still there to tell you *where* the work ran, which is worth logging, but they are not a correctness gate:

```go
result, err := session.ExecuteNearestQuery(ds, queries, accel.WorkloadEstimate{
    Precision: accel.PrecisionFloat32,
})
// result.Index is populated either way.
if !result.Accelerated {
    log.Printf("ran on the CPU: %s", result.FallbackReason)
}
```

Two cases deliberately return nothing instead. When the request itself is what excluded the device — `precision-not-accepted`, `dtype-not-eligible`, `workload-unsupported` — the CPU result would be exactly what you declined, so you get no result and the reason says why. And strict GPU mode returns an error rather than falling back, which is the whole point of asking for it.

## Still Not Implemented

- One operation only: `sum` over a numeric column
- Single-device execution; multi-device plans run on the highest-weighted device
- No CUDA-native path — NVIDIA hardware is reached through Vulkan or DirectX 12
- No implicit acceleration of `DataList.Map(func...)` or `DataTable.Map(func...)`
- No full string-kernel execution path beyond transport and eligibility preparation
- Verified on macOS and Metal only; other platforms are untested

## The shared session

`accel.Default()` returns a session shared by the whole process, created the
first time it is asked for:

```go
session := accel.Default()
result, err := session.ExecuteDataList(dl, accel.WorkloadEstimate{
    Precision: accel.PrecisionFloat32,
})
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

    dl := insyra.NewDataList(1.5, 2.5, nil, 4.5).SetName("numbers")

    // Reduce the column. Runs on a GPU when the host has one, and falls back
    // to the CPU with an observable reason when it does not.
    result, err := session.ExecuteDataList(dl, accel.WorkloadEstimate{
        Precision: accel.PrecisionFloat32,
    })
    if err != nil {
        panic(err)
    }

    fmt.Println(result.Accelerated, result.FallbackReason)
    fmt.Println(result.Reductions["numbers"], result.Counts["numbers"])
    fmt.Println(result.Transfer, result.Dispatch, result.Readback)
}
```

## Notes

- Default backend preference is `CUDA`, then `Metal`, then `WebGPU`.
- Native discovery is best-effort. Env-driven stubs remain available for deterministic testing and non-GPU development.
- Shared-memory devices can derive working-set budgets from host memory when native budget data is unavailable.
- `accel plan` remains a planning/report surface. `accel run <var> --precision float32` executes on a device when a backend module is linked in, and prints the computed value alongside the measured transfer, dispatch, and readback times.
- Execution cost figures are only reported when something actually ran on a device.
