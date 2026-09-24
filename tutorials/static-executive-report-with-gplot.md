# Static Executive Report with gplot

This tutorial focuses on static chart assets for reports and slides using Insyra `gplot`.

## What you will build

You will build a static reporting pipeline that:

- creates KPI DataLists,
- renders line/bar/histogram charts,
- saves PNG chart files,
- exports summary values to CSV.

## Prerequisites

- Go 1.25+.
- Insyra + gplot:

```bash
go get github.com/HazelnutParadise/insyra
go get github.com/HazelnutParadise/insyra/gplot
```

## Scenario

You need non-interactive chart assets for PDF reports and board slides where HTML dashboards are not preferred.

## Step 1: Prepare reporting data

**Goal**  
Create the KPI series used in charts.

**Code**

```go
monthlyRevenue := insyra.NewDataList(10.2, 10.8, 11.5, 12.3, 13.1, 13.7).SetName("Revenue_M")
monthlyCost := insyra.NewDataList(6.1, 6.4, 6.8, 7.1, 7.3, 7.6).SetName("Cost_M")
```

**Expected outcome**  
Two named DataLists are ready for plotting.

## Step 2: Create and save a line chart

**Goal**  
Visualize revenue and cost trends together.

**Code**

```go
line := gplot.CreateLineChart(gplot.LineChartConfig{
	Title: "Revenue vs Cost Trend",
}, []insyra.IDataList{monthlyRevenue, monthlyCost})
gplot.SaveChart(line, "exec_trend.png")
```

**Expected outcome**  
`exec_trend.png` is generated.

## Step 3: Create and save a bar chart

**Goal**  
Compare monthly gross margin in bar format.

**Code**

```go
margin := insyra.NewDataList(4.1, 4.4, 4.7, 5.2, 5.8, 6.1).SetName("GrossMargin_M")
bar := gplot.CreateBarChart(gplot.BarChartConfig{
	Title: "Gross Margin by Month",
	XAxis: []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun"},
}, margin)
gplot.SaveChart(bar, "exec_margin.png")
```

**Expected outcome**  
`exec_margin.png` is generated.

## Step 4: Create and save a histogram

**Goal**  
Inspect the distribution of transaction values.

**Code**

```go
txn := insyra.NewDataList(120, 180, 200, 210, 260, 310, 340, 360, 390, 420).SetName("TxnValue")
hist := gplot.CreateHistogram(gplot.HistogramConfig{
	Title: "Transaction Distribution",
	Bins:  6,
}, txn)
gplot.SaveChart(hist, "exec_distribution.png")
```

**Expected outcome**  
`exec_distribution.png` is generated.

## Step 5: Build summary table

**Goal**  
Create a compact KPI summary for report appendix.

**Code**

```go
summary := insyra.NewDataTable(
	insyra.NewDataList("RevenueMean", "CostMean", "MarginMean").SetName("Metric"),
	insyra.NewDataList(monthlyRevenue.Mean(), monthlyCost.Mean(), margin.Mean()).SetName("Value"),
)
summary.ShowRange(10)
```

**Expected outcome**  
A small summary table is displayed and ready to export.

## Step 6: Export summary CSV

**Goal**  
Persist report summary in tabular form.

**Code**

```go
if err := summary.ToCSV("exec_summary.csv", false, true, false); err != nil {
	log.Fatal(err)
}
```

**Expected outcome**  
`exec_summary.csv` is generated.

## Complete runnable Go program

```go
package main

import (
	"log"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/gplot"
)

func main() {
	monthlyRevenue := insyra.NewDataList(10.2, 10.8, 11.5, 12.3, 13.1, 13.7).SetName("Revenue_M")
	monthlyCost := insyra.NewDataList(6.1, 6.4, 6.8, 7.1, 7.3, 7.6).SetName("Cost_M")
	margin := insyra.NewDataList(4.1, 4.4, 4.7, 5.2, 5.8, 6.1).SetName("GrossMargin_M")
	txn := insyra.NewDataList(120, 180, 200, 210, 260, 310, 340, 360, 390, 420).SetName("TxnValue")

	line := gplot.CreateLineChart(gplot.LineChartConfig{Title: "Revenue vs Cost Trend"}, []insyra.IDataList{monthlyRevenue, monthlyCost})
	gplot.SaveChart(line, "exec_trend.png")

	bar := gplot.CreateBarChart(gplot.BarChartConfig{
		Title: "Gross Margin by Month",
		XAxis: []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun"},
	}, margin)
	gplot.SaveChart(bar, "exec_margin.png")

	hist := gplot.CreateHistogram(gplot.HistogramConfig{Title: "Transaction Distribution", Bins: 6}, txn)
	gplot.SaveChart(hist, "exec_distribution.png")

	summary := insyra.NewDataTable(
		insyra.NewDataList("RevenueMean", "CostMean", "MarginMean").SetName("Metric"),
		insyra.NewDataList(monthlyRevenue.Mean(), monthlyCost.Mean(), margin.Mean()).SetName("Value"),
	)
	if err := summary.ToCSV("exec_summary.csv", false, true, false); err != nil {
		log.Fatal(err)
	}
}
```

## CLI/.isr equivalent workflow (appendix)

`gplot` is currently a Go package workflow (no direct dedicated CLI chart command for this static API).

### Closest CLI preprocessing

```bash
insyra newdl 10.2 10.8 11.5 12.3 13.1 13.7 as revenue
insyra newdl 6.1 6.4 6.8 7.1 7.3 7.6 as cost
insyra summary revenue
insyra summary cost
```

Then run the Go program to generate static PNG report assets.

## Where to go next

- [gplot](../gplot.md)
- [plot](../plot.md)
- [DataList](../DataList.md)
