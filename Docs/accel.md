# [ accel ] Package

The `accel` package defines the opt-in acceleration runtime surface for Insyra.

This package is intentionally backend-agnostic in its first implementation step. It freezes the public runtime shape before CUDA, Metal, WebGPU-native discovery, VRAM/shared-memory caching, heterogeneous scheduling, and CLI/DSL controls are implemented.

## Current Scope

- Session-scoped runtime entry: `Open(...)` / `NewSession(...)`
- Runtime policy object: `Config`
- Normalized runtime types: `Device`, `Report`, `Buffer`, `Dataset`
- CPU-side typed projection helpers:
  - `ProjectDataList(*insyra.DataList)`
  - `ProjectDataTable(*insyra.DataTable)`

## Non-Goals For This Step

- No real GPU backend discovery yet
- No CUDA / Metal / WebGPU-native execution yet
- No implicit acceleration of `DataList.Map(func...)` or `DataTable.Map(func...)`
- No CLI / DSL accel surface yet

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

- The current default report is intentionally conservative. It reports a non-accelerated state until a real backend is selected.
- This package exists to stabilize the runtime contract first. Backend execution, cache residency, and multi-GPU planning are handled in later changes.
