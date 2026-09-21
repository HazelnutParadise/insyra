## MODIFIED Requirements

### Requirement: Keywords and out-of-range references

`TRUE`／`FALSE`／`NULL`／`NIL` 不分大小寫 SHALL 是字面值。其餘裸識別字 SHALL 只解析為 Excel 式欄索引，賦值左右兩側同一套規則；欄名 SHALL 只能以 `['name']` 指涉。解析 SHALL NOT 退回欄名表，識別字含數字或底線時 SHALL 回報錯誤而非改查欄名。綁定後引用超過最後一欄的 Excel 式索引 SHALL 使 `AddColUsingCCL`／`EditCol*UsingCCL`／`ExecuteCCL` 回報錯誤（`Err()`），SHALL NOT 產生整欄 nil。錯誤訊息 SHALL 在該表存在同名欄位時指出改寫成 `['name']`，該資訊 SHALL NOT 影響解析結果。

#### Scenario: Reference past the last column
- **WHEN** 單欄表執行 `AddColUsingCCL("r", "E + 1")`
- **THEN** `Err()` 非 nil 且表仍是一欄

#### Scenario: A bare word that names a column

- **WHEN** 欄位名為 `price` 的表執行 `AddColUsingCCL("r", "price * 2")`
- **THEN** `Err()` 非 nil，訊息說明 `price` 被讀成欄索引並指出改寫成 `['price']`
- **AND** 同一張表的 `['price'] * 2` 照常求值

#### Scenario: A bare word that cannot be an index at all

- **WHEN** 欄位名為 `qty_1` 的表執行 `AddColUsingCCL("r", "qty_1 * 2")`
- **THEN** `Err()` 非 nil，訊息指出改寫成 `['qty_1']`，SHALL NOT 因為它不是純字母就改查欄名

#### Scenario: An assignment target follows the same rule

- **WHEN** 欄位名為 `price` 的表執行 `ExecuteCCL("price = ['price'] * 10")`
- **THEN** `Err()` 非 nil 且訊息指出改寫成 `['price']`
- **AND** `ExecuteCCL("['price'] = ['price'] * 10")` 與 `ExecuteCCL("A = A * 10")` 照常執行
