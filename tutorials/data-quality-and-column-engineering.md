# Data Quality and Column Engineering

This tutorial shows a practical data quality pipeline: clean messy sales records, engineer business columns with CCL, and export a production-ready table.

## What you will build

You will build a complete workflow that:

- creates a messy CSV dataset (self-contained),
- loads it into `*insyra.DataTable`,
- cleans values with `Replace` and `Filter`,
- creates new business columns via CCL,
- sorts and validates output,
- exports a cleaned CSV.

## Prerequisites

- Go 1.25+.
- Insyra installed:

```bash
go get github.com/HazelnutParadise/insyra
```

## Scenario

You receive order data with placeholders (`N/A`), wrong categories (`UNKNOWN`), and inconsistent rows.  
You need one repeatable script to produce analysis-ready data for downstream reporting.

## Step 1: Prepare a messy CSV dataset

**Goal**  
Create a small dataset with realistic quality issues.

**Code**

```go
csv := `OrderID,Region,Product,Quantity,UnitPrice,DiscountRate
2001,North,Laptop,2,1200,0.05
2002,UNKNOWN,Mouse,10,25,0
2003,East,Monitor,N/A,300,0.10
2004,West,Keyboard,5,N/A,0.05
2005,South,Dock,4,150,0.00
2006,North,Laptop,1,1250,0.08
`
_ = os.WriteFile("dq_raw.csv", []byte(csv), 0644)
```

**Expected outcome**  
`dq_raw.csv` is created with intentional missing/invalid fields.

## Step 2: Load into DataTable

**Goal**  
Load and preview the dataset before cleaning.

**Code**

```go
dt, err := insyra.ReadCSV_File("dq_raw.csv", false, true)
if err != nil {
	log.Fatal(err)
}
dt.ShowRange(6)
```

**Expected outcome**  
You can see rows with `UNKNOWN` and `N/A` values.

## Step 3: Clean values and create a Filter-based quality view

**Goal**  
Normalize placeholder values and create a quick filtered view of suspicious rows.

**Code**

```go
dt.Replace("N/A", 0)
qualityView := dt.Filter(func(rowIndex int, columnIndex string, value any) bool {
	return columnIndex == "B" && (value == "UNKNOWN" || value == nil)
})
qualityView.ShowRange(5)
```

**Expected outcome**  
`N/A` values are normalized, and `qualityView` highlights rows with suspicious `Region` values.

## Step 4: Add engineered columns using CCL

**Goal**  
Add derived business fields.

**Code**

```go
dt.AddColUsingCCL("GrossSales", "['Quantity'] * ['UnitPrice']")
dt.AddColUsingCCL("NetSales", "['GrossSales'] * (1 - ['DiscountRate'])")
dt.AddColUsingCCL("OrderTag", "IF(['NetSales'] > 1000, 'HighValue', 'Standard')")
```

**Expected outcome**  
The table gains `GrossSales`, `NetSales`, and `OrderTag`.

## Step 5: Sort and validate key output

**Goal**  
Sort by net sales to validate engineered results quickly.

**Code**

```go
dt.SortBy(insyra.DataTableSortConfig{
	ColumnName: "NetSales",
	Descending: true,
})
dt.ShowRange(5)
```

**Expected outcome**  
Highest-value transactions appear first.

## Step 6: Export the cleaned table

**Goal**  
Persist the final table for downstream use.

**Code**

```go
if err := dt.ToCSV("dq_clean.csv", false, true, false); err != nil {
	log.Fatal(err)
}
```

**Expected outcome**  
`dq_clean.csv` is generated with cleaned and engineered columns.

## Complete runnable Go program

```go
package main

import (
	"log"
	"os"

	"github.com/HazelnutParadise/insyra"
)

func main() {
	csv := `OrderID,Region,Product,Quantity,UnitPrice,DiscountRate
2001,North,Laptop,2,1200,0.05
2002,UNKNOWN,Mouse,10,25,0
2003,East,Monitor,N/A,300,0.10
2004,West,Keyboard,5,N/A,0.05
2005,South,Dock,4,150,0.00
2006,North,Laptop,1,1250,0.08
`
	if err := os.WriteFile("dq_raw.csv", []byte(csv), 0644); err != nil {
		log.Fatal(err)
	}

	dt, err := insyra.ReadCSV_File("dq_raw.csv", false, true)
	if err != nil {
		log.Fatal(err)
	}

	dt.Replace("N/A", 0)
	qualityView := dt.Filter(func(rowIndex int, columnIndex string, value any) bool {
		return columnIndex == "B" && (value == "UNKNOWN" || value == nil)
	})
	qualityView.ShowRange(5)

	dt.AddColUsingCCL("GrossSales", "['Quantity'] * ['UnitPrice']")
	dt.AddColUsingCCL("NetSales", "['GrossSales'] * (1 - ['DiscountRate'])")
	dt.AddColUsingCCL("OrderTag", "IF(['NetSales'] > 1000, 'HighValue', 'Standard')")

	dt.SortBy(insyra.DataTableSortConfig{ColumnName: "NetSales", Descending: true})
	dt.ShowRange(5)

	if err := dt.ToCSV("dq_clean.csv", false, true, false); err != nil {
		log.Fatal(err)
	}
}
```

## CLI/.isr equivalent workflow (appendix)

### One-shot CLI

```bash
insyra --env dq load dq_raw.csv as t
insyra --env dq addcolccl t GrossSales "['Quantity'] * ['UnitPrice']"
insyra --env dq addcolccl t NetSales "['GrossSales'] * (1 - ['DiscountRate'])"
insyra --env dq addcolccl t OrderTag "IF(['NetSales'] > 1000, 'HighValue', 'Standard')"
insyra --env dq sort t NetSales desc
insyra --env dq summary t
insyra --env dq save t dq_clean.csv
```

### `.isr` script

```text
load dq_raw.csv as t
addcolccl t GrossSales "['Quantity'] * ['UnitPrice']"
addcolccl t NetSales "['GrossSales'] * (1 - ['DiscountRate'])"
addcolccl t OrderTag "IF(['NetSales'] > 1000, 'HighValue', 'Standard')"
sort t NetSales desc
summary t
save t dq_clean.csv
```

## Where to go next

- [DataTable](../DataTable.md)
- [CCL](../CCL.md)
- [CLI + DSL Guide](../cli-dsl.md)
