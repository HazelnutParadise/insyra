## ADDED Requirements

### Requirement: Commands that replace or remove data say so
會取代或刪除既有資料的指令 SHALL 在 `insyra help <command>` 的 `Forms` 與 `Docs/cli-dsl.md` 說明這件事，至少包含：`env delete` 刪除整個環境；`env clear` 清空變數，沒有 `--keep-history` 時也清空歷史；`env open` 以開啟的環境取代工作階段的變數，無法存進環境的變數因此消失；`env export` 取代已存在的輸出檔；`env import --force` 取代目標環境；`save` 寫 CSV、JSON 或 Parquet 時取代已存在的檔案；`plot` 取代已存在的輸出檔，沒有 `save` 時寫到工作目錄的 `<type>.html`。

#### Scenario: Help says save replaces a file
- **WHEN** 使用者執行 `insyra help save`
- **THEN** 輸出說明存到已存在的 CSV、JSON 或 Parquet 檔會取代它

#### Scenario: Help says where plot writes
- **WHEN** 使用者執行 `insyra help plot`
- **THEN** 輸出說明沒有 `save` 時寫到工作目錄的 `<type>.html`，並取代同名檔案
