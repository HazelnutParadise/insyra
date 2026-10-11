## ADDED Requirements

### Requirement: A pandas DataFrame's name is never one of its columns

When `insyra.Return` sends a pandas DataFrame, the table SHALL take a `name` as its name only from the frame itself: a column called `name` SHALL NOT be read as the name, while a name set on the frame, which pandas keeps in the instance's `__dict__`, SHALL be.

#### Scenario: A column called name
- **WHEN** Python returns `pd.DataFrame({"name": ["x", "y"], "v": [1, 2]})`
- **THEN** the table has no name and its columns are `name` and `v`

#### Scenario: A name set on the DataFrame
- **WHEN** Python sets `df.name = "scores"` on a DataFrame without a column called `name` and returns it
- **THEN** the table is named `scores`

#### Scenario: A name column beside an explicit name
- **WHEN** a DataFrame has a column called `name` and a name set on the frame itself
- **THEN** the table is named that name
