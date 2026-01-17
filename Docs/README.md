## Welcome to the Insyra Documentation

Insyra is a Go library for data analysis, providing intuitive data structures and tools for statistics, data manipulation, and visualization.

### Quick Start

```go
import "github.com/HazelnutParadise/insyra"

// Create a DataList (similar to a column or array)
dl := insyra.NewDataList(1, 2, 3, 4, 5)
fmt.Println(dl.Mean())  // Output: 3

// Create a DataTable (similar to a spreadsheet)
dt := insyra.NewDataTable(
    insyra.NewDataList("Alice", "Bob", "Charlie").SetName("Name"),
    insyra.NewDataList(25, 30, 35).SetName("Age"),
)
dt.Show()
```

### Documentation Structure

#### Core Data Structures

| Document                          | Description                                                  |
| --------------------------------- | ------------------------------------------------------------ |
| [DataList](DataList.md)           | One-dimensional data container with statistical methods      |
| [DataTable](DataTable.md)         | Two-dimensional table structure with row/column operations   |
| [Configuration](Configuration.md) | Global settings for logging, error handling, and performance |

#### Data Processing Languages

| Document      | Description                                                 |
| ------------- | ----------------------------------------------------------- |
| [CCL](CCL.md) | Column Calculation Language for DataTable column operations |
| [isr](isr.md) | Syntax sugar for fluent, readable code                      |

#### File I/O

| Document              | Description                                                |
| --------------------- | ---------------------------------------------------------- |
| [csvxl](csvxl.md)     | CSV and Excel file reading/writing with encoding detection |
| [parquet](parquet.md) | Apache Parquet file support with streaming                 |

#### Data Acquisition

| Document                  | Description                                                                                       |
| ------------------------- | ------------------------------------------------------------------------------------------------- |
| [datafetch](datafetch.md) | Google Maps store review crawler and Yahoo Finance wrapper (network required for remote fetchers) |

#### Statistical Analysis

| Document          | Description                                             |
| ----------------- | ------------------------------------------------------- |
| [stats](stats.md) | Correlation, hypothesis testing, regression, ANOVA, PCA |

#### Visualization

| Document          | Description                                                |
| ----------------- | ---------------------------------------------------------- |
| [gplot](gplot.md) | Static charts using Gonum (bar, line, scatter, histogram)  |
| [plot](plot.md)   | Interactive charts using ECharts (web-based visualization) |

#### Optimization

| Document          | Description                           |
| ----------------- | ------------------------------------- |
| [lp](lp.md)       | Linear programming solver using GLPK  |
| [lpgen](lpgen.md) | LP model generator with LINGO support |

#### Marketing Analytics

| Document      | Description                              |
| ------------- | ---------------------------------------- |
| [mkt](mkt.md) | RFM analysis and Customer Activity Index |

#### Integration & Utilities

| Document                | Description                                                 |
| ----------------------- | ----------------------------------------------------------- |
| [py](py.md)             | Execute Python code from Go with an auto-managed Python env |
| [parallel](parallel.md) | Simple parallel execution of functions                      |
| [utils](utils.md)       | Helper functions for type conversion and data processing    |
| [pd](pd.md)             | Pandas-like DataFrame helpers and conversions               |

### Installation

```bash
go get github.com/HazelnutParadise/insyra
```

Install every package at once:

```bash
go get github.com/HazelnutParadise/insyra/allpkgs
```

For sub-packages, install them individually:

```bash
go get github.com/HazelnutParadise/insyra/stats
go get github.com/HazelnutParadise/insyra/plot
# ... and so on
```

### Choosing the Right Tool

| If you need to...                         | Use                             |
| ----------------------------------------- | ------------------------------- |
| Store and analyze a single column of data | [DataList](DataList.md)         |
| Work with tabular data (rows and columns) | [DataTable](DataTable.md)       |
| Calculate new columns from existing data  | [CCL](CCL.md)                   |
| Read/write CSV or Excel files             | [csvxl](csvxl.md)               |
| Perform statistical tests                 | [stats](stats.md)               |
| Create static charts for reports          | [gplot](gplot.md)               |
| Create interactive web charts             | [plot](plot.md)                 |
| Solve optimization problems               | [lp](lp.md) + [lpgen](lpgen.md) |
| Analyze customer behavior                 | [mkt](mkt.md)                   |
| Use Python libraries from Go              | [py](py.md)                     |
| Run functions in parallel                 | [parallel](parallel.md)         |

### Requirements & Notes

- Go 1.25+ (per `go.mod`).
- Some packages download external tools or use network access (see each package doc for details).
