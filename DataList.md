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
- [Error Handling](#error-handling)
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
    atomicActor core.AtomicActor
}
```

**Field Descriptions:**

- `data`: Slice containing the actual data elements of any type
- `name`: Optional name for the DataList
- `creationTimestamp`: Unix timestamp when the DataList was created
- `lastModifiedTimestamp`: Unix timestamp when the DataList was last modified
- `atomicActor`: Internal actor used by `AtomicDo` to provide serialized execution without external locks

### Naming Conventions

- **List Names**: Use snake-style Pascal case (e.g., `Factor_Loadings`, `Communalities`) to avoid spelling errors caused by spaces.

## Creating DataList

### NewDataList

```go
func NewDataList(values ...any) *DataList
```

**Description:** Creates a new DataList instance with variadic parameters, automatically flattening nested slices but not arrays.

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

## Data Access

### Get

```go
func (dl *DataList) Get(index int) any
```

**Description:** Retrieves an element at a specific index with support for negative indexing.

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

```go
func (dl *DataList) Data() []any
```

**Description:** Returns the underlying data slice.

**Parameters:**

- None.

**Returns:**

- `[]any`: The underlying data slice

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3)
data := dl.Data() // []any{1, 2, 3}
```

### Len

```go
func (dl *DataList) Len() int
```

**Description:** Returns the number of elements in the DataList.

**Parameters:**

- None.

**Returns:**

- `int`: Number of elements in the DataList

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 4, 5)
fmt.Println(dl.Len()) // 5
```

## Data Manipulation

### Append

```go
func (dl *DataList) Append(values ...any) *DataList
```

**Description:** Adds new elements to the end of the DataList. Returns the DataList to support chaining calls.

**Parameters:**

- `values`: Variadic list of elements to append

**Returns:**

- `*DataList`: Reference to the modified DataList

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3)
dl.Append(4, 5, 6)
// dl now contains: [1, 2, 3, 4, 5, 6]

// Method chaining example
dl.Append(7, 8).Sort().Reverse()
```

### Concat

```go
func (dl *DataList) Concat(other IDataList) *DataList
```

**Description:** Creates a new DataList by appending another DataList to the current DataList.

**Parameters:**

- `other`: Another DataList to concatenate

**Returns:**

- `*DataList`: Return value.

**Example:**

```go
dl1 := insyra.NewDataList(1, 2, 3)
dl2 := insyra.NewDataList(4, 5, 6)
newDL := dl1.Concat(dl2)
// newDL contains: [1, 2, 3, 4, 5, 6]
// dl1 and dl2 remain unchanged
```

### AppendDataList

```go
func (dl *DataList) AppendDataList(other IDataList) *DataList
```

**Description:** Appends another DataList to the current DataList. Returns the DataList to support chaining calls.

**Parameters:**

- `other`: Another DataList to append

**Returns:**

- `*DataList`: Return value.

**Example:**

```go
dl1 := insyra.NewDataList(1, 2, 3)
dl2 := insyra.NewDataList(4, 5, 6)
dl1.AppendDataList(dl2)
// dl1 now contains: [1, 2, 3, 4, 5, 6]
```

### Update

```go
func (dl *DataList) Update(index int, value any) *DataList
```

**Description:** Updates an element at a specific index. Returns the DataList to support chaining calls.

**Parameters:**

- `index`: Zero-based index position
- `value`: New value to set at the index

**Returns:**

- `*DataList`: Return value.

**Example:**

```go
// single call
dl := insyra.NewDataList(1, 2, 3)
dl.Update(1, 99)
// dl now contains: [1, 99, 3]

// chaining example
dl.Update(1, 99).InsertAt(2, 100)
```

### InsertAt

```go
func (dl *DataList) InsertAt(index int, value any) *DataList
```

**Description:** Inserts a new element at a specific index, shifting existing elements to the right. Returns the DataList to support chaining calls.

**Parameters:**

- `index`: Zero-based index position for insertion
- `value`: Value to insert

**Returns:**

- `*DataList`: Return value.

**Example:**

