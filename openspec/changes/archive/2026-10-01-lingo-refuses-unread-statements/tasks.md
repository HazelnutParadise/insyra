# Tasks: lingo-refuses-unread-statements

## 1. Tests first

- [x] 1.1 新測試：未加分號的最後一句、認不得的敘述、讀不出變數的宣告都回傳含行號與敘述的錯誤；註解被略過；`@GIN`、`@FREE`、`@BND` 讀進對應欄位；舊名稱仍照舊略過（舊程式碼不報錯也不讀這三種宣告，即為紅）
- [x] 1.2 新測試（`lp` 套件）：含 `@FREE` 的 LINGO 模型經 `lp.Solve` 得到 -3

## 2. Implementation

- [x] 2.1 `lpgen/lingo.go`：解析器分嚴格與寬鬆兩種，新名稱用嚴格版，Deprecated 名稱用寬鬆版（與過去相同）；嚴格版讀 `@GIN`、`@FREE`、`@BND`、略過註解、拒絕其餘敘述與未結束的最後一句

## 3. Docs, changelog, ledger

- [x] 3.1 `Docs/lpgen.md`：`ParseLingo`／`ParseLingoFile` 讀什麼、拒絕什麼
- [x] 3.2 `CHANGELOG.md`／`CHANGELOG_TW.md`：`lpgen`
- [x] 3.3 `AGENTS.md`：移除已解決的 follow-up；`delivery-status.md`

## 4. Verification

- [x] 4.1 gofmt、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`、`openspec validate lingo-refuses-unread-statements --strict`
