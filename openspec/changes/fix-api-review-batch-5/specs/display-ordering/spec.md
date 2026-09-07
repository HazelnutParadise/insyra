## ADDED Requirements

### Requirement: Columns display in their real order

`ShowTypes` 系列 SHALL 以欄位位置排序欄位，與 `Show` 一致；超過 26 欄時 SHALL NOT 出現 `A, AA, AB, B` 這種字串排序。

#### Scenario: 28 columns
- **WHEN** 對 28 欄的表分別呼叫 `ShowRangeTo` 與 `ShowTypesRangeTo`
- **THEN** 兩者的欄位標題順序相同

### Requirement: ShowRange's documented range matches the code

`ShowRange` 的文件 SHALL 說明 end 為排除、負數 end 由尾端往回數且仍為排除（與 Python slice 相同），並說明傳 `nil` 表示到最後。

#### Scenario: Negative end
- **WHEN** 五個元素的 list 呼叫 `ShowRange(2, -1)`
- **THEN** 顯示 index 2 與 3，不含最後一個