```go
// single call
dl := insyra.NewDataList(1, 3, 4)
dl.InsertAt(1, 2)
// dl now contains: [1, 2, 3, 4]

// chaining example
dl.InsertAt(1, 2).Update(2, 99)
```

### Pop

```go
func (dl *DataList) Pop() any
```

**Description:** Removes and returns the last element from the DataList.

**Parameters:**

- None.

**Returns:**

- `any`: The removed last element

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 4, 5)
lastElement := dl.Pop() // returns 5
// dl now contains: [1, 2, 3, 4]
```

### Drop

```go
func (dl *DataList) Drop(index int) *DataList
```

**Description:** Removes an element at a specific index.

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

```go
func (dl *DataList) DropAll(values ...any) *DataList
```

**Description:** Removes all occurrences of specified values from the DataList. Supports `math.NaN()`.

**Parameters:**

- `values`: Variadic list of values to remove

**Returns:**

- `*DataList`: Reference to the modified DataList

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 2, 4, 2, 5, math.NaN())
dl.DropAll(2, 4, math.NaN())
// dl now contains: [1, 3, 5]
```

### DropIfContains

```go
func (dl *DataList) DropIfContains(substring string) *DataList
```

**Description:** Removes all string elements that contain a specified substring. Non-string elements are kept.

**Parameters:**

- `substring`: Substring to search for in string elements

**Returns:**

- `*DataList`: Reference to the modified DataList

**Example:**

```go
dl := insyra.NewDataList("apple", "banana", "grape", "orange", 123)
dl.DropIfContains("an")
// dl now contains: ["apple", "grape", 123]
```

### Clear

```go
func (dl *DataList) Clear() *DataList
```

**Description:** Removes all elements from the DataList.

**Parameters:**

- None.

**Returns:**

- `*DataList`: Reference to the cleared DataList

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 4, 5)
dl.Clear()
// dl now contains: []
```

### Sort

```go
func (dl *DataList) Sort(ascending ...bool) *DataList
```

**Description:** Sorts the elements in the DataList using mixed sorting logic for different data types.

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

```go
func (dl *DataList) Reverse() *DataList
```

**Description:** Reverses the order of elements in the DataList.

**Parameters:**

- None.

**Returns:**

- `*DataList`: Reference to the reversed DataList

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 4, 5)
dl.Reverse()
// dl now contains: [5, 4, 3, 2, 1]
```

### Map

```go
func (dl *DataList) Map(mapFunc func(int, any) any) *DataList
```

**Description:** Applies a transformation function to all elements with index-aware processing.

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

```go
func (dl *DataList) Filter(predicate func(any) bool) *DataList
```

**Description:** Filters elements based on a predicate function.

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

```go
func (dl *DataList) ClearStrings() *DataList
```

**Description:** Removes all string elements from the DataList.

**Parameters:**

- None.

**Returns:**

- `*DataList`: Reference to the modified DataList

**Example:**

```go
dl := insyra.NewDataList(1, "hello", 2, "world", 3)
dl.ClearStrings()
// dl now contains: [1, 2, 3]
```

### ClearNumbers

```go
func (dl *DataList) ClearNumbers() *DataList
```

**Description:** Removes all numeric elements from the DataList.

**Parameters:**

- None.

**Returns:**

- `*DataList`: Reference to the modified DataList

**Example:**

```go
dl := insyra.NewDataList(1, "hello", 2, "world", 3)
dl.ClearNumbers()
// dl now contains: ["hello", "world"]
```

### ClearNaNs

```go
func (dl *DataList) ClearNaNs() *DataList
```

**Description:** Removes all NaN (Not a Number) elements from the DataList.

**Parameters:**

- None.

**Returns:**

- `*DataList`: Reference to the modified DataList

**Example:**

```go
dl := insyra.NewDataList(1.0, math.NaN(), 2.0, math.NaN(), 3.0)
dl.ClearNaNs()
// dl now contains: [1.0, 2.0, 3.0]
```

### ClearNils

```go
func (dl *DataList) ClearNils() *DataList
```

**Description:** Removes all nil (null) elements from the DataList.

**Parameters:**

- None.

**Returns:**

- `*DataList`: Reference to the modified DataList

**Example:**

