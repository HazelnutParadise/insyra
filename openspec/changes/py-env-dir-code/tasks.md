# Tasks: py-env-dir-code

## 1. 測試先行

- [x] 1.1 `py/environment_pins_test.go`：釘住的 Python 與 `envDirPython` 不同時失敗；代號不符合 `py<兩位數年份><字母>` 時失敗；環境目錄用的是這個代號。在舊程式上確認失敗（常數不存在）

## 2. 實作

- [x] 2.1 `py/const.go`：`envDirCode = "py26a"`、`envDirPython = "3.12.14"`，`installDir` 改用代號；`py/py.go` 的 `ReinstallPyEnv` 註解不寫死代號；1.1 通過

## 3. 文件與紀錄

- [x] 3.1 `Docs/py.md`：目錄改為 `py26a`；說明 Python 換版就換目錄、只換套件時原地同步；升級步驟加入目錄代號
- [x] 3.2 `CHANGELOG.md`、`CHANGELOG_TW.md`：更正 `py-pinned-environment` 條目中「舊環境原地轉換」的說法，寫明舊的 `py25c` 目錄不再使用、可刪除；`ReinstallPyEnv` 條目改為 `py26a`
- [x] 3.3 `delivery-status.md`：Latest Milestones 最上方新增條目，記下擁有者的命名規則

## 4. 驗證

- [x] 4.1 `gofmt -s -l`、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`（0 issues）、`openspec validate py-env-dir-code --strict`
