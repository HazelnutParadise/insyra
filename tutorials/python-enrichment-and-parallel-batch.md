# Python Enrichment and Parallel Batch Processing

This tutorial combines Python-based feature enrichment (`py`) with concurrent batch calculations (`parallel`).

## What you will build

You will build a workflow that:

- creates a local source dataset,
- enriches it in Python with `py.RunCode` / `py.RunCodef`,
- returns enriched data as `*insyra.DataTable`,
- runs independent KPI calculations in parallel,
- writes a combined output table and summary.

## Prerequisites

- Go 1.25+.
- Insyra + py + parallel:

```bash
go get github.com/HazelnutParadise/insyra
go get github.com/HazelnutParadise/insyra/py
go get github.com/HazelnutParadise/insyra/parallel
```

## Scenario

You want Python's rich data transformation ecosystem, but keep orchestration in Go and scale summary calculations via parallel execution.

## Step 1: Create base dataset

**Goal**  
Prepare a small table for enrichment.

**Code**

```go
base := insyra.NewDataTable(
	insyra.NewDataList("U1", "U2", "U3", "U4", "U5").SetName("UserID"),
	insyra.NewDataList(30, 45, 12, 75, 54).SetName("Sessions"),
	insyra.NewDataList(120.0, 260.0, 40.0, 420.0, 310.0).SetName("Revenue"),
)
```

**Expected outcome**  
A DataTable with `UserID`, `Sessions`, and `Revenue`.

## Step 2: Enrich with py.RunCode (Python DataFrame return)

**Goal**  
Use Python to derive ARPU and return a DataFrame back to Go.

**Code**

```go
var enriched *insyra.DataTable
err := py.RunCode(&enriched, `
import pandas as pd
df = pd.DataFrame({
    "UserID": ["U1","U2","U3","U4","U5"],
    "Sessions": [30,45,12,75,54],
    "Revenue": [120.0,260.0,40.0,420.0,310.0],
})
df["ARPU"] = df["Revenue"] / df["Sessions"]
insyra.Return(df)
`)
if err != nil {
	log.Fatal(err)
}
```

**Expected outcome**  
`enriched` contains Python-derived `ARPU`.

## Step 3: Enrich with py.RunCodef (templated threshold)

**Goal**  
Inject a Go threshold and tag high-value users in Python.

**Code**

```go
var tagged *insyra.DataTable
err = py.RunCodef(&tagged, `
import pandas as pd
threshold = $v1
df = pd.DataFrame({
    "UserID": ["U1","U2","U3","U4","U5"],
    "Revenue": [120.0,260.0,40.0,420.0,310.0],
})
df["Tag"] = df["Revenue"].apply(lambda x: "HighValue" if x >= threshold else "Standard")
insyra.Return(df)
`, 250.0)
if err != nil {
	log.Fatal(err)
}
```

**Expected outcome**  
`tagged` includes a `Tag` column controlled by Go input.

## Step 4: Run batch KPIs in parallel

**Goal**  
Compute multiple independent metrics concurrently.

**Code**

```go
results := parallel.GroupUp(
	func() any { return enriched.GetColByName("ARPU").Mean() },
	func() any { return enriched.GetColByName("Revenue").Sum() },
	func() any { return enriched.GetColByName("Sessions").Sum() },
).Run().AwaitResult()
```

**Expected outcome**  
`results` contains three computed KPI values.

## Step 5: Build summary DataTable

**Goal**  
Convert parallel outputs into a report table.

**Code**

```go
summary := insyra.NewDataTable(
	insyra.NewDataList("ARPU_Mean", "Revenue_Sum", "Sessions_Sum").SetName("Metric"),
	insyra.NewDataList(results[0][0], results[1][0], results[2][0]).SetName("Value"),
)
summary.ShowRange(10)
```

**Expected outcome**  
A compact KPI summary table is available.

## Step 6: Export enriched outputs

**Goal**  
Persist both enriched user table and KPI summary.

**Code**

```go
_ = enriched.ToCSV("py_enriched_users.csv", false, true, false)
_ = summary.ToCSV("py_parallel_summary.csv", false, true, false)
```

**Expected outcome**  
Two CSV outputs are generated.

## Complete runnable Go program

```go
package main

import (
	"log"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/parallel"
	"github.com/HazelnutParadise/insyra/py"
)

func main() {
	var enriched *insyra.DataTable
	err := py.RunCode(&enriched, `
import pandas as pd
df = pd.DataFrame({
    "UserID": ["U1","U2","U3","U4","U5"],
    "Sessions": [30,45,12,75,54],
    "Revenue": [120.0,260.0,40.0,420.0,310.0],
})
df["ARPU"] = df["Revenue"] / df["Sessions"]
insyra.Return(df)
`)
	if err != nil {
		log.Fatal(err)
	}

	var tagged *insyra.DataTable
	err = py.RunCodef(&tagged, `
import pandas as pd
threshold = $v1
df = pd.DataFrame({
    "UserID": ["U1","U2","U3","U4","U5"],
    "Revenue": [120.0,260.0,40.0,420.0,310.0],
})
df["Tag"] = df["Revenue"].apply(lambda x: "HighValue" if x >= threshold else "Standard")
insyra.Return(df)
`, 250.0)
	if err != nil {
		log.Fatal(err)
	}
	_ = tagged // available for campaign output if needed

	results := parallel.GroupUp(
		func() any { return enriched.GetColByName("ARPU").Mean() },
		func() any { return enriched.GetColByName("Revenue").Sum() },
		func() any { return enriched.GetColByName("Sessions").Sum() },
	).Run().AwaitResult()

	summary := insyra.NewDataTable(
		insyra.NewDataList("ARPU_Mean", "Revenue_Sum", "Sessions_Sum").SetName("Metric"),
		insyra.NewDataList(results[0][0], results[1][0], results[2][0]).SetName("Value"),
	)

	_ = enriched.ToCSV("py_enriched_users.csv", false, true, false)
	_ = summary.ToCSV("py_parallel_summary.csv", false, true, false)
}
```

## CLI/.isr equivalent workflow (appendix)

`py` and `parallel.GroupUp` are Go package workflows (no direct dedicated CLI commands).

### Closest CLI preprocessing

```bash
insyra newdl 120 260 40 420 310 as revenue
insyra newdl 30 45 12 75 54 as sessions
insyra summary revenue
insyra summary sessions
```

Then run the Go program for Python enrichment + parallel batch computation.

## Where to go next

- [py](../py.md)
- [parallel](../parallel.md)
- [DataTable](../DataTable.md)
