# Tasks: throttle-spacing-contract

## 1. Tests
- [x] 1.1 `TestTWStockThrottle` 從第一個請求之前取的時間點量起，斷言第二個請求至少晚 `Interval` 送到 transport，不加任何寬限
- [x] 1.2 `TestWait_AllowedCallsAreNeverCloserThanTheInterval` 改用排序後第 k 個放行時間至少晚起點 (k−1)·interval 的界限，拿掉半個 interval 的寬限
- [x] 1.3 證明兩個測試在節流被拿掉時會失敗，並以 `-count=200` 跑過

## 2. Wording
- [x] 2.1 限速器型別註解、`TWStockConfig`／`YFinanceConfig`／`TWGeocodingConfig` 的 `Interval` 註解、`Docs/datafetch.md` 三處 `Interval` 說明改成實際的保證
- [x] 2.2 `openspec validate throttle-spacing-contract --strict`、`go test ./datafetch/...`、`golangci-lint run`
