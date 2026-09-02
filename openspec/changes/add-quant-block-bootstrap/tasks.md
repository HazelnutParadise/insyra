# Tasks: add-quant-block-bootstrap

## 1. Tests first (quant/bootstrap_test.go)

- [x] 1.1 形狀測試：`Returns` 為 Paths × Horizon、`Equity` 為 Paths × (Horizon+1)、`Equity[p][0] == 1.0`
- [x] 1.2 連續性測試：`BlockSize == n`、`Horizon == n`、moving block 時每條 `Returns[p]` 等於原序列，`Equity[p]` 等於其複利
- [x] 1.3 複利測試：常數報酬 `r` 時 `Equity[p][t] == (1+r)^t`（容差內）
- [x] 1.4 可重現測試：同 seed 兩次逐值相等；不同 seed 的 `Returns` 不相等；固定小輸入與 seed 的 golden 值
- [x] 1.5 stationary 測試：平均區塊長度接近 `BlockSize`；跨尾端的區塊環狀接回 `returns[0]`（用可辨識的序列驗證）
- [x] 1.6 驗證測試：`Horizon <= 0`、`Paths <= 0`、`BlockSize < 1`、`BlockSize > n`、空序列、nil、含 `"n/a"`／NaN／Inf 的序列各回錯，錯誤訊息含欄位名或列號
- [x] 1.7 `PercentileBands` 測試：對隨機矩陣逐值對照 `insyra.NewDataList(col...).Percentile(p)`；呼叫者順序保留；百分位單調；空 paths、ragged paths、空 percentiles、超出 [0,100] 各回錯

## 2. Implementation (quant/bootstrap.go)

- [x] 2.1 `numericSeries(dl insyra.IDataList, label string) ([]float64, error)`：`AtomicDo` 讀取、`insyra.ToFloat64Safe` 轉換、拒絕 NaN/Inf 並指出列號
- [x] 2.2 `BootstrapConfig`、`BootstrapResult` 型別與 doc comment（seed 永遠生效、`BlockSize` 兩種語意、Equity 慣例）
- [x] 2.3 PCG 種子化（`rand.NewPCG(seed, seed^0x9E3779B97F4A7C15)`）與 quant 內的均勻整數（拒絕抽樣）、`(0,1]` 浮點縮減
- [x] 2.4 `blockBootstrapF64(returns []float64, cfg) (*BootstrapResult, error)`：驗證、moving block 與 stationary 兩種抽樣、複利累積
- [x] 2.5 `BlockBootstrap` 匯出包裝：轉換後呼叫 F64 核心
- [x] 2.6 `quantileType7`（quant 內複本）與 `PercentileBands`：驗證、每個時點排序一次、依呼叫者順序輸出
- [x] 2.7 更新 `quant/init.go` 的套件 doc comment，補上 bootstrap 一句

## 3. Docs, changelog & skills

- [x] 3.1 `Docs/quant.md`：Overview 加 bullet；新增「Probabilistic forecasting」章節（型別、兩個函式、參數、回傳、錯誤條件、統計建議）；Usage Examples 加扇形圖範例；Related Packages 提到 `DataList.Percentile` 同一 type-7
- [x] 3.2 `Docs/README.md`、`README.md`、`README_TW.md` 的 quant 列加入 block bootstrap 路徑模擬
- [x] 3.3 `skills/insyra/SKILL.md` 新增 quant 段落，示範 `BlockBootstrap` + `PercentileBands` 並指向 `Docs/quant.md`
- [x] 3.4 `CHANGELOG.md` 與 `CHANGELOG_TW.md` `## Unreleased` 各新增 `` ### `quant` `` 條目（附 issue #199 連結）

## 4. Verification

- [x] 4.1 `go test ./quant/...` 與 `go test ./...` 全綠；`golangci-lint run` 無新錯誤
- [x] 4.2 `openspec validate add-quant-block-bootstrap --strict` 通過
