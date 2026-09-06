# plot-png-export-policy Specification

## Purpose
`plot.SavePNG` 的資料外送政策：PNG 匯出只在本機渲染，線上退回服務是逐次明確選擇的行為，不是預設。

## Requirements
### Requirement: PNG export never leaves the host without opt-in

`plot.SavePNG(chart, path)` 在本機渲染失敗時 SHALL 回傳錯誤，SHALL NOT 對線上服務發出任何請求；只有呼叫端明確傳入 `true` 時才使用線上服務。錯誤訊息 SHALL 說明如何選擇線上服務。

#### Scenario: Local render fails without opt-in
- **WHEN** 本機無法渲染且未傳入 `true`
- **THEN** 回傳錯誤且無網路請求

