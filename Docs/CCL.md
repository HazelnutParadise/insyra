# Column Calculation Language (CCL)

CCL (Column Calculation Language) is a specialized expression language in Insyra used for column calculations in DataTables. CCL provides a concise yet powerful way to define how to generate new columns based on existing data.

## Table of Contents

- [Execution Modes](#execution-modes)
- [Basic Syntax](#basic-syntax)
- [Assignment Syntax](#assignment-syntax)
- [Creating New Columns](#creating-new-columns)
- [Data Types](#data-types)
- [Operators](#operators)
- [Column References](#column-references)
- [Functions](#functions)
- [Conditional Expressions](#conditional-expressions)
- [Chained Comparisons](#chained-comparisons)
- [Examples](#examples)
- [Best Practices](#best-practices)
- [Performance](#performance)
- [Troubleshooting](#troubleshooting)

## Execution Modes

CCL supports two execution modes:

### 1. Expression Mode

Expression mode evaluates a CCL expression and applies the result to a column. This mode **only supports pure expressions** (no assignment syntax or NEW function).

> **Note:** If you attempt to use assignment syntax (`=`) or the `NEW()` function in Expression Mode, the operation will be rejected and an error will be logged. Use `ExecuteCCL` (Statement Mode) for these features.

#### `AddColUsingCCL` - Add New Column

Evaluates an expression and adds the result as a new column.

```go
// Evaluate expression and add result as new column "result"
dt.AddColUsingCCL("result", "A + B * C")

// ❌ This will be rejected (assignment syntax not allowed):
dt.AddColUsingCCL("result", "B = A + 1")  // Error: assignment syntax not supported

// ❌ This will be rejected (NEW function not allowed):
dt.AddColUsingCCL("result", "NEW('col')")  // Error: NEW function not supported
```

#### `EditColByIndexUsingCCL` - Edit Column by Index

Evaluates an expression and replaces the content of an existing column by its index (Excel-style: A, B, C, ..., AA, AB, ...).

```go
dt.EditColByIndexUsingCCL("A", "A * 10")      // Multiply first column by 10
dt.EditColByIndexUsingCCL("B", "A + ['C']")   // Set second column to A + C

// ❌ This will be rejected:
dt.EditColByIndexUsingCCL("A", "B = A + 1")  // Error: assignment syntax not supported
```

#### `EditColByNameUsingCCL` - Edit Column by Name

Evaluates an expression and replaces the content of an existing column by its name.

```go
dt.EditColByNameUsingCCL("price", "['price'] * 1.1")        // Increase price by 10%
dt.EditColByNameUsingCCL("total", "['quantity'] * ['price']")

// ❌ This will be rejected:
dt.EditColByNameUsingCCL("price", "price = ['price'] * 1.1")  // Error: assignment syntax not supported
```

### 2. Statement Mode (`ExecuteCCL`)

Executes CCL statements that can modify existing columns or create new ones. Supports:

- Assignment syntax (`column = expression`)
- NEW function for creating columns (`NEW('colName') = expression`)
- Multiple statements separated by `;` or newline

```go
// Modify existing column
dt.ExecuteCCL("A = A * 2")

// Create new column
dt.ExecuteCCL("NEW('total') = A + B + C")

// Multiple statements
dt.ExecuteCCL(`
    A = A * 10
    B = B + 5
    NEW('sum') = A + B
`)

// Or use semicolons
dt.ExecuteCCL("A = A + 1; NEW('doubled') = A * 2")
```

## Basic Syntax

CCL expressions consist of operators, functions, column references, and constants. These expressions are evaluated and applied to each row of a DataTable.

Basic operation examples:

```
"A + B"         // Add column A and column B
"A * 2"         // Multiply each value in column A by 2
"(A + B) / C"   // Add columns A and B, then divide by column C
```

## Assignment Syntax

In Statement Mode (`ExecuteCCL`), you can use assignment syntax to modify existing columns:

```
column = expression
```

### Column Index Assignment

```go
// Modify column A (first column)
dt.ExecuteCCL("A = A * 2")

// Modify column B using values from A and C
dt.ExecuteCCL("B = A + C")
```

### Column Name Assignment

Use `['colName']` syntax to assign by column name:

```go
dt.ExecuteCCL("['price'] = ['price'] * 1.1")  // Increase price by 10%
dt.ExecuteCCL("['total'] = ['quantity'] * ['price']")
```

**Note**: Assignment to non-existent columns will result in an error. Use `NEW()` to create new columns.

## Creating New Columns

Use the `NEW()` function to create new columns in Statement Mode:

```
NEW('columnName') = expression
```

Examples:

```go
// Create a new column named "sum"
dt.ExecuteCCL("NEW('sum') = A + B + C")

// Create column with calculated values
dt.ExecuteCCL("NEW('profit') = ['revenue'] - ['cost']")

// Create column with conditional logic
dt.ExecuteCCL("NEW('status') = IF(A > 100, 'High', 'Low')")
```

## Data Types

CCL supports the following data types:

1. **Numbers** - Integers and floating-point numbers

   ```
   "42"    // Integer
   "3.14"  // Floating-point number
   ```

2. **Strings** - Enclosed in single quotes

   ```
   "'Hello, World!'"   // String
   "'123'"             // Numeric string
   ```

3. **Boolean Values** - `true` or `false`

   ```
   "true"              // Boolean true
   "false"             // Boolean false
   ```

4. **Nil/Null** - `nil` or `null`

   ```
   "nil"               // Nil value
   "null"              // Alias for nil
   ```

## Operators

### Arithmetic Operators

- `+` : Addition
- `-` : Subtraction
- `*` : Multiplication
- `/` : Division
- `^` : Exponentiation

```
"A + B"     // Add column A and column B
"A - B"     // Subtract column B from column A
"A * B"     // Multiply column A by column B
"A / B"     // Divide column A by column B
"A ^ 2"     // Square the values in column A
```

### Comparison Operators

- `>` : Greater than
- `<` : Less than
- `>=` : Greater than or equal to
- `<=` : Less than or equal to
- `==` : Equal to
- `!=` : Not equal to

```
"A > B"      // Whether values in column A are greater than those in column B
"A < 10"     // Whether values in column A are less than 10
"A == B"     // Whether values in column A are equal to those in column B
"A != B"     // Whether values in column A are not equal to those in column B
"A == nil"   // Check if column A is nil
```

> **Note on Nil/Null:**
>
> - `== nil` or `== null` can be used to check for missing values.
> - In arithmetic operations, `nil` is treated as `0`.
> - In logical operations, `nil` is treated as `false`.

### Logical Operators

- `&&` : Logical AND (equivalent to `AND()` function)
- `||` : Logical OR (equivalent to `OR()` function)

```
"A > 10 && B < 20"   // true if A > 10 AND B < 20
"A > 10 || B > 10"   // true if A > 10 OR B > 10
"(A > 0 && B > 0) || C"  // Combined logical operations
```

### String Concatenation Operator

- `&` : String concatenation (equivalent to `CONCAT()` function)

```
"A & B"          // Concatenate A and B (e.g., "Hello" & "World" = "HelloWorld")
"A & '-' & B"    // Concatenate with separator (e.g., "Hello-World")
"A & B & C"      // Chain multiple concatenations
```

## Type Coercion and Comparison Behavior

CCL uses dynamic typing and performs automatic type coercion to handle operations between different data types. Understanding these behaviors is crucial for writing correct CCL expressions.

### Numeric Comparison and Arithmetic

When performing arithmetic operations or comparisons, CCL attempts to convert operands to numbers:

```go
// These are equivalent after type coercion
"123" == 123        // true (string "123" is converted to number 123)
"45.5" > 40         // true (string "45.5" is converted to number 45.5)
"100" + 50          // 150 (string "100" is converted to number 100)
```

**Important Notes:**

- If both operands can be converted to numbers, numeric comparison is used
- String-to-number conversion follows standard parsing rules
- Non-numeric strings cannot be used in arithmetic or numeric comparisons and will result in an error

```go
// These will cause errors
"abc" + 10          // Error: cannot convert "abc" to number
"hello" > 5         // Error: cannot convert "hello" to number
```

### String Concatenation

The `&` operator always performs string concatenation by converting all operands to strings:

```go
"Hello" & " " & "World"     // "Hello World"
"Price: " & 123             // "Price: 123" (number converted to string)
"Value: " & 45.67           // "Value: 45.67"
```

### Handling `nil` Values

CCL has specific rules for handling `nil` values in different operations:

#### Equality and Inequality

```go
nil == nil          // true (both are nil)
nil == 0            // false (nil is not equal to 0)
nil == ""           // false (nil is not equal to empty string)
nil != 123          // true (nil is not equal to any non-nil value)
```

#### Comparison Operations

```go
nil > 10            // false (nil cannot be compared with numbers)
nil < 5             // false
nil >= 0            // false
nil <= 100          // false
```

**Rule:** `nil` with any value in size comparison (`>`, `<`, `>=`, `<=`) always returns `false`.

#### Arithmetic Operations

```go
nil + 10            // 10 (nil is treated as 0)
5 - nil             // 5 (nil is treated as 0)
nil * 3             // 0 (nil is treated as 0)
10 / nil            // Error: division by zero (nil is treated as 0)
```

**Rule:** In arithmetic operations, `nil` is treated as `0`.

#### String Concatenation with `nil`

```go
"Hello" & nil       // "Hello<nil>" (nil is converted to string "<nil>")
nil & " World"      // "<nil> World"
```

### Boolean Operations

Logical operators require boolean operands:

```go
true && false       // false
true || false       // true
(A > 10) && (B < 20)    // Evaluate both conditions

// These will cause errors
"yes" && true       // Error: "yes" is not a boolean
1 && 0              // Error: numbers are not booleans
```

### Type Coercion Summary

| Operation | Left Type | Right Type | Behavior |
|-----------|-----------|------------|----------|
| `+`, `-`, `*`, `/`, `^` | Number/String | Number/String | Convert both to numbers, then calculate |
| `>`, `<`, `>=`, `<=` | Number/String | Number/String | Convert both to numbers, then compare |
| `==`, `!=` | Number/String | Number/String | Convert both to numbers if possible, then compare |
| `==`, `!=` | nil | any | Special nil handling (see above) |
| `&` | any | any | Convert both to strings, then concatenate |
| `&&`, `\|\|` | Boolean | Boolean | Must be boolean, no coercion |

### Best Practices for Type Safety

1. **Be explicit with types**: When dealing with mixed types, consider using conversion functions (if available) to make your intent clear.

2. **Handle nil values carefully**: Always consider how `nil` values might affect your calculations:

   ```go
   // Good: Handle nil explicitly
   "IF(A == nil, 0, A) + B"
   
   // Risky: Relies on automatic nil-to-zero conversion
   "A + B"  // What if A is nil?
   ```

3. **Avoid mixing types unnecessarily**: While CCL supports type coercion, it's clearer to work with consistent types when possible.

4. **Test edge cases**: Test your CCL expressions with various input types, including `nil`, to ensure they behave as expected.

## Column References

CCL provides three ways to reference columns in your expressions:

### 1. Direct Column Index (Excel-style)

The basic way to reference columns uses Excel-style letter identifiers:

```
"A"          // Reference to the first column
"B"          // Reference to the second column
"C"          // Reference to the third column
```

### 2. Bracket Column Index `[colIndex]`

Use brackets with column letters to explicitly reference columns by their index. This is useful when column letters might conflict with function names or reserved words:

```
"[A]"        // Explicit reference to the first column
"[B]"        // Explicit reference to the second column
"[AA]"       // Reference to the 27th column
```

Example - avoiding conflicts:

```go
// If "A" might be confused with a function name, use [A] instead
dt.AddColUsingCCL("result", "[A] + [B] * 2")
```

### 3. Bracket Column Name `['colName']`

Use brackets with single-quoted strings to reference columns by their name. This is particularly useful when you have named columns:

```
"['Sales']"          // Reference to column named "Sales"
"['Product Name']"   // Reference to column named "Product Name"
"['price']"          // Reference to column named "price" (case-sensitive)
```

Example with named columns:

```go
dt := NewDataTable()
dt.AppendCols(NewDataList(100, 200, 300), NewDataList(10, 20, 30))
dt.SetColNames([]string{"revenue", "cost"})

// Calculate profit using column names
dt.AddColUsingCCL("profit", "['revenue'] - ['cost']")
```

### Mixed Syntax

You can mix different reference styles in the same expression:

```go
// Using both column index and column name references
dt.AddColUsingCCL("mixed", "[A] * 2 + ['cost']")

// Using direct reference with bracket syntax
dt.AddColUsingCCL("calc", "A + [B] + ['total']")
```

## Functions

### IF Conditional Function

```
"IF(condition, valueIfTrue, valueIfFalse)"
```

Example:

```
"IF(A > 10, 'High', 'Low')"  
// Returns 'High' if the value in column A is greater than 10, otherwise returns 'Low'
```

### AND and OR Logical Functions

```
"AND(condition1, condition2, ...)"  // Returns true if all conditions are true
"OR(condition1, condition2, ...)"   // Returns true if any condition is true
```

Examples:

```
"IF(AND(A > 0, B < 100), 'Valid', 'Invalid')"  
// Returns 'Valid' if A > 0 and B < 100, otherwise returns 'Invalid'

"IF(OR(A > 90, B > 90), 'Excellent', 'Good')"  
// Returns 'Excellent' if A > 90 or B > 90, otherwise returns 'Good'
```

### CONCAT String Concatenation Function

```
"CONCAT(value1, value2, ...)"
```

Example:

```
"CONCAT(A, '-', B)"  
// Concatenates the value in column A, a hyphen, and the value in column B
```

### ISNA Missing Value Check Function

```
"ISNA(value)"
```

Example:

```
"IF(ISNA(A), 0, A)"  
// Returns 0 if the value in column A is a numeric NaN or the string "#N/A", otherwise returns the value in column A
```

> **Note:** `ISNA` handles numeric `NaN` and the string `"#N/A"`. It does **not** return true for `nil` values.

### IFNA Function

```
"IFNA(value, valueIfNA)"
```

Example:

```
"IFNA(A, 0)"  
// Returns 0 if the value in column A is NaN or "#N/A", otherwise returns the value in column A
```

### CASE Multiple Condition Function

```
"CASE(condition1, result1, condition2, result2, ..., defaultValue)"
```

Example:

```
"CASE(A > 90, 'A', A > 80, 'B', A > 70, 'C', 'F')"  
// Returns 'A' if A > 90, 'B' if A > 80, 'C' if A > 70, otherwise returns 'F'
```

## Conditional Expressions

Conditional expressions are used in functions like IF, AND, OR, and CASE, returning boolean values (true or false).

```
"A > B"          // Whether values in column A are greater than those in column B
"A == 10"        // Whether values in column A are equal to 10
"A != B"         // Whether values in column A are not equal to those in column B
```

## Chained Comparisons

CCL supports chained comparison operations, allowing concise syntax for range checks and multi-value comparisons. You can mix different comparison operators in a single chain.

### Basic Range Checks

```
"1 < A < 10"     // Whether A is greater than 1 and less than 10
"0 <= A <= 100"  // Whether A is between 0 and 100 (inclusive)
"A <= B <= C"    // Check if three columns are in ascending order
```

### Mixed Operator Chains

You can combine different comparison operators (`<`, `>`, `<=`, `>=`, `==`, `!=`) in the same chain:

```
"A == B > C"         // A equals B AND B is greater than C
"A != B < C"         // A is not equal to B AND B is less than C
"A == B > C < D"     // A equals B AND B > C AND C < D
"A < B <= C < D"     // A < B AND B <= C AND C < D
"C >= B >= A"        // C >= B AND B >= A (descending order check)
```

### Equivalence

Chained comparisons are equivalent to using the AND operator:

```
"1 < A < 10"         // Equivalent to: AND(1 < A, A < 10)
"A == B > C"         // Equivalent to: AND(A == B, B > C)
"A < B <= C < D"     // Equivalent to: AND(A < B, B <= C, C < D)
```

## Examples

### Conditional Calculations

```go
// Age classification
dt.AddColUsingCCL("age_group", "IF(A < 18, 'Minor', IF(A < 65, 'Adult', 'Senior'))")

// Calculating discounts
dt.AddColUsingCCL("discount", "IF(B > 1000, B * 0.15, B * 0.05)")

// Multiple condition checking
dt.AddColUsingCCL("status", "IF(AND(A >= 0, B <= 100), 'Valid', 'Invalid')")
```

### Mathematical Operations

```go
// Basic arithmetic
dt.AddColUsingCCL("total", "A + B + C")
dt.AddColUsingCCL("average", "(A + B + C) / 3")

// Exponentiation
dt.AddColUsingCCL("square", "A ^ 2")
dt.AddColUsingCCL("cube", "A ^ 3")
```

### String Operations

```go
// String concatenation
dt.AddColUsingCCL("full_name", "CONCAT(A, ' ', B)")

// Conditional formatting
dt.AddColUsingCCL("label", "CONCAT('ID-', A, ': ', IF(B > 50, 'Pass', 'Fail'))")
```

### Range Checks

```go
// Simple range
dt.AddColUsingCCL("in_range", "IF(10 <= A <= 20, 'Yes', 'No')")

// Multi-column order check
dt.AddColUsingCCL("ascending", "IF(A <= B <= C, 'Ascending', 'Not Ascending')")
```

### Using ExecuteCCL for Complex Operations

```go
// Create a sales analysis table
dt := NewDataTable(
    NewDataList(100, 200, 150, 300).SetName("quantity"),
    NewDataList(10.5, 20.0, 15.5, 25.0).SetName("price"),
)

// Execute multiple CCL statements
dt.ExecuteCCL(`
    NEW('revenue') = ['quantity'] * ['price']
    NEW('tax') = ['revenue'] * 0.1
    NEW('total') = ['revenue'] + ['tax']
`)

// Modify existing columns and create new ones
dt.ExecuteCCL(`
    ['price'] = ['price'] * 1.05
    NEW('adjusted_revenue') = ['quantity'] * ['price']
`)
```

## Best Practices

1. **Choose the Right Mode**:
   - Use `AddColUsingCCL` for simple calculations that add a single new column
   - Use `ExecuteCCL` when you need to modify existing columns or perform multiple operations

2. **Improve Readability**: Break complex expressions into multiple simple column calculations

   ```go
   // Using ExecuteCCL for step-by-step calculations:
   dt.ExecuteCCL(`
       NEW('subtotal') = ['quantity'] * ['price']
       NEW('tax') = ['subtotal'] * 0.1
       NEW('total') = ['subtotal'] + ['tax']
   `)
   
   // Rather than one complex expression:
   dt.AddColUsingCCL("total", "['quantity'] * ['price'] * 1.1")
   ```

3. **Avoid Deep Nesting**: Deeply nested IF conditions are hard to maintain; consider using the CASE function

   ```go
   // Better practice:
   dt.AddColUsingCCL("grade", "CASE(A >= 90, 'A', A >= 80, 'B', A >= 70, 'C', A >= 60, 'D', 'F')")
   
   // Rather than:
   dt.AddColUsingCCL("grade", "IF(A >= 90, 'A', IF(A >= 80, 'B', IF(A >= 70, 'C', IF(A >= 60, 'D', 'F'))))")
   ```

4. **Mind Data Types**: Ensure you compare values of compatible types; check for null values or strings in columns

5. **Use Chained Comparisons**: For range checks, use chained comparisons to make expressions more concise

6. **Use Column Names for Clarity**: When columns have meaningful names, use `['colName']` syntax for better readability

   ```go
   // More readable:
   dt.ExecuteCCL("NEW('profit') = ['revenue'] - ['cost']")
   
   // Less readable:
   dt.ExecuteCCL("NEW('profit') = A - B")
   ```

## Performance

CCL is optimized for high-performance batch processing. The formula is compiled (tokenized and parsed) only once, and the resulting AST (Abstract Syntax Tree) is reused for all rows.

### Benchmark Results

Test environment: 100,000 rows × 3 columns

| Formula Type | Example | Time | Per Row |
|--------------|---------|------|---------|
| Simple arithmetic | `A + B + C` | ~32ms | ~0.32μs |
| Bracket syntax | `[A] + ['colName'] + [C]` | ~43ms | ~0.43μs |
| With function | `IF(A > 50000, 1, 0)` | ~59ms | ~0.59μs |
| Complex expression | `IF(AND(A > 10000, B < 150000), A * 2 + B, C)` | ~103ms | ~1.03μs |

### Performance Tips

1. **Prefer simple expressions**: Arithmetic operations are faster than function calls
2. **Minimize function nesting**: Each function call adds overhead
3. **Use bracket syntax when needed**: `[A]` and `['name']` have minimal overhead compared to direct references
4. **Batch operations**: Process all rows at once using `AddColUsingCCL` rather than row-by-row operations

## Troubleshooting

### Common Issues

1. **Assignment to Non-Existent Column**
   - Error: `assignment target column 'X' does not exist`
   - Solution: Use `NEW('X') = expression` to create new columns, not assignment syntax

2. **Expression Evaluation Timeout**
   - Simplify complex expressions
   - Check if you have a very large dataset

3. **Type Errors**
   - Ensure all columns involved in operations have compatible data types
   - Use conditional conversions for columns that may contain different types of data

4. **Column Reference Errors**
   - Verify that column references are correct (A, B, C...)
   - For named columns, ensure the name is spelled correctly (case-sensitive)
   - Check if referenced columns exist in the DataTable

5. **Invalid Expressions**
   - Check for matching parentheses
   - Ensure function syntax is correct (e.g., IF requires three parameters)
   - For `NEW()`, ensure the syntax is `NEW('name') = expression`

6. **Edit Column Not Found**
   - `EditColByNameUsingCCL`: Ensure the column name exists and is spelled correctly (case-sensitive)
   - `EditColByIndexUsingCCL`: Ensure the column index is within range (A, B, C, ... for columns 1, 2, 3, ...)

### Mode Selection Guide

| Scenario | Recommended Method |
|----------|-------------------|
| Add single calculated column | `AddColUsingCCL` |
| Modify single column by index | `EditColByIndexUsingCCL` |
| Modify single column by name | `EditColByNameUsingCCL` |
| Modify existing column (statement style) | `ExecuteCCL` with assignment |
| Create multiple columns | `ExecuteCCL` with multiple `NEW()` |
| Complex multi-step calculation | `ExecuteCCL` with multiple statements |
