# Tasks: thread-safety-switch-reversible

## 1. Tests first

- [x] 1.1 新測試：`TurnOnThreadSafety` 後其他設定不變；關閉時建立的 DataList 重新開啟後 64×100 次並行 `Append` 長度正確（舊程式碼沒有這個方法，編譯失敗即為紅）
- [x] 1.2 新測試：只有從關到開的那一次記錄 Info
- [x] 1.3 新測試：`Dangerously_TurnOffThreadSafety` 的註解提到 `AtomicDo`、`AtomicDoAll`、仍安全的部分與 `TurnOnThreadSafety`（舊程式碼為紅）

## 2. Implementation

- [x] 2.1 `config.go`：`TurnOnThreadSafety`；改寫 `Dangerously_TurnOffThreadSafety` 的註解

## 3. Docs, changelog

- [x] 3.1 `Docs/Configuration.md`：關掉的範圍與開回來的方法
- [x] 3.2 `CHANGELOG.md`／`CHANGELOG_TW.md`：Core
- [x] 3.3 `AGENTS.md`：thread safety 那條 Key Convention；`delivery-status.md`

## 4. Verification

- [x] 4.1 gofmt、`go build ./...`、`go vet ./...`、`go test ./...`（根套件另跑 `-race`）、`golangci-lint run`、`openspec validate thread-safety-switch-reversible --strict`
