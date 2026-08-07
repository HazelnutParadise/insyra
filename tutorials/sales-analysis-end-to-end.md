# Sales Analysis End-to-End

This tutorial walks through a complete sales analysis workflow with Insyra, from raw CSV data to an enriched output table.

## What you will build

By the end of this guide, you will have a runnable pipeline that:

- writes a small sales dataset to CSV (self-contained setup),
- loads it into `*insyra.DataTable`,
- adds business metrics with CCL,
- sorts and previews top transactions,
- computes headline metrics with `DataList` methods,
- exports an enriched CSV for downstream reporting.

## Prerequisites

- Go 1.25+ (per this repository's `go.mod`)
- Insyra installed in your module:

```bash
go get github.com/HazelnutParadise/insyra
```

## Scenario

You have raw order-level sales data and want a quick repeatable script to answer:

- Which transactions drive the most net sales?
- What is total net sales and average order value?
- How much total profit did we generate?

You also want the cleaned/enriched result saved as a new CSV.

## Step 1: Prepare a self-contained CSV dataset

**Goal**  
Create a local `sales.csv` file directly from code, so anyone can run this tutorial without external files.

**Code**

```go
package main

import "os"

func main() {
	csv := `OrderID,Region,Product,Quantity,UnitPrice,DiscountRate,Cost
1001,North,Laptop,2,1200,0.05,1800
1002,South,Mouse,10,25,0,140
1003,East,Monitor,3,300,0.10,700
1004,West,Keyboard,5,80,0.05,250
1005,North,Laptop,1,1250,0.08,900
1006,South,Dock,4,150,0.00,420
1007,East,Headset,6,60,0.15,250
1008,West,Monitor,2,320,0.05,480
`
	_ = os.WriteFile("sales.csv", []byte(csv), 0644)
}
```

**Expected outcome**  
A file named `sales.csv` exists in your working directory with 8 data rows.

## Step 2: Load data into DataTable

**Goal**  
Read `sales.csv` into `*insyra.DataTable` and check the shape.

**Code**

```go
package main

import (
	"fmt"
	"log"

	"github.com/HazelnutParadise/insyra"
)

func main() {
	dt, err := insyra.ReadCSV_File("sales.csv", false, true)
	if err != nil {
		log.Fatal(err)
	}

	rows, cols := dt.Size()
	fmt.Printf("Loaded rows=%d cols=%d\n", rows, cols)
	dt.ShowRange(5)
}
```

**Expected outcome**  
You should see `rows=8 cols=7` and a preview of the first transactions.

## Step 3: Add business columns with CCL

**Goal**  
Create derived metrics directly in the table:

- `GrossSales = Quantity * UnitPrice`
- `NetSales = GrossSales * (1 - DiscountRate)`
- `Profit = NetSales - Cost`

**Code**

```go
dt.AddColUsingCCL("GrossSales", "['Quantity'] * ['UnitPrice']")
dt.AddColUsingCCL("NetSales", "['GrossSales'] * (1 - ['DiscountRate'])")
dt.AddColUsingCCL("Profit", "['NetSales'] - ['Cost']")

rows, cols := dt.Size()
fmt.Printf("After enrichment rows=%d cols=%d\n", rows, cols)
dt.ShowRange(5)
```

**Expected outcome**  
Your table now has 10 columns (the original 7 plus 3 derived business columns).

## Step 4: Sort and inspect top transactions

**Goal**  
Sort the table by `NetSales` descending to identify top transactions.

**Code**

```go
dt.SortBy(insyra.DataTableSortConfig{
	ColumnName: "NetSales",
	Descending: true,
})

dt.ShowRange(5)
```

**Expected outcome**  
Top rows should start with the highest net sales orders (for this dataset: `OrderID=1001`, then `1005`).

## Step 5: Compute headline metrics

**Goal**  
Use `GetColByName` + `DataList` aggregation methods to compute key KPIs.

**Code**

```go
netSales := dt.GetColByName("NetSales")
profit := dt.GetColByName("Profit")
if netSales == nil || profit == nil {
	log.Fatal("required derived columns not found")
}

totalNetSales := netSales.Sum()
avgOrderValue := netSales.Mean()
totalProfit := profit.Sum()

fmt.Printf("Total Net Sales: %.2f\n", totalNetSales)
fmt.Printf("Average Order Value: %.2f\n", avgOrderValue)
fmt.Printf("Total Profit: %.2f\n", totalProfit)
```

**Expected outcome**  
For this sample dataset:

- `Total Net Sales` is `6384.00`
- `Average Order Value` is `798.00`
- `Total Profit` is `1444.00`

## Step 6: Export the enriched table

**Goal**  
Save the transformed dataset to a new CSV for sharing or downstream use.

**Code**

```go
if err := dt.ToCSV("sales_enriched.csv", false, true, false); err != nil {
	log.Fatal(err)
}
fmt.Println("Wrote sales_enriched.csv")
```

**Expected outcome**  
You get a new file `sales_enriched.csv` containing original fields plus `GrossSales`, `NetSales`, and `Profit`.

## Complete runnable Go program

```go
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/HazelnutParadise/insyra"
)

func main() {
	csv := `OrderID,Region,Product,Quantity,UnitPrice,DiscountRate,Cost
1001,North,Laptop,2,1200,0.05,1800
1002,South,Mouse,10,25,0,140
1003,East,Monitor,3,300,0.10,700
1004,West,Keyboard,5,80,0.05,250
1005,North,Laptop,1,1250,0.08,900
1006,South,Dock,4,150,0.00,420
1007,East,Headset,6,60,0.15,250
1008,West,Monitor,2,320,0.05,480
`
	if err := os.WriteFile("sales.csv", []byte(csv), 0644); err != nil {
		log.Fatal(err)
	}

	dt, err := insyra.ReadCSV_File("sales.csv", false, true)
	if err != nil {
		log.Fatal(err)
	}

	dt.AddColUsingCCL("GrossSales", "['Quantity'] * ['UnitPrice']")
	dt.AddColUsingCCL("NetSales", "['GrossSales'] * (1 - ['DiscountRate'])")
	dt.AddColUsingCCL("Profit", "['NetSales'] - ['Cost']")

	dt.SortBy(insyra.DataTableSortConfig{
		ColumnName: "NetSales",
		Descending: true,
	})

	netSales := dt.GetColByName("NetSales")
	profit := dt.GetColByName("Profit")
	if netSales == nil || profit == nil {
		log.Fatal("required derived columns not found")
	}

	fmt.Printf("Total Net Sales: %.2f\n", netSales.Sum())
	fmt.Printf("Average Order Value: %.2f\n", netSales.Mean())
	fmt.Printf("Total Profit: %.2f\n", profit.Sum())

	dt.ShowRange(5)

	if err := dt.ToCSV("sales_enriched.csv", false, true, false); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Wrote sales_enriched.csv")
}
```

## CLI/.isr equivalent workflow (appendix)

The same flow can be run with Insyra CLI commands.

1. Create `sales.csv` (same content as above).
2. Run commands in one-shot mode or via `.isr`.

### One-shot CLI commands

```bash
insyra --env sales-tutorial load sales.csv as sales
insyra --env sales-tutorial addcolccl sales GrossSales "['Quantity'] * ['UnitPrice']"
insyra --env sales-tutorial addcolccl sales NetSales "['GrossSales'] * (1 - ['DiscountRate'])"
insyra --env sales-tutorial addcolccl sales Profit "['NetSales'] - ['Cost']"
insyra --env sales-tutorial sort sales NetSales desc
insyra --env sales-tutorial summary sales
insyra --env sales-tutorial save sales sales_enriched.csv
```

### `.isr` script version

Create `sales_pipeline.isr`:

```text
load sales.csv as sales
addcolccl sales GrossSales "['Quantity'] * ['UnitPrice']"
addcolccl sales NetSales "['GrossSales'] * (1 - ['DiscountRate'])"
addcolccl sales Profit "['NetSales'] - ['Cost']"
sort sales NetSales desc
summary sales
save sales sales_enriched.csv
```

Run it:

```bash
insyra --env sales-tutorial run sales_pipeline.isr
```

## Where to go next

- [CCL](../CCL.md): learn more expression patterns and statement mode.
- [DataTable](../DataTable.md): table operations, filtering, merging, and export methods.
- [CLI + DSL Guide](../cli-dsl.md): complete CLI/REPL/`.isr` command workflow.
