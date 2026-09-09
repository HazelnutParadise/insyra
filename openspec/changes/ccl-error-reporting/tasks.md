# Tasks: ccl-error-reporting

## 1. Tests first

- [x] 1.1 `internal/ccl/errors_test.go`：編譯錯帶偏移量與該處文字、執行錯帶列號與 Unwrap、訊息不含 `{` 或 `0x`
- [x] 1.2 `ccl_error_reporting_test.go`（根套件）：`errors.As(dt.Err(), &ccl.CompileError{})`、執行期錯誤指出列號、編譯與執行訊息可區分

## 2. Implementation

- [x] 2.1 `internal/ccl/errors.go`：`CompileError`、`EvalError`
- [x] 2.2 `ccl_tokenizer.go`：token 記錄 byte offset
- [x] 2.3 `ccl_parser.go`：錯誤改報 byte offset 與該處文字；移除 `%v` 印 struct
- [x] 2.4 `ccl_compiler.go`：`CompileExpression`／`CompileMultiline` 包成 `CompileError`
- [x] 2.5 `ccl_evaluator.go`：`invalid range operands` 改描述運算元
- [x] 2.6 `error_buffer.go`：`ErrorInfo.Cause` 與 `Unwrap`
- [x] 2.7 `datatable.go`：`failErr` 記錄原始錯誤
- [x] 2.8 `ccl.go`：逐列求值錯誤包成 `EvalError`（含列號）
- [x] 2.9 `datatable_ccl.go`：四個進入點改用 `failErr`，去掉 elapsed 前綴
- [x] 2.10 `engine/ccl/ccl.go`：re-export 兩個型別

## 3. Docs, changelog, review ledger

- [x] 3.1 `Docs/CCL.md`：Troubleshooting 說明兩種錯誤與如何判讀
- [x] 3.2 `CHANGELOG.md` 與 `CHANGELOG_TW.md`
- [x] 3.3 `api-review.md`：CCL-16 標已修正；關閉 #354

## 4. Verification

- [x] 4.1 `go test ./...` 全綠；`golangci-lint run` 0 issues
- [x] 4.2 `openspec validate ccl-error-reporting --strict` 通過
