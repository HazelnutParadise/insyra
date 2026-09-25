# Tasks: core-settings-batch

## 1. 實作
- [x] 1.1 測試先紅：表格補值指定欄補不了要報錯、未指定時跳過；表格內插可外插；`SimpleImputer` 設定包（預設平均、常數、錯配、兩包、未知策略）；`ShowRange` 系列嚴格化；`ShowHead`／`ShowTail`；CLI `fillna` 表格外插與失敗不存檔。
- [x] 1.2 `fillColumns` 依有沒有指定欄位決定報錯或跳過；`FillByInterpolation(extrapolate, cols...)`；`NewSimpleImputer(opts ...SimpleImputerOptions)`；`showRangeProblem` 與 `ShowHead`／`ShowTail`；CLI `fillna`。舊的 CLI 測試照裁定改成報錯。

## 2. 文件與紀錄
- [x] 2.1 `Docs/DataTable.md`、`Docs/DataList.md`、`Docs/cli-dsl.md`、兩個 skills；兩份 CHANGELOG。
- [x] 2.2 全套驗證；`api-review.md`、`delivery-status.md`；歸檔、寫 Purpose；在 #213 回報。
