# Tasks: enforce-chainable-never-nil

- [x] 1.1 `chainable_contract_test.go`：走訪模組原始碼，找出「回傳接收者型別、無 error 回傳值」的方法，對該位置的 `return nil` 失敗
- [x] 1.2 反向驗證：暫時加一個違規方法，確認測試會抓到
- [x] 1.3 防呆：檔案數與方法數低於門檻時失敗，不讓掃描壞掉時靜默通過
- [x] 2.1 `go test ./...` 全綠；`golangci-lint run` 0 issues
- [x] 2.2 `openspec validate enforce-chainable-never-nil --strict` 通過
