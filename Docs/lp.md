# [ lp ] Package

The `lp` package provides functionality to solve Linear Programming (LP) problems using the GLPK library. It allows you to define and solve LP models, and provides tools to handle the results.

> [!NOTE]
> This package will automatically install GLPK on your system if it is not already installed.

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
result, info := lp.SolveLPWithGLPK("model.lp", 10)
result.Show()
info.Show()

// convert to csv
result.ToCSV("solution.csv", false, false)
info.ToCSV("info.csv", true, true)
```

> [!TIP]
> Using `ToCSV` method, you can easily export the result to a CSV file.