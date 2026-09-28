# Tasks: one-json-library

## 1. Migration

- [x] 1.1 十個非測試檔改 import `github.com/goccy/go-json`，呼叫與型別不變
- [x] 1.2 `.golangci.yml` 加 `depguard`：非測試檔禁止 `encoding/json` 與其他常見 JSON 庫
- [x] 1.3 `BenchmarkJSONPaths`：固定種子的 100,000 × 5 表，量 `ToJSON_Bytes` 與 `ReadJSON`
- [x] 1.4 `nn/safetensors.go` 判斷標頭後面沒有多餘值時改用 `errors.Is(err, io.EOF)`：errorlint 只放行標準庫回傳的未包裝 `io.EOF`，換庫後會報；go-json 在乾淨結尾回傳的就是 `io.EOF`，行為不變

## 2. Verification

- [x] 2.1 在非測試檔暫時 import `encoding/json`，`golangci-lint run` 失敗；拿掉後 0 issues
- [x] 2.2 `go build ./...`、`go vet ./...`、`go test ./...` 全綠
- [x] 2.3 `INSYRA_ACCEL_GPU_TESTS=1 go test ./accel/...` 在 M3 上全綠（`native_probe.go` 有改）
- [x] 2.4 CLI 環境舊版 `state.json` 讀取與存檔測試全綠；同一個腳本新舊版 CLI 存出的 `state.json` 只差 `lastAccess` 時間戳，`config.json` 逐位元組相同，新版讀舊版的 state 再存回與舊版自己存回的相同
- [x] 2.5 govulncheck v1.3.0 用 Go 1.25.14 跑完，結果與 take-go-json-v0-11 相同

## 3. Records

- [x] 3.1 `AGENTS.md` 的 Key Conventions 寫明規則、現用的庫、換庫的條件與做法
- [x] 3.2 `openspec validate one-json-library --strict` 通過
