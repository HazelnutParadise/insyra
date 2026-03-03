## Why

insyra 目前只能作為 Go library 嵌入程式碼中使用，不具備獨立的命令列介面。使用者必須撰寫 Go 程式才能載入資料、做分析。這對非 Go 開發者、快速探索性分析、或自動化腳本工作流而言門檻過高。需要一個 CLI + REPL 工具，讓使用者可以直接在終端進行互動式資料分析，並透過環境管理機制保存工作進度。

## What Changes

- 新增 `cmd/insyra/main.go` 作為 CLI 程式入口，使用 cobra 框架
- 新增 `cli/` 套件，包含所有命令邏輯、REPL 引擎、環境管理
- CLI 子命令與 REPL 中的 DSL 語法完全統一（共用同一套 command handler）
- 支援環境管理：在 `~/.insyra/envs/` 下建立多個獨立工作空間，保存變數狀態與命令歷史
- 支援約 80+ 條自定義 DSL 命令，涵蓋資料載入/匯出、顯示/摘要、欄列操作、篩選/排序、取代/清理、統計、視覺化、資料擷取、CCL、合併等
- 支援 `.isr` 腳本檔執行
- 新增依賴：`github.com/spf13/cobra`、`github.com/ergochat/readline`

## Capabilities

### New Capabilities
- `cli-entry`: CLI 程式入口與 cobra 命令樹（root command、全域 flags、子命令註冊）
- `repl-engine`: 互動式 REPL 迴圈（readline 行編輯、prompt、信號處理、Tab 補全）
- `command-registry`: 統一命令註冊表與路由機制（CLI/REPL/腳本共用同一套 handler）
- `env-management`: 環境 CRUD（建立/刪除/列出/開啟/重命名）與狀態持久化（變數快照 + 命令歷史）
- `dsl-commands`: 全部 DSL 命令實作（資料 I/O、顯示、欄列操作、篩選排序、統計、視覺化、CCL 等）
- `script-runner`: `.isr` 腳本檔解析與執行

### Modified Capabilities

（無修改既有 capabilities）

## Impact

- **新增程式碼**：`cmd/insyra/` 入口 + `cli/` 套件（commands/、repl/、env/ 子目錄）
- **新增依賴**：`github.com/spf13/cobra`、`github.com/ergochat/readline` 加入 `go.mod`
- **安裝方式**：使用者可透過 `go install github.com/HazelnutParadise/insyra/cmd/insyra@latest` 安裝 CLI
- **既有 API**：不影響，純新增。CLI 透過 insyra 的公開 API（ReadCSV_File、DataTable、DataList 等）運作
- **檔案系統**：CLI 會在使用者 home 目錄建立 `~/.insyra/` 資料夾存放環境資料
