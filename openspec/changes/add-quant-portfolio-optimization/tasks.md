# Tasks: add-quant-portfolio-optimization

## 1. Tests first (quant/portfolio_test.go, quant/portfolio_cvxpy_test.go)

- [ ] 1.1 閉式解測試：內部最小變異數與內部目標報酬各對照 `gonum/mat` 算出的閉式解（1e-8）
- [ ] 1.2 long-only 對照 simplex 格點窮舉（步長 1e-3）；box bounds 內且和為 1；不可行界回錯；不可達目標回錯
- [ ] 1.3 `MaximumSharpe` 不劣於 50 點 frontier 最大值；`EfficientFrontier` 單調、端點、`points < 2` 回錯
- [ ] 1.4 `OptimizePortfolioMoments` 與表格入口一致；非對稱／非 PSD 拒絕；`Weight(name)` 命中與未命中
- [ ] 1.5 驗證測試：nil、少於 2 欄、觀察數不足、不可讀格子含欄名列號、界長度不符、`lo > hi`、未知 Objective；未收斂時 `Converged == false`（用極小 `MaxIterations`）
- [ ] 1.6 `portfolio_cvxpy_test.go`：`INSYRA_RUN_CVXPY=1` 才跑，隨機 long-only 與 box 問題各 20 組對照 cvxpy 目標值（1e-6）；未設定時 Skip；檔頭寫安裝指令

## 2. Implementation

- [ ] 2.1 `quant/portfolio_solver.go`：bounded-simplex 投影（bisection）、power iteration 求 L、加速投影梯度含 restart、augmented Lagrangian 處理目標報酬、可達範圍的貪婪計算、golden-section 求最大 Sharpe
- [ ] 2.2 `quant/portfolio.go`：型別、`OptimizePortfolio`（`numericSeries` 逐欄、均值與樣本共變異數）、`OptimizePortfolioMoments`（對稱與 PSD 檢查）、`EfficientFrontier`、`Weight`
- [ ] 2.3 更新 `quant/init.go` 套件 doc comment

## 3. Docs, changelog & skills

- [ ] 3.1 `Docs/quant.md`：Overview 加 **Portfolio optimization** bullet；新增「Portfolio Optimization」章節（型別、三個目標、界、解法與收斂旗標、每期 Sharpe 換算、限制與 Non-Goals）；Usage Examples 加「Long-only minimum variance and the frontier」；Error Handling 加三列
- [ ] 3.2 `Docs/README.md`、`README.md`、`README_TW.md` 的 quant 列加入投組最適化
- [ ] 3.3 `skills/insyra/SKILL.md` quant 段落加 `OptimizePortfolio` 一句
- [ ] 3.4 `CHANGELOG.md` 與 `CHANGELOG_TW.md` `## Unreleased` `### quant` 段末各加一條

## 4. Verification

- [ ] 4.1 `go test ./quant/...` 與 `go test ./...` 全綠；`golangci-lint run` 無新錯誤；`INSYRA_RUN_CVXPY=1 go test ./quant/ -run CVXPY` 在本機通過一次（venv 先 `pip install cvxpy`）
- [ ] 4.2 `openspec validate add-quant-portfolio-optimization --strict` 通過
