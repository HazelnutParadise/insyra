# [ lp ] Package

The `lp` package provides functionality to solve Linear Programming (LP) problems using the GLPK library. It allows you to define and solve LP models, and provides tools to handle the results.

> [!NOTE]
> - This package will automatically install GLPK on your system if it is not already installed (requires network access).
> - Linux/macOS require build tools (`make`, compiler, or Xcode) for source builds.
> - You can set `GLPK_PATH` to point to an existing `glpsol` binary.

## Installation

To install the `lp` package, use the following command:

```bash
go get github.com/HazelnutParadise/insyra/lp
```

## Features

### `SolveFromFile(lpFile string, timeoutSeconds ...int) (*DataTable, *DataTable)`

Solves an LP file using GLPK and returns the result as two DataTable. First DataTable contains the solution, and the second one contains additional information.

#### Parameters

- `lpFile string`: The path to the LP file.
- `timeoutSeconds int`(optional): The timeout for the solver in seconds.

#### Returns

- `*DataTable`: The solution DataTable(the column name and the row name will not be set).
- `*DataTable`: The additional information DataTable(the column name and the row name will be set).

#### Example

```go
result, info := lp.SolveFromFile("model.lp", 10)
result.Show()
info.Show()

// convert to csv
result.ToCSV("solution.csv", false, false, false)
info.ToCSV("info.csv", true, true, false)
```

> [!TIP]
> Using `ToCSV` method, you can easily export the result to a CSV file.

### `SolveModel(model *lpgen.LPModel, timeoutSeconds ...int) (*DataTable, *DataTable)`

Solves an LPModel directly by passing the model to GLPK without generating a model file.

#### Parameters

- `model *lpgen.LPModel`: The LPModel to solve.
- `timeoutSeconds int`(optional): The timeout for the solver in seconds.

#### Returns

- `*DataTable`: The solution DataTable(the column name and the row name will not be set).
- `*DataTable`: The additional information DataTable(the column name and the row name will be set).

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
result.ToCSV("solution.csv", false, false, false)
info.ToCSV("info.csv", true, true, false)
```
