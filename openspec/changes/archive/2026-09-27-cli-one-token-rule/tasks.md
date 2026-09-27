# Tasks: cli-one-token-rule

## 0. Decision

- [x] 0.1 在 #315 留言記下擁有者 2026-09-27 的裁定與理由（csvkit、xsv、pandas 的對照），再開始實作

## 1. Tests first

- [x] 1.1 `col_token_test.go`：欄與列 token 的每種讀法、前綴、衝突、超出範圍、找不到、重名；經由 `Dispatch` 讓八個指令與欄清單各用名稱、字母、編號執行，並在衝突時拒絕；失敗不會污染下一個指令（先紅：舊程式上 `sort t B`、`swap … C`、`dropcol … C`、`set t r1 …` 失敗，`get t 0 price` 印出 `<nil>`，`sort` 回報上一個指令的錯誤）

## 2. Implementation

- [x] 2.1 `targets.go`：`resolveColumnToken`、`resolveRowToken`、`resolveColumnTokens`、`colSelector(s)`、`splitColumnSpec`；移除舊的 `resolveColumn`、`requireColumnName`、`requireRowIndex`、`requireRowName`
- [x] 2.2 傳給程式庫時，名稱能單獨選中該欄就傳名稱，否則傳編號（`resample` 以選取值命名輸出欄，傳編號會把 `Open` 變成 `B`）
- [x] 2.3 `col`、`row`、`get`、`set`、`sort`、`swap`、`dropcol`、`droprow`、`fillna`、`groupby`、`describe`、`encode`、`parsedates`、`pivot`、`unpivot`、`resample`、`scale`、`merge … on` 改用規則；Usage 改為 `<col>`／`<row>`
- [x] 2.4 `registry.go`：每個指令開始前清除變數上殘留的錯誤

## 3. Docs, changelog, follow-ups

- [x] 3.1 `Docs/cli-dsl.md`：規則、衝突訊息範例、前綴；命令索引的 Usage
- [x] 3.2 `skills/use-insyra-cli/SKILL.md` 與兩份 references 的 Usage
- [x] 3.3 `cli/AGENTS.md`：解析 helper 列入必用清單
- [x] 3.4 `CHANGELOG.md`／`CHANGELOG_TW.md`：CLI 段 BREAKING 與錯誤殘留的修正
- [x] 3.5 `api-review.md` CLI-6 標為已修正；`delivery-status.md`

## 4. Verification

- [x] 4.1 隔離 HOME 實跑：衝突訊息、`sort t 0` 衝突、`name:0`、`get t -1 x`
- [x] 4.2 `go test ./...`、`golangci-lint run`、`openspec validate cli-one-token-rule --strict`