```go
dl := insyra.NewDataList(1, nil, 2, nil, 3)
dl.ClearNils()
// dl now contains: [1, 2, 3]
```

### ClearNilsAndNaNs

```go
func (dl *DataList) ClearNilsAndNaNs() *DataList
```

**Description:** Removes all nil and NaN (Not a Number) elements from the DataList.

**Parameters:**

- None.

**Returns:**

- `*DataList`: Reference to the modified DataList

**Example:**

```go
dl := insyra.NewDataList(1.0, nil, math.NaN(), 2.0, nil, 3.0)
dl.ClearNilsAndNaNs()
// dl now contains: [1.0, 2.0, 3.0]
```

### ClearOutliers

```go
func (dl *DataList) ClearOutliers(stdDev float64) *DataList
```

**Description:** Removes values outside a specified number of standard deviations from the mean.

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

```go
func (dl *DataList) Normalize() *DataList
```

**Description:** Normalizes DataList elements to a specified range (default: 0 to 1).

**Parameters:**

- None.

**Returns:**

- `*DataList`: Reference to the normalized DataList

**Example:**

```go
dl := insyra.NewDataList(10, 20, 30, 40, 50)
dl.Normalize()
// Values are normalized to range [0, 1]
```

### Standardize

```go
func (dl *DataList) Standardize() *DataList
```

**Description:** Standardizes DataList elements using z-score normalization (mean=0, std=1).

**Parameters:**

- None.

**Returns:**

- `*DataList`: Reference to the standardized DataList

**Example:**

```go
dl := insyra.NewDataList(10, 20, 30, 40, 50)
dl.Standardize()
// Values are standardized to have mean=0 and std=1
```

### FillNaNWithMean

```go
func (dl *DataList) FillNaNWithMean() *DataList
```

**Description:** Replaces all NaN values with the mean value of numeric elements.

**Parameters:**

- None.

**Returns:**

- `*DataList`: Reference to the modified DataList

**Example:**

```go
dl := insyra.NewDataList(1.0, math.NaN(), 3.0, math.NaN(), 5.0)
dl.FillNaNWithMean()
// NaN values are replaced with mean (3.0)
```

### ReplaceOutliers

```go
func (dl *DataList) ReplaceOutliers(stdDevs float64, replacement float64) *DataList
```

**Description:** Replaces outliers beyond specified standard deviations with a replacement value. Values whose distance from the mean exceeds the threshold (stdDevs × standard deviation) will be replaced.

**Parameters:**

- `stdDevs`: Number of standard deviations to use as threshold (e.g., 2.0 means values beyond ±2σ from the mean will be replaced)
- `replacement`: Value to replace outliers with

**Returns:**

- `*DataList`: Reference to the modified DataList

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 4, 5, 100)
dl.ReplaceOutliers(2.0, 6.0) // Replace outliers with 6.0
```

### ReplaceNaNsWith

```go
func (dl *DataList) ReplaceNaNsWith(value any) *DataList
```

**Description:** Replaces all NaN (Not a Number) values in the DataList with the specified value.

**Parameters:**

- `value`: The value to replace NaN values with

**Returns:**

- `*DataList`: Reference to the modified DataList

**Example:**

```go
dl := insyra.NewDataList(1.0, math.NaN(), 3.0, math.NaN(), 5.0)
dl.ReplaceNaNsWith(0.0)
// dl now contains: [1.0, 0.0, 3.0, 0.0, 5.0]
```

### ReplaceNilsWith

```go
func (dl *DataList) ReplaceNilsWith(value any) *DataList
```

**Description:** Replaces all nil (null) values in the DataList with the specified value.

**Parameters:**

- `value`: The value to replace nil values with

**Returns:**

- `*DataList`: Reference to the modified DataList

**Example:**

```go
dl := insyra.NewDataList(1, nil, 3, nil, 5)
dl.ReplaceNilsWith(0)
// dl now contains: [1, 0, 3, 0, 5]
```

### ReplaceNaNsAndNilsWith

```go
func (dl *DataList) ReplaceNaNsAndNilsWith(value any) *DataList
```

**Description:** Replaces all NaN and nil values in the DataList with the specified value.

**Parameters:**

- `value`: The value to replace NaN and nil values with

**Returns:**

- `*DataList`: Reference to the modified DataList

**Example:**

```go
dl := insyra.NewDataList(1.0, nil, math.NaN(), 2.0, nil, 3.0)
dl.ReplaceNaNsAndNilsWith(0.0)
// dl now contains: [1.0, 0.0, 0.0, 2.0, 0.0, 3.0]
```

## Statistical Analysis

### Sum

```go
func (dl *DataList) Sum() float64
```

**Description:** Calculates the sum of numeric elements in the DataList.

**Parameters:**

- None.

**Returns:**

- `float64`: Sum of all numeric elements

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 4, 5)
sum := dl.Sum() // 15.0
```

