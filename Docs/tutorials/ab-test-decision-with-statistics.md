# A/B Test Decision with Statistics

This tutorial shows how to move from raw experiment numbers to a decision using Insyra `DataList` and `stats`.

## What you will build

You will build a script that:

- creates synthetic A/B outcomes,
- runs a two-sample t-test,
- checks spend-vs-conversion correlation,
- computes lift and practical effect,
- builds a compact decision table,
- exports the result for reporting.

## Prerequisites

- Go 1.25+.
- Insyra + stats:

```bash
go get github.com/HazelnutParadise/insyra
go get github.com/HazelnutParadise/insyra/stats
```

## Scenario

A growth team tested new onboarding copy (Variant B).  
You must recommend ship/hold based on statistical signal and business impact.

## Step 1: Prepare synthetic test data

**Goal**  
Create control/variant conversion-rate samples and spend data.

**Code**

```go
control := insyra.NewDataList(0.08, 0.09, 0.07, 0.10, 0.09, 0.08, 0.09)
variant := insyra.NewDataList(0.10, 0.11, 0.09, 0.12, 0.10, 0.11, 0.10)
spend := insyra.NewDataList(120, 130, 110, 150, 140, 135, 145)
```

**Expected outcome**  
You have two experiment groups and one spend series.

## Step 2: Run two-sample t-test

**Goal**  
Test whether conversion means differ between groups.

**Code**

```go
tt, err := stats.TwoSampleTTest(control, variant, false, 0.95)
if err != nil {
	log.Fatal(err)
}
fmt.Printf("t-test p-value: %.6f\n", tt.PValue)
```

**Expected outcome**  
You get a valid p-value to support decisioning.

## Step 3: Check correlation with spend

**Goal**  
Evaluate whether higher spend is associated with stronger conversion outcomes.

**Code**

```go
corr, err := stats.Correlation(spend, variant, stats.PearsonCorrelation)
if err != nil {
	log.Fatal(err)
}
fmt.Printf("correlation: %.4f (p=%.6f)\n", corr.Statistic, corr.PValue)
```

**Expected outcome**  
You obtain correlation strength and significance.

## Step 4: Compute business lift

**Goal**  
Quantify relative conversion lift for product stakeholders.

**Code**

```go
controlMean := control.Mean()
variantMean := variant.Mean()
lift := (variantMean - controlMean) / controlMean
fmt.Printf("control=%.4f variant=%.4f lift=%.2f%%\n", controlMean, variantMean, lift*100)
```

**Expected outcome**  
You can communicate effect size as a percentage lift.

## Step 5: Build decision rule

**Goal**  
Turn statistics into a clear recommendation.

**Code**

```go
decision := "HOLD"
if tt.PValue < 0.05 && lift > 0 {
	decision = "SHIP_VARIANT_B"
}
fmt.Println("decision:", decision)
```

**Expected outcome**  
A deterministic recommendation is produced.

## Step 6: Export a decision table

**Goal**  
Save a compact decision artifact for experiment logs.

**Code**

```go
report := insyra.NewDataTable(
	insyra.NewDataList("ControlMean", "VariantMean", "LiftPct", "PValue", "Decision").SetName("Metric"),
	insyra.NewDataList(controlMean, variantMean, lift*100, tt.PValue, decision).SetName("Value"),
)
if err := report.ToCSV("ab_test_decision.csv", false, true, false); err != nil {
	log.Fatal(err)
}
```

**Expected outcome**  
`ab_test_decision.csv` is generated.

## Complete runnable Go program

```go
package main

import (
	"fmt"
	"log"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

func main() {
	control := insyra.NewDataList(0.08, 0.09, 0.07, 0.10, 0.09, 0.08, 0.09)
	variant := insyra.NewDataList(0.10, 0.11, 0.09, 0.12, 0.10, 0.11, 0.10)
	spend := insyra.NewDataList(120, 130, 110, 150, 140, 135, 145)

	tt, err := stats.TwoSampleTTest(control, variant, false, 0.95)
	if err != nil {
		log.Fatal(err)
	}

	corr, err := stats.Correlation(spend, variant, stats.PearsonCorrelation)
	if err != nil {
		log.Fatal(err)
	}

	controlMean := control.Mean()
	variantMean := variant.Mean()
	lift := (variantMean - controlMean) / controlMean

	decision := "HOLD"
	if tt.PValue < 0.05 && lift > 0 {
		decision = "SHIP_VARIANT_B"
	}

	fmt.Printf("t-test p-value: %.6f\n", tt.PValue)
	fmt.Printf("correlation: %.4f (p=%.6f)\n", corr.Statistic, corr.PValue)
	fmt.Printf("lift: %.2f%% decision: %s\n", lift*100, decision)

	report := insyra.NewDataTable(
		insyra.NewDataList("ControlMean", "VariantMean", "LiftPct", "PValue", "Decision").SetName("Metric"),
		insyra.NewDataList(controlMean, variantMean, lift*100, tt.PValue, decision).SetName("Value"),
	)
	if err := report.ToCSV("ab_test_decision.csv", false, true, false); err != nil {
		log.Fatal(err)
	}
}
```

## CLI/.isr equivalent workflow (appendix)

### One-shot CLI

```bash
insyra newdl 0.08 0.09 0.07 0.10 0.09 0.08 0.09 as control
insyra newdl 0.10 0.11 0.09 0.12 0.10 0.11 0.10 as variant
insyra ttest two control variant unequal
insyra ztest two control variant 0.02
insyra chisq gof variant 0.1,0.1,0.1,0.1,0.1,0.1,0.4
```

### `.isr` script

```text
newdl 0.08 0.09 0.07 0.10 0.09 0.08 0.09 as control
newdl 0.10 0.11 0.09 0.12 0.10 0.11 0.10 as variant
ttest two control variant unequal
ztest two control variant 0.02
```

## Where to go next

- [stats](../stats.md)
- [DataList](../DataList.md)
- [cli-dsl](../cli-dsl.md)
