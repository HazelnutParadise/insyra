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

### 1. Expression Mode (`AddColUsingCCL`)

Evaluates an expression and adds the result as a new column. This mode only supports expressions (no assignment).

```go
// Evaluate expression and add result as new column "result"
dt.AddColUsingCCL("result", "A + B * C")
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
```

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

CCL supports chained comparison operations, allowing concise syntax for range checks:

```
"1 < A < 10"     // Whether A is greater than 1 and less than 10
"0 <= A <= 100"  // Whether A is between 0 and 100 (inclusive)
"A <= B <= C"    // Check if three columns are in ascending order
```

This syntax is equivalent to using the AND operator:

```
"AND(1 < A, A < 10)"     // Equivalent to "1 < A < 10"
"AND(0 <= A, A <= 100)"  // Equivalent to "0 <= A <= 100"
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

### Mode Selection Guide

| Scenario | Recommended Method |
|----------|-------------------|
| Add single calculated column | `AddColUsingCCL` |
| Modify existing column | `ExecuteCCL` with assignment |
| Create multiple columns | `ExecuteCCL` with multiple `NEW()` |
| Complex multi-step calculation | `ExecuteCCL` with multiple statements |
