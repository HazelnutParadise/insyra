## 1. Project Skeleton & Dependencies

- [x] 1.1 新增 `github.com/spf13/cobra` 和 `github.com/ergochat/readline` 至 go.mod
- [x] 1.2 建立 `cmd/insyra/main.go` 程式入口（cobra root command 初始化）
- [x] 1.3 建立 `cli/root.go`（root command 定義、全域 flags：`--env`、`--no-color`、`--log-level`）
- [x] 1.4 建立 `cli/commands/registry.go`（CommandHandler 介面、ExecContext 結構、Registry map、Dispatch 函式）

## 2. Environment Management

- [x] 2.1 建立 `cli/env/manager.go`（Create / Delete / List / Open / Rename / Info 環境 CRUD，操作 `~/.insyra/envs/` 目錄）
- [x] 2.2 建立 `cli/env/state.go`（SaveState / LoadState：變數快照 JSON 序列化與反序列化）
- [x] 2.3 建立 `cli/env/config.go`（全域 config.json 讀寫、環境層級 config 覆寫）
- [x] 2.4 實作首次使用自動建立 `default` 環境
- [x] 2.5 建立 `cli/commands/env.go`（`env create|list|open|delete|rename|info` 子命令，路由到 manager）
- [x] 2.6 建立 `cli/commands/config.go`（`config [key] [value]` 命令）

## 3. REPL Engine

- [x] 3.1 建立 `cli/repl/repl.go`（REPL 主迴圈：readline 初始化、prompt 顯示、迴圈讀取輸入、路由到 Registry.Dispatch）
- [x] 3.2 實作 REPL 啟動時自動 LoadState、退出時自動 SaveState
- [x] 3.3 實作空行 / `#` 註解跳過、Ctrl+C 取消輸入、Ctrl+D 退出
- [x] 3.4 接上 root command：無子命令時進入 REPL（使用預設或 `--env` 指定的環境）
- [x] 3.5 實作 `env open <name>` 進入 REPL 的流程

## 4. Basic Commands

- [x] 4.1 建立 `cli/commands/version.go`（`version` 命令，顯示 insyra.Version 和 VersionName）
- [x] 4.2 建立 `cli/commands/help.go`（`help [command]` 命令，列出所有命令或特定命令說明）
- [x] 4.3 建立 `cli/commands/exit.go`（`exit` / `quit` 命令）
- [x] 4.4 建立 `cli/commands/clear.go`（`clear` 命令）
- [x] 4.5 建立 `cli/commands/history.go`（`history` 命令，顯示命令歷史）
- [x] 4.6 建立 `cli/commands/vars.go`（`vars` 命令，列出所有變數名稱、型別、摘要）
- [x] 4.7 建立 `cli/commands/drop.go`（`drop <var>` 命令）
- [x] 4.8 建立 `cli/commands/rename.go`（`rename <var> <new>` 命令）

## 5. Data I/O Commands

- [x] 5.1 建立 `cli/commands/load.go`（`load <file> [sheet <name>] [as <var>]`，依副檔名路由 ReadCSV_File / ReadExcelSheet / ReadJSON_File / parquet.Read）
- [x] 5.2 建立 `cli/commands/save.go`（`save <var> <file>`，依副檔名路由 ToCSV / ToJSON / parquet.Write）
- [x] 5.3 建立 `cli/commands/read.go`（`read <file>` 快速預覽，不存變數）
- [x] 5.4 建立 `cli/commands/convert.go`（`convert <input> <output>` 格式轉換）
- [x] 5.5 建立 `cli/commands/newdl.go`（`newdl <values...> [as <var>]` 手動建立 DataList）
- [x] 5.6 建立 `cli/commands/newdt.go`（`newdt <dl_vars...> [as <var>]` 從 DataList 組建 DataTable）

## 6. Display & Summary Commands

- [x] 6.1 建立 `cli/commands/show.go`（`show <var> [N] [M]`，支援正數/負數/範圍/`_`，設定 DisableFlagParsing 處理負數）
- [x] 6.2 建立 `cli/commands/types.go`（`types <var>` 顯示型別）
- [x] 6.3 建立 `cli/commands/summary.go`（`summary <var>` 統計摘要）
- [x] 6.4 建立 `cli/commands/shape.go`（`shape <var>` 顯示維度）
- [x] 6.5 建立 `cli/commands/cols.go`（`cols <var>` 列出欄名）
- [x] 6.6 建立 `cli/commands/rows_cmd.go`（`rows <var>` 列出列名）

## 7. Column & Row Operation Commands

