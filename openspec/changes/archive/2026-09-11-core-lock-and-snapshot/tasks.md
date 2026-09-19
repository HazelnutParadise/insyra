# Tasks: core-lock-and-snapshot

## 1. Tests first

- [x] 1.1 `concurrency_batch_test.go`：`AtomicDoAll` 遇 nil 不 panic；`ExecuteCCL` 失敗時表不變；成功時後面的語句看得到前面的結果
- [x] 1.2 `groupby_snapshot_test.go`：GroupBy 後修改父表不影響 Aggregate；`-race` 下不 race
- [x] 1.3 `isr/lockable_test.go`：包裝型別符合 `Lockable`（編譯期檢查）
- [x] 1.4 以 `-race` 實測自訂 CCL 函數讀另一張表的三種寫法，與反向鎖定是否互鎖

## 2. Implementation

- [x] 2.1 `atomic.go`：`Lockable`、`lockHandle`；`AtomicDoAll(...Lockable)`；兩個內部呼叫點與壓力測試改 `[]Lockable`
- [x] 2.2 `datatable_groupby.go`：快照複製欄位資料
- [x] 2.3 `datatable_ccl.go`：`cclWorkingCopy`／`commitCCLWorkingCopy`
- [x] 2.4 測試還原全域設定：`TestMain`、`restoreConfig` 擴充、`stats`／`mkt`／`ml`／`ml/mltest`

## 3. Docs, changelog, review ledger

- [x] 3.1 `Docs/CCL.md`：Custom Functions 章節與鎖定規則；ExecuteCCL 全有或全無
- [x] 3.2 `Docs/DataTable.md`、`Docs/DataList.md`、`skills/insyra/SKILL.md`
- [x] 3.3 `CHANGELOG.md`／`CHANGELOG_TW.md`
- [x] 3.4 `api-review.md`：K-8、T-16、E-2、TS-14、CCL-26 標已修正，CCL-15 更新

## 4. Verification

- [x] 4.1 `go test ./...` 全綠；`-race` 於根套件、`isr`、`stats`；`golangci-lint run` 0 issues
- [x] 4.2 `openspec validate core-lock-and-snapshot --strict` 通過
