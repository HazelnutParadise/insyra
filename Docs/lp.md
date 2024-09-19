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

### `SolveLPWithGLPK(mlpFile string, timeoutSeconds ...int) (*DataTable, *DataTable)`

Solves an LP model using GLPK and returns the result as two DataTable. First DataTable contains the solution, and the second one contains additional information.

#### Parameters

- `mlpFile string`: The path to the LP file.
- `timeoutSeconds int`(optional): The timeout for the solver in seconds.

#### Returns

- `*DataTable`: The solution DataTable.
- `*DataTable`: The additional information DataTable.

#### Example

```go
result, info := lp.SolveLPWithGLPK("model.lp", 10)
result.Show()
info.Show()

// convert to csv
result.ToCSV("solution.csv")
info.ToCSV("info.csv")
```

