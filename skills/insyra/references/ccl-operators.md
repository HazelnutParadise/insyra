# CCL Operators Reference

Source of truth: `Docs/CCL.md` (Operators section). If behavior differs, follow the repo docs/tests for your target version.

## Arithmetic + special operators

| Operator | Meaning | Notes / Examples |
|---|---|---|
| `+` | Addition | `A + B` |
| `-` | Subtraction | `A - B` |
| `*` | Multiplication | `A * B` |
| `/` | Division | `A / B` |
| `^` | Exponentiation | `A ^ 2` |
| `.` | Row access | `A.0`, `['Sales'].10`, `A.(1:5)` |
| `:` | Range | Column range: `A:C` / `[A]:[C]` / `['Start']:['End']`; Row range: `@.0:5`, `A.0:5`, `A.(1:5)` |
| `#` | Current row index (0-based) | `A.#` (same row), `IF(#>0, A.(#-1) - A, NULL)` |

### Range expansion in aggregate functions
When a range is used inside an aggregate function (`SUM`, `AVG`, `MIN`, `MAX`, ...), it expands into a flat list of values.

- `SUM(A:C)` sums all values in columns A, B, C.
- `SUM(@.0:5)` sums all values in rows 0..5 across all columns.
- `AVG(A.(0:10))` averages first 11 values of column A.

Note: raw row ranges like `SUM(0:5)` are NOT supported. Use `@.0:5` or `A.0:5`.

### Bounds checking
Row access (`.`) and ranges (`:`) do strict bounds checking. Out-of-range indices/columns throw an error. Negative indices are not supported.
### Combined column+row ranges (recommended parentheses)
When both a **column range** and a **row range** appear together, prefer explicit parentheses to avoid ambiguity:

- Recommended: `(A:B).(1:5)` (first select columns A..B, then slice rows 1..5)
- Avoid: `A:B.(1:5)` (harder to read; may be mis-parsed by humans)

This is especially useful in nested expressions and aggregate calls (e.g., `SUM((A:B).(1:5))`).


## Comparison operators

| Operator | Meaning | Examples |
|---|---|---|
| `>` | greater than | `A > B`, `A > 10` |
| `<` | less than | `A < 10` |
| `>=` | greater or equal | `A >= B` |
| `<=` | less or equal | `10 <= A <= 20` (chained comparisons supported) |
| `==` | equal | `A == B`, `A == nil` |
| `!=` | not equal | `A != B` |

Nil/Null note:
- `== nil` or `== null` checks missing values.
- In arithmetic operations, `nil` is treated as `0`.
- In logical operations, `nil` is treated as `false`.

## Logical operators

| Operator | Meaning | Examples |
|---|---|---|
| `&&` | AND | `A > 10 && B < 20` |
| `||` | OR | `A > 10 || B > 10` |

## String concatenation

| Operator | Meaning | Examples |
|---|---|---|
| `&` | Concatenate strings | `A & '-' & B`, `CONCAT(A, ' ', B)` |
