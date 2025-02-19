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

In `isr`, we use `DL{}` to represent a DataList. Use `DL{}.From()` to create a DataList. Use `PrtDL()` to convert a `*DataList` to a `*DL`.

#### PtrDL

PtrDL converts a DataList or DL to a *DL.
If you have a DataList created by `insyra` main package, you can use `PtrDL()` to convert a it to a `*DL`.

```go
dataList := insyra.NewDataList(1, 2, 3)
ptrDataList := isr.PtrDL(dataList)

// then you can use ptrDataList to call methods in isr package
value := ptrDataList.At(0)
```

If any DataList fails to call methods from the `isr` package, try it.

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

#### Push

`Push()` is equivalent to `insyra.DataList.Append()`. It appends the specified values to the end of the DataList.

```go
dataList.Push(7, 8, 9)
```

### DataTable

In `isr`, we use `DT{}` to represent a DataTable. Use `DT{}.From()` to create a DataTable. Use `PrtDT()` to convert a `*DataTable` to a `*DT`.

#### PtrDT

PtrDT converts a DataTable or DT to a *DT.
If you have a DataTable created by `insyra` main package, you can use `PtrDT()` to convert a it to a `*DT`.

```go
dataTable := insyra.NewDataTable()
ptrDataTable := isr.PtrDT(dataTable)

// then you can use ptrDataTable to call methods in isr package
colZero := ptrDataTable.Col(0)
```

If any DataTable fails to call methods from the `isr` package, try it.

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

#### Row

`Row()` is equivalent to `insyra.DataTable.GetRow()`. It returns the row at the specified index.

#### Col

`Col()` is equivalent to `insyra.DataTable.GetCol()`. It returns the column at the specified index. The column can be accessed by index(A, B, C...) or number.

#### At

`At()` is equivalent to `insyra.DataTable.GetElement()`. It returns the value at the specified row and column. The column can be accessed by index(A, B, C...) or number.

```go
dataTable.At(0, 1)
```
