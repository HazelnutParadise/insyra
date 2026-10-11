# Tasks: py-windows-arm64-runs

## 1. 測試先行

- [x] 1.1 `py/polars_cpu_check_test.go`：用本機 Python 執行跳過檢查的那段程式，六種情況（x64 Python 在 ARM64 Windows、大小寫不同、原生 arm64 Python、真的 x64 機器、macOS、使用者已設值）；另一個測試確認它在 `import polars as pl` 之前。在沒有常數的程式上確認失敗
- [x] 1.2 修正前的紅燈：x86-64 Python 在 ARM64 Windows 上，`import polars` 以 `unknown feature flag` 失敗

## 2. 實作

- [x] 2.1 `py/py.go`：新增 `skipPolarsCPUCheckUnderEmulation`，放在 `generateDefaultPyCode` 所有 import 之前；1.1 通過

## 3. 文件與紀錄

- [x] 3.1 `Docs/py.md`：Windows arm64 的說明加上 Python 在模擬下執行、為什麼跳過 polars 的檢查
- [x] 3.2 `CHANGELOG.md`、`CHANGELOG_TW.md`：`### py` 新增條目

## 4. 驗證

- [x] 4.1 `gofmt -s -l`、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`（0 issues）
