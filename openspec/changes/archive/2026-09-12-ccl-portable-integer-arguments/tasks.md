# Tasks: ccl-portable-integer-arguments

## 1. Tests first

- [x] 1.1 `portable_integer_args_test.go`：LEFT、RIGHT、MID 的超大計數與位置、NaN 計數、ROUND 的 NaN 位數、DATEADD 各單位、DAY／HOUR／MINUTE／SECOND、日期加減天數、列範圍上下界（先紅：原生 15 項、`GOARCH=amd64` 19 項，兩邊還給出不同的錯誤日期與數值）

## 2. Implementation

- [x] 2.1 `clampedInt`：LEFT、RIGHT、MID 的計數與位置、ROUND 的位數，先夾在 int32 範圍內，NaN 回錯
- [x] 2.2 `durationOf`：DATEADD 的時分秒、DAY／HOUR／MINUTE／SECOND 的秒數、日期加減天數，超出約 292 年回錯；DATEADD 的日月年超出 int32 範圍回錯
- [x] 2.3 列範圍上下界改用 `wholeIndex`

## 3. Docs, changelog

- [x] 3.1 `Docs/CCL.md`：字串函式、ROUND、DATEADD、日期加減、DAY／HOUR／MINUTE／SECOND 的範圍
- [x] 3.2 `CHANGELOG.md`／`CHANGELOG_TW.md`

## 4. Verification

- [x] 4.1 `go test ./internal/ccl/` 原生與 `GOARCH=amd64` 都綠；`go test ./...` 全綠；`golangci-lint run` 0 issues
- [x] 4.2 CI 的 windows 與 ubuntu（兩者都是 amd64）通過（run 34630799983）
- [x] 4.3 `openspec validate ccl-portable-integer-arguments --strict`
