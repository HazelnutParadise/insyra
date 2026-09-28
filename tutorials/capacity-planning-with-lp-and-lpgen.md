# Capacity Planning with lp and lpgen

This tutorial builds a linear-programming capacity plan with `lpgen`, solves it with `lp`, and exports the optimization result.

## What you will build

You will build a model that:

- defines production decision variables,
- sets objective and constraints,
- writes an LP file,
- solves the LP model with GLPK backend,
- reviews decision and meta tables,
- exports optimization outputs.

## Prerequisites

- Go 1.25+.
- Insyra + optimization packages:

```bash
go get github.com/HazelnutParadise/insyra
go get github.com/HazelnutParadise/insyra/lp
go get github.com/HazelnutParadise/insyra/lpgen
```

## Scenario

You have two product lines competing for limited machine and labor capacity, and need the highest-margin feasible production plan.

## Step 1: Define the optimization model

**Goal**  
Create LP objective, constraints, and bounds.

**Code**

```go
model := lpgen.NewLPModel()
model.SetObjective("Maximize", "50 x1 + 70 x2")
model.AddConstraint("2 x1 + 4 x2 <= 240") // machine hours
model.AddConstraint("3 x1 + 2 x2 <= 180") // labor hours
model.AddBound("0 <= x1 <= 80")
model.AddBound("0 <= x2 <= 70")
```

**Expected outcome**  
A complete LP model is assembled in memory.

## Step 2: Generate LP file for auditability

**Goal**  
Persist model text for versioning and review.

**Code**

```go
model.GenerateLPFile("capacity_plan.lp")
```

**Expected outcome**  
`capacity_plan.lp` is generated.

## Step 3: Solve the LP model

**Goal**  
Get optimal decision variables and solver metadata.

**Code**

```go
solution, meta := lp.SolveModel(model, 30)
if solution == nil || meta == nil {
	log.Fatal("solver failed")
}
```

**Expected outcome**  
Two DataTables are returned: solution table and meta table.

## Step 4: Inspect optimization outputs

**Goal**  
Understand selected quantities and objective result.

**Code**

```go
solution.ShowRange(20)
meta.ShowRange(20)
```

**Expected outcome**  
You can see selected values for `x1`, `x2`, and objective diagnostics.

## Step 5: Add a business interpretation table

**Goal**  
Translate LP output into a decision summary.

**Code**

```go
summary := insyra.NewDataTable(
	insyra.NewDataList("Decision", "Description").SetName("Field"),
	insyra.NewDataList("CapacityPlan", "Optimal production under labor/machine constraints").SetName("Value"),
)
summary.ShowRange(10)
```

**Expected outcome**  
A human-readable decision note is prepared.

## Step 6: Export optimization artifacts

**Goal**  
Persist LP outputs for downstream workflows.

**Code**

```go
_ = solution.ToCSV("capacity_solution.csv", false, true, false)
_ = meta.ToCSV("capacity_meta.csv", false, true, false)
_ = summary.ToCSV("capacity_summary.csv", false, true, false)
```

**Expected outcome**  
Three CSV files are generated for solver results and business summary.

## Complete runnable Go program

```go
package main

import (
	"log"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/lp"
	"github.com/HazelnutParadise/insyra/lpgen"
)

func main() {
	model := lpgen.NewLPModel()
	model.SetObjective("Maximize", "50 x1 + 70 x2")
	model.AddConstraint("2 x1 + 4 x2 <= 240")
	model.AddConstraint("3 x1 + 2 x2 <= 180")
	model.AddBound("0 <= x1 <= 80")
	model.AddBound("0 <= x2 <= 70")
	model.GenerateLPFile("capacity_plan.lp")

	solution, meta := lp.SolveModel(model, 30)
	if solution == nil || meta == nil {
		log.Fatal("solver failed")
	}

	summary := insyra.NewDataTable(
		insyra.NewDataList("Decision", "Description").SetName("Field"),
		insyra.NewDataList("CapacityPlan", "Optimal production under labor/machine constraints").SetName("Value"),
	)

	_ = solution.ToCSV("capacity_solution.csv", false, true, false)
	_ = meta.ToCSV("capacity_meta.csv", false, true, false)
	_ = summary.ToCSV("capacity_summary.csv", false, true, false)
}
```

## CLI/.isr equivalent workflow (appendix)

`lpgen`/`lp` optimization modeling is currently a Go package workflow (no direct dedicated CLI command for model building/solving).

### Closest CLI preprocessing

```bash
insyra newdl 50 70 as margin
insyra summary margin
```

Then run the Go program for full LP build/solve/export.

## Where to go next

- [lpgen](../lpgen.md)
- [lp](../lp.md)
- [DataTable](../DataTable.md)
