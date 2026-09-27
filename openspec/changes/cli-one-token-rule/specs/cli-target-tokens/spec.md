## ADDED Requirements

### Requirement: A token is read every way it can be

選取欄或列的 CLI token 若沒有前綴，系統 SHALL 同時嘗試以下讀法：全為數字（可帶負號）時讀作從 0 起算的編號，負數從最後倒數；全為英文字母時讀作 Excel 式欄索引（只適用於欄，不分大小寫）；任何 token 都讀作名稱並精確比對。所有成立的讀法指向同一欄（列）時，系統 SHALL 使用它。

#### Scenario: A name that is also letters past the last column
- **WHEN** 對欄位為 `name, price, qty` 的表格執行 `sort t price`
- **THEN** 依 `price` 欄排序

#### Scenario: Letters and a number
- **WHEN** 對同一張表執行 `sort t B` 或 `sort t 1`
- **THEN** 都依 `price` 欄排序

#### Scenario: A number that is also a column name
- **WHEN** 表格欄位為 `key, 1, 2021`，執行 `col t 1` 與 `col t 2021`
- **THEN** 前者取名為 `1` 的欄（編號 1，兩種讀法一致），後者取名為 `2021` 的欄（當編號超出範圍）

### Requirement: Disagreeing readings are refused

兩種成立的讀法指向不同的欄（列）時，系統 SHALL 回傳錯誤，說明兩種讀法各指向哪一欄，並給出能選定其中一種的前綴寫法；系統 SHALL NOT 自行選擇其中一種，也 SHALL NOT 執行動作。

#### Scenario: Letters against a name
- **WHEN** 表格欄位為 `x, a`，執行 `sort t a`
- **THEN** 回傳錯誤，指出 `index A` 是名為 `x` 的欄、名為 `a` 的欄是編號 1，並建議 `index:a` 或 `name:a`

#### Scenario: A number against a name
- **WHEN** 表格欄位為 `id, 0`，執行 `sort t 0`
- **THEN** 回傳錯誤並建議 `number:0` 或 `name:0`

### Requirement: Prefixes pick one reading

`number:`、`index:`、`name:` 前綴 SHALL 只採用對應的讀法，超出範圍或找不到時 SHALL 回傳錯誤，SHALL NOT 改用其他讀法。列 SHALL 接受 `number:` 與 `name:`；對列使用 `index:` SHALL 回傳錯誤。`groupby` 與 `resample` 的 `<col>:<op>` 規格中，前綴 SHALL 留在欄的部分。

#### Scenario: Forcing the name
- **WHEN** 表格欄位為 `x, a`，執行 `sort t name:a`
- **THEN** 依名為 `a` 的欄排序

#### Scenario: A prefixed spec
- **WHEN** 執行 `groupby t by name agg name:price:sum`
- **THEN** 以名為 `price` 的欄加總

### Requirement: Every command that picks a column or a row uses the rule

`col`、`row`、`get`、`set`、`sort`、`swap`、`dropcol`、`droprow`，以及 `fillna`、`groupby`、`describe`、`encode`、`parsedates`、`pivot`、`unpivot`、`resample`、`scale`、`merge … on` 的欄清單 SHALL 以同一套規則解析 token。多個 token 的指令 SHALL 在任何修改之前解析全部 token，其中一個失敗時資料 SHALL 保持不變。

#### Scenario: get with a column name
- **WHEN** 對欄位為 `name, price, qty` 的表格執行 `get t 0 price`
- **THEN** 印出第 0 列的 `price` 值，而不是 `<nil>`

#### Scenario: set with a row name
- **WHEN** 執行 `set t r1 qty 9`，其中 `r1` 是列名
- **THEN** 寫入該列的 `qty`

### Requirement: A command does not inherit an earlier command's error

每個指令執行前，系統 SHALL 清除所有變數上先前記錄的錯誤，使一個指令的失敗 SHALL NOT 被之後的指令當成自己的錯誤回報。

#### Scenario: A failure followed by a valid command
- **WHEN** `col t B` 失敗後執行 `sort t price`
- **THEN** `sort` 成功