- [x] 7.1 建立 `cli/commands/col.go`（`col <var> <name|index> [as <var>]` 取出欄）
- [x] 7.2 建立 `cli/commands/row.go`（`row <var> <index|name> [as <var>]` 取出列）
- [x] 7.3 建立 `cli/commands/get.go`（`get <var> <row> <col>` 取元素）
- [x] 7.4 建立 `cli/commands/set.go`（`set <var> <row> <col> <value>` 設元素）
- [x] 7.5 建立 `cli/commands/addcol.go`（`addcol <var> <values...>` 新增欄）
- [x] 7.6 建立 `cli/commands/addrow.go`（`addrow <var> <values...>` 新增列）
- [x] 7.7 建立 `cli/commands/dropcol.go`（`dropcol <var> <name|index...>` 刪除欄）
- [x] 7.8 建立 `cli/commands/droprow.go`（`droprow <var> <index|name...>` 刪除列）
- [x] 7.9 建立 `cli/commands/setcolnames.go`（`setcolnames <var> <names...>`）
- [x] 7.10 建立 `cli/commands/setrownames.go`（`setrownames <var> <names...>`）
- [x] 7.11 建立 `cli/commands/swap.go`（`swap <var> col|row <a> <b>`）
- [x] 7.12 建立 `cli/commands/transpose.go`（`transpose <var> [as <var>]`）

## 8. Filter, Sort & Search Commands

- [x] 8.1 建立 `cli/commands/filter.go`（`filter <var> <expr> [as <var>]` CCL 條件篩選）
- [x] 8.2 建立 `cli/commands/sort.go`（`sort <var> <col> [asc|desc]`）
- [x] 8.3 建立 `cli/commands/find.go`（`find <var> <value>` 搜尋）
- [x] 8.4 建立 `cli/commands/sample.go`（`sample <var> <n> [as <var>]` 隨機抽樣）
- [x] 8.5 建立 `cli/commands/clone.go`（`clone <var> [as <var>]` 深複製）

## 9. Replace & Clean Commands

- [x] 9.1 建立 `cli/commands/replace.go`（`replace <var> <old> <new>` / `replace <var> nan|nil <value>`）
- [x] 9.2 建立 `cli/commands/clean.go`（`clean <var> nan|nil|strings|outliers [<stddev>]`）

## 10. DataList Statistics Commands

- [x] 10.1 建立 `cli/commands/stats_dl.go`（`sum`/`mean`/`median`/`mode`/`stdev`/`var`/`min`/`max`/`range` 命令，各自路由到對應 DataList 方法）
- [x] 10.2 建立 `cli/commands/stats_dl_extra.go`（`quartile <var> <q>`/`iqr`/`percentile <var> <p>`/`count`/`counter` 命令）

## 11. DataList Transformation Commands

- [x] 11.1 建立 `cli/commands/transform.go`（`rank`/`normalize`/`standardize`/`reverse`/`upper`/`lower`/`capitalize`/`parsenums`/`parsestrings` 命令）

## 12. Time Series Commands

- [x] 12.1 建立 `cli/commands/timeseries.go`（`movavg <var> <window> [as <var>]`/`expsmooth <var> <alpha> [as <var>]`/`diff <var> [as <var>]`/`fillnan <var> mean`）

## 13. Advanced Statistics Commands

- [ ] 13.1 建立 `cli/commands/stats_adv.go`（`corr`/`corrmatrix`/`cov`/`skewness`/`kurtosis` 命令）
- [ ] 13.2 建立 `cli/commands/regression.go`（`regression <type> <y> <x...>` — linear/poly/exp/log）
- [ ] 13.3 建立 `cli/commands/hypothesis.go`（`ttest`/`ztest`/`anova`/`ftest`/`chisq` 命令）
- [ ] 13.4 建立 `cli/commands/pca.go`（`pca <var> <n>`）

## 14. CCL & Merge Commands

- [ ] 14.1 建立 `cli/commands/ccl.go`（`ccl <var> <expression>` / `addcolccl <var> <name> <expr>`）
- [ ] 14.2 建立 `cli/commands/merge.go`（`merge <var1> <var2> <direction> <mode> [on <cols>] [as <var>]`）

## 15. Visualization & Data Fetch Commands

- [ ] 15.1 建立 `cli/commands/plot.go`（`plot <type> <var> [options...] [save <file>]`）
- [ ] 15.2 建立 `cli/commands/fetch.go`（`fetch yahoo <ticker> <method> [params...] [as <var>]`）

## 16. Script Runner

- [ ] 16.1 建立 `cli/commands/run.go`（`run <script.isr>` 逐行讀取 .isr 檔 → Registry.Dispatch）
- [ ] 16.2 實作 `#` 註解跳過、空行跳過、錯誤顯示行號

## 17. Tab Completion

- [ ] 17.1 建立 `cli/repl/completer.go`（命令名補全：從 Registry 取、變數名補全：從 ExecContext.Vars 取、檔案路徑補全：load/save/read/run 後觸發）

## 18. Polish & Testing

- [ ] 18.1 CLI 有狀態命令自動讀寫環境 state（DisableFlagParsing 處理負數參數）
- [ ] 18.2 彩色輸出與 `--no-color` flag 實作
- [ ] 18.3 錯誤訊息美化（友善的格式與顏色）
- [x] 18.4 建立 `cli/commands/registry_test.go`（命令 handler 單元測試）
- [x] 18.5 建立 `cli/env/manager_test.go`（環境 CRUD 測試）
- [x] 18.6 建立 `cli/repl/repl_test.go`（REPL 整合測試）
- [ ] 18.7 手動端對端測試：`go run ./cmd/insyra` 進入 REPL → load/show/filter/save 完整流程
- [ ] 18.8 跨平台驗證：Windows `%USERPROFILE%\.insyra\` 路徑正確解析
