# quant-options Specification

## Purpose
`quant` 的歐式選擇權定價：含連續股利率的 Black–Scholes–Merton 價格與五個 greeks（vega 每單位波動率、theta 每年），到期時回內含價值；隱含波動率先檢查無套利界、二分法夾住再以 vega 做牛頓修正，有迭代上限。

## Requirements
### Requirement: Black–Scholes–Merton price and greeks

`quant` SHALL 提供 `OptionType`（`OptionCall`、`OptionPut`）、`BSInput{Spot, Strike, Rate, DividendYield, Volatility, TimeToExpiry float64; Type OptionType}` 與 `BlackScholes(in BSInput) (*BSResult, error)`。`BSResult` SHALL 含 `Price`、`Delta`、`Gamma`、`Vega`、`Theta`、`Rho`。公式 SHALL 為含連續股利率的 Black–Scholes–Merton：`d1 = (ln(S/K) + (r − q + σ²/2)T) / (σ√T)`、`d2 = d1 − σ√T`；call `S·e^{−qT}·N(d1) − K·e^{−rT}·N(d2)`，put `K·e^{−rT}·N(−d2) − S·e^{−qT}·N(−d1)`。`Vega` SHALL 為對 σ 每單位（非每 1%）的偏導，`Theta` SHALL 為對 T 每年的 `−∂V/∂T`。`TimeToExpiry == 0` 時 `Price` SHALL 為內含價值，`Delta` 為 0 或 ±1，其餘 greeks 為 0。`Spot`、`Strike`、`Volatility` 非正、`TimeToExpiry` 為負、任一輸入非有限、未知 `Type` SHALL 回錯。

#### Scenario: Hull textbook example

- **WHEN** `S=42, K=40, r=0.10, q=0, σ=0.20, T=0.5`
- **THEN** call `Price` 為 4.7594（1e-3 內），put `Price` 為 0.8086（1e-3 內）

#### Scenario: Put–call parity holds

- **WHEN** 任意合法輸入
- **THEN** `call − put == S·e^{−qT} − K·e^{−rT}`（1e-10 內）

#### Scenario: Greeks match finite differences

- **WHEN** 任意合法輸入，以 `h = 1e-5` 對 `Spot` 中央差分
- **THEN** `Delta` 與差分在 1e-6 內，`Gamma` 與二階差分在 1e-4 內；對 σ、T、r 的差分分別驗證 `Vega`、`Theta`、`Rho`

#### Scenario: Expiry returns intrinsic value

- **WHEN** `TimeToExpiry: 0`，`S=105, K=100`
- **THEN** call `Price == 5`、`Delta == 1`、其餘 greeks 為 0；put `Price == 0`、`Delta == 0`

#### Scenario: Invalid input is refused

- **WHEN** `Volatility: 0` 或 `Spot: −1` 或 `TimeToExpiry: −0.1`
- **THEN** 回傳指出該欄位的錯誤

### Requirement: Implied volatility

`quant` SHALL 提供 `ImpliedVolatility(price float64, in BSInput) (float64, error)`，忽略 `in.Volatility`，回傳使 `BlackScholes(in).Price == price` 的 σ。求解 SHALL 先在 `[1e-6, 10]` 以二分法夾住，再以牛頓法（vega 為導數）收斂到價格誤差 `1e-10` 或 σ 變動 `1e-12`，上限 200 次迭代。`price` 低於內含價值折現下界、或高於無套利上界（call 為 `S·e^{−qT}`，put 為 `K·e^{−rT}`）、或 `TimeToExpiry == 0` SHALL 回錯指出違反的界。

#### Scenario: Round trip recovers volatility

- **WHEN** 以 `σ = 0.35` 定價後把 `Price` 交給 `ImpliedVolatility`
- **THEN** 回傳 0.35（1e-8 內），對 call 與 put、價內與價外各一組

#### Scenario: Price outside bounds is refused

- **WHEN** call `price` 大於 `S·e^{−qT}`，或小於 `max(S·e^{−qT} − K·e^{−rT}, 0)`
- **THEN** 回傳提及上界或下界的錯誤

#### Scenario: Deep out-of-the-money converges

- **WHEN** `S=100, K=150, T=0.25, r=0.02`、`σ=0.6` 定價的 call
- **THEN** 隱含波動率回到 0.6（1e-6 內）

