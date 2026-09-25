## ADDED Requirements

### Requirement: The interfaces list every method, with recorded exceptions

`IDataTable` and `IDataList` SHALL list every exported method of `*DataTable` and `*DataList` except those an extension in the module overrides with its own signature, and each exception SHALL be named with its reason in the test that enforces the list.

#### Scenario: A new method without a place in the interface

- **WHEN** an exported method is added to `*DataTable` but not to `IDataTable` and not to the exceptions
- **THEN** `TestInterfacesListEveryMethod` fails naming it

### Requirement: A type embedding the core table goes wherever a table goes

A type that embeds `*DataTable` SHALL satisfy `IDataTable` and SHALL be accepted by every function taking one, `Merge` included.

#### Scenario: Merging a caller's own table type

- **WHEN** `other.Merge(sales, MergeDirectionVertical, MergeModeOuter)` runs with `sales` a struct embedding `*insyra.DataTable`
- **THEN** the tables are merged

#### Scenario: An isr table given to stats

- **WHEN** an isr table is passed to `stats.PCA`
- **THEN** it compiles and runs without unwrapping
