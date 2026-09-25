# Tasks: refuse-extra-optional-values

## 1. 實作
- [x] 1.1 測試先紅：核心 20 項、`finance`、`nn`、`accel`、`plot`、`datafetch` 各給兩個值時必須報錯（`finance` 另以還原修改的方式確認測試會紅）。
- [x] 1.2 核心加 `extraOptional`，各函式在動任何資料前檢查；`finance` 的 `resolveOpts` 改回傳錯誤；`nn`、`accel` 的建構函式把錯誤留到 `Param`／`Backward`／`Build`／`Discover` 回報。

## 2. 文件與紀錄
- [x] 2.1 `Docs/DataList.md`、`Docs/DataTable.md`、`Docs/finance.md`、`Docs/nn.md`、`Docs/accel.md`、`Docs/plot.md`、`skills/insyra/SKILL.md`；兩份 CHANGELOG。
- [ ] 2.2 全套驗證；`api-review.md`、`delivery-status.md`；歸檔、寫 Purpose；在 #213 回報這一批。