### Max

```go
func (dl *DataList) Max() float64
```

**Description:** Returns the maximum value among numeric elements.

**Parameters:**

- None.

**Returns:**

- `float64`: Maximum numeric value

**Example:**

```go
dl := insyra.NewDataList(1, 5, 3, 9, 2)
max := dl.Max() // 9.0
```

### Min

```go
func (dl *DataList) Min() float64
```

**Description:** Returns the minimum value among numeric elements.

**Parameters:**

- None.

**Returns:**

- `float64`: Minimum numeric value

**Example:**

```go
dl := insyra.NewDataList(1, 5, 3, 9, 2)
min := dl.Min() // 1.0
```

### Mean

```go
func (dl *DataList) Mean() float64
```

**Description:** Calculates the arithmetic mean of numeric elements.

**Parameters:**

- None.

**Returns:**

- `float64`: Arithmetic mean

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 4, 5)
mean := dl.Mean() // 3.0
```

### WeightedMean

```go
func (dl *DataList) WeightedMean(weights any) float64
```

**Description:** Calculates the weighted mean using provided weights.

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

```go
func (dl *DataList) GMean() float64
```

**Description:** Calculates the geometric mean of numeric elements.

**Parameters:**

- None.

**Returns:**

- `float64`: Geometric mean

**Example:**

```go
dl := insyra.NewDataList(1, 2, 4, 8)
gmean := dl.GMean() // 2.83 (approximately)
```

### Median

```go
func (dl *DataList) Median() float64
```

**Description:** Returns the median value after sorting elements.

**Parameters:**

- None.

**Returns:**

- `float64`: Median value

**Example:**

```go
dl := insyra.NewDataList(1, 3, 2, 5, 4)
median := dl.Median() // 3.0
```

### Mode

```go
func (dl *DataList) Mode() []float64
```

**Description:** Returns the most frequent value(s) in the DataList.

**Parameters:**

- None.

**Returns:**

- `[]float64`: Slice of mode values

**Example:**

```go
dl := insyra.NewDataList(1, 2, 2, 3, 3, 3)
modes := dl.Mode() // [3.0]
```

### Stdev

```go
func (dl *DataList) Stdev() float64
```

**Description:** Calculates the sample standard deviation.

**Parameters:**

- None.

**Returns:**

- `float64`: Sample standard deviation

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 4, 5)
stdev := dl.Stdev()
```

### StdevP

```go
func (dl *DataList) StdevP() float64
```

**Description:** Calculates the population standard deviation.

**Parameters:**

- None.

**Returns:**

- `float64`: Population standard deviation

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 4, 5)
stdevp := dl.StdevP()
```

### Var

```go
func (dl *DataList) Var() float64
```

**Description:** Calculates the sample variance.

**Parameters:**

- None.

**Returns:**

- `float64`: Sample variance

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 4, 5)
variance := dl.Var()
```

### VarP

```go
func (dl *DataList) VarP() float64
```

**Description:** Calculates the population variance.

**Parameters:**

- None.

**Returns:**

