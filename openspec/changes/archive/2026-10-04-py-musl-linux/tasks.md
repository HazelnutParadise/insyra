# Tasks: py-musl-linux

## 1. 調查與測試先行

- [x] 1.1 實測：Alpine arm64 容器只裝 `build-base`，用新清單建環境成功，三個套件從原始碼編譯，編譯工具全部照清單，12 個套件 import 成功
- [x] 1.2 `py/environment_pins_test.go`：Linux 的 wheel 規則分成 manylinux 與 musllinux，在缺 `knownSourceBuilds` 條目時確認失敗並列出四個缺口
- [x] 1.3 用舊清單強制編 `scikit-learn`，確認解析失敗（cython 3.3.0 不符合 `<3.2.6`）

## 2. 實作

- [x] 2.1 `knownSourceBuilds` 加上兩個 musl 平台；1.2 通過
- [x] 2.2 build constraints 依四個套件的編譯需求加上 `ninja`、`patchelf` 重新解析，12 個工具、208 個雜湊值，用釘版 uv 重跑 `uv lock`，套件本身的解析不變
- [x] 2.3 端對端測試與編譯檢查的上限放寬到 75 分鐘
- [x] 2.4 `.github/workflows/py-musl.yml`：Alpine 容器裡在 amd64 與 arm64 都跑端對端測試與編譯檢查，最後確認測試真的有跑；移除 macOS 的 `py-build-tools.yml`
- [x] 2.5 編譯檢查改成只編 `knownSourceBuilds` 裡目前平台的套件（以 `/lib/ld-musl-*.so.1` 判斷 musl），沒有要編的平台就跳過。強制編 `blis` 在 Alpine 上 gcc 與 clang 都失敗，那是使用者不會遇到的編譯

## 3. 文件與紀錄

- [x] 3.1 `Docs/py.md`：musl Linux 會編譯哪些套件、要裝 `build-base`、第一次建環境要多久；升級步驟改成對 `knownSourceBuilds` 的每個平台解析
- [x] 3.2 `CHANGELOG.md`、`CHANGELOG_TW.md`：`### py` 新增條目
- [x] 3.3 `delivery-status.md`：Latest Milestones 最上方新增條目；`AGENTS.md` 刪掉已解決的 musl follow-up

## 4. 驗證

- [x] 4.1 本機 Alpine arm64 容器跑 CI 的同一串指令：端對端測試與編譯檢查都通過；macOS 上編譯檢查會跳過
- [x] 4.2 本機 Alpine amd64 容器建環境成功、`scikit-learn` 從原始碼編譯
- [x] 4.3 `gofmt -s -l`、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`（0 issues）、`openspec validate py-musl-linux --strict`
- [x] 4.4 push 後 `Python on musl Linux` workflow 三個 job 都通過
