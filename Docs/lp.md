# [ lp ] Package

The `lp` package solves linear and mixed-integer programs. `Solve` and `SolveFile` use the pure-Go go-milp engine by default. GLPK can be selected explicitly with `Options{Engine: lp.EngineGLPK}`.

> [!NOTE]
>
> - The new `Solve` and `SolveFile` APIs do not install GLPK. The legacy `SolveFromFile` and `SolveModel` functions retain their automatic-install behavior.
> - For the explicit GLPK engine, set `GLPK_PATH` to the `glpsol` executable or its containing directory, or make `glpsol` available on `PATH`.

## Installation

To install the `lp` package, use the following command:

```bash
go get github.com/HazelnutParadise/insyra/lp
```

## Pure-Go Solver

```go
func Solve(model *lpgen.LPModel, opts lp.Options) (*lp.Solution, error)
func SolveFile(path string, opts lp.Options) (*lp.Solution, error)
```

The zero-value `Options{}` selects `EngineMILP`. Set `Options.Engine` to `EngineGLPK` to use an installed `glpsol`; `Options.TimeLimit` is a best-effort limit. With `EngineMILP`, it applies to branch search and may not cover the initial continuous solve, so total solve time can exceed it. Solver outcomes such as infeasible, unbounded, or stopped are returned as a `Solution` with nil error. Invalid input, an unavailable engine, or a solver failure returns an error.

`Solution` provides `Status`, `Objective`, `Values`, `Engine`, `Elapsed`, `Nodes`, and `Log`. `Solution.ToDataTable()` returns the variable/value rows in model order.

> [!NOTE]
> When the linear relaxation at a branch-and-bound node fails numerically, go-milp v0.2.0 skips that node and can still report the result as optimal, even though the skipped part of the search might have held a better solution ([go-milp#2](https://github.com/daniel-sullivan/go-milp/issues/2)). If an integer program's answer matters, solve it once more with `EngineGLPK` and compare the objectives.

## Features

### Solve From LP File

```go
func SolveFromFile(lpFile string, timeoutSeconds ...int) (*DataTable, *DataTable)
```

**Description:** Solves an LP file using GLPK and returns the result as two DataTable. First DataTable contains the solution, and the second one contains additional information.

**Parameters:**

- `lpFile string`: The path to the LP file.
- `timeoutSeconds int`(optional): The timeout for the solver in seconds.

**Returns:**

- `*DataTable`: The solution DataTable(the column name and the row name will not be set). It is `nil` when the solve does not produce a solution: a timeout, a solver error, a temporary file that could not be written, or arguments that were rejected. Check it before calling a method on it.
- `*DataTable`: The additional information DataTable(the column name and the row name will be set). Rows are always in the order Status, Execution Time, Warnings, Full Output, Iterations, Nodes. It is `nil` too when the arguments are rejected, for example more than one `timeoutSeconds`.

#### Example

```go
result, info := lp.SolveFromFile("model.lp", 10)
if result == nil {
    // No solution: info carries the reason in its Status and Warnings rows.
    if info != nil {
        info.Show()
    }
    return
}
result.Show()
info.Show()

// convert to csv
result.ToCSV("solution.csv", false, false, false)
info.ToCSV("info.csv", true, true, false)
```

> [!TIP]
> Using `ToCSV` method, you can easily export the result to a CSV file.

### Solve Model

```go
func SolveModel(model *lpgen.LPModel, timeoutSeconds ...int) (*DataTable, *DataTable)
```

**Description:** Solves an LPModel directly by passing the model to GLPK without generating a model file.

**Parameters:**

- `model *lpgen.LPModel`: The LPModel to solve.
- `timeoutSeconds int`(optional): The timeout for the solver in seconds.

**Returns:**

- `*DataTable`: The solution DataTable(the column name and the row name will not be set). It is `nil` when the solve does not produce a solution: a timeout, a solver error, or a temporary file that could not be created or written. Check it before calling a method on it.
- `*DataTable`: The additional information DataTable(the column name and the row name will be set). Rows are always in the order Status, Execution Time, Warnings, Full Output, Iterations, Nodes. It is `nil` too when the temporary files could not be created. Unlike `SolveFromFile`, `SolveModel` does not reject more than one `timeoutSeconds`: it uses the first and ignores the rest.

#### Example

```go
model := lpgen.NewLPModel()
model.SetObjective("Maximize", "3 x1 + 5 x2 + x3")
model.AddConstraint("x1 + 2 x2 + 3 x3 <= 12")
model.AddConstraint("x2 + x3 + x4 <= 3")
model.AddConstraint("x1 + x2 + x3 + x4 <= 100")
model.AddBound("0 <= x1 <= 4")
model.AddBound("1 <= x2 <= 6")
model.AddBound("0 <= x3 <= 10")
model.AddIntegerVar("x1")
model.AddIntegerVar("x2")
model.AddIntegerVar("x3")
model.AddIntegerVar("x4")
model.AddBinaryVar("x1")
model.AddBinaryVar("x2")

result, info := lp.SolveModel(model, 10)
if result == nil {
    // No solution: info carries the reason in its Status and Warnings rows.
    if info != nil {
        info.Show()
    }
    return
}
result.ToCSV("solution.csv", false, false, false)
info.ToCSV("info.csv", true, true, false)
```
