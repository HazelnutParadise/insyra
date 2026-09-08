# Parquet Inspection, Streaming, and CCL Filtering

This tutorial demonstrates a practical Parquet workflow with Insyra: inspect file metadata, stream in batches, filter with CCL, and write a new Parquet output.

## What you will build

You will build a pipeline that:

- creates a local Parquet file from a DataTable,
- inspects schema and row-group metadata,
- reads selected columns,
- streams data in chunks,
- applies CCL filtering,
- writes a filtered Parquet file.

## Prerequisites

- Go 1.25+.
- Insyra + parquet package:

```bash
go get github.com/HazelnutParadise/insyra
go get github.com/HazelnutParadise/insyra/parquet
```

## Scenario

You are processing medium-size transaction data in Parquet format and want to avoid loading everything into memory while still applying business filters.

## Step 1: Build and write a local Parquet file

**Goal**  
Prepare a reproducible local Parquet source for the rest of the tutorial.

**Code**

```go
dt := insyra.NewDataTable(
	insyra.NewDataList(1, 2, 3, 4, 5, 6).SetName("OrderID"),
	insyra.NewDataList("North", "North", "South", "East", "West", "South").SetName("Region"),
	insyra.NewDataList(2200.0, 430.0, 1200.0, 980.0, 250.0, 1750.0).SetName("NetSales"),
)

if err := parquet.Write(dt, "sales.parquet"); err != nil {
	log.Fatal(err)
}
```

**Expected outcome**  
`sales.parquet` is created.

## Step 2: Inspect metadata

**Goal**  
Read schema and row-group metadata before deciding read strategy.

**Code**

```go
info, err := parquet.Inspect("sales.parquet")
if err != nil {
	log.Fatal(err)
}
fmt.Println("rows:", info.NumRows, "rowGroups:", info.NumRowGroups)
for _, c := range info.Columns {
	fmt.Println("column:", c.Name, c.PhysicalType)
}
```

**Expected outcome**  
You see row count, row group count, and column metadata.

## Step 3: Read selected columns

**Goal**  
Load only needed columns for faster analysis.

**Code**

```go
ctx := context.Background()
small, err := parquet.Read(ctx, "sales.parquet", parquet.ReadOptions{
	Columns: []string{"OrderID", "NetSales"},
})
if err != nil {
	log.Fatal(err)
}
small.ShowRange(6)
```

**Expected outcome**  
Only `OrderID` and `NetSales` are loaded.

## Step 4: Stream in batches

**Goal**  
Process data chunk-by-chunk to keep memory predictable.

**Code**

```go
rowsSeen := 0
batches, errs := parquet.Stream(ctx, "sales.parquet", parquet.ReadOptions{}, 2)
for b := range batches {
	r, _ := b.Size()
	rowsSeen += r
}
if err := <-errs; err != nil {
	log.Fatal(err)
}
fmt.Println("streamed rows:", rowsSeen)
```

**Expected outcome**  
Rows are processed in small chunks (`batchSize=2`).

## Step 5: Filter with CCL

**Goal**  
Apply business rules directly from Parquet using CCL.

**Code**

```go
filtered, err := parquet.FilterWithCCL(ctx, "sales.parquet", "['NetSales'] >= 1000")
if err != nil {
	log.Fatal(err)
}
filtered.ShowRange(6)
```

**Expected outcome**  
Only high-value rows remain.

## Step 6: Write filtered output

**Goal**  
Persist filtered records as a new Parquet dataset.

**Code**

```go
if err := parquet.Write(filtered, "sales_high_value.parquet"); err != nil {
	log.Fatal(err)
}
```

**Expected outcome**  
`sales_high_value.parquet` is generated.

## Complete runnable Go program

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/parquet"
)

func main() {
	dt := insyra.NewDataTable(
		insyra.NewDataList(1, 2, 3, 4, 5, 6).SetName("OrderID"),
		insyra.NewDataList("North", "North", "South", "East", "West", "South").SetName("Region"),
		insyra.NewDataList(2200.0, 430.0, 1200.0, 980.0, 250.0, 1750.0).SetName("NetSales"),
	)
	if err := parquet.Write(dt, "sales.parquet"); err != nil {
		log.Fatal(err)
	}

	info, err := parquet.Inspect("sales.parquet")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("rows:", info.NumRows, "rowGroups:", info.NumRowGroups)

	ctx := context.Background()
	_, err = parquet.Read(ctx, "sales.parquet", parquet.ReadOptions{
		Columns: []string{"OrderID", "NetSales"},
	})
	if err != nil {
		log.Fatal(err)
	}

	batches, errs := parquet.Stream(ctx, "sales.parquet", parquet.ReadOptions{}, 2)
	for range batches {
	}
	if err := <-errs; err != nil {
		log.Fatal(err)
	}

	filtered, err := parquet.FilterWithCCL(ctx, "sales.parquet", "['NetSales'] >= 1000")
	if err != nil {
		log.Fatal(err)
	}

	if err := parquet.Write(filtered, "sales_high_value.parquet"); err != nil {
		log.Fatal(err)
	}
}
```

## CLI/.isr equivalent workflow (appendix)

`insyra` CLI can load Parquet and do table-level operations, but streaming and parquet-native CCL filtering are Go package features.

### CLI part (available)

```bash
insyra load parquet sales.parquet cols OrderID,NetSales as t
insyra summary t
insyra save t sales_selected.csv
```

### Go supplement for parquet-native features

- Use `parquet.Stream(...)` for chunked reads.
- Use `parquet.FilterWithCCL(...)` for direct Parquet filtering.

## Where to go next

- [parquet](../parquet.md)
- [CCL](../CCL.md)
- [DataTable](../DataTable.md)
