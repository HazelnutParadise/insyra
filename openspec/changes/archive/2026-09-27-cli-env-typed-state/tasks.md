# Tasks: cli-env-typed-state

## 1. Scaler 的 JSON 編碼（根套件）
- [x] 1.1 先寫失敗測試：四種 scaler 擬合後 `json.Marshal` 再 `json.Unmarshal` 到同型別的新值，`Params()`、`Transform`、`InverseTransform`、`TransformDataList` 逐位元相同；min-max 用 [-1, 1] 範圍，一欄以欄名、一欄以欄字母擬合；只含缺值的欄 NaN 參數讀回仍是 NaN；把 standard 的 JSON 讀進 `RobustScaler` 回錯誤且 receiver 不變；未擬合的 scaler 讀回仍未擬合。以 `go test -run 'Scaler.*JSON' .` 確認測試先失敗
- [x] 1.2 在 `datatable_scale.go` 實作 `*scaler` 的 `MarshalJSON`、四個具體型別的 `UnmarshalJSON`，並把 `computeColumn` 的仿射係數改由參數推導的共用函式計算，讓 1.1 通過
- [x] 1.3 `Docs/DataTable.md` 的 scaler 段落與 `skills/insyra/SKILL.md` 的 fit 原則補上 JSON 存讀；`CHANGELOG.md`／`CHANGELOG_TW.md` 的 `### Core` 各加一條

## 2. 帶型別的格子與欄編碼（`cli/env`）
- [x] 2.1 先寫失敗測試：每一種可保存的格子型別（含 NaN、±Inf、-0、`int64` 極值、`uint64` 最大值、`time.Time` 的 UTC／Local／固定時區、`[]byte`、scale 保留的 `decimal.Decimal`、`json.Number`、`ReadJSON` 形狀的巢狀 map 與陣列）經欄編碼、`json` 寫出、`UseNumber` 讀回、欄解碼後型別與值相同；同型欄寫 `type`，混型欄寫 `types`；全 nil 欄兩者皆無；不支援的型別（struct、含 `float64` 葉的巢狀 map、負 scale 的 decimal、年份超出 0–9999 的時間）回錯誤並指出型別。以 `go test ./cli/env/` 確認先失敗
- [x] 2.2 新增 `cli/env/cell_codec.go` 實作欄編碼與解碼，讓 2.1 通過

## 3. 變數編碼、存讀與匯入（`cli/env`）
- [x] 3.1 先寫失敗測試：欄位順序非字母序、混型的 DataTable（含列名、表名、`time.Time`、整數值的 `float64`、NaN、nil）、DataList、`int`／`float64`／`int64`／`time.Time` 純量、`[]int`／`[]float64`／`[]bool`、四種 scaler、`*stats.HierarchicalResult` 經 `SaveState`／`RestoreVariables` 後型別與值相同；typed nil 表與清單不 panic；迴歸結果與含 struct 格子的表不寫入並回 `*UnsavedVariablesError`（依名稱排序，含型別與原因），其他變數照寫；`LoadState` 回傳儲存形式；舊格式的 JSON 字串表、`$float` 欄位陣列表、DataList、純量照舊還原；含 NaN 純量的環境能 `Export`，大於 2^53 的 `int64` 經 `Export`／`Import` 不變。以 `go test ./cli/env/` 確認先失敗
- [x] 3.2 實作新格式的變數編碼與解碼、`UnsavedVariablesError`，改寫 `SaveState`／`LoadState`／`RestoreVariables`，舊格式讀取維持原邏輯，`Import` 改用 `UseNumber`；把只釘住舊寫入格式的 `TestSaveStateKeepsLegacyLayoutWithoutSpecialFloats` 與 `TestLoadState_CoercesScalarsBeforeReturning` 改成對應新規格的測試，讓 3.1 與既有 `cli/env` 測試通過
- [x] 3.3 以 10 萬列 × 10 欄的表量測新舊格式的存、讀時間與檔案大小，記在本檔 3.3 之下
  - 2026-09-27 在 M3 上量測，欄型別為 `float64`、`int64`、`string` 輪流，各跑三次：舊格式存 276–303 ms、讀 374–383 ms、檔案 24,563,621 bytes。新格式存 248–314 ms、讀 216–308 ms、檔案 22,964,842 bytes。審查後修正完成、每個字串多了 UTF-8 檢查之後再量一次：存 191–198 ms、讀 169–172 ms，檔案大小不變

