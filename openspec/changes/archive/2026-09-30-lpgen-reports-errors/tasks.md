# Tasks: lpgen-reports-errors

## 1. Tests first

- [x] 1.1 新測試：`GenerateLPFile` 在未知目標型別時回傳錯誤、不留檔也不留暫存檔；目標檔已存在時內容不變；目錄不存在時回傳含路徑的錯誤；成功時內容等於 `WriteLP`（舊程式碼無回傳值，不能編譯，即為紅）
- [x] 1.2 新測試：`ParseLingo` 與 `ParseLingoFile` 對同一模型回傳相同結果且等於 `ParseLingoModel_str`；缺檔回傳 `fs.ErrNotExist`；超長行回傳錯誤；兩個舊名的 doc comment 有指名替代名稱的 `Deprecated:` 段落（舊程式碼缺新函式，即為紅）

## 2. Implementation

- [x] 2.1 使用 `parquet-write-options` 搬到 `internal/utils/atomic_file.go` 的 `WriteFileAtomically`
- [x] 2.2 `lpgen/lpgen.go`：`GenerateLPFile` 回傳 `error`，經 `WriteFileAtomically` 寫入，不再記錄警告
- [x] 2.3 `lpgen/lingo.go`：`ParseLingo`、`ParseLingoFile` 共用一個讀 `io.Reader` 的解析器；兩個舊名改為 Deprecated 包裝，失敗時仍記錄警告並回傳 nil
- [x] 2.4 既有測試中依賴舊行為（只剩橫幅的檔案、以記錄代替錯誤）的部分改為新契約

## 3. Docs, changelog, ledger

- [x] 3.1 `Docs/lpgen.md`、`Docs/tutorials/capacity-planning-with-lp-and-lpgen.md`：`GenerateLPFile` 的回傳值與範例、`ParseLingo`／`ParseLingoFile`、舊名
- [x] 3.2 `CHANGELOG.md`／`CHANGELOG_TW.md`：`lpgen`，標 BREAKING
- [x] 3.3 `AGENTS.md`：下一版移除 Deprecated 名稱的 follow-up
- [x] 3.4 `api-review.md`：LP-3 標為已修正；`delivery-status.md`

## 4. Verification

- [x] 4.1 gofmt、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`、`openspec validate lpgen-reports-errors --strict`