- `float64`: Population variance

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 4, 5)
variancep := dl.VarP()
```

### Range

```go
func (dl *DataList) Range() float64
```

**Description:** Returns the difference between maximum and minimum values.

**Parameters:**

- None.

**Returns:**

- `float64`: Range (max - min)

**Example:**

```go
dl := insyra.NewDataList(1, 5, 3, 9, 2)
range := dl.Range() // 8.0 (9 - 1)
```

### Quartile

```go
func (dl *DataList) Quartile(q int) float64
```

**Description:** Calculates quartile values (Q1, Q2, Q3).

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

```go
func (dl *DataList) IQR() float64
```

**Description:** Calculates the interquartile range (Q3 - Q1).

**Parameters:**

- None.

**Returns:**

- `float64`: Interquartile range

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
iqr := dl.IQR()
```

### Percentile

```go
func (dl *DataList) Percentile(percentile float64) float64
```

**Description:** Calculates the value below which a given percentage of observations fall.

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

```go
func (dl *DataList) MAD() float64
```

**Description:** Calculates the median absolute deviation.

**Parameters:**

- None.

**Returns:**

- `float64`: Median absolute deviation

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 4, 5)
mad := dl.MAD()
```

### Rank

```go
func (dl *DataList) Rank() *DataList
```

**Description:** Assigns ranks to elements based on their values.

**Parameters:**

- None.

**Returns:**

- `*DataList`: New DataList with rank values

**Example:**

```go
dl := insyra.NewDataList(3, 1, 4, 1, 5)
ranks := dl.Rank()
// Returns ranks based on sorted order
```

### Difference

```go
func (dl *DataList) Difference() *DataList
```

**Description:** Calculates differences between consecutive elements.

**Parameters:**

- None.

**Returns:**

- `*DataList`: New DataList with consecutive differences

**Example:**

```go
dl := insyra.NewDataList(1, 3, 6, 10, 15)
diff := dl.Difference()
// Returns: [2, 3, 4, 5] (differences between consecutive elements)
```

### MovingAverage

```go
func (dl *DataList) MovingAverage(windowSize int) *DataList
```

**Description:** Calculates moving average with specified window size.

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

```go
func (dl *DataList) WeightedMovingAverage(windowSize int, weights any) *DataList
```

**Description:** Calculates weighted moving average with specified window size and weights.

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

```go
func (dl *DataList) ExponentialSmoothing(alpha float64) *DataList
```

**Description:** Calculates exponential smoothing with specified smoothing factor.

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

```go
func (dl *DataList) DoubleExponentialSmoothing(alpha, beta float64) *DataList
```

**Description:** Calculates double exponential smoothing with alpha and beta parameters.

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

```go
func (dl *DataList) MovingStdev(windowSize int) *DataList
```

**Description:** Calculates moving standard deviation with specified window size.

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

```go
func (dl *DataList) Summary()
```

**Description:** Displays comprehensive statistical summary to the console.

**Parameters:**

- None.

**Returns:**

- None.

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
dl.Summary()
// Outputs: count, mean, median, min, max, range, std dev, variance, quartiles, IQR
```

## Data Transformation

### Upper

```go
func (dl *DataList) Upper() *DataList
```

**Description:** Converts all string elements to uppercase.

**Parameters:**

- None.

**Returns:**

- `*DataList`: Reference to the modified DataList

**Example:**

```go
dl := insyra.NewDataList("hello", "world", 123)
dl.Upper()
// String elements become: ["HELLO", "WORLD", 123]
```

### Lower

```go
func (dl *DataList) Lower() *DataList
```

**Description:** Converts all string elements to lowercase.

**Parameters:**

- None.

**Returns:**

- `*DataList`: Reference to the modified DataList

**Example:**

```go
dl := insyra.NewDataList("HELLO", "WORLD", 123)
dl.Lower()
// String elements become: ["hello", "world", 123]
```

### Capitalize

```go
func (dl *DataList) Capitalize() *DataList
```

**Description:** Capitalizes the first letter of each string element.

**Parameters:**

- None.

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

```go
func (dl *DataList) LinearInterpolation(x float64) float64
```

**Description:** Performs linear interpolation for a given x value.

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

```go
func (dl *DataList) QuadraticInterpolation(x float64) float64
```

**Description:** Performs quadratic interpolation for a given x value.

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

```go
func (dl *DataList) LagrangeInterpolation(x float64) float64
```

**Description:** Performs Lagrange interpolation for a given x value.

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

