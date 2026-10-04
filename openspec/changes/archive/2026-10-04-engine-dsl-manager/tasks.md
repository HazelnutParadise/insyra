# Tasks: engine-dsl-manager

## 1. 測試先行

- [x] 1.1 `engine/dsl/manager_test.go`：只用 `engine/dsl` 建 Manager、開 session、列環境、讀狀態；`DefaultManager` 每次回傳不同實例且位置是 `~/.insyra/envs`；在舊程式上編譯失敗
- [x] 1.2 `cli/env/deprecated_wrappers_test.go`：七個名稱有指向 `engine/dsl` 的 Deprecated，`Default`、`ConfigKeys`、`ExportPayload` 沒有；在舊程式上失敗

## 2. 實作

- [x] 2.1 `engine/dsl/manager.go`：`Manager`、`NewManager`、`DefaultManager` 與五個型別；`engine/dsl/dsl.go` 的 `NewSession` 參數改寫成 `*Manager`，說明寫明只會自動建立 `default`、`Session` 不能並行使用
- [x] 2.2 `cli/env/env.go`：七個名稱標 Deprecated，`Default` 與套件說明指向 `engine/dsl`
- [x] 2.3 `internal/dsl/env/manager.go`：`NewManager` 說明不再說設定在建立後固定；`internal/dsl/session.go`：nil manager 的錯誤訊息改指向 `dsl.DefaultManager()`
- [x] 2.4 repo 內用到這些舊名稱的測試與 `cli/repl/session.go` 改用 `engine/dsl` 的名稱；`engine/dsl/api_test.go` 刻意保留舊寫法，證明仍可編譯

## 3. 文件與紀錄

- [x] 3.1 `Docs/cli-dsl.md`：範例只 import `engine/dsl`；修正開啟不存在的 `demo` 環境會失敗的範例，並在 scratchpad 的獨立 module 實際編譯、執行
- [x] 3.2 `engine/README.md`：DSL 章節
- [x] 3.3 `CHANGELOG.md`、`CHANGELOG_TW.md`：`### CLI` 段落末尾新增條目
- [x] 3.4 `AGENTS.md`：`cli/env` 的移除 follow-up 加上這七個名稱
- [x] 3.5 `delivery-status.md`：Latest Milestones 最上方新增條目
- [x] 3.6 skills：不需要修改，`use-insyra-cli` 只指向 `engine/dsl` 與文件位置，沒有變

## 4. 驗證

- [x] 4.1 `gofmt -s -l`、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`（0 issues）、`openspec validate engine-dsl-manager --strict`；`go list -deps ./engine/dsl` 沒有 `cli/`、Cobra、readline