## 4. CLI 的儲存與警告
- [x] 4.1 先寫失敗測試：以 `NewRootCommand` 分開執行一次性命令，`load` 一個欄位順序為 `zeta,alpha,when` 的 CSV、`parsedates`，再由新的命令還原後欄位順序、每欄型別（`float64`、`string`、`time.Time`）與列數不變，接著 `resample` 能用 `time.Time` 欄；`scale fit` 與 `scale transform` 分開執行成功且結果與同程序相同；`hclust` 與 `cutree` 分開執行成功；`SaveEnvState` 遇到迴歸結果時輸出一行警告、回傳 nil，同一個 ctx 再存一次不重複警告，變數換成另一種不可保存的型別時再警告；`DSLSession.Execute` 在迴歸結果之後回傳 nil。以 `go test ./cli/...` 確認先失敗
- [x] 4.2 在 `cli/commands` 新增 `SaveEnvState`，`cli/root.go`、`cli/repl/repl.go`（每行與結束時）、`cli/repl/api.go` 改用它，讓 4.1 通過
- [x] 4.3 `Docs/cli-dsl.md` 的 Environment Model 與 scale 段落、`skills/use-insyra-cli/SKILL.md` 的 session 說明改寫成新行為；`CHANGELOG.md`／`CHANGELOG_TW.md` 的 `### CLI` 各加一條

## 5. 整合驗證
- [x] 5.1 以本地建置的 CLI 與暫存 HOME 重跑提案中的重現步驟，確認欄序、型別、scaler、`cutree` 都正常，迴歸結果印出警告
- [x] 5.2 `go test ./...`、`golangci-lint run`、`openspec validate cli-env-typed-state --strict` 通過
- [x] 5.3 `AGENTS.md` Follow-ups 記下「整個 `state.json` 讀不到時，下一個一次性命令會把它蓋掉」

## 6. 審查後修正
- [x] 6.1 `SaveState` 恢復「只在檔案沒寫成時回錯誤」的契約，新增 `Manager.SaveVariables` 回報存不了的變數並取代 `UnsavedVariablesError`，`SaveEnvState` 改用它，警告文字加上「或開啟另一個環境」；以 `go test ./cli/...` 的 `TestSaveVariablesReportsUnsavedVariables`、`TestSaveStateIgnoresUnsavedVariables` 與 `TestSaveEnvState*` 驗證
- [x] 6.2 `LoadState` 繼續回傳已轉型的頂層純量（新格式照儲存型別、舊格式照舊規則），`RestoreVariables` 與 `Export` 改讀未轉型的儲存形式，新格式解碼失敗的變數原樣保留並寫回；以 `TestLoadState_TypesScalars`、`TestLoadStateTypesLegacyScalars`、`TestRestoreKeepsUndecodableVariable`、`TestExportWithNaNScalar` 驗證
- [x] 6.3 格子編碼支援 `time.Duration`，非法 UTF-8 字串逐位元組保存，偏移帶秒的時間改存 UTC 以保留時間點，巢狀值限 64 層；以 `cli/env` 的新單元測試驗證
- [x] 6.4 端到端測試加上 CCL 日期相減產生的 `time.Duration` 欄，並刪掉孤兒註解；以 `go test -run OneShot ./cli/` 驗證
- [x] 6.5 `Docs/cli-dsl.md`、兩份 CHANGELOG 補上 `time.Duration`、`SaveVariables` 與 `env open` 的說明；`AGENTS.md` Follow-ups 記下兩個一次性命令同時存檔會遺失其中一方的變數
