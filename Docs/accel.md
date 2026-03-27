# [ accel ] Package

The `accel` package defines the opt-in acceleration runtime surface for Insyra.

The package is still pre-execution, but it is no longer only a shape freeze. The current slice includes backend discovery seams, typed CPU-side projection, resident-cache accounting, shard planning, and CLI/DSL inspection surfaces.

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
  - allocator registry plus execution ledger via `ExecuteProjectedDataset(...)`, `ExecuteDataList(...)`, and `ExecuteDataTable(...)`
  - builtin homogeneous allocators for `CUDA`, `Metal`, and `WebGPU`, with ledger fallback for heterogeneous plans
  - CLI/DSL surfaces: `accel devices`, `accel cache`, `accel plan`, `accel run <var>`, `show accel.devices`, `show accel.cache`, `config accel.mode`

## Still Not Implemented

- No true CUDA / Metal / WebGPU kernel execution yet
- No backend-native VRAM allocator implementation yet
- No true backend allocator or merge execution path yet; current execution seam has builtin per-backend allocator stubs for homogeneous plans, but they still materialize accounting/residency records rather than GPU allocations or kernels
- No implicit acceleration of `DataList.Map(func...)` or `DataTable.Map(func...)`
- No full string-kernel execution path beyond transport and eligibility preparation

## Installation

`accel` is optional and is not part of `allpkgs` at this stage.

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

    dl := insyra.NewDataList(1, 2, nil, 4).SetName("numbers")
    ds, err := session.ProjectDataList(dl)
    if err != nil {
        panic(err)
    }

    fmt.Println(ds.Name)
    fmt.Println(ds.Rows)
    fmt.Println(ds.Buffers[0].Type)
}
```

## Notes

- Default backend preference is `CUDA`, then `Metal`, then `WebGPU`.
- Native discovery is best-effort. Env-driven stubs remain available for deterministic testing and non-GPU development.
- Shared-memory devices can derive working-set budgets from host memory when native budget data is unavailable.
- `accel plan` remains a planning/report surface. `accel run <var>` now drives execution through builtin backend allocator stubs or the internal ledger fallback, but it still does not launch backend-native GPU kernels.
