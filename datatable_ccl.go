package insyra

import (
	"time"
)

func (dt *DataTable) AddColUsingCCL(newColName, ccl string) *DataTable {
	// 添加 recover 以防止程序崩潰
	defer func() {
		if r := recover(); r != nil {
			LogWarning("DataTable", "AddColUsingCCL", "Panic recovered: %v", r)
		}
	}()

	// 添加超時保護（使用 channel）
	resultChan := make(chan []any, 1)
	errorChan := make(chan error, 1)

	// 重設遞迴深度和調用深度計數器
	resetCCLEvalDepth()
	resetCCLFuncCallDepth()

	// 使用 goroutine 運行 CCL 表達式計算
	go func() {
		result, err := applyCCLOnDataTable(dt, ccl)
		if err != nil {
			errorChan <- err
			return
		}
		resultChan <- result
	}()
	// 等待結果或超時 (增加到15秒)
	timeoutDuration := 15 * time.Second // 增加超時時間
	const timeoutMsg = "CCL evaluation timed out after 15 seconds"

	// 優先記錄表達式開始評估的時間
	startTime := time.Now()
	LogDebug("DataTable", "AddColUsingCCL", "Starting CCL evaluation for %s: %s", newColName, ccl)

	select {
	case result := <-resultChan:
		elapsed := time.Since(startTime)
		LogDebug("DataTable", "AddColUsingCCL", "CCL evaluation completed in %v", elapsed)
		dt.AppendCols(NewDataList(result...).SetName(newColName))
		return dt
	case err := <-errorChan:
		elapsed := time.Since(startTime)
		LogWarning("DataTable", "AddColUsingCCL", "Failed to apply CCL on DataTable after %v: %v", elapsed, err)
		return dt
	case <-time.After(timeoutDuration):
		LogWarning("DataTable", "AddColUsingCCL", timeoutMsg)
		return dt
	}
}