```go
func (dl *DataList) NearestNeighborInterpolation(x float64) float64
```

**Description:** Performs nearest neighbor interpolation for a given x value.

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

```go
func (dl *DataList) NewtonInterpolation(x float64) float64
```

**Description:** Performs Newton interpolation for a given x value.

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

```go
func (dl *DataList) HermiteInterpolation(x float64, derivatives []float64) float64
```

**Description:** Performs Hermite interpolation with derivatives for a given x value.

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

```go
func (dl *DataList) Show()
```

**Description:** Displays DataList content in a clean, colored linear format.

**Parameters:**

- None.

**Returns:**

- None.

**Example:**

```go
dl := insyra.NewDataList(1, "hello", 3.14, true)
dl.Show()
// Displays elements with color coding based on data types
```

### ShowRange

```go
func (dl *DataList) ShowRange(startEnd ...any)
```

**Description:** Displays DataList content within a specified range.

**Parameters:**

- `startEnd`: Variable parameters for range specification
  - No parameters: shows all items
  - Single positive integer (n): shows first n items
  - Single negative integer (-n): shows last n items
  - Two parameters (start, end): shows items from start to end (exclusive)
  - Two parameters (start, nil): shows items from start to end

**Returns:**

- None.

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
dl.ShowRange(3)      // Show first 3 items
dl.ShowRange(-3)     // Show last 3 items
dl.ShowRange(2, 5)   // Show items from index 2 to 4
dl.ShowRange(5, nil) // Show items from index 5 to end
```

### ShowTypes

```go
func (dl *DataList) ShowTypes()
```

**Description:** Displays the data types of each element in the DataList.

**Parameters:**

- None.

**Returns:**

- None.

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

```go
func (dl *DataList) ShowTypesRange(startEnd ...any)
```

**Description:** Displays the data types of DataList elements within a specified range.

**Parameters:**

- `startEnd`: Variable parameters for range specification (same as ShowRange)
  - No parameters: shows all items
  - Single positive integer (n): shows first n items
  - Single negative integer (-n): shows last n items
  - Two parameters (start, end): shows items from start to end (exclusive)
  - Two parameters (start, nil): shows items from start to end

**Returns:**

- None.

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

```go
func (dl *DataList) IsEqualTo(other *DataList) bool
```

**Description:** Checks if the data content is equal to another DataList.

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

```go
func (dl *DataList) IsTheSameAs(other *DataList) bool
```

**Description:** Checks if the DataList is identical to another (including metadata).

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

```go
func (dl *DataList) ParseNumbers() *DataList
```

**Description:** Converts all elements to numeric values where possible.

**Parameters:**

- None.

**Returns:**

- `*DataList`: Reference to the modified DataList

**Example:**

```go
dl := insyra.NewDataList("1", "2.5", "hello", "3")
dl.ParseNumbers()
// Converts "1" -> 1, "2.5" -> 2.5, "3" -> 3, "hello" remains unchanged
```

### ParseStrings

```go
func (dl *DataList) ParseStrings() *DataList
```

**Description:** Converts all elements to string values.

**Parameters:**

- None.

**Returns:**

- `*DataList`: Reference to the modified DataList

**Example:**

```go
dl := insyra.NewDataList(1, 2.5, true, "hello")
dl.ParseStrings()
// All elements become strings: ["1", "2.5", "true", "hello"]
```

### ToF64Slice

```go
func (dl *DataList) ToF64Slice() []float64
```

**Description:** Converts DataList to a slice of float64 values.

**Parameters:**

- None.

**Returns:**

- `[]float64`: Slice containing numeric values (non-numeric values become 0)

**Example:**

```go
dl := insyra.NewDataList(1, 2.5, "hello", 4)
floatSlice := dl.ToF64Slice() // [1.0, 2.5, 0.0, 4.0]
```

### ToStringSlice

```go
func (dl *DataList) ToStringSlice() []string
```

**Description:** Converts DataList to a slice of string values.

**Parameters:**

- None.

**Returns:**

- `[]string`: Slice containing all values as strings

**Example:**

```go
dl := insyra.NewDataList(1, 2.5, true, "hello")
stringSlice := dl.ToStringSlice() // ["1", "2.5", "true", "hello"]
```

### Clone

```go
func (dl *DataList) Clone() *DataList
```

**Description:** Creates a deep copy of the DataList.

**Parameters:**

- None.

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

```go
func (dl *DataList) GetName() string
```

**Description:** Retrieves the name assigned to the DataList.

**Parameters:**

- None.

**Returns:**

- `string`: Name of the DataList

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3)
dl.SetName("My_Data")
name := dl.GetName() // "My_Data"
```

