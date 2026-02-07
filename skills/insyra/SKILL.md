---
name: insyra
description: Use when working in Go and you need DataList/DataTable-style data wrangling, quick previews, parallel transforms, file I/O (CSV/Excel/Parquet), Excel-like column formulas (CCL), or charts—especially to turn messy external data into structured tables for automation, reporting, or agent tools.
---

# Insyra (Go)

## Overview
**Insyra** is a Go library for dataframe-like workflows: ingest → clean/transform → summarize → visualize/export.
It’s useful even when the end goal isn’t “data analysis” (e.g., automation, scraping, QA, reporting).

## When to Use
Use Insyra when you need any of these in Go:

- **ETL / data cleaning:** normalize columns, filter/sort, derive new columns.
- **Quick inspection / debugging:** get a fast console preview of a table/list.
- **Parallel data transforms:** speed up map/filter-style workloads.
- **File chores:** read/write CSV, convert CSV↔Excel, Parquet read/write.
- **Excel-like formulas:** compute derived columns with **CCL**.

## Core mental model
- **`DataList`**: a column/series-like container (stats, sort, transform).
- **`DataTable`**: multiple named `DataList` columns as a table.
- **`isr` syntactic sugar**: preferred entrypoint for new codebases.
- **CCL (Column Calculation Language)**: Excel-like formulas for derived columns.
- **Instance error tracking**: chain fluent ops, then check `Err()` / `ClearErr()`.

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

### 5) Prefer `isr` syntactic sugar for new code

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

### 6) Convert multiple CSV files to one Excel workbook (`csvxl`)

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

## Insyra docs via MCP (recommended for agents)
If you want **up-to-date Insyra documentation inside an MCP-capable client**, prefer these:

- Insyra docs MCP server (GitMCP): https://gitmcp.io/HazelnutParadise/insyra
- Context7 (docs MCP server / alternative):
  - https://context7.com/
  - https://github.com/upstash/context7

## Quick reference (docs)
- Official docs site: https://hazelnutparadise.github.io/insyra/
- Go package docs: https://pkg.go.dev/github.com/HazelnutParadise/insyra
- Docs folder (often newest): https://github.com/HazelnutParadise/insyra/tree/main/Docs
- Releases (API vs version): https://github.com/HazelnutParadise/insyra/releases
