# Tasks: cli-docs-match-registry

## 1. Tests first

- [x] 1.1 `cli/commands/docs_sync_test.go` `TestCLIDocsMatchRegistry`：每個註冊指令在三份文件都有條目，且 usage 行與 `Usage` 完全相同；多出的條目只允許 `completion`、`load sql`、`save sql`；解析數量過少時失敗（先紅：`accel` 三份都缺，21 處 usage 不符）
- [x] 1.2 `TestCLITopicListsNameEveryCommand`：Command Groups 與 `cli-commands.md` 列出每個指令（先紅：Command Groups 缺 `accel`、`describe`、`fillnan`）
- [x] 1.3 `usage_accuracy_test.go`：`pca`、`regression` 的 Usage 含 `[as <var>]`；`count` 不把 value 標成選填；`save … sql` 的用法錯誤列出 `[rownames [true|false]]`（先紅：五項全失敗）

## 2. Implementation

- [x] 2.1 `pca`、`regression`、`count` 的 Usage 與用法錯誤；`save … sql` 的用法錯誤
- [x] 2.2 三份文件的 usage 行對齊 registry；`fetch`／`load`／`rolling` 的詳細形式移到另一行；usage 參考補上 Parquet 選項
- [x] 2.3 補上 `accel`（三份文件與 Command Groups）；Command Groups 補 `describe`、`fillnan`
- [x] 2.4 修正 `merge`、`count`、`ttest`、`ztest`、`anova`、`ftest`、`chisq` 範例；另外把十二個樣板範例，以及 `percentile x 0.9`、`pca x 3` 換成實跑過的呼叫
- [x] 2.5 修正 JSON `headers` 與 `save … sql` `rownames` 的說明
- [x] 2.6 四份文件開頭不再寫「由 help 產生」，改寫實際的維護方式

## 3. Changelog, review ledger, follow-ups

- [x] 3.1 `CHANGELOG.md`／`CHANGELOG_TW.md`：help 文字修正；更正 batch 7 對 `--precision` 的描述
- [x] 3.2 `api-review.md`：CLI-12 標已修正
- [x] 3.3 `AGENTS.md` follow-ups：CLI 整數比對失敗、接受後忽略的引數
- [x] 3.4 `delivery-status.md`：里程碑與剩餘數量

## 4. Verification

- [x] 4.1 `go test ./...` 全綠；`golangci-lint run` 0 issues
- [x] 4.2 改過的範例在隔離的 HOME 逐一實跑，全部 exit 0，結果與手算相符
- [x] 4.3 `openspec validate cli-docs-match-registry --strict` 通過
