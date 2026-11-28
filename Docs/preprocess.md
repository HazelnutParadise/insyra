# [ preprocess ] Package

The `preprocess` package prepares a `DataTable` for a model: it scales numeric columns, encodes categorical columns and fills missing cells with values fitted on training data.

Each step is a fitted object. A fit learns its parameters from one table, and `Transform` applies the same parameters to any other table, so a test set is prepared with statistics from the training set and never with its own. The package takes an `insyra.IDataTable`, so a table from the `isr` package works without conversion. It is a package of its own, the way scikit-learn keeps these in `sklearn.preprocessing` and `sklearn.impute`, because functions that need not be `DataTable` methods stay out of the core. The in-place fill methods on the table (`FillForward`, `FillWithMean` and the rest) and `DataList.Normalize` / `Standardize` are not here: they use the values of the table or list they are called on and keep no fitted state.

```go
import "github.com/HazelnutParadise/insyra/preprocess"
```

## Table of Contents

- [Categorical Encoding](#categorical-encoding)
- [Feature Scaling](#feature-scaling)
- [Fitted Missing-Value Imputation](#fitted-missing-value-imputation)
- [Replacing the core methods](#replacing-the-core-methods)

### Categorical Encoding

```go
func OneHotEncode(dt insyra.IDataTable, opts OneHotOptions) (*insyra.DataTable, *OneHotEncoder, error)
func LabelEncode(dt insyra.IDataTable, opts LabelOptions) (*insyra.DataTable, *LabelEncoder, error)
func OrdinalEncode(dt insyra.IDataTable, opts OrdinalOptions) (*insyra.DataTable, *OrdinalEncoder, error)
```

**Description:** Categorical encoding turns string or mixed-type category columns into numeric columns that can feed `stats.LinearRegression`, KNN, PCA, and clustering. Each function returns a fresh `*DataTable`; the table passed in is not modified. The returned encoder stores the fitted category mapping and can `Transform` another table with the same schema, such as a test set or prediction batch.

Column references are column selectors: a bare string is an Excel-style index (`"A"`, `"B"`, ..., `"AA"`), `insyra.Name("price")` is a column name, and an `int` is a 0-based position (negative counts from the end). An integer category is identified by its value, so `int(1)` and the `int64(1)` a CSV load produces are one category, and an `Order` written as `[]any{1, 2, 3}` matches a CSV column. Other categories keep their type as well as their value, so `int(1)`, `1.0` and the string `"1"` are distinct. Missing means `nil` or `NaN`. For one-hot encoding, two distinct categories that would generate the same indicator column name (for example `int(1)` and `"1"`, both `c_1`, or `nil` and the string `"<nil>"`) are rejected at fit time; rename a category or set a distinct `Prefix`/`Separator`.

**Policies:**

| Type | Values | Behavior |
|---|---|---|
| `NaNPolicy` | `NaNAsCategory`, `NaNError`, `NaNSkip` | Missing becomes its own category, errors, or is skipped (`nil` for label/ordinal; all-zero for one-hot). |
| `UnknownPolicy` | `UnknownIgnore`, `UnknownError`, `UnknownAsNew` | On `Transform`, unseen categories become all-zero/`nil`, error, or are encoded as new outputs for that call. `UnknownAsNew` extends only the returned table; the fitted encoder is left unchanged, so `Transform` is pure and safe to reuse. |
| `LabelSort` | `LabelSortFirstSeen`, `LabelSortLexicographic`, `LabelSortByFrequency` | Controls label id assignment order. |

**Options:**

```go
type OneHotOptions struct {
    Columns        []any // each entry a column selector
    DropFirst      bool
    HandleNaN      NaNPolicy
    Unknown        UnknownPolicy
    Prefix         string
    Separator      string
    KeepOriginal   bool
    SortCategories bool
}

type LabelOptions struct {
    Column       any // a column selector
    NewColumn    string
    SortBy       LabelSort
    HandleNaN    NaNPolicy
    Unknown      UnknownPolicy
    KeepOriginal bool
}

type OrdinalOptions struct {
    Column       any // a column selector
    Order        []any
    NewColumn    string
    HandleNaN    NaNPolicy
    Unknown      UnknownPolicy
    KeepOriginal bool
}
```

`OneHotEncode` emits one `0/1` `int` column per category, named `<prefix><separator><category>`; by default the prefix is the source column name and the separator is `"_"`. A numeric category is written by the same rule as `ToCSV`, so `1500000.0` gives `price_1500000`. `DropFirst` omits the first category as the reference level. `LabelEncode` maps each class to an integer id. `OrdinalEncode` uses the explicit `Order` slice as `0..n-1`.

**Encoder interface and introspection:**

```go
type Encoder interface {
    Transform(dt *DataTable) (*DataTable, error)
    InverseTransform(dt *DataTable) (*DataTable, error)
    Kind() string
}

func (e *OneHotEncoder) Categories() map[string][]any
func (e *OneHotEncoder) OutputColumns() []string
func (e *LabelEncoder) Classes() []any
func (e *LabelEncoder) Inverse(values ...any) ([]any, error)
func (e *OrdinalEncoder) Classes() []any
func (e *OrdinalEncoder) Inverse(values ...any) ([]any, error)
```

**Example — fit, transform, inverse:**

```go
package main

import (
    "fmt"
    "log"

    "github.com/HazelnutParadise/insyra"
    "github.com/HazelnutParadise/insyra/preprocess"
)

func main() {
    train := insyra.NewDataTable(
        insyra.NewDataList("red", "blue", "red").SetName("color"),
        insyra.NewDataList(10, 20, 30).SetName("value"),
    )

    encoded, enc, err := preprocess.OneHotEncode(train, preprocess.OneHotOptions{
        Columns:   []any{insyra.Name("color")},
        DropFirst: true,
        Unknown:   preprocess.UnknownIgnore,
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(encoded.ColNames()) // [color_blue value]

    test := insyra.NewDataTable(
        insyra.NewDataList("blue", "green").SetName("color"),
        insyra.NewDataList(40, 50).SetName("value"),
    )
    encodedTest, err := enc.Transform(test)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(encodedTest.GetColByName("color_blue").Data()) // [1 0]

    originalShape, err := enc.InverseTransform(encoded)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(originalShape.ColNames()) // [color value]
}
```

**Example — categorical predictor into `stats.LinearRegression`:**

```go
package main

import (
    "fmt"
    "log"

    "github.com/HazelnutParadise/insyra"
    "github.com/HazelnutParadise/insyra/preprocess"
    "github.com/HazelnutParadise/insyra/stats"
)

func main() {
    dt := insyra.NewDataTable(
        insyra.NewDataList(100.0, 130.0, 110.0, 150.0).SetName("sales"),
        insyra.NewDataList("basic", "pro", "basic", "pro").SetName("plan"),
    )

    x, _, err := preprocess.OneHotEncode(dt, preprocess.OneHotOptions{
        Columns:   []any{insyra.Name("plan")},
        DropFirst: true, // baseline = first-seen category ("basic")
    })
    if err != nil {
        log.Fatal(err)
    }

    fit, err := stats.LinearRegression(
        dt.GetColByName("sales"),
        x.GetColByName("plan_pro"),
    )
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(fit.Coefficients) // intercept, effect of plan_pro
}
```

### Feature Scaling

```go
func StandardScale(dt insyra.IDataTable, cols ...any) (*insyra.DataTable, *StandardScaler, error)
func MinMaxScale(dt insyra.IDataTable, featureMin, featureMax float64, cols ...any) (*insyra.DataTable, *MinMaxScaler, error)
func RobustScale(dt insyra.IDataTable, cols ...any) (*insyra.DataTable, *RobustScaler, error)
func MaxAbsScale(dt insyra.IDataTable, cols ...any) (*insyra.DataTable, *MaxAbsScaler, error)
```

**Description:** Feature scalers fit numeric scaling parameters once and reuse them. Each function returns a fresh `*DataTable` (the table passed in is not modified) plus a fitted scaler. The scaler can `Transform` and `InverseTransform` other tables with the same parameters.

**Why this is not `DataList.Normalize` / `Standardize`:** `Normalize()` and `Standardize()` are stateless and modify the list in place — they recompute min/max or mean/std from whatever data they are handed. That is exactly what you must *not* do to a test set: scaling test data with its own statistics leaks information and makes train/test results incomparable. A `Scaler` fits on the **training** set and applies those frozen parameters to the test set, which is the correct, leakage-free workflow.

**Choosing a scaler:**

| Scaler | Centers on | Scales by | Use when |
|---|---|---|---|
| `StandardScaler` | mean | sample std dev (matches `Standardize`) | roughly Gaussian features; the common default |
| `MinMaxScaler` | min | range, into `[featureMin, featureMax]` | you need a bounded range (e.g. `[0,1]`) and have few outliers |
| `RobustScaler` | median | IQR (Q3−Q1), type-7 quantile — matches scikit-learn's `RobustScaler` | features have outliers that would distort mean/std |
| `MaxAbsScaler` | 0 | max absolute value, into `[-1,1]` | sparse or sign-meaningful data you don't want to shift |

**Behavior:**

- Column references are column selectors: a bare string is an Excel-style index (`"A"`, `"B"`, ..., `"AA"`), `insyra.Name("price")` is a column name, and an `int` is a 0-based position (negative counts from the end). `cols` is required.
- Only the listed columns are scaled; other columns pass through unchanged, preserving column order, names, table name, and row names.
- `nil` and `NaN` are preserved and excluded from fitting (they do not affect the computed parameters).
- A non-numeric, non-missing value in a target column is an error.
- `Transform` errors if a fitted column is missing from the input. `InverseTransform` only restores fitted columns that are present and passes others through (so a prediction table covering a subset still works).
- Constant / degenerate columns (`std==0`, `max==min`, `IQR==0`, `maxAbs==0`) do not panic: the fitted training data maps to the degenerate output (`0`, or `featureMin` for min-max) and the inverse round-trips.

**Interface and introspection:**

```go
type Scaler interface {
    Fit(dt *DataTable, cols ...any) error
    Transform(dt *DataTable) (*DataTable, error)
    FitTransform(dt *DataTable, cols ...any) (*DataTable, error)
    InverseTransform(dt *DataTable) (*DataTable, error)
    Params() map[string]ScalerParams
    Kind() string
}

func NewStandardScaler() *StandardScaler
func NewMinMaxScaler(featureMin, featureMax float64) *MinMaxScaler
func NewDefaultMinMaxScaler() *MinMaxScaler // [0, 1]
func NewRobustScaler() *RobustScaler
func NewMaxAbsScaler() *MaxAbsScaler
```

`Params()` returns the fitted parameters keyed by column name (`Mean`/`Std`, `Min`/`Max`/`OutputMin`/`OutputMax`, `Median`/`Q1`/`Q3`/`IQR`, or `MaxAbs` depending on the kind).

There is also a DataList-oriented counterpart (`DataListScaler`) on every scaler: `FitDataList`, `TransformDataList`, `FitTransformDataList`, `InverseTransformDataList`, each returning a new `*DataList`.

**Saving a fitted scaler:** every scaler implements `json.Marshaler` and `json.Unmarshaler`, so a scaler fitted in one program can transform tables in another. The JSON holds the kind, the output range, and each fitted column, remembered by its name, or by its position when it has no name, whichever selector it was fitted by; a table the scaler transforms later must name those columns the same way. A column listed twice is refused. Decode it into the type the scaler was created as. JSON from a different kind returns an error and leaves the receiver as it was. A parameter that is NaN or infinite, such as the mean of a column that held only missing values, is written as the string `"NaN"`, `"+Inf"` or `"-Inf"` and read back as that value.

```go
b, err := json.Marshal(sc) // sc is a fitted *preprocess.StandardScaler
if err != nil {
    log.Fatal(err)
}

// Later, possibly in another process:
restored := new(preprocess.StandardScaler)
if err := json.Unmarshal(b, restored); err != nil {
    log.Fatal(err)
}
testScaled, err := restored.Transform(test) // same result as sc.Transform(test)
```

**Example — fit on train, transform test (no leakage):**

```go
package main

import (
    "fmt"
    "log"

    "github.com/HazelnutParadise/insyra"
    "github.com/HazelnutParadise/insyra/preprocess"
)

func main() {
    data := insyra.NewDataTable(
        insyra.NewDataList(20.0, 30.0, 40.0, 50.0, 60.0).SetName("Age"),
        insyra.NewDataList(30.0, 45.0, 50.0, 70.0, 90.0).SetName("Income"),
    )
    train, test := data.TrainTestSplit(0.8, insyra.SamplingOptions{UseSeed: true, Seed: 42})

    // Fit the scaler on the training set only.
    sc := preprocess.NewStandardScaler()
    trainScaled, err := sc.FitTransform(train, insyra.Name("Age"), insyra.Name("Income"))
    if err != nil {
        log.Fatal(err)
    }

    // Apply the SAME parameters to the test set.
    testScaled, err := sc.Transform(test)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(sc.Params()["Age"]) // {Age standard <mean> <std> ...}
    _ = trainScaled
    _ = testScaled

    // Recover the original scale (e.g. for predictions).
    restored, err := sc.InverseTransform(trainScaled)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(restored.GetColByName("Age").Data())
}
```

### Fitted Missing-Value Imputation

```go
type ImputationStrategy string

const (
    ImputeMean     ImputationStrategy = "mean"
    ImputeMedian   ImputationStrategy = "median"
    ImputeMode     ImputationStrategy = "mode"
    ImputeConstant ImputationStrategy = "constant"
)

type SimpleImputerOptions struct {
    Strategy  ImputationStrategy // how each fill value is computed; empty means ImputeMean
    FillValue any                // the value ImputeConstant fills with; nil for every other strategy
}

func NewSimpleImputer(opts ...SimpleImputerOptions) *SimpleImputer
```

`SimpleImputer` learns a fill value once and fills many tables with it. It
fits one replacement value per selected column and reuses those values on
later tables. The name and the four strategies are scikit-learn's. With no
options it fills with each column's mean; `ImputeConstant` needs `FillValue`,
and giving `FillValue` to any other strategy is an error that `Fit` reports. Use it for training, validation, and production
data so the statistics come from the training table only. `Transform` returns
a new table and leaves the source unchanged. Columns not selected during
`Fit` pass through unchanged. Missing values are `nil` or `NaN`.
`FitTransform(dt, cols...)` fits on `dt` and returns its filled copy in one
call, like the scalers' method of the same name; `dt` itself is not changed.

Mean and median require a number column. `Fit` fails if any selected column
holds other values, naming the column and its data type, and learns nothing:
you selected the column, so leaving it unfilled would make the fit look
complete when it is not. The table's `FillWithMean` and `FillWithMedian` report
a column you named and they cannot fill the same way. Use `ImputeMode` or
`ImputeConstant` for a text column. Mode supports mixed values and uses the first-occurring
value to break ties. Every selected column must have at least one observed value
at fit time, and `Fit` reports the column name otherwise. `SimpleImputer` has no
`InverseTransform` and is not an `preprocess.Scaler`, because imputation cannot
recover which cells were originally missing. That absence is deliberate rather
than a gap: a method that always returned an error would still satisfy any
interface asking for it, so code testing for the capability by type assertion
would be told it is present and then refused at the call. Not having the method
is the only form of that answer a type assertion can read.

Do not use the fitted imputer and the in-place `FillWithMean`,
`FillWithMedian`, or `FillWithMode` interchangeably. Use the fitted form in a
model pipeline; use the in-place methods for one-off table cleaning when
recomputing from the table being modified is intentional.

**Example — fit on training data, then reuse the replacement:**

```go
package main

import (
    "log"

    "github.com/HazelnutParadise/insyra"
    "github.com/HazelnutParadise/insyra/preprocess"
)

func main() {
    train := insyra.NewDataTable(
        insyra.NewDataList(10.0, nil, 30.0).SetName("income"),
    )
    test := insyra.NewDataTable(
        insyra.NewDataList(100.0, nil).SetName("income"),
    )

    imputer := preprocess.NewSimpleImputer() // mean; or SimpleImputerOptions{Strategy: preprocess.ImputeMedian}
    if err := imputer.Fit(train, insyra.Name("income")); err != nil {
        log.Fatal(err)
    }
    imputedTest, err := imputer.Transform(test) // nil becomes 20.0
    if err != nil {
        log.Fatal(err)
    }
    _ = imputedTest
}
```

## Replacing the core methods

The scalers, encoders and the fitted imputer used to be `DataTable` methods and root `insyra` names. Those still work in 0.3.x, are marked `Deprecated`, and are removed in 0.4, where this package is the only place they live. This package already has the 0.4 shape, so a program changed to it now needs no further change for 0.4.

| Deprecated | Use |
|---|---|
| `dt.StandardScale(cols...)` | `preprocess.StandardScale(dt, cols...)` |
| `dt.MinMaxScale(min, max, cols...)` | `preprocess.MinMaxScale(dt, min, max, cols...)` |
| `dt.RobustScale(cols...)` | `preprocess.RobustScale(dt, cols...)` |
| `dt.MaxAbsScale(cols...)` | `preprocess.MaxAbsScale(dt, cols...)` |
| `dt.OneHotEncode(opts)` | `preprocess.OneHotEncode(dt, opts)` |
| `dt.LabelEncode(opts)` | `preprocess.LabelEncode(dt, opts)` |
| `dt.OrdinalEncode(opts)` | `preprocess.OrdinalEncode(dt, opts)` |
| `insyra.OneHotOptions` | `preprocess.OneHotOptions` |
| `insyra.LabelEncodeOptions` | `preprocess.LabelOptions` |
| `insyra.OrdinalEncodeOptions` | `preprocess.OrdinalOptions` |
| `insyra.NewStandardScaler`, `NewMinMaxScaler`, `NewDefaultMinMaxScaler`, `NewRobustScaler`, `NewMaxAbsScaler` | the same names in `preprocess` |
| `insyra.StandardScaler`, `MinMaxScaler`, `RobustScaler`, `MaxAbsScaler`, `Scaler`, `DataListScaler`, `ScalerParams` | the same names in `preprocess` |
| `insyra.OneHotEncoder`, `LabelEncoder`, `OrdinalEncoder`, `Encoder` | the same names in `preprocess` |
| `insyra.NaNPolicy`, `UnknownPolicy`, `LabelSort` and their constants | the same names in `preprocess` |
| `insyra.NewSimpleImputer`, `SimpleImputer`, `SimpleImputerOptions`, `ImputationStrategy`, `ImputeMean`, `ImputeMedian`, `ImputeMode`, `ImputeConstant` | the same names in `preprocess` |

Three things differ from the deprecated methods, all of them the 0.4 behaviour:

- **A bare string column is an Excel-style index.** The core methods read `"price"` as a column name and fall back to an index; here `"price"` is refused as an index, and the name is written `insyra.Name("price")`. `"A"` and `0` still pick the first column. A typo in a name is reported instead of silently matching a column whose letters it happens to spell.
- **A large float category keeps its digits in a one-hot column name.** `1500000.0` gives `price_1500000`, where the core method gives `price_1.5e+06`.
- **The fitted objects are separate types.** A `*preprocess.StandardScaler` is not an `*insyra.StandardScaler`, so a variable typed as the core type has to change with the call. `ml`'s pipelines, column transformers and ONNX export take both kinds, and export the same graph for both.

Results and error text are otherwise the same, and the scaler JSON format is the same, so a scaler saved by the core type loads into the `preprocess` type with `json.Unmarshal`. The CLI commands `scale` and `encode` keep using the core types until 0.4 and behave as before.

```go
// Before
scaled, scaler, err := dt.StandardScale("age", "income")

// After
scaled, scaler, err := preprocess.StandardScale(dt, insyra.Name("age"), insyra.Name("income"))
```
