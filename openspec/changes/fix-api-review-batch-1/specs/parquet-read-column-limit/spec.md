## ADDED Requirements

### Requirement: ReadColumn enforces MaxValues before reading

`parquet.ReadColumn` 在 `opt.MaxValues > 0` 時 SHALL 先由檔案 metadata 加總所選 row group（未指定則全部）的列數；列數超過 `MaxValues` 時 SHALL 回傳錯誤，訊息 SHALL 含實際列數與上限，且 SHALL NOT 讀取任何資料值。`MaxValues == 0` SHALL 維持無上限。

#### Scenario: Over the limit is refused without reading

- **WHEN** 檔案有 1000 列，`ReadColumn(ctx, path, "a", ReadColumnOptions{MaxValues: 10})`
- **THEN** 回傳含 `1000` 與 `10` 的錯誤，`DataList` 為 nil

#### Scenario: Within the limit reads normally

- **WHEN** 同檔案，`MaxValues: 1000`
- **THEN** 回傳 1000 個值

#### Scenario: Limit applies to the selected row groups only

- **WHEN** 檔案有兩個 row group 各 500 列，`RowGroups: []int{0}, MaxValues: 500`
- **THEN** 讀取成功，回傳 500 個值
