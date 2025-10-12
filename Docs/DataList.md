# DataList

DataList is a fundamental data structure in Insyra that provides a dynamic, generic way to store and manipulate collections of data. It supports various data types including strings, numbers, booleans, and temporal data, offering powerful data analysis and manipulation capabilities.

## Table of Contents

- [Data Structure](#data-structure)
- [Creating DataList](#creating-datalist)
- [Data Access](#data-access)
- [Data Manipulation](#data-manipulation)
- [Data Filtering](#data-filtering)
- [Data Preprocessing](#data-preprocessing)
- [Statistical Analysis](#statistical-analysis)
- [Data Transformation](#data-transformation)
- [Interpolation Methods](#interpolation-methods)
- [Data Visualization](#data-visualization)
- [Data Comparison](#data-comparison)
- [Data Conversion](#data-conversion)
- [Metadata Management](#metadata-management)
- [Utility Methods](#utility-methods)
- [AtomicDo](#atomicdo)
- [Notes](#notes)

## Data Structure

### DataList Struct

```go
type DataList struct {
    data                  []any
    name                  string
    creationTimestamp     int64
    lastModifiedTimestamp atomic.Int64

    // AtomicDo support (actor-style serialization)
    initOnce sync.Once
    cmdCh    chan func()
    closed   atomic.Bool
}
```

**Field Descriptions:**

- `data`: Slice containing the actual data elements of any type
- `name`: Optional name for the DataList
- `creationTimestamp`: Unix timestamp when the DataList was created
- `lastModifiedTimestamp`: Unix timestamp when the DataList was last modified
- `initOnce`, `cmdCh`, `closed`: Internal fields enabling `AtomicDo` actor-style, serialized execution for thread-safety without external locks

### Naming Conventions

- **List Names**: Use snake-style Pascal case (e.g., `Factor_Loadings`, `Communalities`) to avoid spelling errors caused by spaces.

## AtomicDo

`AtomicDo` provides safe, serialized access to a DataList using an internal actor goroutine. It ensures all mutations and reads inside the function run in order and without races, even across multiple goroutines.

```go
func (dl *DataList) AtomicDo(f func(*DataList))
```

- Single-threaded execution: Functions passed to `AtomicDo` are processed one at a time.
- Reentrant: If called from within `AtomicDo`, the function runs immediately (no deadlock).
- Cross-list nesting: Calling `anotherDl.AtomicDo` inside `dl.AtomicDo` is supported; nested calls execute inline to avoid deadlocks.
- Closed behavior: After `dl.Close()`, `AtomicDo` executes the function inline without scheduling.

Examples

- Batch update safely from multiple goroutines:

```go
dl := insyra.NewDataList()
wg := sync.WaitGroup{}
for i := 0; i < 100; i++ {
    wg.Add(1)
    go func(v int) {
        defer wg.Done()
        dl.AtomicDo(func(dl *insyra.DataList) {
            dl.Append(v)
        })
    }(i)
}
wg.Wait()
```

- Multi-step, consistent transformation:

```go
dl.AtomicDo(func(dl *insyra.DataList) {
    // read-modify-write happens atomically relative to other calls
    if dl.Len() > 0 {
        first := dl.Get(0)
        dl.InsertAt(0, first)
    }
})
```

Guidelines

- Keep functions short; avoid long blocking work inside `AtomicDo`.
- Do heavy computation outside and mutate inside `AtomicDo`.
- Prefer `AtomicDo` for any sequence of dependent operations that must see a consistent view.

## Creating DataList

### NewDataList

Creates a new DataList instance with variadic parameters, automatically flattening nested slices but not arrays.

```go
func NewDataList(values ...any) *DataList
```

**Parameters:**

- `values`: Variadic list of elements to initialize the DataList with

**Returns:**

- `*DataList`: A newly created DataList

**Flattening Behavior:**

- **Slices** are automatically flattened (e.g., `[]int{1, 2}` becomes `1, 2`)
- **Arrays** are kept as single elements (e.g., `[3]int{1, 2, 3}` remains as one element)
- Nested slices are recursively flattened
- Other types are preserved as-is

**Example:**

```go
// Create with individual values
dl := insyra.NewDataList(1, 2, 3, 4, 5)

// Create with mixed types
dl := insyra.NewDataList("Alice", 25, true, 3.14)

// Create with nested slices (automatically flattened)
dl := insyra.NewDataList([]int{1, 2}, []string{"a", "b"}) // Results in: [1, 2, "a", "b"]

// Arrays are not flattened
dl := insyra.NewDataList([3]int{1, 2, 3}, 4) // Results in: [[1, 2, 3], 4]
```

### From

Factory method for creating a DataList with a more modern syntax.

```go
func (_ DataList) From(values ...any) *DataList
```

**Parameters:**

- `values`: Variadic list of elements to initialize the DataList with

**Returns:**

- `*DataList`: A newly created DataList

**Example:**

```go
// Recommended modern syntax
dl := insyra.DataList{}.From(1, 2, 3, 4, 5)

// Using syntax sugar
import "github.com/HazelnutParadise/insyra/isr"
dl := isr.DL{}.From(1, 2, 3, 4, 5)
```

## Data Access

### Get

Retrieves an element at a specific index with support for negative indexing.

```go
func (dl *DataList) Get(index int) any
```

**Parameters:**

- `index`: Zero-based index position (negative values count from the end)

**Returns:**

- `any`: The element at the specified index

**Example:**

```go
dl := insyra.NewDataList(10, 20, 30, 40, 50)
fmt.Println(dl.Get(0))  // 10 (first element)
fmt.Println(dl.Get(-1)) // 50 (last element)
fmt.Println(dl.Get(-2)) // 40 (second to last)
```

### Data

Returns the underlying data slice.

```go
func (dl *DataList) Data() []any
```

**Returns:**

- `[]any`: The underlying data slice

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3)
data := dl.Data() // []any{1, 2, 3}
```

### Len

Returns the number of elements in the DataList.

```go
func (dl *DataList) Len() int
```

**Returns:**

- `int`: Number of elements in the DataList

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 4, 5)
fmt.Println(dl.Len()) // 5
```

## Data Manipulation

### Append

Adds new elements to the end of the DataList.

```go
func (dl *DataList) Append(values ...any)
```

**Parameters:**

- `values`: Variadic list of elements to append

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3)
dl.Append(4, 5, 6)
// dl now contains: [1, 2, 3, 4, 5, 6]
```

### Update

Updates an element at a specific index.

```go
func (dl *DataList) Update(index int, value any)
```

**Parameters:**

- `index`: Zero-based index position
- `value`: New value to set at the index

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3)
dl.Update(1, 99)
// dl now contains: [1, 99, 3]
```

### InsertAt

Inserts a new element at a specific index, shifting existing elements to the right.

```go
func (dl *DataList) InsertAt(index int, value any)
```

**Parameters:**

- `index`: Zero-based index position for insertion
- `value`: Value to insert

**Example:**

```go
dl := insyra.NewDataList(1, 3, 4)
dl.InsertAt(1, 2)
// dl now contains: [1, 2, 3, 4]
```

### Pop

Removes and returns the last element from the DataList.

```go
func (dl *DataList) Pop() any
```

**Returns:**

- `any`: The removed last element

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 4, 5)
lastElement := dl.Pop() // returns 5
// dl now contains: [1, 2, 3, 4]
```

### Drop

Removes an element at a specific index.

```go
func (dl *DataList) Drop(index int) *DataList
```

**Parameters:**

- `index`: Zero-based index position of element to remove

**Returns:**

- `*DataList`: Reference to the modified DataList

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 4, 5)
dl.Drop(2) // removes element at index 2
// dl now contains: [1, 2, 4, 5]
```

### DropAll

Removes all occurrences of specified values from the DataList.

```go
func (dl *DataList) DropAll(values ...any) *DataList
```

**Parameters:**

- `values`: Variadic list of values to remove

**Returns:**

- `*DataList`: Reference to the modified DataList

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 2, 4, 2, 5)
dl.DropAll(2, 4)
// dl now contains: [1, 3, 5]
```

### DropIfContains

Removes all elements that contain a specified substring (for string elements).

```go
func (dl *DataList) DropIfContains(substring any) *DataList
```

**Parameters:**

- `substring`: Substring to search for in string elements

**Returns:**

- `*DataList`: Reference to the modified DataList

**Example:**

```go
dl := insyra.NewDataList("apple", "banana", "grape", "orange")
dl.DropIfContains("an")
// dl now contains: ["apple", "grape"]
```

### Clear

Removes all elements from the DataList.

```go
func (dl *DataList) Clear() *DataList
```

**Returns:**

- `*DataList`: Reference to the cleared DataList

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 4, 5)
dl.Clear()
// dl now contains: []
```

### Sort

Sorts the elements in the DataList using mixed sorting logic for different data types.

```go
func (dl *DataList) Sort(ascending ...bool) *DataList
```

**Parameters:**

- `ascending`: Optional boolean to specify sort order (default: true for ascending)

**Returns:**

- `*DataList`: Reference to the sorted DataList

**Type Priority for Mixed Types:**

When sorting DataLists containing different data types, elements are first sorted by type priority, then by their natural order within the same type. The type priority (from highest to lowest in ascending order) is:

1. `nil` - Null values
2. `bool` - Boolean values (false before true)
3. Numeric types (`int`, `uint`, `float`, etc.) - Numbers by value
4. `string` - Strings in lexicographical order
5. `time.Time` - Time values chronologically
6. Other types - Custom types by string representation

**Example:**

```go
dl := insyra.NewDataList(3, 1, 4, 1, 5, 9)
dl.Sort() // ascending order
// dl now contains: [1, 1, 3, 4, 5, 9]

dl.Sort(false) // descending order
// dl now contains: [9, 5, 4, 3, 1, 1]

// Mixed types example
dl2 := insyra.NewDataList(nil, true, 42, "hello", time.Now())
dl2.Sort()
// dl2 will be sorted: [nil, true, 42, "hello", time_value]
```

### Reverse

Reverses the order of elements in the DataList.

```go
func (dl *DataList) Reverse() *DataList
```

**Returns:**

- `*DataList`: Reference to the reversed DataList

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 4, 5)
dl.Reverse()
// dl now contains: [5, 4, 3, 2, 1]
```

### Map

Applies a transformation function to all elements with index-aware processing.

```go
func (dl *DataList) Map(mapFunc func(int, any) any) *DataList
```

**Parameters:**

- `mapFunc`: Function that takes index and value, returns transformed value

**Returns:**

- `*DataList`: New DataList with transformed values

**Example:**

```go
// Add index to each element
dl := insyra.NewDataList("a", "b", "c")
result := dl.Map(func(index int, value any) any {
    return fmt.Sprintf("Index %d: %s", index, value.(string))
})
// Result: ["Index 0: a", "Index 1: b", "Index 2: c"]

// Apply different transformations based on index
numbers := insyra.NewDataList(1, 2, 3, 4, 5)
result := numbers.Map(func(index int, value any) any {
    if index%2 == 0 {
        return value.(int) * 2  // Even indices: multiply by 2
    }
    return value.(int) + 10     // Odd indices: add 10
})
// Result: [2, 12, 6, 14, 10]
```

## Data Filtering

### Filter

Filters elements based on a predicate function.

```go
func (dl *DataList) Filter(predicate func(any) bool) *DataList
```

**Parameters:**

- `predicate`: Function that returns true for elements to keep

**Returns:**

- `*DataList`: New DataList with filtered elements

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
evens := dl.Filter(func(value any) bool {
    return value.(int)%2 == 0
})
// evens contains: [2, 4, 6, 8, 10]
```

### ClearStrings

Removes all string elements from the DataList.

```go
func (dl *DataList) ClearStrings() *DataList
```

**Returns:**

- `*DataList`: Reference to the modified DataList

**Example:**

```go
dl := insyra.NewDataList(1, "hello", 2, "world", 3)
dl.ClearStrings()
// dl now contains: [1, 2, 3]
```

### ClearNumbers

Removes all numeric elements from the DataList.

```go
func (dl *DataList) ClearNumbers() *DataList
```

**Returns:**

- `*DataList`: Reference to the modified DataList

**Example:**

```go
dl := insyra.NewDataList(1, "hello", 2, "world", 3)
dl.ClearNumbers()
// dl now contains: ["hello", "world"]
```

### ClearNaNs

Removes all NaN (Not a Number) elements from the DataList.

```go
func (dl *DataList) ClearNaNs() *DataList
```

**Returns:**

- `*DataList`: Reference to the modified DataList

**Example:**

```go
dl := insyra.NewDataList(1.0, math.NaN(), 2.0, math.NaN(), 3.0)
dl.ClearNaNs()
// dl now contains: [1.0, 2.0, 3.0]
```

### ClearOutliers

Removes values outside a specified number of standard deviations from the mean.

```go
func (dl *DataList) ClearOutliers(stdDev float64) *DataList
```

**Parameters:**

- `stdDev`: Number of standard deviations to use as threshold

**Returns:**

- `*DataList`: Reference to the modified DataList

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 4, 5, 100) // 100 is an outlier
dl.ClearOutliers(2.0) // Remove values beyond 2 standard deviations
// dl now contains: [1, 2, 3, 4, 5]
```

## Data Preprocessing

### Normalize

Normalizes DataList elements to a specified range (default: 0 to 1).

```go
func (dl *DataList) Normalize() *DataList
```

**Returns:**

- `*DataList`: Reference to the normalized DataList

**Example:**

```go
dl := insyra.NewDataList(10, 20, 30, 40, 50)
dl.Normalize()
// Values are normalized to range [0, 1]
```

### Standardize

Standardizes DataList elements using z-score normalization (mean=0, std=1).

```go
func (dl *DataList) Standardize() *DataList
```

**Returns:**

- `*DataList`: Reference to the standardized DataList

**Example:**

```go
dl := insyra.NewDataList(10, 20, 30, 40, 50)
dl.Standardize()
// Values are standardized to have mean=0 and std=1
```

### FillNaNWithMean

Replaces all NaN values with the mean value of numeric elements.

```go
func (dl *DataList) FillNaNWithMean() *DataList
```

**Returns:**

- `*DataList`: Reference to the modified DataList

**Example:**

```go
dl := insyra.NewDataList(1.0, math.NaN(), 3.0, math.NaN(), 5.0)
dl.FillNaNWithMean()
// NaN values are replaced with mean (3.0)
```

### ReplaceOutliers

Replaces outliers beyond specified standard deviations with a replacement value.

```go
func (dl *DataList) ReplaceOutliers(stdDev float64, replacement float64) *DataList
```

**Parameters:**

- `stdDev`: Number of standard deviations to use as threshold
- `replacement`: Value to replace outliers with

**Returns:**

- `*DataList`: Reference to the modified DataList

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 4, 5, 100)
dl.ReplaceOutliers(2.0, 6.0) // Replace outliers with 6.0
```

## Statistical Analysis

### Sum

Calculates the sum of numeric elements in the DataList.

```go
func (dl *DataList) Sum() float64
```

**Returns:**

- `float64`: Sum of all numeric elements

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 4, 5)
sum := dl.Sum() // 15.0
```

### Max

Returns the maximum value among numeric elements.

```go
func (dl *DataList) Max() float64
```

**Returns:**

- `float64`: Maximum numeric value

**Example:**

```go
dl := insyra.NewDataList(1, 5, 3, 9, 2)
max := dl.Max() // 9.0
```

### Min

Returns the minimum value among numeric elements.

```go
func (dl *DataList) Min() float64
```

**Returns:**

- `float64`: Minimum numeric value

**Example:**

```go
dl := insyra.NewDataList(1, 5, 3, 9, 2)
min := dl.Min() // 1.0
```

### Mean

Calculates the arithmetic mean of numeric elements.

```go
func (dl *DataList) Mean() float64
```

**Returns:**

- `float64`: Arithmetic mean

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 4, 5)
mean := dl.Mean() // 3.0
```

### WeightedMean

Calculates the weighted mean using provided weights.

```go
func (dl *DataList) WeightedMean(weights any) float64
```

**Parameters:**

- `weights`: Weights as DataList or slice of float64

**Returns:**

- `float64`: Weighted mean

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 4, 5)
weights := insyra.NewDataList(0.1, 0.2, 0.3, 0.2, 0.2)
wmean := dl.WeightedMean(weights)
```

### GMean

Calculates the geometric mean of numeric elements.

```go
func (dl *DataList) GMean() float64
```

**Returns:**

- `float64`: Geometric mean

**Example:**

```go
dl := insyra.NewDataList(1, 2, 4, 8)
gmean := dl.GMean() // 2.83 (approximately)
```

### Median

Returns the median value after sorting elements.

```go
func (dl *DataList) Median() float64
```

**Returns:**

- `float64`: Median value

**Example:**

```go
dl := insyra.NewDataList(1, 3, 2, 5, 4)
median := dl.Median() // 3.0
```

### Mode

Returns the most frequent value(s) in the DataList.

```go
func (dl *DataList) Mode() []float64
```

**Returns:**

- `[]float64`: Slice of mode values

**Example:**

```go
dl := insyra.NewDataList(1, 2, 2, 3, 3, 3)
modes := dl.Mode() // [3.0]
```

### Stdev

Calculates the sample standard deviation.

```go
func (dl *DataList) Stdev() float64
```

**Returns:**

- `float64`: Sample standard deviation

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 4, 5)
stdev := dl.Stdev()
```

### StdevP

Calculates the population standard deviation.

```go
func (dl *DataList) StdevP() float64
```

**Returns:**

- `float64`: Population standard deviation

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 4, 5)
stdevp := dl.StdevP()
```

### Var

Calculates the sample variance.

```go
func (dl *DataList) Var() float64
```

**Returns:**

- `float64`: Sample variance

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 4, 5)
variance := dl.Var()
```

### VarP

Calculates the population variance.

```go
func (dl *DataList) VarP() float64
```

**Returns:**

- `float64`: Population variance

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 4, 5)
variancep := dl.VarP()
```

### Range

Returns the difference between maximum and minimum values.

```go
func (dl *DataList) Range() float64
```

**Returns:**

- `float64`: Range (max - min)

**Example:**

```go
dl := insyra.NewDataList(1, 5, 3, 9, 2)
range := dl.Range() // 8.0 (9 - 1)
```

### Quartile

Calculates quartile values (Q1, Q2, Q3).

```go
func (dl *DataList) Quartile(q int) float64
```

**Parameters:**

- `q`: Quartile number (1, 2, or 3)

**Returns:**

- `float64`: Quartile value

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
q1 := dl.Quartile(1) // First quartile
q2 := dl.Quartile(2) // Second quartile (median)
q3 := dl.Quartile(3) // Third quartile
```

### IQR

Calculates the interquartile range (Q3 - Q1).

```go
func (dl *DataList) IQR() float64
```

**Returns:**

- `float64`: Interquartile range

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
iqr := dl.IQR()
```

### Percentile

Calculates the value below which a given percentage of observations fall.

```go
func (dl *DataList) Percentile(percentile float64) float64
```

**Parameters:**

- `percentile`: Percentile value (0-100)

**Returns:**

- `float64`: Value at the specified percentile

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
p75 := dl.Percentile(75) // 75th percentile
```

### MAD

Calculates the median absolute deviation.

```go
func (dl *DataList) MAD() float64
```

**Returns:**

- `float64`: Median absolute deviation

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 4, 5)
mad := dl.MAD()
```

### Rank

Assigns ranks to elements based on their values.

```go
func (dl *DataList) Rank() *DataList
```

**Returns:**

- `*DataList`: New DataList with rank values

**Example:**

```go
dl := insyra.NewDataList(3, 1, 4, 1, 5)
ranks := dl.Rank()
// Returns ranks based on sorted order
```

### Difference

Calculates differences between consecutive elements.

```go
func (dl *DataList) Difference() *DataList
```

**Returns:**

- `*DataList`: New DataList with consecutive differences

**Example:**

```go
dl := insyra.NewDataList(1, 3, 6, 10, 15)
diff := dl.Difference()
// Returns: [2, 3, 4, 5] (differences between consecutive elements)
```

### MovingAverage

Calculates moving average with specified window size.

```go
func (dl *DataList) MovingAverage(windowSize int) *DataList
```

**Parameters:**

- `windowSize`: Size of the moving window

**Returns:**

- `*DataList`: New DataList with moving averages

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
ma := dl.MovingAverage(3) // 3-period moving average
```

### WeightedMovingAverage

Calculates weighted moving average with specified window size and weights.

```go
func (dl *DataList) WeightedMovingAverage(windowSize int, weights any) *DataList
```

**Parameters:**

- `windowSize`: Size of the moving window
- `weights`: Weights for the moving average

**Returns:**

- `*DataList`: New DataList with weighted moving averages

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
weights := insyra.NewDataList(0.5, 0.3, 0.2)
wma := dl.WeightedMovingAverage(3, weights)
```

### ExponentialSmoothing

Calculates exponential smoothing with specified smoothing factor.

```go
func (dl *DataList) ExponentialSmoothing(alpha float64) *DataList
```

**Parameters:**

- `alpha`: Smoothing factor (0 < alpha < 1)

**Returns:**

- `*DataList`: New DataList with exponentially smoothed values

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
smoothed := dl.ExponentialSmoothing(0.3)
```

### DoubleExponentialSmoothing

Calculates double exponential smoothing with alpha and beta parameters.

```go
func (dl *DataList) DoubleExponentialSmoothing(alpha, beta float64) *DataList
```

**Parameters:**

- `alpha`: Level smoothing factor (0 < alpha < 1)
- `beta`: Trend smoothing factor (0 < beta < 1)

**Returns:**

- `*DataList`: New DataList with double exponentially smoothed values

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
smoothed := dl.DoubleExponentialSmoothing(0.3, 0.2)
```

### MovingStdev

Calculates moving standard deviation with specified window size.

```go
func (dl *DataList) MovingStdev(windowSize int) *DataList
```

**Parameters:**

- `windowSize`: Size of the moving window

**Returns:**

- `*DataList`: New DataList with moving standard deviations

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
mstdev := dl.MovingStdev(3)
```

### Summary

Displays comprehensive statistical summary to the console.

```go
func (dl *DataList) Summary()
```

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
dl.Summary()
// Outputs: count, mean, median, min, max, range, std dev, variance, quartiles, IQR
```

## Data Transformation

### Upper

Converts all string elements to uppercase.

```go
func (dl *DataList) Upper() *DataList
```

**Returns:**

- `*DataList`: Reference to the modified DataList

**Example:**

```go
dl := insyra.NewDataList("hello", "world", 123)
dl.Upper()
// String elements become: ["HELLO", "WORLD", 123]
```

### Lower

Converts all string elements to lowercase.

```go
func (dl *DataList) Lower() *DataList
```

**Returns:**

- `*DataList`: Reference to the modified DataList

**Example:**

```go
dl := insyra.NewDataList("HELLO", "WORLD", 123)
dl.Lower()
// String elements become: ["hello", "world", 123]
```

### Capitalize

Capitalizes the first letter of each string element.

```go
func (dl *DataList) Capitalize() *DataList
```

**Returns:**

- `*DataList`: Reference to the modified DataList

**Example:**

```go
dl := insyra.NewDataList("hello", "world", 123)
dl.Capitalize()
// String elements become: ["Hello", "World", 123]
```

## Interpolation Methods

### LinearInterpolation

Performs linear interpolation for a given x value.

```go
func (dl *DataList) LinearInterpolation(x float64) float64
```

**Parameters:**

- `x`: The x-value for interpolation

**Returns:**

- `float64`: Interpolated y-value

**Example:**

```go
dl := insyra.NewDataList(1, 4, 9, 16, 25) // y-values for x = 0, 1, 2, 3, 4
interpolated := dl.LinearInterpolation(2.5) // Interpolate at x=2.5
```

### QuadraticInterpolation

Performs quadratic interpolation for a given x value.

```go
func (dl *DataList) QuadraticInterpolation(x float64) float64
```

**Parameters:**

- `x`: The x-value for interpolation

**Returns:**

- `float64`: Interpolated y-value

**Example:**

```go
dl := insyra.NewDataList(1, 4, 9, 16, 25)
interpolated := dl.QuadraticInterpolation(2.5)
```

### LagrangeInterpolation

Performs Lagrange interpolation for a given x value.

```go
func (dl *DataList) LagrangeInterpolation(x float64) float64
```

**Parameters:**

- `x`: The x-value for interpolation

**Returns:**

- `float64`: Interpolated y-value

**Example:**

```go
dl := insyra.NewDataList(1, 4, 9, 16, 25)
interpolated := dl.LagrangeInterpolation(2.5)
```

### NearestNeighborInterpolation

Performs nearest neighbor interpolation for a given x value.

```go
func (dl *DataList) NearestNeighborInterpolation(x float64) float64
```

**Parameters:**

- `x`: The x-value for interpolation

**Returns:**

- `float64`: Interpolated y-value

**Example:**

```go
dl := insyra.NewDataList(1, 4, 9, 16, 25)
interpolated := dl.NearestNeighborInterpolation(2.3) // Returns nearest value
```

### NewtonInterpolation

Performs Newton interpolation for a given x value.

```go
func (dl *DataList) NewtonInterpolation(x float64) float64
```

**Parameters:**

- `x`: The x-value for interpolation

**Returns:**

- `float64`: Interpolated y-value

**Example:**

```go
dl := insyra.NewDataList(1, 4, 9, 16, 25)
interpolated := dl.NewtonInterpolation(2.5)
```

### HermiteInterpolation

Performs Hermite interpolation with derivatives for a given x value.

```go
func (dl *DataList) HermiteInterpolation(x float64, derivatives []float64) float64
```

**Parameters:**

- `x`: The x-value for interpolation
- `derivatives`: Derivative values at data points

**Returns:**

- `float64`: Interpolated y-value

**Example:**

```go
dl := insyra.NewDataList(1, 4, 9, 16, 25)
derivatives := []float64{2, 4, 6, 8, 10} // Derivative values
interpolated := dl.HermiteInterpolation(2.5, derivatives)
```

## Data Visualization

### Show

Displays DataList content in a clean, colored linear format.

```go
func (dl *DataList) Show()
```

**Example:**

```go
dl := insyra.NewDataList(1, "hello", 3.14, true)
dl.Show()
// Displays elements with color coding based on data types
```

### ShowRange

Displays DataList content within a specified range.

```go
func (dl *DataList) ShowRange(startEnd ...any)
```

**Parameters:**

- `startEnd`: Variable parameters for range specification
  - No parameters: shows all items
  - Single positive integer (n): shows first n items
  - Single negative integer (-n): shows last n items
  - Two parameters (start, end): shows items from start to end (exclusive)
  - Two parameters (start, nil): shows items from start to end

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
dl.ShowRange(3)      // Show first 3 items
dl.ShowRange(-3)     // Show last 3 items
dl.ShowRange(2, 5)   // Show items from index 2 to 4
dl.ShowRange(5, nil) // Show items from index 5 to end
```

### ShowTypes

Displays the data types of each element in the DataList.

```go
func (dl *DataList) ShowTypes()
```

**Example:**

```go
dl := insyra.NewDataList(1, "hello", 3.14, true, nil)
dl.ShowTypes()
// Displays: Index  Type
//          0      int
//          1      string
//          2      float64
//          3      bool
//          4      nil
```

### ShowTypesRange

Displays the data types of DataList elements within a specified range.

```go
func (dl *DataList) ShowTypesRange(startEnd ...any)
```

**Parameters:**

- `startEnd`: Variable parameters for range specification (same as ShowRange)
  - No parameters: shows all items
  - Single positive integer (n): shows first n items
  - Single negative integer (-n): shows last n items
  - Two parameters (start, end): shows items from start to end (exclusive)
  - Two parameters (start, nil): shows items from start to end

**Example:**

```go
dl := insyra.NewDataList(1, "hello", 3.14, true, nil, 42, "world")
dl.ShowTypesRange(3)      // Show types for first 3 items
dl.ShowTypesRange(-2)     // Show types for last 2 items
dl.ShowTypesRange(1, 4)   // Show types for items from index 1 to 3
dl.ShowTypesRange(2, nil) // Show types from index 2 to end
```

## Data Comparison

### IsEqualTo

Checks if the data content is equal to another DataList.

```go
func (dl *DataList) IsEqualTo(other *DataList) bool
```

**Parameters:**

- `other`: Another DataList to compare with

**Returns:**

- `bool`: True if data content is equal

**Example:**

```go
dl1 := insyra.NewDataList(1, 2, 3)
dl2 := insyra.NewDataList(1, 2, 3)
isEqual := dl1.IsEqualTo(dl2) // true
```

### IsTheSameAs

Checks if the DataList is identical to another (including metadata).

```go
func (dl *DataList) IsTheSameAs(other *DataList) bool
```

**Parameters:**

- `other`: Another DataList to compare with

**Returns:**

- `bool`: True if completely identical (data, name, timestamps)

**Example:**

```go
dl1 := insyra.NewDataList(1, 2, 3)
dl2 := dl1 // Same reference
isSame := dl1.IsTheSameAs(dl2) // true
```

## Data Conversion

### ParseNumbers

Converts all elements to numeric values where possible.

```go
func (dl *DataList) ParseNumbers() *DataList
```

**Returns:**

- `*DataList`: Reference to the modified DataList

**Example:**

```go
dl := insyra.NewDataList("1", "2.5", "hello", "3")
dl.ParseNumbers()
// Converts "1" -> 1, "2.5" -> 2.5, "3" -> 3, "hello" remains unchanged
```

### ParseStrings

Converts all elements to string values.

```go
func (dl *DataList) ParseStrings() *DataList
```

**Returns:**

- `*DataList`: Reference to the modified DataList

**Example:**

```go
dl := insyra.NewDataList(1, 2.5, true, "hello")
dl.ParseStrings()
// All elements become strings: ["1", "2.5", "true", "hello"]
```

### ToF64Slice

Converts DataList to a slice of float64 values.

```go
func (dl *DataList) ToF64Slice() []float64
```

**Returns:**

- `[]float64`: Slice containing numeric values (non-numeric values become 0)

**Example:**

```go
dl := insyra.NewDataList(1, 2.5, "hello", 4)
floatSlice := dl.ToF64Slice() // [1.0, 2.5, 0.0, 4.0]
```

### ToStringSlice

Converts DataList to a slice of string values.

```go
func (dl *DataList) ToStringSlice() []string
```

**Returns:**

- `[]string`: Slice containing all values as strings

**Example:**

```go
dl := insyra.NewDataList(1, 2.5, true, "hello")
stringSlice := dl.ToStringSlice() // ["1", "2.5", "true", "hello"]
```

### Clone

Creates a deep copy of the DataList.

```go
func (dl *DataList) Clone() *DataList
```

**Returns:**

- `*DataList`: New DataList with copied data

**Example:**

```go
dl1 := insyra.NewDataList(1, 2, 3)
dl2 := dl1.Clone()
// dl2 is an independent copy of dl1
```

## Metadata Management

### GetName

Retrieves the name assigned to the DataList.

```go
func (dl *DataList) GetName() string
```

**Returns:**

- `string`: Name of the DataList

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3)
dl.SetName("My_Data")
name := dl.GetName() // "My_Data"
```

### SetName

Assigns a name to the DataList. Use snake-style Pascal case (e.g., `Factor_Loadings`) to avoid spelling errors caused by spaces.

```go
func (dl *DataList) SetName(name string) *DataList
```

**Parameters:**

- `name`: Name to assign to the DataList (recommended: snake-style Pascal case)

**Returns:**

- `*DataList`: Reference to the DataList

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3)
dl.SetName("Factor_Loadings")
```

### GetCreationTimestamp

Returns the creation timestamp in Unix format.

```go
func (dl *DataList) GetCreationTimestamp() int64
```

**Returns:**

- `int64`: Unix timestamp when DataList was created

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3)
timestamp := dl.GetCreationTimestamp()
```

### GetLastModifiedTimestamp

Returns the last modification timestamp in Unix format.

```go
func (dl *DataList) GetLastModifiedTimestamp() int64
```

**Returns:**

- `int64`: Unix timestamp when DataList was last modified

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3)
dl.Append(4) // This updates the timestamp
timestamp := dl.GetLastModifiedTimestamp()
```

## Utility Methods

### Count

Returns the number of occurrences of a specified value.

```go
func (dl *DataList) Count(value any) int
```

**Parameters:**

- `value`: Value to count occurrences of

**Returns:**

- `int`: Number of occurrences

**Example:**

```go
dl := insyra.NewDataList(1, 2, 2, 3, 2, 4)
count := dl.Count(2) // 3
```

### Counter

Returns a map showing the count of each unique value.

```go
func (dl *DataList) Counter() map[any]int
```

**Returns:**

- `map[any]int`: Map of values to their occurrence counts

**Example:**

```go
dl := insyra.NewDataList(1, 2, 2, 3, 2, 4)
counter := dl.Counter()
// Returns: map[1:1 2:3 3:1 4:1]
```

### FindFirst

Finds the first occurrence of a value and returns its index.

```go
func (dl *DataList) FindFirst(value any) any
```

**Parameters:**

- `value`: Value to search for

**Returns:**

- `any`: Index of first occurrence, or nil if not found

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 2, 4)
index := dl.FindFirst(2) // 1 (first occurrence at index 1)
```

### FindLast

Finds the last occurrence of a value and returns its index.

```go
func (dl *DataList) FindLast(value any) any
```

**Parameters:**

- `value`: Value to search for

**Returns:**

- `any`: Index of last occurrence, or nil if not found

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 2, 4)
index := dl.FindLast(2) // 3 (last occurrence at index 3)
```

### FindAll

Finds all occurrences of a value and returns their indices.

```go
func (dl *DataList) FindAll(value any) []int
```

**Parameters:**

- `value`: Value to search for

**Returns:**

- `[]int`: Slice of indices where the value occurs

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 2, 4, 2)
indices := dl.FindAll(2) // [1, 3, 5]
```

### ReplaceFirst

Replaces the first occurrence of a value with a new value.

```go
func (dl *DataList) ReplaceFirst(oldValue, newValue any)
```

**Parameters:**

- `oldValue`: Value to search for
- `newValue`: Value to replace with

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 2, 4)
dl.ReplaceFirst(2, 99)
// dl now contains: [1, 99, 3, 2, 4]
```

### ReplaceLast

Replaces the last occurrence of a value with a new value.

```go
func (dl *DataList) ReplaceLast(oldValue, newValue any)
```

**Parameters:**

- `oldValue`: Value to search for
- `newValue`: Value to replace with

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 2, 4)
dl.ReplaceLast(2, 99)
// dl now contains: [1, 2, 3, 99, 4]
```

### ReplaceAll

Replaces all occurrences of a value with a new value.

```go
func (dl *DataList) ReplaceAll(oldValue, newValue any)
```

**Parameters:**

- `oldValue`: Value to search for
- `newValue`: Value to replace with

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 2, 4, 2)
dl.ReplaceAll(2, 99)
// dl now contains: [1, 99, 3, 99, 4, 99]
```

