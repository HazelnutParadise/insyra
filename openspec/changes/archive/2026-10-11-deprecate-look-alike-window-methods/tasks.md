# Tasks: deprecate-look-alike-window-methods

## 1. Tests first

- [x] 1.1 測試：解析 `datalist.go`，四個方法的 doc comment 都有指名替代呼叫的 `Deprecated: use` 段落並註明在 0.4 移除，`ExponentialSmoothing` 沒有。原行為由特性測試檔固定。

## 2. Implementation

- [x] 2.1 `datalist.go`：四個 doc comment 加 Deprecated 段落；`interfaces.go`：`IDataList` 的四個方法同樣標 Deprecated。
- [x] 2.2 `cli/commands/timeseries.go`：`movavg`、`diff` 呼叫保留。

## 3. Docs, changelog, ledger

- [x] 3.1 `Docs/DataList.md`：說明四個方法已 Deprecated，註記在 0.4 移除。
- [x] 3.2 `CHANGELOG.md`／`CHANGELOG_TW.md`：Core。
- [x] 3.3 `delivery-status.md`。

## 4. Verification

- [x] 4.1 全套驗證通過。
