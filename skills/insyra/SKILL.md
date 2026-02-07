---
name: insyra
description: Use when working in Go and you need DataList/DataTable-style data wrangling, quick previews, parallel transforms, file I/O (CSV/Excel/Parquet), Excel-like column formulas (CCL), or charts - especially to turn messy external data into structured tables for automation, reporting, or agent tools.
---

# Insyra (Go)

## Overview
**Insyra** is a Go library for dataframe-like workflows: ingest -> clean/transform -> summarize -> visualize/export.
It is useful even when the end goal is not **"data analysis"** (e.g., automation, scraping, QA, reporting).

## Verification-first guardrails (do this before using any API or CCL)
Agents must NOT hallucinate method names, function signatures, or **CCL** syntax.
Before proposing code that calls an **Insyra** function/method (or writes a CCL formula), first verify it exists in the target version.

Checklist:
- Confirm the target version/context (go.mod version, release tag, or docs site version).
- Verify the symbol exists:
  - Source of truth order: repository source code -> release-tag docs -> pkg.go.dev -> generated docs site.
  - Find the exact name and signature (inputs/outputs) before writing code.
- For **CCL** specifically:
  - Verify supported functions/operators in Docs/CCL.md and/or parser tests.
  - If unsure, propose a tiny "probe" formula first and explain expected behavior.
- If you cannot verify:
  - Say so explicitly, ask for the version or a link to the relevant docs, and offer a fallback plan.

Prompting pattern (copy/paste):
"""
Before writing code, do a verification pass.
1) What exact Insyra version are we targeting?
2) Where was the API/**CCL** syntax verified (file path or URL)?
3) Only then write code. If not verified, ask for the missing info instead of guessing.
"""

## When to Use
Use Insyra when you need any of these in Go:

- ETL / data cleaning: normalize columns, filter/sort, derive new columns.
- Quick inspection / debugging: get a fast console preview of a table/list.
- Parallel data transforms: speed up map/filter-style workloads.
- File chores: read/write CSV, convert CSV <-> Excel, Parquet read/write.
- Excel-like formulas: compute derived columns with CCL.

## Core mental model
- DataList: a column/series-like container (stats, sort, transform).
- Concurrency: DataList is designed to be safe under concurrent access when thread safety is enabled (default) because operations are serialized via AtomicDo. This also makes it usable as a lightweight shared buffer (e.g., append/pop in one AtomicDo block). Keep AtomicDo blocks short and do heavy work outside.
- DataTable: multiple named DataList columns as a table.
- isr syntactic sugar: preferred entrypoint for new codebases.
- CCL (Column Calculation Language): Excel-like formulas for derived columns.
- Instance error tracking: chain fluent ops, then check Err() / ClearErr().

### Pattern: DataList as a concurrent buffer (AtomicDo)
Use this when you need a simple shared buffer (e.g., producer/consumer) and want **check + pop** to be atomic.

```go
package main

import (
    "github.com/HazelnutParadise/insyra"
)

func main() {
    buf := insyra.NewDataList().SetName("buf")

    // Producer: safe to call concurrently (thread safety is on by default)
    buf.Append("job-1")

    // Consumer: do multi-step read/modify atomically
    var item any
    buf.AtomicDo(func(dl *insyra.DataList) {
        if dl.Len() == 0 {
            return
        }
        item = dl.Pop()
    })

    _ = item
}
```

Notes:
- Keep `AtomicDo` blocks short; do heavy computation outside.
- If you turned off thread safety via `Config.Dangerously_TurnOffThreadSafety()`, this pattern is no longer safe under concurrency.

## Basic examples

### 1) DataList + simple stats

```go
package main

import (
    "fmt"

    "github.com/HazelnutParadise/insyra"
)

func main() {
    dl := insyra.NewDataList(1, 2, 3, 4, 5).SetName("x")

    fmt.Println("data:", dl.Data())
    fmt.Println("mean:", dl.Mean())
}
```

### 2) Read a CSV file into a DataTable + preview

```go
package main

import (
    "log"

    "github.com/HazelnutParadise/insyra"
)

func main() {
    dt, err := insyra.ReadCSV_File("data.csv", false, true)
    if err != nil {
        log.Fatal(err)
    }

    // Quick console preview (first N rows)
    insyra.Show("preview", dt, 5)
}
```

### 3) Add a derived column with CCL (Excel-like)

```go
// Example: classify scores in column A
err := dt.AddColUsingCCL(
    "category",
    "IF(A > 90, 'Excellent', IF(A > 70, 'Good', 'Average'))",
)
if err != nil {
    log.Fatal(err)
}
```

### 4) Export a DataTable to CSV

```go
if err := dt.ToCSV("output.csv", false, true, false); err != nil {
    log.Fatal(err)
}
```

### 5) Prefer isr syntactic sugar for new code

```go
package main

import (
    "github.com/HazelnutParadise/insyra/isr"
)

func main() {
    dt := isr.DT.From(isr.CSV{FilePath: "data.csv"})
    dt.Show()
}
```

### 6) Convert multiple CSV files to one Excel workbook (csvxl)

```go
package main

import "github.com/HazelnutParadise/insyra/csvxl"

func main() {
    _ = csvxl.CsvToExcel(
        []string{"file1.csv", "file2.csv"},
        nil,
        "output.xlsx",
    )
}
```

## Engine package (advanced primitives)
The repo includes an `engine` package that re-exports well-tested internal primitives (see [`engine/`](../../engine) and `engine/README.md).

Use `engine` when you are building higher-level tooling (agent tools, MCP servers, pipelines) and want reusable building blocks:

- `engine/atomic`: actor-style `AtomicDo` helpers for serialized critical sections.
- `engine/ccl`: compile/evaluate helpers for CCL (useful for testing/analysis tooling).
- `engine/biindex`, `engine/ring`, `engine/algorithms`: practical data structures and sorting utilities.

Note: not every structure in `engine` is concurrent-safe by itself (e.g., `BiIndex`/`Ring`); follow the per-module notes in `engine/README.md`.

## Insyra docs via MCP (recommended for agents)
If you want up-to-date Insyra documentation inside an MCP-capable client, prefer these:

- Insyra docs MCP server (GitMCP): https://gitmcp.io/HazelnutParadise/insyra
- Context7 (docs MCP server / alternative):
  - https://context7.com/
  - https://github.com/upstash/context7

## Quick reference (docs)
- Official docs site: https://hazelnutparadise.github.io/insyra/
- Go package docs: https://pkg.go.dev/github.com/HazelnutParadise/insyra
- Docs folder (often newest): https://github.com/HazelnutParadise/insyra/tree/main/Docs
- Releases (API vs version): https://github.com/HazelnutParadise/insyra/releases