## Notes

### Thread Safety

DataList operations are thread-safe thanks to internal mutex synchronization. Multiple goroutines can safely access and modify DataList instances concurrently.

### Memory Management

DataList includes automatic memory reorganization features that optimize memory usage during operations. The system performs background memory cleanup to maintain efficient memory allocation.

### Data Type Handling

DataList is designed to handle mixed data types gracefully. Statistical operations automatically filter out non-numeric data, while string operations work only on string elements. This allows for flexible data manipulation without type conflicts.

### Performance Considerations

- Large DataLists benefit from the internal memory optimization system
- Statistical operations are optimized for numeric data processing
- Interpolation methods assume equally spaced data points for best results

### Error Handling

Most DataList operations handle errors gracefully by:

- Skipping invalid data types for type-specific operations
- Preserving original values when transformations fail
- Providing meaningful error messages for debugging

### Best Practices

1. **Naming**: Use descriptive names with `SetName()` in snake-style Pascal case (e.g., `Factor_Loadings`) to avoid spelling errors caused by spaces
2. **Type Consistency**: While DataList supports mixed types, keeping similar data types together improves performance
3. **Memory**: For large datasets, prefer operations that modify in-place over creating new DataLists
4. **Interpolation**: Ensure your data is suitable for the chosen interpolation method
