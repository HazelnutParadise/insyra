# Market Trend Forecasting with Yahoo Finance

This tutorial combines online market data (`datafetch.YFinance`) with an offline fallback path, then fits a simple trend model and saves visual output.

## What you will build

You will build a workflow that:

- fetches recent price history from Yahoo Finance,
- falls back to local CSV when network fetch fails,
- engineers daily return with CCL,
- fits linear regression trend,
- renders a trend chart,
- exports enriched price data.

## Prerequisites

- Go 1.25+.
- Insyra packages:

```bash
go get github.com/HazelnutParadise/insyra
go get github.com/HazelnutParadise/insyra/datafetch
go get github.com/HazelnutParadise/insyra/stats
go get github.com/HazelnutParadise/insyra/plot
```

## Scenario

You need a lightweight trend briefing for one ticker.  
The pipeline must still run in offline environments by using a local snapshot.

## Step 1: Try online fetch from Yahoo Finance

**Goal**  
Get recent bars using `YFinance -> Ticker -> History`.

**Code**

```go
yf, err := datafetch.YFinance(datafetch.YFinanceConfig{})
if err != nil {
	log.Fatal(err)
}

prices, err := yf.Ticker("AAPL").History(datafetch.YFHistoryParams{
	Period:   "3mo",
	Interval: "1d",
})
```

**Expected outcome**  
You either get `prices` from Yahoo or an error for fallback handling.

## Step 2: Add offline fallback snapshot

**Goal**  
Guarantee the tutorial runs even without network.

**Code**

```go
if err != nil || prices == nil {
	snapshot := `Date,Close
2025-01-02,189.2
2025-01-03,190.1
2025-01-06,191.4
2025-01-07,190.8
2025-01-08,192.5
2025-01-09,193.0
`
	_ = os.WriteFile("aapl_snapshot.csv", []byte(snapshot), 0644)
	prices, err = insyra.ReadCSV_File("aapl_snapshot.csv", false, true)
	if err != nil {
		log.Fatal(err)
	}
}
```

**Expected outcome**  
`prices` is always available (online or offline path).

## Step 3: Engineer return features

**Goal**  
Create a simple engineered close-price feature.

**Code**

```go
var closeExpr string
var closeDL *insyra.DataList
for _, name := range prices.ColNames() {
	if strings.EqualFold(name, "close") {
		closeExpr = "['" + name + "']"
		closeDL = prices.GetColByName(name)
		break
	}
}
if closeDL == nil {
	log.Fatalf("Close column missing. available columns: %v", prices.ColNames())
}

prices.AddColUsingCCL("CloseScaled", closeExpr+" / 100")
prices.AddColUsingCCL("CloseSquared", closeExpr+" * "+closeExpr)
prices.ShowRange(8)
```

**Expected outcome**  
`CloseScaled` and `CloseSquared` columns are added.

## Step 4: Fit a trend line with linear regression

**Goal**  
Estimate slope and R-squared for close-price trend.

**Code**

```go
closeDL.SetName("Close")

x := insyra.NewDataList()
for i := 0; i < closeDL.Len(); i++ {
	x.Append(float64(i + 1))
}

lr, err := stats.LinearRegression(closeDL, x)
if err != nil {
	log.Fatal(err)
}
fmt.Printf("slope=%.6f r2=%.4f\n", lr.Slope, lr.RSquared)
```

**Expected outcome**  
You get a numeric slope and fit quality.

## Step 5: Render an interactive chart

**Goal**  
Generate an HTML chart for stakeholder review.

**Code**

```go
closeDL.SetName("Close")
line := plot.CreateLineChart(plot.LineChartConfig{
	Title: "AAPL Close Trend",
}, closeDL)
if err := plot.SaveHTML(line, "aapl_trend.html"); err != nil {
	log.Fatal(err)
}
```

**Expected outcome**  
`aapl_trend.html` is generated.

## Step 6: Export enriched data

**Goal**  
Save transformed dataset for further analysis.

**Code**

```go
if err := prices.ToCSV("aapl_enriched.csv", false, true, false); err != nil {
	log.Fatal(err)
}
```

**Expected outcome**  
`aapl_enriched.csv` is generated.

## Complete runnable Go program

```go
package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/datafetch"
	"github.com/HazelnutParadise/insyra/plot"
	"github.com/HazelnutParadise/insyra/stats"
)

func main() {
	yf, err := datafetch.YFinance(datafetch.YFinanceConfig{})
	if err != nil {
		log.Fatal(err)
	}

	prices, err := yf.Ticker("AAPL").History(datafetch.YFHistoryParams{
		Period:   "3mo",
		Interval: "1d",
	})
	if err != nil || prices == nil {
		snapshot := `Date,Close
2025-01-02,189.2
2025-01-03,190.1
2025-01-06,191.4
2025-01-07,190.8
2025-01-08,192.5
2025-01-09,193.0
`
		if err := os.WriteFile("aapl_snapshot.csv", []byte(snapshot), 0644); err != nil {
			log.Fatal(err)
		}
		prices, err = insyra.ReadCSV_File("aapl_snapshot.csv", false, true)
		if err != nil {
			log.Fatal(err)
		}
	}

	var closeExpr string
	var closeDL *insyra.DataList
	for _, name := range prices.ColNames() {
		if strings.EqualFold(name, "close") {
			closeExpr = "['" + name + "']"
			closeDL = prices.GetColByName(name)
			break
		}
	}
	if closeDL == nil {
		log.Fatalf("Close column missing. available columns: %v", prices.ColNames())
	}

	prices.AddColUsingCCL("CloseScaled", closeExpr+" / 100")
	prices.AddColUsingCCL("CloseSquared", closeExpr+" * "+closeExpr)

	closeDL.SetName("Close")
	x := insyra.NewDataList()
	for i := 0; i < closeDL.Len(); i++ {
		x.Append(float64(i + 1))
	}

	lr, err := stats.LinearRegression(closeDL, x)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("slope=%.6f r2=%.4f\n", lr.Slope, lr.RSquared)

	closeDL.SetName("Close")
	line := plot.CreateLineChart(plot.LineChartConfig{Title: "AAPL Close Trend"}, closeDL)
	if err := plot.SaveHTML(line, "aapl_trend.html"); err != nil {
		log.Fatal(err)
	}

	if err := prices.ToCSV("aapl_enriched.csv", false, true, false); err != nil {
		log.Fatal(err)
	}
}
```

## CLI/.isr equivalent workflow (appendix)

### One-shot CLI

```bash
insyra fetch yahoo AAPL history period=3mo interval=1d as px
insyra addcolccl px CloseScaled "['close'] / 100"
insyra addcolccl px CloseSquared "['close'] * ['close']"
insyra regression linear Close px_index as trend
insyra plot line px save aapl_trend.html
insyra save px aapl_enriched.csv
```

### `.isr` script

```text
fetch yahoo AAPL history period=3mo interval=1d as px
addcolccl px CloseScaled "['close'] / 100"
addcolccl px CloseSquared "['close'] * ['close']"
plot line px save aapl_trend.html
save px aapl_enriched.csv
```

## Where to go next

- [datafetch](../datafetch.md)
- [stats](../stats.md)
- [plot](../plot.md)
