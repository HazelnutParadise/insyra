## ADDED Requirements

### Requirement: A pandas DataFrame's name is never one of its columns

When `insyra.Return` sends a pandas DataFrame, the table SHALL take the DataFrame's `name` attribute as its name only when `name` is not a label of the DataFrame's columns. A DataFrame with a column labelled `name`, at any level of its column index, SHALL come back with no table name.

#### Scenario: A column called name
- **WHEN** Python returns `pd.DataFrame({"name": ["x", "y"], "v": [1, 2]})`
- **THEN** the table has no name and its columns are `name` and `v`

#### Scenario: A name set on the DataFrame
- **WHEN** Python sets `df.name = "scores"` on a DataFrame without a column called `name` and returns it
- **THEN** the table is named `scores`
