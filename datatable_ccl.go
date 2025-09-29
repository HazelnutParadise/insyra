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

	// 重設遞歸深度和調用深度計數器
	resetCCLEvalDepth()
	resetCCLFuncCallDepth()

	resultDtChan := make(chan *DataTable, 1)

	dt.AtomicDo(func(dt *DataTable) {
		// 優先記錄表達式開始評估的時間
		startTime := time.Now()
		LogDebug("DataTable", "AddColUsingCCL", "Starting CCL evaluation for %s: %s", newColName, ccl)

		result, err := applyCCLOnDataTable(dt, ccl)
		if err != nil {
			elapsed := time.Since(startTime)
			LogWarning("DataTable", "AddColUsingCCL", "Failed to apply CCL on DataTable after %v: %v", elapsed, err)
		} else {
			elapsed := time.Since(startTime)
			LogDebug("DataTable", "AddColUsingCCL", "CCL evaluation completed in %v", elapsed)
			dt.AppendCols(NewDataList(result...).SetName(newColName))
		}
		resultDtChan <- dt
	})
	return <-resultDtChan
}
