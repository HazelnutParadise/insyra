# Interactive KPI Dashboard with plot

This tutorial builds an interactive KPI dashboard using Insyra `plot` package and exports both HTML and PNG artifacts.

## What you will build

You will create:

- KPI DataLists for monthly metrics,
- a line chart (revenue trend),
- a bar chart (new customers),
- exported interactive HTML charts,
- exported PNG chart snapshots,
- a KPI summary CSV.

## Prerequisites

- Go 1.25+.
- Insyra + plot:

```bash
go get github.com/HazelnutParadise/insyra
go get github.com/HazelnutParadise/insyra/plot
```

## Scenario

A weekly business review needs a lightweight dashboard artifact that can be opened in a browser and attached to a report.

## Step 1: Prepare KPI series

**Goal**  
Define monthly KPI data as named DataLists.

**Code**

```go
months := []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun"}
revenue := insyra.NewDataList(12000, 13200, 14100, 15300, 16000, 17200).SetName("Revenue")
customers := insyra.NewDataList(220, 245, 260, 280, 295, 320).SetName("NewCustomers")
```

**Expected outcome**  
You have reusable DataLists for chart generation.

## Step 2: Create revenue line chart

**Goal**  
Generate an interactive line chart for revenue trend.

**Code**

```go
revLine := plot.CreateLineChart(plot.LineChartConfig{
	Title: "Monthly Revenue Trend",
	XAxis: months,
}, revenue)
```

**Expected outcome**  
A `*charts.Line` chart object is created.

## Step 3: Create customer bar chart

**Goal**  
Generate an interactive bar chart for acquisition.

**Code**

```go
custBar := plot.CreateBarChart(plot.BarChartConfig{
	Title: "Monthly New Customers",
	XAxis: months,
}, customers)
```

**Expected outcome**  
A `*charts.Bar` chart object is created.

## Step 4: Save interactive HTML files

**Goal**  
Publish browser-ready chart files.

**Code**

```go
if err := plot.SaveHTML(revLine, "kpi_revenue.html"); err != nil {
	log.Fatal(err)
}
if err := plot.SaveHTML(custBar, "kpi_customers.html"); err != nil {
	log.Fatal(err)
}
```

**Expected outcome**  
`kpi_revenue.html` and `kpi_customers.html` are generated.

## Step 5: Save PNG snapshots

**Goal**  
Generate static images for documents/slides.

**Code**

```go
if err := plot.SavePNG(revLine, "kpi_revenue.png"); err != nil {
	log.Println("revenue png failed:", err)
}
if err := plot.SavePNG(custBar, "kpi_customers.png"); err != nil {
	log.Println("customers png failed:", err)
}
```

**Expected outcome**  
PNG snapshots are generated (or logged if runtime lacks screenshot backend).

## Step 6: Export KPI summary table

**Goal**  
Save a compact KPI summary CSV.

**Code**

```go
summary := insyra.NewDataTable(
	insyra.NewDataList("RevenueTotal", "RevenueMean", "CustomerTotal", "CustomerMean").SetName("Metric"),
	insyra.NewDataList(revenue.Sum(), revenue.Mean(), customers.Sum(), customers.Mean()).SetName("Value"),
)
if err := summary.ToCSV("kpi_summary.csv", false, true, false); err != nil {
	log.Fatal(err)
}
```

**Expected outcome**  
`kpi_summary.csv` is generated.

## Complete runnable Go program

```go
package main

import (
	"log"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/plot"
)

func main() {
	months := []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun"}
	revenue := insyra.NewDataList(12000, 13200, 14100, 15300, 16000, 17200).SetName("Revenue")
	customers := insyra.NewDataList(220, 245, 260, 280, 295, 320).SetName("NewCustomers")

	revLine := plot.CreateLineChart(plot.LineChartConfig{
		Title: "Monthly Revenue Trend",
		XAxis: months,
	}, revenue)
	custBar := plot.CreateBarChart(plot.BarChartConfig{
		Title: "Monthly New Customers",
		XAxis: months,
	}, customers)

	if err := plot.SaveHTML(revLine, "kpi_revenue.html"); err != nil {
		log.Fatal(err)
	}
	if err := plot.SaveHTML(custBar, "kpi_customers.html"); err != nil {
		log.Fatal(err)
	}

	_ = plot.SavePNG(revLine, "kpi_revenue.png")
	_ = plot.SavePNG(custBar, "kpi_customers.png")

	summary := insyra.NewDataTable(
		insyra.NewDataList("RevenueTotal", "RevenueMean", "CustomerTotal", "CustomerMean").SetName("Metric"),
		insyra.NewDataList(revenue.Sum(), revenue.Mean(), customers.Sum(), customers.Mean()).SetName("Value"),
	)
	if err := summary.ToCSV("kpi_summary.csv", false, true, false); err != nil {
		log.Fatal(err)
	}
}
```

## CLI/.isr equivalent workflow (appendix)

### One-shot CLI

```bash
insyra newdl 12000 13200 14100 15300 16000 17200 as revenue
insyra newdl 220 245 260 280 295 320 as customers
insyra plot line revenue save kpi_revenue.html
insyra plot bar customers save kpi_customers.html
insyra summary revenue
insyra summary customers
```

### `.isr` script

```text
newdl 12000 13200 14100 15300 16000 17200 as revenue
newdl 220 245 260 280 295 320 as customers
plot line revenue save kpi_revenue.html
plot bar customers save kpi_customers.html
summary revenue
summary customers
```

## Where to go next

- [plot](../plot.md)
- [DataList](../DataList.md)
- [cli-dsl](../cli-dsl.md)
