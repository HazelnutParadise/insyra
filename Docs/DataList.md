## DataList

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
* **Append:** Adds a new element to the end of the DataList.
* **Get:** Retrieves the element at a specific index. Supports negative indexing for accessing elements from the end.
* **Pop:** Removes and returns the last element from the DataList.
* **Drop:** Removes the element at a specific index.
* **DropAll:** Removes all occurrences of specified values from the DataList.
* **Clear:** Removes all elements from the DataList.
* **Len:** Returns the number of elements currently stored in the DataList.

**Data Manipulation:**

* **Sort:** Sorts the elements in the DataList using a mixed sorting logic that handles strings, numbers (various integer and float types), and time data types. If sorting fails, the original order is restored.
* **Reverse:** Reverses the order of elements in the DataList.

**Data Filtering:**

* **ClearStrings:** Removes all string elements from the DataList.
* **ClearNumbers:** Removes all numeric elements (int, float, etc.) from the DataList.

**Data Analysis:**

* **Max:** Returns the maximum value in the DataList. Handles different data types by converting them to a common base (float64) for comparison.
* **Min:** Returns the minimum value in the DataList. Similar logic to Max is applied for data type handling.
* **Mean:** Calculates the arithmetic mean (average) of the DataList elements. Excludes non-numeric data types.
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

**Data Conversion:**

* **ToF64Slice:** Converts the DataList elements to a slice of float64 values. Useful for operations requiring numerical data.

**Timestamps:**

* **GetCreationTimestamp:** Returns the creation timestamp of the DataList in Unix timestamp format (seconds since epoch).
* **GetLastModifiedTimestamp:** Returns the last modification timestamp of the DataList in Unix timestamp format.
* **updateTimestamp:** Internally updates the `lastModifiedTimestamp` whenever the DataList is modified.

**Names (Optional):**

* **GetName:** Retrieves the assigned name for the DataList.
* **SetName:** Sets a name for the DataList. (**Note:** Currently allows any string value.)

**Error Handling:**

* Several methods (e.g., Max, Min, Sort) handle data type mismatches and potential errors during calculations. In such cases, informative messages are printed, and the operation might return `nil` or restore the original state.

**Code Structure:**

The provided code demonstrates the implementation of the `DataList` type and its associated functionalities. It includes method definitions for all the features mentioned above.

**Overall, the `DataList` type provides a versatile and user-friendly data structure  for various use cases within the `insyra` package.**