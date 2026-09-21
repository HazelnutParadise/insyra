# Tasks: one-column-selector

## 1. 選擇器本體
- [x] 1.1 根套件新增 `Name` 與其型別，並寫出三種形式的解析器（字串=Excel 索引、`Name`=欄名、int=位置，負數從尾端算）。
- [x] 1.2 解析失敗的訊息指名收到的型別與三種可用寫法；欄名恰好也是範圍內索引時記警告並仍用索引。
- [x] 1.3 測試先紅：三種形式、未知型別、警告那一條、欄名當裸字串時的錯誤訊息。

## 2. 核心存取器
- [x] 2.1 `GetCol`、`UpdateCol`、`SetColToRowNames`、四個 `Replace*InCol` 改收選擇器，拿掉大寫欄名 fallback。
- [x] 2.2 補 `GetColByIndex`、`UpdateColByName`、`SetColToRowNamesByName`、四個 `Replace*InColByName`。
- [x] 2.3 既有 `ByName`／`ByNumber` 方法改為薄包裝，行為與通用選擇器一致。

## 3. 分析家族
- [x] 3.1 `resolveColForGroup` 改用共用解析器，欄名優先的順序消失。
- [x] 3.2 `GroupBy`、`Pivot`、`Unpivot`、`Resample`、十個 window 方法與 `AggregateConfig`、`ResampleAgg`、`PivotConfig`、`UnpivotConfig` 改收選擇器。

## 4. 設定結構
- [x] 4.1 `DataTableSortConfig` 的三個欄位收成一個 `Col`，移除優先順序與警告，空設定仍排第一欄。

## 5. 週邊
- [x] 5.1 `isr` 的 `name` 改成根套件型別的別名，`isr.Name` 維持可用。
- [x] 5.2 CLI 的欄位 token 解析改走共用規則，Usage 與說明同步。

## 6. 文件與紀錄
- [x] 6.1 `Docs/DataTable.md` 的欄位定址說明與範例、`AGENTS.md` 的慣例那一行。
- [x] 6.2 兩份 CHANGELOG 標 BREAKING 並寫出遷移寫法；`skills/insyra/`、`skills/use-insyra-cli/` 同步。
- [x] 6.3 全套驗證（gofmt、build、vet、test、golangci-lint）。
- [x] 6.4 `api-review.md` T-11 與 issue 對照列、`delivery-status.md`；歸檔並關閉 #225。