### SetName

```go
func (dl *DataList) SetName(name string) *DataList
```

**Description:** Assigns a name to the DataList. Use snake-style Pascal case (e.g., `Factor_Loadings`) to avoid spelling errors caused by spaces.

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

```go
func (dl *DataList) GetCreationTimestamp() int64
```

**Description:** Returns the creation timestamp in Unix format.

**Parameters:**

- None.

**Returns:**

- `int64`: Unix timestamp when DataList was created

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3)
timestamp := dl.GetCreationTimestamp()
```

### GetLastModifiedTimestamp

```go
func (dl *DataList) GetLastModifiedTimestamp() int64
```

**Description:** Returns the last modification timestamp in Unix format.

**Parameters:**

- None.

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

```go
func (dl *DataList) Count(value any) int
```

**Description:** Returns the number of occurrences of a specified value. Supports `math.NaN()`.

**Parameters:**

- `value`: Value to count occurrences of

**Returns:**

- `int`: Number of occurrences

**Example:**

```go
dl := insyra.NewDataList(1, 2, 2, 3, 2, 4, math.NaN())
count := dl.Count(2) // 3
countNaN := dl.Count(math.NaN()) // 1
```

### Counter

```go
func (dl *DataList) Counter() map[any]int
```

**Description:** Returns a map showing the count of each unique value.

**Parameters:**

- None.

**Returns:**

- `map[any]int`: Map of values to their occurrence counts

**Example:**

```go
dl := insyra.NewDataList(1, 2, 2, 3, 2, 4)
counter := dl.Counter()
// Returns: map[1:1 2:3 3:1 4:1]
```

### FindFirst

```go
func (dl *DataList) FindFirst(value any) any
```

**Description:** Finds the first occurrence of a value and returns its index. Supports `math.NaN()`.

**Parameters:**

- `value`: Value to search for

**Returns:**

- `any`: Index of first occurrence, or nil if not found

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 2, 4, math.NaN())
index := dl.FindFirst(2) // 1
indexNaN := dl.FindFirst(math.NaN()) // 5
```

### FindLast

```go
func (dl *DataList) FindLast(value any) any
```

**Description:** Finds the last occurrence of a value and returns its index. Supports `math.NaN()`.

**Parameters:**

- `value`: Value to search for

**Returns:**

- `any`: Index of last occurrence, or nil if not found

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 2, 4, math.NaN())
index := dl.FindLast(2) // 3
indexNaN := dl.FindLast(math.NaN()) // 5
```

### FindAll

```go
func (dl *DataList) FindAll(value any) []int
```

**Description:** Finds all occurrences of a value and returns their indices. Supports `math.NaN()`.

**Parameters:**

- `value`: Value to search for

**Returns:**

- `[]int`: Slice of indices where the value occurs

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 2, 4, 2, math.NaN())
indices := dl.FindAll(2) // [1, 3, 5]
indicesNaN := dl.FindAll(math.NaN()) // [6]
```

### ReplaceFirst

```go
func (dl *DataList) ReplaceFirst(oldValue, newValue any) *DataList
```

**Description:** Replaces the first occurrence of a value with a new value.

**Parameters:**

- `oldValue`: Value to search for
- `newValue`: Value to replace with

**Returns:**

- `*DataList`: The DataList itself (for method chaining)

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 2, 4)
dl.ReplaceFirst(2, 99)
// dl now contains: [1, 99, 3, 2, 4]
```

### ReplaceLast

```go
func (dl *DataList) ReplaceLast(oldValue, newValue any) *DataList
```

**Description:** Replaces the last occurrence of a value with a new value.

**Parameters:**

- `oldValue`: Value to search for
- `newValue`: Value to replace with

**Returns:**

- `*DataList`: The DataList itself (for method chaining)

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 2, 4)
dl.ReplaceLast(2, 99)
// dl now contains: [1, 2, 3, 99, 4]
```

