# [ py ] Package

The `py` package allows Golang programs to execute Python code seamlessly and interactively. It provides functionality to pass Go variables into Python scripts, and execute the Python code. Results from the Python script can be sent back to the Go program automatically.

The `py` package automatically installs common Python libraries, allowing you to use them directly in your Python code. You can also install additional dependencies as needed.

## Functions

### `RunCode`

```go
func RunCode(out any, code string) error
```

This function is used to execute arbitrary Python code and bind the result to the provided struct pointer. It appends default Python code required for communication and runs the code.

#### Parameters

- `out` (any): A pointer to a struct to bind the Python result to. You can optionally use `json` tags on struct fields for custom mapping, otherwise field names are matched to Python dictionary keys by default. If you don't need the Python return result, pass `nil`.
- `code` (string): The Python code to be executed.

#### Returns

- `error`: Returns an error if execution failed or result binding failed, `nil` otherwise.

#### Example

```go
type ResultData struct {
    Message string `json:"message"`
    Value   int    `json:"value"`
}

var result ResultData
err := RunCode(&result, `
print("Hello from Python")
insyra.Return({"message": "Hello from Python", "value": 123})
`)
if err != nil {
    fmt.Println("Error:", err)
} else {
    fmt.Println(result)
}
```

---

### `RunCodef`

```go
func RunCodef(out any, code string, args ...any) error
```

This function is used to execute Python code with variables passed from Go using `$v1`, `$v2`, etc. placeholders and bind the result to the provided struct pointer. The function replaces these placeholders with the provided arguments.

In the Python code template, use `$v1`, `$v2`, `$v3`, etc. as placeholders for the arguments passed to the function.

#### Parameters

- `out` (any): A pointer to a struct to bind the Python result to. You can optionally use `json` tags on struct fields for custom mapping, otherwise field names are matched to Python dictionary keys by default. If you don't need the Python return result, pass `nil`.
- `code` (string): The Python code template with `$v1`, `$v2`, etc. placeholders.
- `args` (`...any`): A variable-length argument list of Go variables to be substituted into the template.

#### Returns

- `error`: Returns an error if execution failed or result binding failed, `nil` otherwise.

#### Example

```go
package main

import (
 "fmt"
 "github.com/HazelnutParadise/insyra"
 "github.com/HazelnutParadise/insyra/py"
)

type PlotResult struct {
 Success bool   `json:"success"`
 Message string `json:"message"`
}

func main() {
 // Create DataList
 xData := insyra.NewDataList(45, 50, 55, 60, 65, 70, 75, 80, 85, 90)
 yData := insyra.NewDataList(110, 120, 135, 145, 150, 160, 170, 180, 190, 200)

 // Submit Code to Python
 var result PlotResult
 err := py.RunCodef(&result, `
x = $v1
y = $v2

sns.set(style="whitegrid")
sns.scatterplot(x=x, y=y)

plt.title($v3)
plt.xlabel($v4)
plt.ylabel($v5)

plt.show()
insyra.Return({"success": True, "message": "Plot created"})
`, xData.Data(), yData.Data(), "Scatter Plot from Go DataList", "X Values", "Y Values")
 if err != nil {
     fmt.Println("Error:", err)
 } else {
     fmt.Println("Result:", result)
 }
}
```

### `RunFile`

Run Python code from a file and bind the result to the provided struct pointer.

#### Parameters

- `out` (any): A pointer to a struct to bind the Python result to. You can optionally use `json` tags on struct fields for custom mapping, otherwise field names are matched to Python dictionary keys by default. If you don't need the Python return result, pass `nil`.
- `filepath` (string): The Python file to be executed.

#### Returns

- `error`: Returns an error if execution failed or result binding failed, `nil` otherwise.

### `RunFilef`

Run Python code from a file with variables passed from Go using `$v1`, `$v2`, etc. placeholders and bind the result to the provided struct pointer.

#### Parameters

- `out` (any): A pointer to a struct to bind the Python result to. You can optionally use `json` tags on struct fields for custom mapping, otherwise field names are matched to Python dictionary keys by default. If you don't need the Python return result, pass `nil`.
- `filepath` (string): The Python file to be executed.
- `args` (`...any`): A variable-length argument list of Go variables to be substituted into the template.

#### Returns

- `error`: Returns an error if execution failed or result binding failed, `nil` otherwise.

---

### Context-aware `RunCode` / `RunFile`

The `py` package provides context-aware variants that accept a `context.Context` so the Python execution can be canceled from Go. When cancellation happens, these functions return the context error (i.e., `ctx.Err()`), so callers can use `errors.Is(err, context.DeadlineExceeded)` or `errors.Is(err, context.Canceled)` to check the cancellation reason.

#### Functions

```go
func RunCodeContext(ctx context.Context, out any, code string) error
func RunCodefContext(ctx context.Context, out any, code string, args ...any) error
func RunFileContext(ctx context.Context, out any, filepath string) error
func RunFilefContext(ctx context.Context, out any, filepath string, args ...any) error

// Convenience helper:
func RunCodeWithTimeout(timeout time.Duration, out any, code string) error
```

