## Context

insyra 是一個 Go 語言的資料分析 library，提供 DataList（類似 Series）和 DataTable（類似 DataFrame）兩大核心結構，以及統計分析、視覺化、資料擷取等子套件。目前只能嵌入 Go 程式碼使用，沒有 CLI 或互動式介面。

本設計新增一個 CLI 工具，讓使用者可以：
- 在終端直接執行資料分析指令（`insyra load data.csv as t1`）
- 進入 REPL 互動式操作（直接輸入 `insyra`）
- 管理多個持久化環境（`~/.insyra/envs/`）
- 執行 `.isr` 腳本檔

既有的 `isr` 套件已提供高階 API 封裝，`Show()`/`ShowRange()` 已處理終端寬度偵測，`ToJSON`/`ReadJSON` 提供序列化能力——這些都可直接為 CLI 所用。

## Goals / Non-Goals

**Goals:**
- CLI 與 REPL 命令語法完全統一（使用者只需學一套語法）
- 提供約 80+ 條 DSL 命令，涵蓋 insyra 絕大部分功能
- 環境管理支援多個獨立工作空間，保存變數狀態與歷史記錄
- 支援 Tab 補全、命令歷史、彩色輸出
- 支援腳本檔批次執行
- `go install` 即可安裝

**Non-Goals:**
- 不實作圖形化介面（GUI / TUI dashboard）
- 不取代 Go library API（CLI 是上層消費者，不修改核心 API）
- 不實作 Go 語法 REPL（不是 Go playground）
- 不做遠端 / 多人協作環境
- 第一版不支援 SQL 資料來源的 CLI 操作（需 GORM driver 設定，複雜度過高）

## Decisions

### 1. CLI 框架：cobra

**選擇**：`github.com/spf13/cobra`

**理由**：Go 生態最成熟的 CLI 框架，支援子命令、flag、補全生成。cobra 的子命令結構天然對應我們的 DSL 命令（如 `insyra load`、`insyra show`）。

**替代方案**：
- `urfave/cli`：功能足夠但社群較小，子命令模式不如 cobra 直觀
- 標準 `flag`：不支援子命令，需大量手動解析
- 手寫解析器：開發成本高，無自動補全

### 2. CLI 與 REPL 命令統一：共用 Command Registry

**選擇**：所有命令邏輯統一寫在 `cli/commands/` 中的 handler，cobra 子命令和 REPL 都路由到同一個 handler。

**結構**：
```
CommandHandler {
    Name        string
    Aliases     []string
    Usage       string
    Description string
    Run         func(ctx *ExecContext, args []string) error
}

ExecContext {
    Vars    map[string]any    // 變數名 → *DataTable / *DataList
    EnvName string
    EnvPath string
    Output  io.Writer
}
```

**路由機制**：
- Cobra 端：遍歷 Registry，為每個 handler 生成 `cobra.Command`
- REPL 端：讀取輸入，拆 tokens，以第一個 token 查 Registry
- 腳本端：逐行讀取 `.isr` 檔，同樣路由到 Registry

**理由**：避免 CLI 和 REPL 各寫一套邏輯，減少維護成本，保證語法一致性。

### 3. DSL 語法風格：`verb subject args`

**選擇**：命令格式為 `verb subject [args] [as alias]`，如 `show t1 10`、`filter t1 "revenue > 1000" as t_high`。

**理由**：
- 直接映射為 cobra 子命令（verb = 子命令名，subject = 第一個 arg）
- 對非程式設計師友善，語義清晰
- 支援 `as <var>` 語法將結果存入指定變數名

**替代方案**：
- 物件導向 `t1.show 10`：無法直接映射 cobra 子命令，需客製解析器
- SQL 語法 `SELECT * FROM t1`：學習成本高，與 insyra API 不對稱

### 4. REPL 行編輯：ergochat/readline

**選擇**：`github.com/ergochat/readline`

**理由**：純 Go 實作、活躍維護、支援 Windows、提供行編輯 / 歷史 / Tab 補全。

**替代方案**：
- `chzyer/readline`：已停止維護
- `peterh/liner`：功能較少
- `charmbracelet/bubbletea`：TUI 框架，對 REPL 用途過重

### 5. 環境存放位置：`~/.insyra/envs/`

**選擇**：全域 home 目錄 `~/.insyra/envs/<name>/`。

**結構**：
```
~/.insyra/
  config.json              # 全域設定（預設環境名等）
  envs/
    default/
      state.json           # 變數快照（DataTable/DataList JSON 序列化）
      history.txt          # 命令歷史
      config.json          # 環境層級設定覆寫
```

**理由**：環境是使用者級的工作空間，不綁定特定專案目錄。使用者可能在不同路徑下分析同一組資料。

**替代方案**：
- 專案級 `.insyra/`：類似 `.git`，但 CLI 用途不同於版本控制，不適合
- 可設定路徑：增加複雜度，第一版用固定路徑即可

### 6. 狀態序列化：JSON

**選擇**：變數快照以 JSON 格式存於 `state.json`，利用 insyra 既有的 `ToJSON_String()` / `ReadJSON()` 方法。

**格式**：
```json
{
  "variables": {
    "t1": { "type": "DataTable", "data": {...}, "colNames": [...], "rowNames": [...] },
    "d1": { "type": "DataList", "data": [...] }
  },
  "lastAccess": "2026-02-28T10:00:00Z"
}
```

**理由**：insyra 已有完整的 JSON 序列化 / 反序列化支援，無需引入額外格式。

### 7. 有狀態 CLI 命令

**選擇**：CLI 模式下有狀態命令（如 `insyra show t1`）自動讀取當前環境的 `state.json`，執行後自動存回。用 `--env` flag 指定環境。

**理由**：讓使用者不進 REPL 也能用一行行 CLI 命令串接操作（如在 shell script 中使用）。

### 8. 隱式變數 `$result`

**選擇**：未使用 `as <var>` 的運算結果自動存入 `$result`。

**理由**：方便快速探索，不用每次都命名。`$result` 可被後續命令引用。

## Risks / Trade-offs

- **[cobra 負數參數衝突]** → `show t1 -5` 中 `-5` 可能被 cobra 解讀為 flag。**緩解**：對 `show` 等命令設定 `DisableFlagParsing: true`，或用 `--` 分隔。
- **[狀態序列化效能]** → 大型 DataTable（數百萬列）序列化 / 反序列化可能緩慢。**緩解**：第一版先以 JSON 實作，未來可換用 Parquet 或 binary 格式。加入大小警告。
- **[依賴膨脹]** → cobra 和 readline 引入新依賴。**緩解**：兩者都是純 Go、無 CGO，依賴圖乾淨。
- **[DSL 語法擴展性]** → 固定 verb-subject 語法可能在未來遇到表達力不足的場景。**緩解**：保留 `ccl` 命令作為 escape hatch，複雜計算可用 CCL 表達式。
- **[跨平台路徑]** → Windows 下 `~` 需轉為 `%USERPROFILE%`。**緩解**：使用 `os.UserHomeDir()` 標準函式。
