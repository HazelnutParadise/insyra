# Tasks: require-go-1-26

## 1. 版本
- [x] 1.1 `go.mod` 的 `go` 指示改為 `1.26.8`，`go mod tidy` 後確認沒有任何依賴版本變動。
- [x] 1.2 govulncheck 工作流程改用最新的 1.26 修補版。

## 2. 文件與紀錄
- [x] 2.1 `Docs/README.md` 與所有教學的「Go 1.25+」改為「Go 1.26+」。
- [x] 2.2 AGENTS.md 的 chromedp 待辦註明 Go 1.26 的前提在 `0.4` 已成立。
- [x] 2.3 兩份 CHANGELOG 的 BREAKING 條目。
- [x] 2.4 `delivery-status.md` 的里程碑與決策紀錄。

## 3. 驗證
- [x] 3.1 以 Go 1.26.8 執行 `gofmt -l .`、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`。
- [x] 3.2 `govulncheck ./...`。
