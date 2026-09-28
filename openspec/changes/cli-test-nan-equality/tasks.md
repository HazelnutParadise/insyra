# Tasks: cli-test-nan-equality

## 1. Tests first

- [x] 1.1 `TestApproxEqualAnyTreatsNaNOnlyAsNaN`（先紅：NaN 對數字在舊 helper 下被當成相等）

## 2. Implementation

- [x] 2.1 `approxEqualAny`：NaN 只等於 NaN
- [x] 2.2 重跑 CLI 測試，逐一調查因此失敗的斷言

## 3. Ledger

- [x] 3.1 `AGENTS.md`：移除這條 follow-up；`delivery-status.md`

## 4. Verification

- [x] 4.1 gofmt、`go vet ./...`、`go test ./...`、`golangci-lint run`、`openspec validate cli-test-nan-equality --strict`