#### Parameters

- `ctx` (`context.Context`): The context used to control cancellation and deadlines for the Python execution.
- Other parameters are the same as their non-context counterparts (`out`, `code`, `filepath`, `args`).

#### Returns

- `error`: Non-nil when execution failed; if execution was canceled via the provided `ctx`, the function returns `ctx.Err()` (typically `context.Canceled` for manual cancellation or `context.DeadlineExceeded` for timeouts). Use `errors.Is` to check for these.

#### Examples

Timeout (convenience):

```go
err := py.RunCodeWithTimeout(1*time.Second, nil, `
import time
time.sleep(10)
insyra.Return({"ok": True})
`)
if errors.Is(err, context.DeadlineExceeded) {
    fmt.Println("python run timed out")
} else if err != nil {
    fmt.Println("python failed:", err)
}
```

Manual cancel (WithCancel):

```go
ctx, cancel := context.WithCancel(context.Background())
go func() {
    time.Sleep(500 * time.Millisecond)
    cancel()
}()

err := py.RunCodeContext(ctx, nil, `
import time
time.sleep(5)
insyra.Return({"ok": True})
`)
if errors.Is(err, context.Canceled) {
    fmt.Println("python run canceled")
} else if err != nil {
    fmt.Println("python failed:", err)
}
```

#### Notes

- The context-aware functions use `exec.CommandContext` under the hood. When the context is done, the underlying Python process is killed and the function returns `ctx.Err()`.

- Note: initialization errors are now propagated to callers. The Python environment initializer `pyEnvInit()` no longer calls fatal logging to terminate the process; instead it returns an `error` when initialization fails (for example: failing to ensure `uv` is installed, failing to prepare the install directory, failing to set up the uv environment, or failing to install dependencies). Callers of py functions (e.g., `RunCode`, `RunFile`, `RunCodeContext`, `PipInstall`, `PipList`, `PipFreeze`, etc.) will return that initialization `error` — be sure to check and handle the returned `error` in your code.
- For platform-specific process group / child-process cleanup semantics, consider the platform behavior; if you need robust group termination, let us know and we can add process-group management to the runner.

### `PipInstall`

```go
func PipInstall(dep string) error
```

This function installs Python dependencies using uv pip. It executes the install command and returns an `error` if the installation fails. It does not terminate the program; callers should handle the error.

#### Parameters

- `dep` (string): The name of the dependency to be installed.

#### Returns

- `error`: Non-nil if installation failed; nil otherwise.

### `PipUninstall`

```go
func PipUninstall(dep string) error
```

This function uninstalls Python dependencies using uv pip. It returns an `error` if the uninstallation fails; callers should handle the error. It does not terminate the program.

#### Parameters

- `dep` (string): The name of the dependency to be uninstalled.

#### Returns

- `error`: Non-nil if uninstallation failed; nil otherwise.

### `PipList`

```go
func PipList() (map[string]string, error)
```

This function lists currently installed Python packages in the uv-managed environment. It returns a map of package name to version (e.g., `{"requests":"2.31.0"}`) and an error if the listing fails.

#### Returns

- `map[string]string`: Map of package name -> version.
- `error`: Non-nil if listing failed.

#### Example

```go
pkgs, err := py.PipList()
if err != nil {
    fmt.Println("Failed to list packages:", err)
} else {
    for name, ver := range pkgs {
        fmt.Printf("%s==%s\n", name, ver)
    }
}
```

### `PipFreeze`

```go
func PipFreeze() ([]string, error)
```

This function returns the lines from `pip freeze` (suitable for a `requirements.txt`), one line per package (e.g., `requests==2.31.0`). It returns an error if the command fails.

#### Returns

- `[]string`: Each element is usually of the form `package==version` or an editable/VCS entry.
- `error`: Non-nil if the command failed.

#### Example

```go
lines, err := py.PipFreeze()
if err != nil {
    fmt.Println("Failed to run pip freeze:", err)
} else {
    for _, l := range lines {
        fmt.Println(l)
    }
}
```

### `ReinstallPyEnv`

```go
func ReinstallPyEnv() error
```

This function completely reinstalls the Python environment by removing the existing virtual environment and reinstalling all dependencies. Useful when you want to reset the Python environment, update dependencies, or fix environment-related issues.

#### Returns

- `error`: Returns an error if the reinstallation fails, `nil` otherwise.

#### Example

```go
package main

import (
 "fmt"
 "github.com/HazelnutParadise/insyra/py"
)

func main() {
 err := py.ReinstallPyEnv()
 if err != nil {
     fmt.Println("Failed to reinstall Python environment:", err)
 } else {
     fmt.Println("Python environment reinstalled successfully!")
 }
}
```

## Concurrency Support

The `py` package supports concurrent execution of Python code. Multiple goroutines can call `RunCode`, `RunCodef`, `RunFile`, and `RunFilef` simultaneously without interference. Each execution gets a unique ID and processes independently.

