# DataList

This document describes the `DataList` type and its functionalities within the `insyra` package.

**DataList**

The `DataList` type provides a dynamic and generic way to store and manage a collection of elements. It supports various data types, including strings, numbers, booleans, and even time data.

**Key Features:**

* **Generic:** Accepts elements of any data type.
* **Dynamic:** Grows and shrinks as elements are added or removed.
* **Functionality:** Provides methods for manipulating and analyzing the data stored in the list.
* **Timestamps:** Tracks creation and modification timestamps for each DataList instance.
* **Optional Names:** Allows assigning names to DataLists for better organization.

**Data Handling:**

* **NewDataList:** Creates a new DataList instance. It accepts a variadic list of elements and flattens them before storing them internally.
* **Append:** Adds new elements to the end of the DataList.
* **Get:** Retrieves the element at a specific index. Supports negative indexing for accessing elements from the end.
* **Update:** Updates the element at a specific index with a new value.
* **InsertAt:** Inserts a new element at a specific index, shifting the existing elements to the right.
* **FindFirst:** Finds the first occurrence of a specified value in the DataList and returns its index.
* **FindLast:** Finds the last occurrence of a specified value in the DataList and returns its index.
* **FindAll:** Finds all occurrences of a specified value in the DataList and returns their indices.
* **Filter**: Filters the DataList based on a provided function that returns a boolean value for each element.
* **ReplaceFirst:** Replaces the first occurrence of a specified value with a new value.
* **ReplaceLast:** Replaces the last occurrence of a specified value with a new value.
* **ReplaceAll:** Replaces all occurrences of a specified value with a new value.
* **ReplaceOutliers:** Replaces values from the DataList that are outside the specified number of standard deviations with a specified value.
* **Pop:** Removes and returns the last element from the DataList.
* **Drop:** Removes the element at a specific index.
* **DropAll:** Removes all occurrences of specified values from the DataList.
* **DropIfContains:** Removes all elements that contain a specified substring.
* **Clear:** Removes all elements from the DataList.
* **Len:** Returns the number of elements currently stored in the DataList.

**Data Manipulation:**

* **Sort:** Sorts the elements in the DataList using a mixed sorting logic that handles strings, numbers (various integer and float types), and time data types. If sorting fails, the original order is restored.
* **Reverse:** Reverses the order of elements in the DataList.
* **Upper:** Converts all string elements in the DataList to uppercase.
* **Lower:** Converts all string elements in the DataList to lowercase.
* **Capitalize:** Capitalizes the first letter of each string element in the DataList.

**Data Filtering:**

* **ClearStrings:** Removes all string elements from the DataList.
* **ClearNumbers:** Removes all numeric elements (int, float, etc.) from the DataList.
* **ClearNaNs:** Removes all NaN (Not a Number) elements from the DataList.

**Data Preprocessing:**
* **Normalize:** Normalizes the DataList elements to a specified range (default: 0 to 1).
* **Standardize:** Standardizes the DataList elements by subtracting the mean and dividing by the standard deviation.
* **FillNaNWithMean:** Replaces all NaN (Not a Number) elements with the mean value of the DataList.
* **ClearOutliers:** Removes values from the DataList that are outside the specified number of standard deviations. This method modifies the original DataList and returns it.

**Data Analysis:**

* **Rank:** Assigns a rank to each element in the DataList based on their values. Returns a new DataList with the ranks.
* **Max:** Returns the maximum value in the DataList. Handles different data types by converting them to a common base (float64) for comparison.
* **Min:** Returns the minimum value in the DataList. Similar logic to Max is applied for data type handling.
* **Mean:** Calculates the arithmetic mean (average) of the DataList elements. Excludes non-numeric data types.
* **WeightedMean:** Calculates the weighted mean of the DataList elements based on the provided weights.
* **GMean:** Calculates the geometric mean of the DataList elements. Excludes non-numeric data types.
* **Median:** Returns the median value of the DataList after sorting the elements.
* **Mode:** Returns the most frequent value (mode) in the DataList.
* **Stdev:** Calculates the sample standard deviation of the DataList elements. Excludes non-numeric data types.
* **StdevP:** Calculates the population standard deviation of the DataList elements. Excludes non-numeric data types.
* **Var:** Calculates the sample variance of the DataList elements. Excludes non-numeric data types.
* **VarP:** Calculates the population variance of the DataList elements. Excludes non-numeric data types.
* **Range:** Returns the difference between the maximum and minimum values in the DataList.
* **Quartile:** Calculates the quartile value (Q1, Q2, or Q3) based on the provided input.
* **IQR:** Calculates the interquartile range (IQR) of the DataList, which represents the range between the first and third quartiles.
* **Percentile:** Percentile: Calculates the percentile value based on the provided input, which represents the value below which a given percentage of observations fall. For example, entering 25 (the input scale is 0 to 100) would return the value at the 25th percentile, also known as the first quartile (Q1).
* **MAD:** Calculates the median absolute deviation (MAD) of the DataList elements.
* **Difference:** Calculates the difference between consecutive elements in the DataList. Returns a new DataList with the differences.
* **MovingAverage:** Calculates the moving average of the DataList elements using a specified window size. Returns a new DataList with the moving average values.
* **WeightedMovingAverage:** Calculates the weighted moving average of the DataList elements using a specified window size and weights. Returns a new DataList with the weighted moving average values.
* **ExponentialSmoothing:** Calculates the exponential smoothing of the DataList elements using a specified smoothing factor (alpha). Returns a new DataList with the smoothed values.
* **DoubleExponentialSmoothing:** Calculates the double exponential smoothing of the DataList elements using specified smoothing factors (alpha and beta). Returns a new DataList with the smoothed values.
* **MovingStdev:** Calculates the moving standard deviation of the DataList elements using a specified window size. Returns a new DataList with the moving standard deviation values.

**Data Conversion:**

* **ParseNumbers:** Converts all elements in the DataList to numeric values (float) if possible.

* **ParseStrings:** Converts all elements in the DataList to string values.

* **ToF64Slice:** Converts the DataList to a slice of float64 values. Useful for operations requiring numerical data.

* **ToStringSlice:** Converts the DataList to a slice of string values.

**Timestamps:**

* **GetCreationTimestamp:** Returns the creation timestamp of the DataList in Unix timestamp format (seconds since epoch).
* **GetLastModifiedTimestamp:** Returns the last modification timestamp of the DataList in Unix timestamp format.
* **updateTimestamp:** Internally updates the `lastModifiedTimestamp` whenever the DataList is modified.

**Names (Optional):**

* **GetName:** Retrieves the assigned name for the DataList.
* **SetName:** Sets a name for the DataList.

**Error Handling:**

* Several methods (e.g., Max, Min, Sort) handle data type mismatches and potential errors during calculations. In such cases, informative messages are printed, and the operation might return `nil` or restore the original state.

**Code Structure:**

The provided code demonstrates the implementation of the `DataList` type and its associated functionalities. It includes method definitions for all the features mentioned above.

**Overall, the `DataList` type provides a versatile and user-friendly data structure  for various use cases within the `insyra` package.**