## ADDED Requirements

### Requirement: A NaN or an infinity in a result comes back as one

When the value Python passes to `insyra.Return` holds a NaN, an `inf` or a `-inf`, the runners SHALL decode it as `math.NaN()`, `math.Inf(1)` or `math.Inf(-1)` wherever the result type can hold a float64: a table or list cell, a float field, a slice, array or map element, or an `any`. Python's `None` SHALL still come back as `nil`. Decoding a NaN or an infinity into a type that cannot hold one SHALL be an error. Text inside strings SHALL be left as it is, and a result holding no NaN or infinity SHALL decode exactly as before.

#### Scenario: A DataFrame with a missing value
- **WHEN** Python returns a DataFrame whose column `a` holds `1.0` and `NaN`, and the result type is `*insyra.DataTable`
- **THEN** the table's second cell in `a` is a NaN and the call returns no error

#### Scenario: Floats outside tables
- **WHEN** Python returns `{"x": [1.5, nan, inf, -inf]}` and the result type is `map[string][]float64`, or `any`
- **THEN** the values are 1.5, a NaN, `+Inf` and `-Inf`

#### Scenario: A NaN for an int
- **WHEN** Python returns a NaN and the result type is `int`
- **THEN** the call returns an error naming the NaN

#### Scenario: The names inside text
- **WHEN** Python returns the string `NaN Infinity`
- **THEN** the string comes back unchanged

### Requirement: A result the Go side cannot read is an error

When the Go side cannot decode the message `insyra.Return` sends, it SHALL answer with an error, and unless the run then delivers another result, the call SHALL return an error saying why rather than `nil`. `insyra.Return` SHALL raise when the answer is an error or when the connection closes without an answer.

#### Scenario: An integer too large for a float64
- **WHEN** Python returns `10**400`
- **THEN** the call returns an error saying the number does not fit in a float64

#### Scenario: Another result after the refusal
- **WHEN** the code catches the exception `insyra.Return(10**400)` raises and returns `"fallback"`
- **THEN** the call returns `"fallback"` and no error

#### Scenario: No answer
- **WHEN** the Go side closes the connection without answering
- **THEN** `insyra.Return` raises in Python, and the process exits with an error
