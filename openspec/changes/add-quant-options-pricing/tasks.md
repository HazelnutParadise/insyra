# Tasks: add-quant-options-pricing

## 1. Tests first (quant/options_test.go)

- [ ] 1.1 Hull 範例 call 4.7594 與 put 0.8086（1e-3）；含股利率 `q` 的第二組 golden（手算並附註解）
- [ ] 1.2 Put–call parity（隨機輸入多組，1e-10）
- [ ] 1.3 五個 greeks 對中央差分（Spot、σ、T、r），容差依 spec
- [ ] 1.4 `TimeToExpiry: 0` 的內含價值與極限 greeks
- [ ] 1.5 驗證測試：`Spot`、`Strike`、`Volatility` 非正、`T < 0`、NaN／Inf、未知 `Type` 各回錯並指出欄位
- [ ] 1.6 隱含波動率：round trip（call／put × 價內／價外）、深度價外 `K/S=1.5`、上界與下界外的價格回錯、`T == 0` 回錯

## 2. Implementation (quant/options.go)

- [ ] 2.1 `OptionType`、`BSInput`、`BSResult` 與 doc comment（單位：小數利率、年、每單位 σ、每年 theta）
- [ ] 2.2 `validateBSInput`、`normPDF`、`blackScholesF64`（含 `T == 0` 分支）、`BlackScholes`
- [ ] 2.3 `impliedVolatilityF64`（界檢查 → 二分夾住 → 牛頓 → 迭代上限）、`ImpliedVolatility`
- [ ] 2.4 更新 `quant/init.go` 套件 doc comment

## 3. Docs, changelog & skills

- [ ] 3.1 `Docs/quant.md`：Overview 加 **Options** bullet；新增「Options (Black–Scholes–Merton)」章節（型別、公式、greeks 單位與換算、隱含波動率界、錯誤條件）；Usage Examples 加「從 datafetch 選擇權鏈算隱含波動率」範例；Error Handling 加兩列
- [ ] 3.2 `Docs/README.md`、`README.md`、`README_TW.md` 的 quant 列加入選擇權定價
- [ ] 3.3 `skills/insyra/SKILL.md` quant 段落加 `BlackScholes`／`ImpliedVolatility` 一句
- [ ] 3.4 `CHANGELOG.md` 與 `CHANGELOG_TW.md` `## Unreleased` `### quant` 段末各加一條

## 4. Verification

- [ ] 4.1 `go test ./quant/...` 與 `go test ./...` 全綠；`golangci-lint run` 無新錯誤
- [ ] 4.2 `openspec validate add-quant-options-pricing --strict` 通過
