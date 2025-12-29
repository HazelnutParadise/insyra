# Insyra - Crafting Your Art of Data

[![Test](https://github.com/HazelnutParadise/insyra/actions/workflows/test.yml/badge.svg)](https://github.com/HazelnutParadise/insyra/actions/workflows/test.yml)
[![GolangCI-Lint](https://github.com/HazelnutParadise/insyra/actions/workflows/golangci-lint.yml/badge.svg)](https://github.com/HazelnutParadise/insyra/actions/workflows/golangci-lint.yml)
[![Govulncheck](https://github.com/HazelnutParadise/insyra/actions/workflows/govulncheck.yml/badge.svg)](https://github.com/HazelnutParadise/insyra/actions/workflows/govulncheck.yml)
[![Go version](https://img.shields.io/github/go-mod/go-version/HazelnutParadise/insyra.svg)](https://github.com/HazelnutParadise/insyra)
[![Go Report Card](https://goreportcard.com/badge/github.com/HazelnutParadise/insyra)](https://goreportcard.com/report/github.com/HazelnutParadise/insyra)
[![GoDoc](https://godoc.org/github.com/HazelnutParadise/insyra?status.svg)](https://pkg.go.dev/github.com/HazelnutParadise/insyra)
[![MIT license](https://img.shields.io/badge/license-MIT-brightgreen.svg)](https://opensource.org/licenses/MIT)

A next-generation data analysis library for Golang. Supports **parallel processing**, **data visualization**, and **seamless integration with Python**.

**Official Website: <https://insyra.hazelnut-paradise.com>**

**Documentation site:** Docs are served by **Docsify** from the `docs/` folder and automatically deployed to the `gh-pages` branch via GitHub Actions on pushes to `main`/`master` (see `.github/workflows/deploy-docs.yml`). After the first successful run, your site will be available at `https://<GITHUB_USER>.github.io/insyra` (replace `<GITHUB_USER>` with the owner/org name).

> [!NOTE]
> This project is evolving rapidly—please star and watch the repository to stay up to date with the latest changes!

![logo](logo/logo_transparent.png)

**[繁體中文版 README](README_TW.md)**

Welcome to join [**Side Project Taiwan**(Discord Server)](https://discord.com/channels/1205906503073140776/1280539658551558368) for further discussions.

## Fast, Lovely, Easy To Use

The **Insyra** library is a dynamic and versatile tool designed for managing and analyzing data in Go. It offers a rich set of features for data manipulation, statistical calculations, data visualization, and more, making it an essential toolkit for developers handling complex data structures.

> [!TIP]
> We got brand new `isr` package, which provides **Sytax Sugar**!<br/>
> Any new project is recommended to use `isr` package instead of calling `insyra` main package directly.<br/>
> For more details, please refer to the **[Documentation](/Docs/isr.md)**.

> [!NOTE]
> If some functions or methods in the documentation are not working, it may be because the feature is not yet included in the latest release. Please refer to the documentation in the source code of the corresponding version in **[Releases](https://github.com/HazelnutParadise/insyra/releases)**.

> [!IMPORTANT]
> **For any functions or methods not explicitly listed in Insyra documents, it indicates that the feature is still under active development. These experimental features might provide unstable results.** <br/>
> Please refer to our latest updates in **[Docs](/Docs)** folder for more details.

## [Idensyra](https://github.com/HazelnutParadise/idensyra)

We provide a mini Go IDE, `Idensyra`, which aims to make data analysis even more easier (though Insyra has already made it very easy).

`Idensyra` comes with Insyra pre-installed, and allows you to run Go code without installing Go environment!

**[Know more about Idensyra](https://github.com/HazelnutParadise/idensyra)**

## Getting Started

### For those new to Golang

> [!TIP]
> Jump to [Installation](#installation) or [Quick Example](#quick-example) if you are familiar with Go.

1. Download and install Golang from [here](https://golang.org/dl/).
2. Set up your editor, we recommend using [VSCode](https://code.visualstudio.com/). Or even lighter weight, [Idensyra](https://github.com/HazelnutParadise/idensyra).
3. Open or create a folder for your project, and open it in the editor.

4. Create a new project by running the following command:

    ```sh
    go mod init your_project_name
    ```

5. Install **Insyra**:

    ```sh
    go get github.com/HazelnutParadise/insyra/allpkgs
    ```

6. Create a new file, e.g., `main.go`, and write the following code:

    ```go
    package main

    import (
        "fmt"
        "github.com/HazelnutParadise/insyra"
    )

    func main() {
        // Your code here
    }
    ```

7. Run your project:

    ```sh
    go run main.go
    ```

### Installation

- To start using **Insyra**, install it with the following command:

    ```sh
    go get github.com/HazelnutParadise/insyra/allpkgs
    ```

- Update **Insyra** to the latest version:

    ```sh
    go get -u github.com/HazelnutParadise/insyra/allpkgs
    ```

    or

    ```sh
    go get -u github.com/HazelnutParadise/insyra/allpkgs@latest
    ```

### Quick Example

```go
package main

import (
    "fmt"
    "github.com/HazelnutParadise/insyra"
)

func main() {
    dl := insyra.NewDataList(1, 2, 3, 4, 5)
    dl.Append(6)
    fmt.Println("DataList:", dl.Data())
    fmt.Println("Mean:", dl.Mean())
}
```

#### Syntactic Sugar

It is strongly recommended to use syntactic sugar since it is much more power and easier to use. For example, the above code can be written as:

```go
package main

import (
 "fmt"

 "github.com/HazelnutParadise/insyra/isr"
)

func main() {
 dl := isr.DL.Of(1, 2, 3, 4, 5)
 dl.Append(6)
 dl.Show()
 fmt.Println("Mean:", dl.Mean())
}
```

To use the syntactic sugar, import `github.com/HazelnutParadise/insyra/isr`.

### Console Preview with `insyra.Show`

Need a quick labelled look at any showable structure (like `DataTable` or `DataList`)? Use the package-level `Show` helper, which delegates to `ShowRange` under the hood and supports the same range arguments:

```go
func main() {
    dt := insyra.NewDataTable(
        insyra.NewDataList("Alice", "Bob", "Charlie").SetName("Name"),
        insyra.NewDataList(28, 34, 29).SetName("Age"),
    ).SetName("Team Members")

    insyra.Show("Preview", dt, 2) // First two rows
}
```

### Configuration

**Insyra** provides a global `Config` object for managing library behavior. You can customize logging, error handling, and performance settings:

#### Log Level Management

Control what level of messages are logged:

```go
// Set log level - only messages at this level or above will be logged
insyra.Config.SetLogLevel(insyra.LogLevelDebug)    // Most verbose
insyra.Config.SetLogLevel(insyra.LogLevelInfo)     // Default
insyra.Config.SetLogLevel(insyra.LogLevelWarning)  // Only warnings and errors
insyra.Config.SetLogLevel(insyra.LogLevelFatal)    // Only fatal errors

// Get current log level
level := insyra.Config.GetLogLevel()
```

#### Colored Output

Control whether terminal output is colored:

```go
// Enable / disable colored output
insyra.Config.SetUseColoredOutput(true)

// Check colored output status
usesColor := insyra.Config.GetDoesUseColoredOutput()
```

#### Error Handling

Configure how errors are handled:

```go
// Prevent panics and handle errors gracefully instead
insyra.Config.SetDontPanic(true)

// Check panic prevention status
isPanicPrevented := insyra.Config.GetDontPanicStatus()

// Set custom error handling function for all errors
insyra.Config.SetDefaultErrHandlingFunc(func(errType insyra.LogLevel, packageName, funcName, errMsg string) {
    // Your custom error handling logic
    // errType: The severity level of the error
    // packageName: The package where the error occurred
    // funcName: The function where the error occurred
    // errMsg: The error message
    // Use %v to print LogLevel values reliably
    fmt.Printf("[%v] %s.%s: %s\n", errType, packageName, funcName, errMsg)
})

// Get the current error handling function
handler := insyra.Config.GetDefaultErrHandlingFunc()
```

#### Performance Configuration

Fine-tune performance for your use case:

```go
// ⚠️ DANGER: Turn off thread safety for extreme performance
// Use ONLY when you are sure there are no concurrent accesses
// Data consistency is NOT guaranteed when this is disabled!
insyra.Config.Dangerously_TurnOffThreadSafety()

// If you need to reset all configs back to library defaults, call:
// Note: defaults are usually set on initialization, but this can be
// useful during tests or when switching configurations at runtime.
insyra.SetDefaultConfig()
```

#### Complete Example

```go
package main

import (
    "fmt"
    "github.com/HazelnutParadise/insyra"
)

func main() {
    // Initialize with custom configuration
    insyra.Config.SetLogLevel(insyra.LogLevelDebug)
    insyra.Config.SetDontPanic(true)
    
    // Custom error handler
    insyra.Config.SetDefaultErrHandlingFunc(func(errType insyra.LogLevel, pkg, fn, msg string) {
        fmt.Printf("ERROR in %s.%s: %s\n", pkg, fn, msg)
    })
    
    // Now use Insyra with these settings
    dl := insyra.NewDataList(1, 2, 3, 4, 5)
    fmt.Println(dl.Mean())
}
```

For implementation details, see the [config.go](config.go) source file.

## Thread Safety and Defensive Copies

- **Defensive copies:** Insyra returns defensive copies for all public data accessors. Any method that exposes internal slices, maps, or other mutable structures returns a copy so callers cannot mutate internal state unintentionally.
- **Atomic operations:** For safe concurrent multi-step operations, use the helper `AtomicDo`. `AtomicDo` serializes all operations for an instance via a dedicated actor goroutine and a command channel (see [atomic.go](atomic.go)), avoiding mutexes.

## [DataList](/Docs/DataList.md)

The `DataList` is the core structure in **Insyra**, enabling the storage, management, and analysis of dynamic data collections. It offers various methods for data manipulation and statistical analysis.

For a complete list of methods and features, please refer to the **[DataList Documentation](/Docs/DataList.md)**.

## [DataTable](/Docs/DataTable.md)

The `DataTable` structure provides a tabular data representation, allowing for the storage and manipulation of data in a structured format. It offers methods for data filtering, sorting, and aggregation, making it a powerful tool for data analysis.

**You can also convert between DataTables and CSV files with simply one line of code, enabling seamless integration with external data sources.**

### [Column Calculation Language (CCL)](/Docs/CCL.md)

**Insyra** features a powerful **Column Calculation Language (CCL)** that works just like Excel formulas!

With CCL, you can:

- Create calculated columns using familiar Excel-like syntax
- Reference columns using Excel-style notation (A, B, C...)
- Use conditional logic with `IF`, `AND`, `OR`, and `CASE` functions
- Perform mathematical operations and string manipulations
- Execute chained comparisons like `1 < A <= 10` for range checks
- Access specific rows using the `.` operator (e.g., `A.0`) and reference all columns with `@`
- Use aggregate functions like `SUM`, `AVG`, `COUNT`, `MAX`, and `MIN`

```go
// Add a column that classifies data based on values in column A
dt.AddColUsingCCL("category", "IF(A > 90, 'Excellent', IF(A > 70, 'Good', 'Average'))")

// Perform calculations just like in Excel
dt.AddColUsingCCL("total", "A + B + C")
dt.AddColUsingCCL("average", "AVG(A + B + C)")

// Use aggregate functions on rows or columns
dt.AddColUsingCCL("row_sum", "SUM(@.0)")

// Use range checks with chained comparisons (try this in Excel!)
dt.AddColUsingCCL("in_range", "IF(10 <= A <= 20, 'Yes', 'No')")
```

#### Parquet Integration

CCL can be applied **directly during Parquet file reading** to filter data at the source:

```go
// Filter rows while reading - only matching rows are loaded into memory
dt, err := parquet.FilterWithCCL(ctx, "sales_data.parquet", "(['amount'] > 1000) && (['status'] = 'Active')")

// Apply CCL transformations directly on parquet files (streaming mode)
err := parquet.ApplyCCL(ctx, "data.parquet", "NEW('total') = A + B + C")
```

This approach reduces memory usage when working with large datasets by processing data in batches.

For a complete guide to CCL syntax and features, see the **[CCL Documentation](/Docs/CCL.md)**.

For a complete list of DataTable methods and features, please refer to the **[DataTable Documentation](/Docs/DataTable.md)**.

## Packages

**Insyra** also provides several expansion packages, each focusing on a specific aspect of data analysis.

### **[isr](/Docs/isr.md)**

Provides **Syntactic Sugar** for **Insyra**. It is designed to simplify the usage of **Insyra** and make it more intuitive.

### **[stats](/Docs/stats.md)**

Provides statistical functions for data analysis, including skewness, kurtosis, and moment calculations.

### **[parallel](/Docs/parallel.md)**

Offers parallel processing capabilities for data manipulation and analysis. Allows you to execute any function and automatically wait for all goroutines to complete.

### **[plot](/Docs/plot.md)**

Provides a wrapper around the powerful [github.com/go-echarts/go-echarts](https://github.com/go-echarts/go-echarts) library, designed to simplify data visualization.

### **[gplot](/Docs/gplot.md)**

A visualization package based on [github.com/gonum/plot](https://github.com/gonum/plot). Fast and no need for Chrome. Even supports function plot.

### **[csvxl](/Docs/csvxl.md)**

Work with Excel and CSV files. Such as convert CSV to Excel.

### **[parquet](/Docs/parquet.md)**

Provides read and write support for the Apache Parquet file format, deeply integrated with Insyra's `DataTable` and `DataList`. Supports streaming, column-level reading, and CCL filtering.

### **[mkt](/Docs/mkt.md)**

Provides marketing-related data analysis functions, such as RFM analysis. No need to worry about how to calculate, one function does it all!

### **[py](/Docs/py.md)**

Execute Python code in Go without manually installing Python environment and dependencies. Allows passing variables between Go and Python.

### **[datafetch](/Docs/datafetch.md)**

Allows you to fetch data easily. It currently supports fetching comments from stores on Google Maps.

### **[lpgen](/Docs/lpgen.md)**

Provides a **super simple** and intuitive way to generate linear programming (LP) models and save them as `.lp` files. It supports setting objectives, adding constraints, defining variable bounds, and specifying binary or integer variables.

### **[lp](/Docs/lp.md)**

Fully automatic linear programming (LP) solver using [GLPK](https://www.gnu.org/software/glpk/).

## Advanced Usage

Beyond basic usage, **Insyra** provides extensive capabilities for handling different data types and performing complex statistical operations. Explore more in the **[detailed documentation](/Docs)**.

## Contributing

Contributions are welcome! You can contribute to **Insyra** by:

- **[Issues](https://github.com/HazelnutParadise/insyra/issues):** Reporting issues or suggesting new features.
- **[Pull Requests](https://github.com/HazelnutParadise/insyra/pulls):** Submitting pull requests to enhance the library.
- **[Discussions](https://github.com/HazelnutParadise/insyra/discussions):** Sharing your feedback and ideas to improve the project.
<!-- For more details, see the [contributing guidelines](https://github.com/HazelnutParadise/insyra/blob/main/CONTRIBUTING.md). -->

## Contributors

[![contributors](https://contrib.rocks/image?repo=HazelnutParadise/insyra)](https://github.com/HazelnutParadise/insyra/contributors)

## License

Insyra is licensed under the MIT License. See the [LICENSE](LICENSE) file for more information.
