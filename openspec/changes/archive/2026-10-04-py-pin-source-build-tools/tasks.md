# Tasks: py-pin-source-build-tools

## 1. 骨架與測試先行

- [x] 1.1 骨架：`py/environment_build_test.go` 放 `readBuildRequirements` 與受 `INSYRA_PY_E2E` 控制的 `TestSourceBuildsUseOnlyPinnedTools`，兩者都是空格子
- [x] 1.2 `py/environment_pins_test.go`：`TestSourceBuildToolsArePinned` 與讀取 build constraints 的 helper；`knownSourceBuilds` 的註解改成編譯工具已釘版。在舊的 `pyproject.toml` 上確認失敗
- [x] 1.3 `py/environment_buildlog_test.go`：`TestBuildRequirementsAreReadFromTheUVLog`，在空格子上確認失敗

## 2. 實作

- [x] 2.1 `py/environment/pyproject.toml` 加入十個編譯工具的 `build-constraint-dependencies`（以釘版 uv 對 Windows arm64 解析，附每個檔案的 SHA-256），用釘版 uv 重跑 `uv lock`；確認 lock 只多了 `[manifest] build-constraints`；1.2 通過
- [x] 2.2 `py/environment_build_test.go`：填 `readBuildRequirements` 與 `TestSourceBuildsUseOnlyPinnedTools`；1.3 通過
- [x] 2.3 `.github/workflows/py-build-tools.yml`：在 `macos-latest` 上跑編譯檢查，只在 pins 相關路徑變動時觸發，最後確認測試真的有跑。原本放在 `windows-11-arm`，實跑發現那裡用的是 x86-64 Python、不會編譯，改到 macOS

## 3. 文件與紀錄

- [x] 3.1 `Docs/py.md`：Windows arm64 的說明改成 uv 在那裡用 x86-64 Python、不從原始碼編譯，原生 arm64 Python 才會編，編譯工具已釘版；「Pinned versions」加上 build constraints 與升級步驟
- [x] 3.2 `CHANGELOG.md`、`CHANGELOG_TW.md`：`### py` 新增條目
- [x] 3.3 `delivery-status.md`：Latest Milestones 最上方新增條目

## 4. 驗證

- [x] 4.1 在 Mac 上以 `INSYRA_PY_E2E=1` 跑 `TestSourceBuildsUseOnlyPinnedTools`，通過；拿掉 lock 裡的 `ninja` 再跑，確認失敗並指出 `ninja`
- [x] 4.2 突變檢查：改掉 lock 裡一個雜湊值、讓 `numpy` 的編譯版本與環境不同，`TestSourceBuildToolsArePinned` 都會失敗
- [x] 4.3 `gofmt -s -l`、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`（0 issues）、`openspec validate py-pin-source-build-tools --strict`
- [x] 4.4 push 後 `Python Build Tools` workflow 通過，從紀錄確認兩個套件都是從原始碼編譯、裝進來的編譯工具都在清單裡
