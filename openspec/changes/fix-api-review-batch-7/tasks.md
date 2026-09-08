# Tasks: fix-api-review-batch-7

## 1. Tests first

- [x] 1.1 `cli/commands/batch7_test.go`：七個指令失敗時回錯、五種拼錯的列舉值被拒、config 的 key 與值驗證、accel Usage 與旗標註冊、plot／fetch 拒絕多餘引數、sample／setcolnames 範圍檢查

## 2. Implementation

- [x] 2.1 `cli/commands/targets.go`：`resolveColumn`、`requireColumnName`、`requireRowIndex`、`requireRowName`、`checkTableErr`、`parseSortDirection`
- [x] 2.2 `sort.go`、`dropcol.go`、`droprow.go`、`swap.go`、`ccl.go`：先檢查目標，事後檢查 `PopErr()`
- [x] 2.3 `clean.go`、`merge.go`、`hypothesis.go`：封閉式列舉解析，`parseAlternativeHypothesis` 改回傳錯誤，新增 `parseEqualVariance`
- [x] 2.4 `cli/env/config.go`：key 白名單、`log-level`／`no-color`／`accel-mode` 值驗證、`ConfigKeys()`
- [x] 2.5 `accel.go`、`registry.go`：Usage 去掉 `run`、加上 `--precision` 並註冊
- [x] 2.6 `plot.go`、`fetch.go`：拒絕不認識的引數；Usage 移除 `[options...]`
- [x] 2.7 `sample.go`、`setcolnames.go`：範圍與數量檢查

## 3. Docs, changelog, review ledger

- [x] 3.1 `Docs/cli-dsl.md`、`skills/use-insyra-cli/references/cli-command-usage.md`：新的錯誤行為與 accel／plot 的 Usage
- [x] 3.2 `CHANGELOG.md` 與 `CHANGELOG_TW.md`：CLI
- [x] 3.3 `api-review.md`：CLI-7、CLI-8、CLI-9、CLI-10、CLI-13、CLI-14 標已修正；關閉 #316、#317、#318、#319、#322、#323

## 4. Verification

- [x] 4.1 `go test ./...` 全綠；`golangci-lint run` 0 issues
- [x] 4.2 `openspec validate fix-api-review-batch-7 --strict` 通過
