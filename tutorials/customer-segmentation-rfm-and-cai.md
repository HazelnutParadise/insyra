# Customer Segmentation with RFM and CAI

This tutorial shows how to segment customers with Insyra marketing analytics features (`mkt.RFM` and `mkt.CustomerActivityIndex`).

## What you will build

You will build a workflow that:

- creates transaction history data,
- computes RFM scores,
- computes CAI (Customer Activity Index),
- identifies priority customers,
- exports segmentation outputs.

## Prerequisites

- Go 1.25+.
- Insyra + mkt:

```bash
go get github.com/HazelnutParadise/insyra
go get github.com/HazelnutParadise/insyra/mkt
```

## Scenario

A CRM team needs two views:

- static value segmentation (RFM),
- momentum of customer activity (CAI).

Both are needed to decide retention and upsell actions.

## Step 1: Prepare transaction data

**Goal**  
Create a self-contained transaction dataset.

**Code**

```go
csv := `CustomerID,TradingDay,Amount
C001,2025-01-01,120
C001,2025-01-18,80
C001,2025-02-02,140
C001,2025-02-19,200
C002,2025-01-03,40
C002,2025-01-30,30
C002,2025-02-20,35
C003,2025-01-05,220
C003,2025-01-20,260
C003,2025-02-04,300
C003,2025-02-25,340
`
dt, err := insyra.ReadCSV_String(csv, false, true)
if err != nil {
	log.Fatal(err)
}
```

**Expected outcome**  
You have one DataTable with customer, date, and amount columns.

## Step 2: Compute RFM segmentation

**Goal**  
Assign customers into R/F/M score groups.

**Code**

```go
rfm := mkt.RFM(dt, mkt.RFMConfig{
	CustomerIDColName: "CustomerID",
	TradingDayColName: "TradingDay",
	AmountColName:     "Amount",
	NumGroups:         5,
	DateFormat:        "YYYY-MM-DD",
	TimeScale:         mkt.TimeScaleDaily,
})
if rfm == nil {
	log.Fatal("rfm failed")
}
rfm.ShowRange(10)
```

**Expected outcome**  
You get R/F/M scores plus combined RFM score per customer.

## Step 3: Compute CAI

**Goal**  
Measure customer activity trend over time.

**Code**

```go
cai := mkt.CustomerActivityIndex(dt, mkt.CAIConfig{
	CustomerIDColName: "CustomerID",
	TradingDayColName: "TradingDay",
	DateFormat:        "YYYY-MM-DD",
	TimeScale:         mkt.TimeScaleDaily,
})
if cai == nil {
	log.Fatal("cai failed")
}
cai.ShowRange(10)
```

**Expected outcome**  
You get CAI values showing momentum of each customer's activity.

## Step 4: Identify priority segments

**Goal**  
Quickly inspect who is high-value and who is declining.

**Code**

```go
rfm.SortBy(insyra.DataTableSortConfig{ColumnName: "RFM_Score", Descending: true})
cai.SortBy(insyra.DataTableSortConfig{ColumnName: "CAI", Descending: false})

rfm.ShowRange(5) // top value segment
cai.ShowRange(5) // most declining activity first
```

**Expected outcome**  
You get an actionable shortlist for retention and upsell.

## Step 5: Add simple campaign tags

**Goal**  
Attach immediate campaign tags to RFM output using CCL.

**Code**

```go
rfm.AddColUsingCCL("CampaignTag", "IF(['RFM_Score'] >= 12, 'VIP_Upsell', 'Nurture')")
rfm.ShowRange(10)
```

**Expected outcome**  
A `CampaignTag` column appears for campaign routing.

## Step 6: Export segmentation outputs

**Goal**  
Save RFM and CAI tables for CRM workflows.

**Code**

```go
if err := rfm.ToCSV("customer_rfm.csv", false, true, false); err != nil {
	log.Fatal(err)
}
if err := cai.ToCSV("customer_cai.csv", false, true, false); err != nil {
	log.Fatal(err)
}
```

**Expected outcome**  
`customer_rfm.csv` and `customer_cai.csv` are generated.

## Complete runnable Go program

```go
package main

import (
	"log"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/mkt"
)

func main() {
	csv := `CustomerID,TradingDay,Amount
C001,2025-01-01,120
C001,2025-01-18,80
C001,2025-02-02,140
C001,2025-02-19,200
C002,2025-01-03,40
C002,2025-01-30,30
C002,2025-02-20,35
C003,2025-01-05,220
C003,2025-01-20,260
C003,2025-02-04,300
C003,2025-02-25,340
`
	dt, err := insyra.ReadCSV_String(csv, false, true)
	if err != nil {
		log.Fatal(err)
	}

	rfm := mkt.RFM(dt, mkt.RFMConfig{
		CustomerIDColName: "CustomerID",
		TradingDayColName: "TradingDay",
		AmountColName:     "Amount",
		NumGroups:         5,
		DateFormat:        "YYYY-MM-DD",
		TimeScale:         mkt.TimeScaleDaily,
	})
	if rfm == nil {
		log.Fatal("rfm failed")
	}

	cai := mkt.CustomerActivityIndex(dt, mkt.CAIConfig{
		CustomerIDColName: "CustomerID",
		TradingDayColName: "TradingDay",
		DateFormat:        "YYYY-MM-DD",
		TimeScale:         mkt.TimeScaleDaily,
	})
	if cai == nil {
		log.Fatal("cai failed")
	}

	rfm.SortBy(insyra.DataTableSortConfig{ColumnName: "RFM_Score", Descending: true})
	cai.SortBy(insyra.DataTableSortConfig{ColumnName: "CAI", Descending: false})
	rfm.AddColUsingCCL("CampaignTag", "IF(['RFM_Score'] >= 12, 'VIP_Upsell', 'Nurture')")

	if err := rfm.ToCSV("customer_rfm.csv", false, true, false); err != nil {
		log.Fatal(err)
	}
	if err := cai.ToCSV("customer_cai.csv", false, true, false); err != nil {
		log.Fatal(err)
	}
}
```

## CLI/.isr equivalent workflow (appendix)

`mkt.RFM` and `mkt.CustomerActivityIndex` are Go package capabilities (no direct CLI command today).

### Closest CLI preprocessing flow

```bash
insyra load customer_txn.csv as t
insyra summary t
insyra save t customer_txn_clean.csv
```

Then run the Go program for RFM/CAI scoring.

## Where to go next

- [mkt](../mkt.md)
- [DataTable](../DataTable.md)
- [CCL](../CCL.md)
