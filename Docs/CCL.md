# Column Calculation Language (CCL)

CCL (Column Calculation Language) is a specialized expression language in Insyra used for column calculations in DataTables. CCL provides a concise yet powerful way to define how to generate new columns based on existing data.

## Table of Contents

- [Basic Syntax](#basic-syntax)
- [Data Types](#data-types)
- [Operators](#operators)
- [Column References](#column-references)
- [Functions](#functions)
- [Conditional Expressions](#conditional-expressions)
- [Chained Comparisons](#chained-comparisons)
- [Examples](#examples)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

## Basic Syntax

CCL expressions consist of operators, functions, column references, and constants. These expressions are evaluated and applied to each row of a DataTable, generating a new column.

```
NewColumn = CCLExpression
```

Basic operation examples:
```
"A + B"         // Add column A and column B
"A * 2"         // Multiply each value in column A by 2
"(A + B) / C"   // Add columns A and B, then divide by column C
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

CCL uses Excel-like column references, designating columns with letters: A, B, C, etc.

```
"A"          // Reference to the first column
"B"          // Reference to the second column
"C"          // Reference to the third column
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

## Best Practices

1. **Improve Readability**: Break complex expressions into multiple simple column calculations
   ```go
   // Better practice:
   dt.AddColUsingCCL("temp1", "A + B")
   dt.AddColUsingCCL("temp2", "temp1 * C")
   dt.AddColUsingCCL("result", "IF(temp2 > 100, 'High', 'Low')")
   
   // Rather than:
   dt.AddColUsingCCL("result", "IF((A + B) * C > 100, 'High', 'Low')")
   ```

2. **Avoid Deep Nesting**: Deeply nested IF conditions are hard to maintain; consider using the CASE function
   ```go
   // Better practice:
   dt.AddColUsingCCL("grade", "CASE(A >= 90, 'A', A >= 80, 'B', A >= 70, 'C', A >= 60, 'D', 'F')")
   
   // Rather than:
   dt.AddColUsingCCL("grade", "IF(A >= 90, 'A', IF(A >= 80, 'B', IF(A >= 70, 'C', IF(A >= 60, 'D', 'F'))))")
   ```

3. **Mind Data Types**: Ensure you compare values of compatible types; check for null values or strings in columns

4. **Use Chained Comparisons**: For range checks, use chained comparisons to make expressions more concise

## Troubleshooting

### Common Issues

1. **Expression Evaluation Timeout**
   - Simplify complex expressions
   - Check if you have a very large dataset

2. **Type Errors**
   - Ensure all columns involved in operations have compatible data types
   - Use conditional conversions for columns that may contain different types of data

3. **Column Reference Errors**
   - Verify that column references are correct (A, B, C...)
   - Check if referenced columns exist in the DataTable

4. **Invalid Expressions**
   - Check for matching parentheses
   - Ensure function syntax is correct (e.g., IF requires three parameters)
