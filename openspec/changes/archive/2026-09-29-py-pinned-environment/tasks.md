# Tasks: py-pinned-environment

## 1. 釘版資料

- [x] 1.1 以 uv 0.12.20 在暫存目錄解析 12 個套件目前的穩定版，產生 `py/environment/pyproject.toml`（`requires-python = "==3.12.14"`、`[tool.uv] required-version = "==0.12.20"`、每個套件 `name==version`）與 `uv.lock`；在暫存目錄以 `uv sync --locked --inexact --managed-python` 實際建好環境，13 個 import 全部成功
- [x] 1.2 `py/environment/uv-sha256.sum`：0.12.20 release 的 `sha256.sum` 原檔，以 `cmp` 確認與 release 資產逐位元組相同；八個平台壓縮檔下載後以 `shasum -c` 核對
- [x] 1.3 `.gitattributes`：`py/environment/*` 不做換行轉換

## 2. 測試先行

- [x] 2.1 釘版一致性測試：讀得出 uv 與 Python 版本；前導程式 import 的每個套件都釘死；lock 與釘版、Python 需求一致；六個平台都有壓縮檔與 checksum；不支援的平台回錯。在舊程式上確認失敗（函式不存在）
- [x] 2.2 下載測試（`httptest`）：checksum 相符時從 `.tar.gz` 與 `.zip` 取出執行檔；不符時回錯且不寫檔；壓縮檔裡沒有 uv 時回錯
- [x] 2.3 環境同步測試（以測試執行檔充當 uv）：第一次建置執行一次 `uv sync` 並寫入標記；標記相符時不呼叫 uv；標記不同時重新同步；同步失敗時回傳 uv 的錯誤輸出、不寫標記、下一次重試；context 到期時很快回傳 `DeadlineExceeded`；`ReinstallPyEnv` 刪掉整個環境目錄但保留 uv
- [x] 2.4 端對端測試（`INSYRA_PY_E2E=1` 才執行）：在暫存目錄從零下載 uv、建環境，`RunCode` 回傳 DataFrame，`PipList` 的版本等於釘版
- [x] 2.5 審查後補強：`uv sync` 用 `--frozen`；子行程移除使用者的 `UV_PYTHON_PREFERENCE`、`UV_PROJECT_ENVIRONMENT`，`UV_PYTHON_DOWNLOADS` 固定為 `automatic`；lock 只取自 PyPI、各平台都有 wheel（Windows arm64 的 blis 與 statsmodels 列為例外）；等待別人準備環境時遵守 context；`ReinstallPyEnv` 刪除失敗不留下就緒標記；寫入 uv 前先 fsync。各項測試先在舊程式上失敗

## 3. 實作

- [x] 3.1 `py/environment.go`：嵌入三個檔案、讀取釘版、平台對照、下載與驗證、解壓、同步與標記
- [x] 3.2 `py/init.go`：`pyEnvInit(ctx)` 改走釘版環境，失敗不標記完成；移除安裝腳本與逐一安裝套件的程式碼
- [x] 3.3 `py/const.go`、`py/py.go`：移除 `pythonVersion`，新增 `uvPath`；`Pip…` 改用釘版 uv；執行函式把 context 傳給環境準備；`ReinstallPyEnv` 取得鎖、刪整個目錄後重建；2.1 到 2.3 通過；在本機跑 2.4 通過

## 4. 文件與紀錄

- [x] 4.1 `Docs/py.md`：首次使用做了什麼、釘版內容在哪、如何升級釘版、`ReinstallPyEnv` 刪除的範圍、context 也涵蓋環境準備
- [x] 4.2 `CHANGELOG.md`、`CHANGELOG_TW.md`：`### py` 段落末尾新增條目，標明 BREAKING（行為改變）
- [x] 4.3 `api-review.md`：SEC-10 標為已修正；PY-1 註明 `ReinstallPyEnv` 文件已補
- [x] 4.4 `delivery-status.md`：Latest Milestones 最上方新增條目
- [x] 4.5 檢查 `skills/insyra/SKILL.md`：不談 Python 環境，本變更不需修改

## 5. 驗證

- [x] 5.1 `gofmt -s -l`、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`（0 issues）、`openspec validate py-pinned-environment --strict`
