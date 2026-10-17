## MODIFIED Requirements

### Requirement: A pandas DataFrame's name is never one of its columns

When `insyra.Return` sends a pandas DataFrame, the table SHALL take the DataFrame's `name` attribute as its name. When `name` is a label of the DataFrame's columns, at any level of its column index, the table SHALL take only a name set as an attribute on the DataFrame object and SHALL NOT take the column as its name; a DataFrame with such a column and no attribute name SHALL come back with no table name.

#### Scenario: A column called name

- **WHEN** Python returns `pd.DataFrame({"name": ["x", "y"], "v": [1, 2]})`
- **THEN** the table has no name and its columns are `name` and `v`

#### Scenario: A name set on the DataFrame

- **WHEN** Python sets `df.name = "scores"` on a DataFrame without a column called `name` and returns it
- **THEN** the table is named `scores`

#### Scenario: A name set on a DataFrame with a column called name

- **WHEN** Python sets `df.name = "custom frame"` on `pd.DataFrame({"name": ["x", "y"], "v": [1, 2]})` and returns it
- **THEN** the table is named `custom frame` and its columns are `name` and `v`

## ADDED Requirements

### Requirement: Result field names match by strings.EqualFold

When a key in a result matches no struct field exactly, the field that takes it SHALL be the first field, in declaration order, whose name `strings.EqualFold` accepts for the key. A key that folds to a field's name under any other rule MUST NOT match that field.

#### Scenario: A dotted capital I does not match I

- **WHEN** the result holds a key `İ` and the struct has a field `I` with no exact match
- **THEN** the field `I` keeps its zero value
