# [ py ] Package

The `py` package allows Go programs to execute Python code and exchange variables/results. It spins up a managed Python environment using `uv` and a local IPC server.

On first use it:

- Installs `uv` if missing
- Creates a virtual environment under `.insyra_env/py<version_code>_<os>_<arch>`
- Installs a preset dependency list (e.g., `numpy`, `pandas`, `polars`, `matplotlib`, `seaborn`, `scikit-learn`)

This setup requires network access and may take a while the first time.

On Windows on arm64, an x86-64 build of CPython runs through Windows' x64 emulation, and polars' CPU check reads the machine as ARM64 under that emulation and can refuse to import. The script `py` generates therefore sets `POLARS_SKIP_CPU_CHECK=1` for an x86-64 Python on ARM64 Windows, unless you set the variable yourself. A native arm64 Python, and every other platform, is left as it is.

### Environment Utilities

```go
// Reinstall the managed Python environment
func ReinstallPyEnv() error
```

## Functions

### Run Code

```go
func RunCode(out any, code string) error
```

**Description:** This function is used to execute arbitrary Python code and bind the result to the provided struct pointer. It appends default Python code required for communication and runs the code.

**Parameters:**

- `out` (any): A pointer to a struct to bind the Python result to. You can optionally use `json` tags on struct fields for custom mapping, otherwise field names are matched to Python dictionary keys by default. If you don't need the Python return result, pass `nil`.
- `code` (string): The Python code to be executed.

**Returns:**

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

### Run Code With Parameters

```go
func RunCodef(out any, code string, args ...any) error
```

**Description:** This function is used to execute Python code with variables passed from Go using `$v1`, `$v2`, etc. placeholders and bind the result to the provided struct pointer. The function replaces these placeholders with the provided arguments.

**Parameters:**

- `out` (any): A pointer to a struct to bind the Python result to. You can optionally use `json` tags on struct fields for custom mapping, otherwise field names are matched to Python dictionary keys by default. If you don't need the Python return result, pass `nil`.
- `code` (string): The Python code template with `$v1`, `$v2`, etc. placeholders.
- `args` (`...any`): A variable-length argument list of Go variables to be substituted into the template.

**Returns:**

- `error`: Returns an error if execution failed or result binding failed, `nil` otherwise.

In the Python code template, use `$v1`, `$v2`, `$v3`, etc. as placeholders for the arguments passed to the function.

Each placeholder is replaced from the template as you wrote it, in one pass. Text an argument puts into the script is not read again, so an argument holding `$v2` stays the text `$v2`. A placeholder is read by its whole number: `$v10` is the tenth argument. One with no argument at its position, or written with a leading zero such as `$v01`, is left in the script as written. Only the arguments the template uses are converted, and one that cannot be written as a Python value, such as `math.NaN()`, or a slice or map holding it, makes the call return an error that names its placeholder. Python does not start in that case.

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

### Run Python File

Run Python code from a file and bind the result to the provided struct pointer.

**Parameters:**

- `out` (any): A pointer to a struct to bind the Python result to. You can optionally use `json` tags on struct fields for custom mapping, otherwise field names are matched to Python dictionary keys by default. If you don't need the Python return result, pass `nil`.
- `filepath` (string): The Python file to be executed.

**Returns:**

- `error`: Returns an error if execution failed or result binding failed, `nil` otherwise.

### Run Python File With Parameters

Run Python code from a file with variables passed from Go using `$v1`, `$v2`, etc. placeholders and bind the result to the provided struct pointer.

**Parameters:**

- `out` (any): A pointer to a struct to bind the Python result to. You can optionally use `json` tags on struct fields for custom mapping, otherwise field names are matched to Python dictionary keys by default. If you don't need the Python return result, pass `nil`.
- `filepath` (string): The Python file to be executed.
- `args` (`...any`): A variable-length argument list of Go variables to be substituted into the template.

**Returns:**

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

**Parameters:**

- `ctx` (`context.Context`): The context used to control cancellation and deadlines for the Python execution.
- Other parameters are the same as their non-context counterparts (`out`, `code`, `filepath`, `args`).

**Returns:**

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

### Install Python Dependency

```go
func PipInstall(dep string) error
```

**Description:** This function installs Python dependencies using uv pip. It executes the install command and returns an `error` if the installation fails. It does not terminate the program; callers should handle the error.

**Parameters:**

- `dep` (string): The name of the dependency to be installed.

**Returns:**

- `error`: Non-nil if installation failed; nil otherwise.

### Uninstall Python Dependency

```go
func PipUninstall(dep string) error
```

**Description:** This function uninstalls Python dependencies using uv pip. It returns an `error` if the uninstallation fails; callers should handle the error. It does not terminate the program.

**Parameters:**

- `dep` (string): The name of the dependency to be uninstalled.

**Returns:**