## Automatic Type Conversion

When passing Go variables to Python code using `RunCodef` or `RunFilef`, certain Insyra types are automatically converted to their Python equivalents:

- **DataTable** (`insyra.IDataTable`): Automatically converted to a `pandas.DataFrame`. The DataFrame will include column names and row names if they are set in the original DataTable.
- **DataList** (`insyra.IDataList`): Automatically converted to a `pandas.Series`. The Series will include the name if it is set in the original DataList.

This conversion allows seamless integration between Go's Insyra data structures and Python's data analysis libraries.

## Return Type Binding (Python -> Go)

When `out` is a `DataTable` or `DataList` pointer, `insyra.Return` recognizes common pandas/polars structures and binds them back to Insyra types automatically.

### Supported Return Mappings

- **pandas.DataFrame** -> `*insyra.DataTable`
  - Columns become `ColNames`.
  - Index becomes `RowNames` (converted to strings).
  - `DataFrame.name` (if set) becomes `DataTable.Name`.
- **polars.DataFrame** -> `*insyra.DataTable`
  - Columns become `ColNames`.
  - Polars has no index; row names are not set.
- **pandas.Series** -> `*insyra.DataList`
  - `Series.name` becomes `DataList.Name` (converted to string).
- **polars.Series** -> `*insyra.DataList`
  - `Series.name` becomes `DataList.Name` (converted to string).
- **1D list/tuple** -> `*insyra.DataList`
- **2D list** -> `*insyra.DataTable`
- **dict** -> `*insyra.DataTable` (single row, keys become column names)
- **list of dict** -> `*insyra.DataTable` (multiple rows, keys become column names)

### Example: DataFrame -> DataTable

```go
var dt *insyra.DataTable
err := py.RunCode(&dt, `
import pandas as pd
df = pd.DataFrame({"a": [1, 2], "b": [3, 4]}, index=["r1", "r2"])
insyra.Return(df)
`)
```

### Example: Series -> DataList

```go
var dl *insyra.DataList
err := py.RunCode(&dl, `
import polars as pl
s = pl.Series("s1", [10, 20])
insyra.Return(s)
`)
```

## Functions for Python Code

Here are some functions that are useful when writing Python code to be executed with `RunCode` or `RunCodef`.

### `insyra.Return`

```python
insyra.Return(result=None, error=None, url)
```

This function is used to return data from Python to Go.

If `result` is a pandas/polars DataFrame or Series, it will be normalized and returned as a typed payload so that Go can bind it directly into `DataTable` or `DataList`.

#### Parameters

- `result` (any): The result data to be returned to Go.
- `error` (string): The error message if an error occurred, None otherwise. The Insyra framework will automatically deal with errors, you don't need to set it manually.
- `url` (string): The URL to send the data to. Insyra will automatically set it, you don't need to set it manually.

#### Example

```python
insyra.Return({
 "message": "Hello from Python",
 "value": 123,
})
```

### `insyra_return`

```python
insyra_return(result=None, error=None, url)
```

This is an alias for `insyra.Return` for convenience. It provides the same functionality as `insyra.Return`.

#### Parameters

- `result` (any): The result data to be returned to Go.
- `error` (string): The error message if an error occurred, None otherwise. The Insyra framework will automatically deal with errors, you don't need to set it manually.
- `url` (string): The URL to send the data to. Insyra will automatically set it, you don't need to set it manually.

#### Example

```python
insyra_return({
 "message": "Hello from Python",
 "value": 123,
})
```

### `insyra.execution_id`

```python
insyra.execution_id
```

This variable contains the unique execution ID for the current Python code run. It can be used for logging, debugging, or tracking purposes.

#### Example

```python
print(f"Current execution ID: {insyra.execution_id}")
insyra.Return({"execution_id": insyra.execution_id, "data": "some data"})
```

## Pre-installed Dependencies

- **Python Environment**: Insyra automatically installs Python environment using `uv` in the `.insyra_py_env` directory in your project root.
- **Python Libraries**: Insyra automatically installs following Python libraries, you can use them directly in your Python code:

```go
pyDependencies   = map[string]string{
 "import requests":                   "requests",       // HTTP requests
 "import json":                       "",               // JSON data processing (built-in module)
 "import numpy as np":                "numpy",          // Numerical operations
 "import pandas as pd":               "pandas",         // Data analysis and processing
 "import polars as pl":               "polars",         // Data analysis and processing (faster alternative to pandas)
 "import matplotlib.pyplot as plt":   "matplotlib",     // Data visualization
 "import seaborn as sns":             "seaborn",        // Data visualization
 "import scipy":                      "scipy",          // Scientific computing
 "import sklearn":                    "scikit-learn",   // Machine learning
 "import statsmodels.api as sm":      "statsmodels",    // Statistical modeling
 "import plotly.graph_objects as go": "plotly",         // Interactive data visualization
 "import spacy":                      "spacy",          // Efficient natural language processing
 "import bs4":                        "beautifulsoup4", // Web scraping
}
```