### ReplaceAll

```go
func (dl *DataList) ReplaceAll(oldValue, newValue any) *DataList
```

**Description:** Replaces all occurrences of a value with a new value.

**Parameters:**

- `oldValue`: Value to search for
- `newValue`: Value to replace with

**Returns:**

- `*DataList`: The DataList itself (for method chaining)

**Example:**

```go
dl := insyra.NewDataList(1, 2, 3, 2, 4, 2)
dl.ReplaceAll(2, 99)
// dl now contains: [1, 99, 3, 99, 4, 99]
```

## Error Handling

Insyra provides both a global error buffer and instance-level error tracking for `DataList`. For fluent/chained operations, use the instance-level `Err()` method to check for errors after a chain and `ClearErr()` to clear them before continuing.

### Instance-Level Error Checking

After performing chained operations, you can check if any errors occurred using the `Err()` method:

```go
// Perform chained operations
dl.Append(1,2,3).Sort().Reverse()

// Check for errors after the chain
if err := dl.Err(); err != nil {
    fmt.Printf("Error occurred: %s\n", err.Message)
    // Handle the error
}

// Clear the error for future operations and continue chaining
dl.ClearErr()
```

#### Available Methods

| Method                 | Description                                                                                     |
| ---------------------- | ----------------------------------------------------------------------------------------------- |
| `Err() *ErrorInfo`     | Returns the last error that occurred during a chained operation, or `nil` if no error occurred. |
| `ClearErr() *DataList` | Clears the last error and returns the DataList for continued chaining.                          |

> **Note:** `setError` is an internal helper used by methods to record the last error on the instance. Most chainable methods will call it when an operation fails.

## AtomicDo

```go
func (dl *DataList) AtomicDo(f func(*DataList))
```

**Description:** `AtomicDo` provides safe, serialized access to a DataList using an internal actor goroutine. It ensures all mutations and reads inside the function run in order and without races, even across multiple goroutines.

**Parameters:**

- `f`: Input value for `f`. Type: `func(*DataList)`.

**Returns:**

- None.

**Behavior:**

- Single-threaded execution: Functions passed to `AtomicDo` are processed one at a time.
- Reentrant: If called from within `AtomicDo`, the function runs immediately (no deadlock).
- Cross-list nesting: Calling `anotherDl.AtomicDo` inside `dl.AtomicDo` is supported.

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

## Notes

### Thread Safety

DataList operations are serialized through `AtomicDo` when thread safety is enabled (default). You can disable this with `Config.Dangerously_TurnOffThreadSafety()` if you are sure there is no concurrent access.

### Memory Management

DataList relies on Go's garbage collector. Large lists may increase memory pressure, so prefer in-place operations when possible.

### Data Type Handling

DataList allows mixed data types. Numeric statistics use `ToFloat64Safe` and skip values that cannot be converted; some methods may treat unsupported values as `0`.

### Performance Considerations

- Keep data types consistent when running numeric methods
- Do heavy computation outside `AtomicDo` and only mutate inside
- Interpolation methods assume evenly spaced data points for best results

### Error Handling

Refer to the [Error Handling](#error-handling) section above for full details. In short, `DataList` supports instance-level error tracking via `Err()` and `ClearErr()`; most operations also handle errors gracefully by:

- Skipping invalid data types for type-specific operations
- Preserving original values when transformations fail
- Providing meaningful error messages for debugging

### Best Practices

1. **Naming**: Use descriptive names with `SetName()` in snake-style Pascal case (e.g., `Factor_Loadings`) to avoid spelling errors caused by spaces
2. **Type Consistency**: While DataList supports mixed types, keeping similar data types together improves performance
3. **Memory**: For large datasets, prefer operations that modify in-place over creating new DataLists
4. **Interpolation**: Ensure your data is suitable for the chosen interpolation method
