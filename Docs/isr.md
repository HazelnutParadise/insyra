# [ isr ] Package

The `isr` package provides syntax sugar for insyra. **"isr"** stands for **"insyra"**, it's the first letters of the three syllables.

Every method in `isr` package supports method chaining. For example, you can use `DL{}.From().At()` to get the value at the specified index.

## Installation and Import

```bash
go get github.com/HazelnutParadise/insyra/isr
```

```go
import "github.com/HazelnutParadise/insyra/isr"
```

## Usage

### DataList

In `isr`, we use `DL{}` to represent a DataList. Use `DL{}.From()` to create a DataList.

#### From

```go
dataList := isr.DL{}.From(
    "A", "B", "C"
    ,1, 2, 3,
    ,4, 5, 6,
)
```

`DL{}.From()` is equivalent to `insyra.NewDataList()`. Hoever, `DL{}.From()` returns `*DL{}`, which can be used to call methods in `isr` package.

#### At

`At()` is equivalent to `insyra.DataList.Get()`. It returns the value at the specified index.

```go
dataList.At(0, 1) // 2
```

### DataTable

In `isr`, we use `DT{}` to represent a DataTable. Use `DT{}.From()` to create a DataTable.

#### From

`DT{}.From()` is equivalent to `insyra.NewDataTable()`. Hoever, `DT{}.From()` returns `*DT{}`, which can be used to call methods in `isr` package.

`DT{}.From()` is far more powerful, it converts a `DataList`, `DL`, `Row`, `Col`, `[]Row`, `[]Col`, `CSV`, `map[string]any`, or `map[int]any` to a DataTable.

##### From DataList

```go
dataTable := isr.DT{}.From(
    isr.DL{}.From("A", "B", "C"),
)
```

##### From Multiple DataLists

```go
dataTable := isr.DT{}.From(
    isr.DLs{
        isr.DL{}.From("A", "B", "C"),
        isr.DL{}.From(1, 2, 3),
        isr.DL{}.From(4, 5, 6),
    },
)
```

`DLs` is a type alias for `[]*DL`.

##### From Row

```go
dataTable := isr.DT{}.From(
    isr.Row{
        "A": 1,
        "B": 2,
        "C": 3,
    }
)
```

##### From []Col

```go
dataTable := isr.DT{}.From(
    []isr.Col{{
        isr.Name("Henrry"): 1,
        isr.Name("Peggy"):  2,
        isr.Name("Melody"): 3,
    }, {
        0: 4,
        1: 5,
        2: 6,
    }},
)
```

##### From CSV

```go
dataTable := isr.DT{}.From(
    isr.CSV{FilePath: "data.csv"},
)

// with options
dataTable := isr.DT{}.From(
    isr.CSV{
        FilePath: "data.csv",
        LoadOpts: isr.CSV_inOpts{
            FirstCol2RowNames: true,
            FirstRow2ColNames: true,
        },
    },
)
```