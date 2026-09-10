# CCL Operators Reference

Source of truth: `Docs/CCL.md` (Operators section). If behavior differs, follow the repo docs/tests for your target version.

## Arithmetic + special operators

| Operator | Meaning | Notes / Examples |
|---|---|---|
| `+` | Addition | `A + B` |
| `-` | Subtraction | `A - B` |
| `*` | Multiplication | `A * B` |
| `/` | Division | `A / B` |
| `%` | Remainder | `A % 3` (same as `MOD`; `A % 0` is an error) |
| `^` | Exponentiation | `A ^ 2` (left-associative: `2^3^2` = 64) |
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

Indices must be **whole numbers**: `A.(1.7)`, `A.(0:1.9)` and `ROLLING_MEAN(A, 2.9)` are errors, not silently truncated. NaN and infinity are errors too.

### A range means different things in different places
Inside an aggregate, `A:C` is every value in those columns (`SUM(A:C)`). On its own it is the **current row** restricted to those columns, so `AddColUsingCCL("r", "A:B")` gives each cell that row's A and B values as a slice — same as `(A:B).#`, and the same defaulting that makes bare `A` mean `A.#`.

A **row** range has no standalone reading: `1:5` says which rows but not of what, so attach it to a column (`A.(1:5)`).

`LAG` and `LEAD` accept a whole row: `LAG(@, 1)` gives every row the one before it, `LAG(A:B, 1)` the same for those columns. Other sequence functions do arithmetic per value and reject a row (`CUMSUM(@)` is an error).

### Combined column+row ranges (recommended parentheses)
When both a **column range** and a **row range** appear together, prefer explicit parentheses to avoid ambiguity:

- Recommended: `(A:B).(1:5)` (first select columns A..B, then slice rows 1..5)
- Avoid: `A:B.(1:5)` (harder to read; may be mis-parsed by humans)

This is especially useful in nested expressions and aggregate calls (e.g., `SUM((A:B).(1:5))`).


### Case
Function names, Excel-style column indices and keywords ignore case: `sum(a)` == `SUM(A)`, `nil` == `NULL`. **Column names do not**: `['price']` and `['Price']` are different columns, and a name that is not there is an error.

### Numbers rendered as text
`&`, `CONCAT`, `TOSTR`, `LEN` and the string functions use Go's default number formatting, so very large and very small `float64` values come out in scientific notation: `'x' & 0.0000001` is `"x1e-07"`. A value read from an integer column keeps its type (`LEN(A)` on the integer `1000000` is `7`), but a literal written in the expression is always `float64` (`LEN(1000000)` is `5`, because it renders as `"1e+06"`). Use `TOSTR(x, fmt)` when the exact text matters.

### Names vs indexes (quoting rule of thumb)
- **Names** use quotes (typically single quotes):
  - Column name: `['Price']`
  - Row name: `.'Peter'` or `.(0:'Peter')` / `.('Row1':'Row5')`
- **Indexes** do **not** use quotes:
  - Column index: `A`, `[A]`, `A:C`, `[A]:[C]`
  - Row index: `.0`, `.(0:5)`, `@.0:5`

You can **mix** name + index in the same expression. For readability, parenthesize when ranges are involved:

- Example (mixed column range + mixed row range): `([A]:['Price']).(0:'Peter')`

## Comparison operators

| Operator | Meaning | Examples |
|---|---|---|
| `>` | greater than | `A > B`, `A > 10` |
| `<` | less than | `A < 10` |
| `>=` | greater or equal | `A >= B` |
| `<=` | less or equal | `10 <= A <= 20` (chained comparisons supported) |
| `==` | equal | `A == B`, `A == nil` |
| `!=` | not equal | `A != B` |

What gets compared: numbers first (including numeric strings, so `'10' > '9'` is true); if neither side is a number and both are strings, they compare as text (`'apple' < 'banana'` is true); a word against a number is an error.

Nil/Null note:
- `== nil` or `== null` checks missing values.
- In arithmetic operations, `nil` is treated as `0`.
- In logical operations, `nil` is treated as `false`.

## Logical operators

| Operator | Meaning | Examples |
|---|---|---|
| `&&` | AND | `A > 10 && B < 20` |
| `||` | OR | `A > 10 || B > 10` |

Both **short-circuit**, so a guard works: `B != 0 && A / B > 1` returns false for `B = 0` instead of failing. `IF`, `AND()`, `OR()` and `CASE()` do the same. `AND()`/`OR()` need at least two arguments.

Operands are read as booleans, not required to be booleans: numbers (`0` is false), and the strings `true`/`yes`/`1`/`false`/`no`/`0`/`''`. Any other string is an error.

## String concatenation

| Operator | Meaning | Examples |
|---|---|---|
| `&` | Concatenate strings | `A & '-' & B`, `CONCAT(A, ' ', B)` |

`nil` concatenates as the empty string: `nil & 'x'` is `"x"`.

## Precedence

Tightest first: `:` → `.` → `^` → `*` `/` `%` → `+` `-` → `&` → comparisons → `&&` → `||`.

`&` binds **looser** than arithmetic, as in Excel: `'a' & 1 + 2` is `"a3"`. Unary minus binds tighter than `^`: `-2^2` is `4`.

Function arguments take exactly one comma between them. `SUM(A B)`, `IF(A > 1, 1, 0,)` and `SUM(A,,B)` are all errors.
