## ADDED Requirements

### Requirement: Shuffle reorders a list in place

`DataList.Shuffle(options ...SamplingOptions)` SHALL put the list's own elements in random order and return the list itself, keeping its name. Shuffling an empty list SHALL NOT record an error.

#### Scenario: Shuffling a list
- **WHEN** 對 20 個元素、名為 `ids` 的 list 呼叫 `Shuffle(SamplingOptions{UseSeed: true, Seed: 9})`
- **THEN** 回傳值就是該 list 本身，名稱仍是 `ids`，順序改變，排序後與原本的元素相同，`Err()` 為 nil

#### Scenario: An empty list
- **WHEN** 對空 list 呼叫 `Shuffle()`
- **THEN** 回傳該 list 本身，`Err()` 為 nil

### Requirement: Shuffle reorders a table in place

`DataTable.Shuffle(options ...SamplingOptions)` SHALL put the table's own rows in random order and return the table itself. Every column and the row names SHALL move with their rows, the table's and the columns' names SHALL stay, and a column shorter than the table SHALL be padded with `nil`.

#### Scenario: Shuffling a table
- **WHEN** 對有列名的 5 列表格呼叫 `Shuffle(SamplingOptions{UseSeed: true, Seed: 4})`
- **THEN** 回傳值就是該表格本身，欄名與表名不變，每一列的各欄與列名仍對齊

### Requirement: A failed Shuffle leaves its receiver

Given more than one `SamplingOptions`, `DataList.Shuffle` and `DataTable.Shuffle` SHALL record an error, leave the receiver as it was, and return it.

#### Scenario: Two options
- **WHEN** 呼叫 `dl.Shuffle(seed, seed)` 或 `dt.Shuffle(seed, seed)`
- **THEN** 回傳接收者本身，內容不變，`Err()` 非 nil

### Requirement: Shuffled returns a shuffled copy

`DataList.Shuffled` and `DataTable.Shuffled` SHALL return a new value holding the receiver's elements or rows in random order, named after the receiver with `_Shuffled` appended, and SHALL leave the receiver unchanged.

#### Scenario: The receiver keeps its order
- **WHEN** 對名為 `ids` 的 list 呼叫 `Shuffled(seed)`
- **THEN** 回傳另一個名為 `ids_Shuffled` 的 list，原本的 list 順序不變

### Requirement: Shuffle and Shuffled agree for one seed

For the same `SamplingOptions` seed, `Shuffled` SHALL give the order `Shuffle` gives, row names included for a table.

#### Scenario: One seed, one order
- **WHEN** 對相同內容的兩個 list（或兩張表）分別以同一個 seed 呼叫 `Shuffle` 與 `Shuffled`，seed 從 0 到 49
- **THEN** 兩者的順序（表格連同列名）完全相同，包括欄長不一的表格

### Requirement: A failed Shuffled reports on the copy

Given more than one `SamplingOptions`, `DataList.Shuffled` SHALL return an empty list carrying the error and record it on the receiver too, and `DataTable.Shuffled` SHALL record the error on the table and return an empty table. Neither SHALL change the receiver.

#### Scenario: A failed copy
- **WHEN** 呼叫 `dl.Shuffled(seed, seed)`
- **THEN** 回傳長度 0、`Err()` 非 nil 的新 list，接收者的 `Err()` 也非 nil，內容不變

### Requirement: The CLI's shuffle stores a copy

The CLI's `shuffle <var> … as <name>` SHALL store a shuffled copy under `<name>` and SHALL leave `<var>` in its order, for a DataList and a DataTable alike.

#### Scenario: Shuffling a variable
- **WHEN** 執行 `x shuffle seed 42 as out`
- **THEN** `out` 是另一個值，`x` 的順序不變
