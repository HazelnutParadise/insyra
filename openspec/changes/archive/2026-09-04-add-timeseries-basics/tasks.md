# Tasks: add-timeseries-basics

## 1. Fixtures and tests first

- [x] 1.1 `testdata/gen_window_fixtures.py` 新增 `ewm_mean`、`ewm_var`、`ewm_std` 案例：alpha／span／halflife 各一組、`adjust` 真假、`bias` 真假、含 nil 的輸入、`min_periods`；重新產生 `testdata/window_fixtures.json`（fixture 結構加 `alpha`/`span`/`halflife`/`adjust`/`bias` 欄位）
- [x] 1.2 `datalist_window_crosslang_test.go`：`runFixtureOp` 加入三個 ewm op 並逐值比對（1e-9，nil 對 nil）
- [x] 1.3 `datalist_ewm_test.go`：衰減參數缺失或重複時 warning 與空結果；`Adjust:false, Alpha:0.5` 的 `[1,2,3] → [1,1.5,2.25]`；`MinObs` 抑制前段輸出；`DataTable.EWMCol` 對欄名與字母索引皆可解析
- [x] 1.4 `datalist_window_test.go`：`Rolling.Cov` 全視窗等於 `stats.Covariance`；`Rolling.Beta` 全視窗等於 `quant.Beta`；縮放 benchmark 的滾動 beta 為常數；平坦 benchmark 視窗 `Beta` 為 nil、`Cov` 為 0；nil 配對略過；`other == nil` 回空結果
- [x] 1.5 `datatable_resample_test.go`：日線→月線 OHLCV；週一到週日分桶與週日標籤；跳過整月不補列；非 `time.Time` 格子回錯並指出列號；`As` 命名；輸入亂序結果相同；混合時區各自截斷

## 2. Implementation

- [x] 2.1 `datalist_ewm.go`：`EWMOptions`、`EWMDataList`、`DataList.EWM`、`Mean/Var/Std`（pandas adjust／bias 語意、nil 不重置累積、`MinObs`）
- [x] 2.2 `datalist_window.go`：抽出 `pairWindow` 私有 helper 供 `Corr/Cov/Beta` 共用；新增 `Cov`、`Beta`
- [x] 2.3 `datatable_window.go`：新增 `EWMCol`
- [x] 2.4 `datatable_resample.go`：`ResampleFreq`、`ResampleAgg`、`DataTable.Resample`（截斷成期間鍵 → `GroupBy` 聚合 → 期間末日標籤 → 遞增排序）
- [x] 2.5 `interfaces.go`：`IDataList` 加 `EWM`；`IDataTable` 加 `EWMCol`、`Resample`

## 3. Docs, changelog & skills

- [x] 3.1 `Docs/DataList.md`：window 章節新增 EWM（選項表、三個 reducer、pandas 對應）與 `Rolling.Cov`/`Beta`
- [x] 3.2 `Docs/DataTable.md`：新增 `EWMCol` 與 `Resample`（頻率、標籤慣例、OHLCV 範例、錯誤條件）
- [x] 3.3 `skills/insyra/SKILL.md` 與 `skills/insyra/references/` 中列出 window 函式的位置加上 EWM、Rolling.Cov/Beta、Resample
- [x] 3.4 `CHANGELOG.md` 與 `CHANGELOG_TW.md` `## Unreleased` `### Core` 段末各加一條（含介面新增方法的提醒）

## 4. Verification

- [x] 4.1 `go test ./...` 全綠；`golangci-lint run` 無新錯誤；fixture 由 `python3 testdata/gen_window_fixtures.py` 重新產生且 diff 只增不改既有案例
- [x] 4.2 `openspec validate add-timeseries-basics --strict` 通過
