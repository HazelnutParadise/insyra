# [ py ] Package

The `py` package allows Go programs to execute Python code and exchange variables/results. It spins up a managed Python environment using `uv` and a local IPC server.

On first use it:

- Downloads the pinned [uv](https://github.com/astral-sh/uv) release for your platform from GitHub, checks it against the SHA-256 the release publishes, and keeps it in `.insyra_env/uv-<version>_<os>_<arch>/` under the working directory. It does not run uv's install script or use a `uv` already on your `PATH`.
- Has that uv build a virtual environment in `.insyra_env/py26a_<os>_<arch>` with the pinned CPython, a build uv downloads and verifies itself, and the pinned packages (`numpy`, `pandas`, `polars`, `matplotlib`, `seaborn`, `scikit-learn`, …) at the exact versions in the lock file, downloaded from the PyPI URLs the lock records and checked against the hashes recorded there, whatever index your own uv configuration names.

This setup needs network access and takes a while the first time. A setup that fails is run again by the next call, and the context passed to a `…Context` function also bounds it: cancelling the context stops a download or an install in progress, or a wait for another call's setup to finish.

To choose when this happens, call `py.Setup(ctx)` yourself, for example when a server starts, so a network failure shows up there instead of on its first request. Calling it is optional: without it, the first call prepares the environment as described above.

uv keeps the CPython build and its download cache in its usual per-user directories (on macOS and Linux `~/.local/share/uv/python` and `~/.cache/uv`, unless `UV_PYTHON_INSTALL_DIR` or `UV_CACHE_DIR` say otherwise), so deleting `.insyra_env` does not remove them. The setup ignores `UV_PYTHON_PREFERENCE` and lets uv download Python whatever `UV_PYTHON_DOWNLOADS` says, because the environment always runs the pinned, uv-managed CPython; your other uv settings, such as a proxy or certificates, still apply.

On Windows on arm64, uv installs the x86-64 build of CPython, its default there, and Windows runs it through its x64 emulation, so every package installs from its `win_amd64` wheel and nothing is built from source. Python code therefore runs under emulation there, slower than on a native Python. polars' CPU check reads the machine as ARM64 under that emulation and would refuse to import, so the script `py` generates sets `POLARS_SKIP_CPU_CHECK=1` for an x86-64 Python on ARM64 Windows, unless you set the variable yourself. GitHub's Windows 11 arm64 runner builds the environment and runs Python through the package on every change to it. PyPI has no Windows arm64 wheel of `blis` (a dependency of spaCy) or `statsmodels`, so a native arm64 Python would build those two; the tools such a build installs are pinned and checked against their hashes like every other package (see [Pinned versions](#pinned-versions)).

On musl Linux, such as Alpine, uv installs its musl build of CPython, and PyPI has no musllinux wheel of `scikit-learn`, nor on arm64 of `matplotlib` or `statsmodels`, so uv builds those from source there, with the pinned build tools. Install a C and C++ compiler first; on Alpine, `apk add build-base`. The first setup takes longer: building the three packages took about 17 minutes in an Alpine arm64 container on an Apple M3. `matplotlib`'s build downloads the sources of FreeType, HarfBuzz, libraqm, Qhull and SheenBidi and checks each against a SHA-256 its source archive pins, so it needs access to those sites. uv keeps the wheels it builds in its cache, so another environment on the same machine reuses them.

Later runs find a marker file in the environment directory and start without calling uv. The directory is named for the Python it runs (`py26a` is CPython 3.12.14). When an insyra upgrade changes only package versions, the first run afterwards brings the environment in the same directory to them: each pinned package moves to its pinned version, and packages you added with `PipInstall` stay. When an upgrade changes the Python version, the environment gets a new directory and is built there from scratch; the old directory, such as `.insyra_env/py25c_<os>_<arch>` from before CPython 3.12.14 was pinned, is no longer used, and you can delete it.

### Pinned versions

Every version the environment uses is recorded in [`py/environment/`](https://github.com/HazelnutParadise/insyra/tree/main/py/environment):

- `pyproject.toml` pins uv (`[tool.uv] required-version`), Python (`requires-python`) and each package (`name==version`). Under `[tool.uv] build-constraint-dependencies` it also pins every tool uv installs to build a package from source, with the SHA-256 of each file of that version.
- `uv.lock` records the resolution of those packages and everything they depend on, with the SHA-256 of every file, and the build constraints with their hashes. The setup takes the build constraints from the lock, so a build tool whose download does not match is refused.
- `uv-sha256.sum` is the `sha256.sum` file the uv release publishes, unchanged; the downloaded uv archive is checked against it.

To bump them:

1. For a new uv, replace `uv-sha256.sum` with that release's `sha256.sum`, downloaded from `https://github.com/astral-sh/uv/releases/download/<version>/sha256.sum` and left unchanged, and set `required-version = "==<version>"` in `pyproject.toml`.
2. For a new Python, set `requires-python = "==<x.y.z>"` to a CPython patch uv offers on every supported platform; `uv python list --all-versions --all-platforms --all-arches --only-downloads` lists them. Then give the environment directory a new code in `py/const.go`: set `envDirCode` to `py<two-digit year><letter>`, `a` for the year's first new code and `b` for the second, and `envDirPython` to the new version. The tests fail until both are set.
3. Set each package to the version you want as `name==version`. Every package the Python preamble imports (`pyDependencies` in `py/const.go`) must be pinned, and nothing else.
4. Run `uv lock` in `py/environment/` with the pinned uv version, with `UV_NO_CONFIG=1` and none of `UV_INDEX_URL`, `UV_DEFAULT_INDEX` or `UV_EXCLUDE_NEWER` set, so the lock takes every file from PyPI.
5. Rebuild the build constraints when a package uv builds from source (`knownSourceBuilds` in `py/environment_pins_test.go`) changed version or joined that list, or when the lock moved a package that is also a build tool, such as `numpy`, `setuptools` or `packaging`. The constraints are one list for every source build, so write one requirements file with the `[build-system] requires` of every package in `knownSourceBuilds`, read from its source archive on PyPI, each range as written, plus `ninja` and `patchelf`, which meson-python asks for when they are not on `PATH`; a tool the environment also installs goes in at the version `uv.lock` resolves it to. Run `uv pip compile <file> --python-version <python> --generate-hashes` with the pinned uv and `UV_NO_CONFIG=1` for each platform in `knownSourceBuilds` (`--python-platform x86_64-unknown-linux-musl`, `aarch64-unknown-linux-musl` and `aarch64-pc-windows-msvc`), check that they resolve the same versions, write each package as `{ requirement = "name==version", hashes = [...] }` with all of its hashes, and run `uv lock` again so the lock records them.
6. Run `go test ./py/`, which fails when the three files disagree, when the lock takes a package from anywhere but PyPI, when a supported platform lacks a wheel the lock had before, or when a build constraint is not an exact pin with hashes. Then run, with `INSYRA_PY_E2E=1`, `go test ./py/ -run TestPinnedEnvironmentEndToEnd`, which builds the environment from nothing in a temporary directory and runs Python in it, and `go test ./py/ -run TestSourceBuildsUseOnlyPinnedTools`, which builds from source the packages `knownSourceBuilds` lists for the platform it runs on and fails if a build installed a tool the build constraints do not pin. The `Python on musl Linux` workflow runs both in an Alpine container on amd64 and arm64 whenever these files change.

### Environment Utilities

```go
// Prepare the managed Python environment now instead of on first use (optional)
func Setup(ctx context.Context) error

// Delete the managed Python environment and build it again from the pinned versions
func ReinstallPyEnv() error
```

## Functions

### Run

```go
func Run[T any](ctx context.Context, code string, args ...any) (T, error)
```

**Description:** Runs the Python code and returns the value it passes to `insyra.Return`, decoded into a `T`. You name the type at the call and there is no variable to declare and pass by address; the value is still matched against `T` when it comes back, so a result of another shape is an error then. `T` follows the same rules `out` does for `RunCode`: `*insyra.DataTable` or `*insyra.DataList` for a DataFrame or a Series, and a struct, map, slice or scalar through JSON. `$v1`, `$v2`, … in the code are replaced from `args` as `RunCodef` replaces them. `T` cannot be `insyra.DataTable` or `insyra.DataList` itself, which cannot be copied: use `*insyra.DataTable` or `*insyra.DataList`. `Run` refuses the value types before it starts Python.

**Parameters:**

- `ctx` (`context.Context`): bounds the environment setup on first use and the Python process. Cancelling it stops the process, and `Run` returns `ctx.Err()`. A `nil` context is an error.
- `code` (string): the Python code, optionally with `$v1`, `$v2`, … placeholders.
- `args` (`...any`): the values for the placeholders.

**Returns:**

- `T`: the decoded result, or `T`'s zero value when the run or the decoding fails.
- `error`: non-nil when execution or decoding failed.

#### Example

```go
type Summary struct {
    Mean float64 `json:"mean"`
    N    int     `json:"n"`
}

s, err := py.Run[Summary](ctx, `
import statistics
data = $v1
insyra.Return({"mean": statistics.mean(data), "n": len(data)})
`, []float64{1, 2, 3, 4})
if err != nil {
    return err
}
fmt.Println(s.Mean, s.N) // 2.5 4

dt, err := py.Run[*insyra.DataTable](ctx, `insyra.Return(pd.DataFrame({"a": [1, 2], "b": [3, 4]}))`)
```

---

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
func Run[T any](ctx context.Context, code string, args ...any) (T, error)
func RunCodeContext(ctx context.Context, out any, code string) error
func RunCodefContext(ctx context.Context, out any, code string, args ...any) error
func RunFileContext(ctx context.Context, out any, filepath string) error
func RunFilefContext(ctx context.Context, out any, filepath string, args ...any) error
func PipInstallContext(ctx context.Context, dep string) error
func PipUninstallContext(ctx context.Context, dep string) error

// Deprecated: use RunCodeContext with context.WithTimeout.
func RunCodeWithTimeout(timeout time.Duration, out any, code string) error
```

The plain functions (`RunCode`, `RunCodef`, `RunFile`, `RunFilef`, `PipInstall`, `PipUninstall`) call these with `context.Background()`. `RunCodeWithTimeout` is `RunCodeContext` with a context from `context.WithTimeout`; it is deprecated and will be removed in the release after the one that deprecated it.

**Parameters:**

- `ctx` (`context.Context`): The context used to control cancellation and deadlines for the Python execution.
- Other parameters are the same as their non-context counterparts (`out`, `code`, `filepath`, `args`).

**Returns:**

- `error`: Non-nil when execution failed; if execution was canceled via the provided `ctx`, the function returns `ctx.Err()` (typically `context.Canceled` for manual cancellation or `context.DeadlineExceeded` for timeouts). Use `errors.Is` to check for these. A context that is already done returns `ctx.Err()` before anything starts, and a `nil` context is an error rather than a panic.

#### Examples

Timeout:

```go
ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
defer cancel()

err := py.RunCodeContext(ctx, nil, `
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

- The context-aware functions use `exec.CommandContext` under the hood. When the context is done, the underlying Python process, or the `uv pip` command of `PipInstallContext` and `PipUninstallContext`, is killed and the function returns `ctx.Err()`.

- Note: initialization errors are now propagated to callers. The Python environment initializer `pyEnvInit()` no longer calls fatal logging to terminate the process; instead it returns an `error` when initialization fails (for example: failing to download or verify uv, or uv failing to build the environment). Callers of py functions (e.g., `RunCode`, `RunFile`, `RunCodeContext`, `PipInstall`, `PipList`, `PipFreeze`, etc.) will return that initialization `error` — be sure to check and handle the returned `error` in your code.
- For platform-specific process group / child-process cleanup semantics, consider the platform behavior; if you need robust group termination, let us know and we can add process-group management to the runner.

### Install Python Dependency

```go
func PipInstall(dep string) error
func PipInstallContext(ctx context.Context, dep string) error
```

**Description:** This function installs Python dependencies using uv pip. It executes the install command and returns an `error` if the installation fails. It does not terminate the program; callers should handle the error. `PipInstallContext` takes a context that bounds the environment setup and the install; `PipInstall` uses `context.Background()`. Cancelling it stops `uv pip install` where it is, which can leave the package half installed: install it again, or use `ReinstallPyEnv`.

**Parameters:**

- `dep` (string): The name of the dependency to be installed.

**Returns:**

- `error`: Non-nil if installation failed; nil otherwise.

### Uninstall Python Dependency

```go
func PipUninstall(dep string) error
func PipUninstallContext(ctx context.Context, dep string) error
```

**Description:** This function uninstalls Python dependencies using uv pip. It returns an `error` if the uninstallation fails; callers should handle the error. It does not terminate the program. `PipUninstallContext` takes a context that bounds the environment setup and the uninstall; `PipUninstall` uses `context.Background()`.

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

### Prepare the Python Environment

```go
func Setup(ctx context.Context) error
```

**Description:** Prepares the managed Python environment now: it downloads and verifies the pinned uv if it is missing, and has uv bring `.insyra_env/py26a_<os>_<arch>` to the pinned versions, which is what the first `RunCode`, `PipInstall` or other call would otherwise do. Calling it is optional. Call it at start-up to fail there rather than on a first request, and to bound the downloads with a context. On an environment that is already prepared it returns `nil` without running uv, so calling it on every start costs little.

**Parameters:**

- `ctx` (`context.Context`): bounds the uv download, the sync, and the wait for a setup another call has already started. A `nil` context is an error, and a context that is already done returns `ctx.Err()` before anything starts.

**Returns:**

- `error`: Non-nil if the environment could not be prepared. A setup that fails is not marked ready, so `Setup` or the next call tries again.

#### Example

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
defer cancel()
if err := py.Setup(ctx); err != nil {
    log.Fatalf("preparing the Python environment: %v", err)
}
```

### Reinstall Python Environment

```go
func ReinstallPyEnv() error
```

**Description:** This function deletes the whole environment directory, `.insyra_env/py26a_<os>_<arch>` under the working directory, with everything in it, the packages you installed with `PipInstall` included, and builds the environment again from the pinned versions. The pinned uv, kept beside that directory, is not downloaded again. Use it to reset an environment that no longer works.

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

An integer keeps every digit. Bound into an integer field such as `int64` or `uint64`, it is exact whenever the field holds it. In a table or list cell, or anything decoded into `any`, an integer within ±2^53 comes back as a float64, as every number always has, since a float64 holds each of them exactly. An integer above 2^53 in magnitude, such as a database id or a nanosecond timestamp, comes back as an `int64`, or a `uint64` above the `int64` range, instead of a float64 with its last digits changed. An integer beyond 64 bits comes back as the nearest float64. A number an integer field cannot hold, such as `2**63` for an `int64`, is an error. A DataFrame is sent column by column, so an integer column beside a float one stays integers. A column with missing values, such as pandas `Int64` with `NA` or polars integers with nulls, is turned into floats by pandas or polars before it is sent, so a large integer in it is not exact.

A result the Go side cannot read, such as an integer too large for a float64 (`10**400`), makes the call return an error that says why, and `insyra.Return` raises in Python. Code that catches that exception and returns another value gets that value back instead.

### Tables and lists inside a result

A table or a list can also sit inside the value you return. `insyra.Return` turns a DataFrame or Series anywhere inside a dict, list or tuple into the payload it sends for one at the top level. On the Go side, when the type you decode into has a `*insyra.DataTable`, `*insyra.DataList`, `insyra.IDataTable` or `insyra.IDataList` in a struct field, a map value, a slice or array element or behind a pointer, that part is decoded as a table or a list, replacing what it held, and everything else exactly as `encoding/json` decodes it: struct fields are matched by their `json` tag or Go name, an exact match before one that ignores case, with its rules for embedded structs, `-` and `,string`; map keys can be strings, integers or a type with `UnmarshalText`; decoding into a value that already holds something keeps what the result leaves out; and `None` sets a pointer, slice, map or interface to nil and leaves a struct as it is. A type with its own `UnmarshalJSON` or `UnmarshalText` decodes itself.

```go
type Scored struct {
    Table *insyra.DataTable `json:"table"`
    Score float64           `json:"score"`
}
s, err := py.Run[Scored](ctx, `insyra.Return({"table": df, "score": 7})`)

splits, err := py.Run[map[string]*insyra.DataTable](ctx, `insyra.Return({"train": train, "test": test})`)
```

An embedded `*insyra.DataTable` or `*insyra.DataList` follows the rule for an embedded struct. Without a tag it is decoded from the whole value around it, as the `isr` types (`struct{ *insyra.DataTable }`) are, at the top of the result or inside it, and a value of the other kind, such as a DataFrame for an `isr` list, is an error. Beside other fields it takes the value only when Python sent a DataFrame, a Series or a list; a dict is then the struct's own object and goes to its other fields. With a tag it is a field under that name, so ``struct{ *insyra.DataTable `json:"table"`; Score float64 }`` reads the table from the key `table`. An embedded `insyra.IDataTable` or `insyra.IDataList` is a field named after its type, as an embedded interface is in `encoding/json`. A field that is only named `DataTable` is decoded as that field, like any other.

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

- **Python Environment**: Insyra installs a managed environment under `.insyra_env/py26a_<os>_<arch>` in the working directory, running the pinned CPython.
- **Python Libraries**: Insyra installs the following libraries at the versions pinned in `py/environment/pyproject.toml` (see [Pinned versions](#pinned-versions)) and imports them at the start of every script:

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
