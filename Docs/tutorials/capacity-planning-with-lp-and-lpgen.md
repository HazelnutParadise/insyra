# Capacity Planning with lp and lpgen

This tutorial builds a linear-programming capacity plan with `lpgen`, solves it with `lp`, and exports the optimization result.

## What you will build

You will build a model that:

- defines production decision variables,
- sets objective and constraints,
- writes an LP file,
- solves the model with `lp`'s default engine, which needs nothing installed,
- reads the solution status, total margin and production quantities,
- exports optimization outputs.

## Prerequisites

- Go 1.26+.
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
Get the optimal production quantities.

**Code**

```go
sol, err := lp.Solve(model, lp.Options{TimeLimit: 30 * time.Second})
if err != nil {
	log.Fatal(err)
}
if sol.Status != lp.StatusOptimal {
	log.Fatalf("no optimal plan: %v", sol.Status)
}
```

**Expected outcome**  
`err` is nil and `sol.Status` is `lp.StatusOptimal`. An error means the model could not be solved as written, for example because of a typo in a constraint. A model with no feasible plan returns a nil error with `sol.Status` set to `lp.StatusInfeasible`.

## Step 4: Inspect optimization outputs

**Goal**  
Understand selected quantities and objective result.

**Code**

```go
fmt.Printf("total margin: %g\n", sol.Objective)
solution := sol.ToDataTable()
solution.ShowRange(20)
```

**Expected outcome**  
The total margin is 4650, and the table shows `x1` = 30 and `x2` = 45 in its `Variable` and `Value` columns. Both machine hours (2 × 30 + 4 × 45 = 240) and labor hours (3 × 30 + 2 × 45 = 180) are fully used.

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
_ = summary.ToCSV("capacity_summary.csv", false, true, false)
```

**Expected outcome**  
Two CSV files are generated: the production quantities and the business summary.

## Complete runnable Go program

```go
package main

import (
	"fmt"
	"log"
	"time"

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

	sol, err := lp.Solve(model, lp.Options{TimeLimit: 30 * time.Second})
	if err != nil {
		log.Fatal(err)
	}
	if sol.Status != lp.StatusOptimal {
		log.Fatalf("no optimal plan: %v", sol.Status)
	}
	fmt.Printf("total margin: %g\n", sol.Objective)

	solution := sol.ToDataTable()
	summary := insyra.NewDataTable(
		insyra.NewDataList("Decision", "Description").SetName("Field"),
		insyra.NewDataList("CapacityPlan", "Optimal production under labor/machine constraints").SetName("Value"),
	)

	_ = solution.ToCSV("capacity_solution.csv", false, true, false)
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
