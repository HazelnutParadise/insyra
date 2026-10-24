# Tasks: lpgen-reports-errors

## 1. Tests

- [x] 1.1 `lpgen/write_lp_compat_test.go`：`WriteLP` 與 `GenerateLPFile` 寫出的位元組相同，並與 0.3.x 寫出的檔案相同
- [x] 1.2 `lpgen/lingo_errors_test.go`：`ParseLingo` 與 `ParseLingoFile` 對同一模型回傳相同結果；缺檔回傳 `fs.ErrNotExist`；超長行回傳錯誤；兩個舊名的 doc comment 有指名替代名稱的 `Deprecated:` 段落

## 2. Implementation

- [x] 2.1 `lpgen/write_lp.go`：`(*LPModel).WriteLP(io.Writer) error`，`GenerateLPFile` 的輸出與它相同
- [x] 2.2 `lpgen/lingo.go`：`ParseLingo`、`ParseLingoFile` 共用一個讀 `io.Reader` 的解析器；兩個舊名改為 Deprecated 包裝，失敗時仍記錄警告並回傳 nil

## 3. Docs and changelog

- [x] 3.1 `Docs/lpgen.md`：`WriteLP`、`ParseLingo`／`ParseLingoFile`，與兩個舊名的 Deprecated 說明
- [x] 3.2 `CHANGELOG.md`／`CHANGELOG_TW.md`：`lpgen` 條目
