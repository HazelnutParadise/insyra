# Tasks: parse-numbers-types-like-csv

## 1. Tests first

- [x] 1.1 新測試檔：規格中 ParseNumbers 的六個情境，含 2^53+1 保留、`int8`、空字串、`nil` 不報錯、一次錯誤的計數與列號、與 CSV 讀入逐格比對（先紅）
- [x] 1.2 `datalist_test.go` 的 `TestDataListParseNumbers` 改成期望 `int64`
- [x] 1.3 Capitalize 的根規則測試（新舊程式都通過，因為 English 在 x/text 沒有 tailoring）

## 2. Implementation

- [x] 2.1 `datalist.go` `ParseNumbers`：兩段式分類與轉換，一次 `fail`
- [x] 2.2 `datalist.go` `Capitalize`：`language.Und`

## 3. Docs, changelog, ledger

- [x] 3.1 `Docs/DataList.md`：`ParseNumbers` 的型別規則、未轉換的值與錯誤；`Capitalize` 的語言規則
- [x] 3.2 `Docs/cli-dsl.md`：`parsenums` 的說明
- [x] 3.3 `CHANGELOG.md`／`CHANGELOG_TW.md`：Core 新增 BREAKING
- [x] 3.4 `api-review.md`：D-16 標為已修正
- [x] 3.5 `delivery-status.md`

## 4. Verification

- [x] 4.1 gofmt、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`、`openspec validate parse-numbers-types-like-csv --strict`