- `error`: Non-nil if uninstallation failed; nil otherwise.

### List Installed Python Packages

```go
func PipList() (map[string]string, error)
```

**Description:** This function lists currently installed Python packages in the uv-managed environment. It returns a map of package name to version (e.g., `{"requests":"2.31.0"}`) and an error if the listing fails.

**Parameters:**

- None.

**Returns:**

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

### Pip Freeze

```go
func PipFreeze() ([]string, error)
```

**Description:** This function returns the lines from `pip freeze` (suitable for a `requirements.txt`), one line per package (e.g., `requests==2.31.0`). It returns an error if the command fails.

**Parameters:**

- None.

**Returns:**

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

### Reinstall Python Environment

```go
func ReinstallPyEnv() error
```

**Description:** This function completely reinstalls the Python environment by removing the existing virtual environment and reinstalling all dependencies. Useful when you want to reset the Python environment, update dependencies, or fix environment-related issues.

**Parameters:**

- None.

**Returns:**

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

Python sends the value passed to `insyra.Return` back over a local IPC server: a Unix socket file in the temp directory (`os.TempDir()`), or a named pipe on Windows. The server opens when a call needs it and closes when the last call running at the same time finishes, so the socket file exists only while a call is in progress, the first call's environment setup included. If the server cannot open, for example because the temp directory's path is too long for a Unix socket, the call returns an error that wraps the cause (`errors.Is(err, syscall.EINVAL)` for that case) before it prepares the environment or starts Python, and the next call tries again.

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
  - `DataFrame.name` (if set) becomes `DataTable.Name`. A column called `name` is not taken for it, so such a DataFrame comes back with no table name.
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

An empty DataFrame, such as the result of a filter that matched no rows, comes back as a table with no rows and the DataFrame's column names.

A NaN, `inf` or `-inf` in the result, such as a missing value in a DataFrame, comes back as `math.NaN()`, `math.Inf(1)` or `math.Inf(-1)`: in a table or list cell, a float field, a slice, array or map value, or anything decoded into `any`. Python's `None` comes back as `nil`, so `ClearNaNs`, `ClearNils` and the other methods that tell the two apart work on the result as on any other list. A NaN for a type that cannot hold one, such as `int`, is an error.

A result the Go side cannot read, such as an integer too large for a float64 (`10**400`), makes the call return an error that says why, and `insyra.Return` raises in Python. Code that catches that exception and returns another value gets that value back instead.

### Tables and lists inside a result

A table or a list can also sit inside the value you return. `insyra.Return` turns a DataFrame or Series anywhere inside a dict, list or tuple into the payload it sends for one at the top level. On the Go side, when the type you decode into has a `*insyra.DataTable`, `*insyra.DataList`, `insyra.IDataTable` or `insyra.IDataList` in a struct field, a map value, a slice or array element or behind a pointer, that part is decoded as a table or a list, replacing what it held, and everything else exactly as `encoding/json` decodes it: struct fields are matched by their `json` tag or Go name, an exact match before one that ignores case, with its rules for embedded structs, `-` and `,string`; map keys can be strings, integers or a type with `UnmarshalText`; decoding into a value that already holds something keeps what the result leaves out; and `None` sets a pointer, slice, map or interface to nil and leaves a struct as it is. A type with its own `UnmarshalJSON` or `UnmarshalText` decodes itself.

```go
type Scored struct {
    Table *insyra.DataTable `json:"table"`
    Score float64           `json:"score"`
}
var s Scored
err := py.RunCode(&s, `insyra.Return({"table": df, "score": 7})`)

var splits map[string]*insyra.DataTable
err = py.RunCode(&splits, `insyra.Return({"train": train, "test": test})`)
```

An embedded `*insyra.DataTable` or `*insyra.DataList` follows the rule for an embedded struct. Without a tag it is decoded from the whole value around it, as the `isr` types (`struct{ *insyra.DataTable }`) are, at the top of the result or inside it, and a value of the other kind, such as a DataFrame for an `isr` list, is an error. Beside other fields it takes the value only when Python sent a DataFrame, a Series or a list; a dict is then the struct's own object and goes to its other fields. With a tag it is a field under that name, so ``struct{ *insyra.DataTable `json:"table"`; Score float64 }`` reads the table from the key `table`. An embedded `insyra.IDataTable` or `insyra.IDataList` is a field named after its type, as an embedded interface is in `encoding/json`. A struct field named `DataTable` or `DataList`, embedded or not, takes the whole result when the result is itself a DataFrame, a Series or a list; a `json` tag that points the field at another key opts it out, and a dict is decoded key by key as usual.

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

**Parameters:**

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

**Parameters:**

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

- **Python Environment**: Insyra installs a managed environment under `.insyra_env/py<version_code>_<os>_<arch>` (project root).
- **Python Libraries**: Insyra installs and imports the following libraries by default:

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
