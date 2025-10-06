package mkt

import (
	"time"

	"github.com/HazelnutParadise/Go-Utils/conv"
	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/internal/utils"
)

type CAIConfig struct {
	CustomerIDColIndex string    // The column index(A, B, C, ...) of customer ID in the data table
	CustomerIDColName  string    // The column name of customer ID in the data table (if both index and name are provided, index takes precedence)
	TradingDayColIndex string    // The column index(A, B, C, ...) of trading day in the data table
	TradingDayColName  string    // The column name of trading day in the data table (if both index and name are provided, index takes precedence)
	DateFormat         string    // The format of the date string (e.g., "YYYY-MM-DD", "DD/MM/YYYY", "yyyy-mm-dd")
	TimeScale          TimeScale // The time scale for analysis (e.g., hourly, daily, weekly, monthly, yearly)
}

// CAI is an alias for CustomerActivityIndex.
var CAI = CustomerActivityIndex

// CustomerActivityIndex calculates the Customer Activity Index (CAI) for each customer based on their transaction history.
// It returns a DataTable containing CustomerID, MLE, WMLE, and CAI for each customer.
//
// Parameters:
//   - dt: Input DataTable containing transaction records.
//   - caiConfig: Configuration for CAI calculation, including column indices/names and date format.
//
// # CAI
//
// CAI (Customer Activity Index) is a metric used to evaluate customer activity based on their transaction history.
// It tells the change in customer activity level over time.
// A positive CAI indicates a customer whose activity is increasing, while a negative CAI indicates a customer whose activity is decreasing.
func CustomerActivityIndex(dt insyra.IDataTable, caiConfig CAIConfig) insyra.IDataTable {
	customerTransactionsTime := make(map[string][]time.Time)
	customerTransactionsIntervals := make(map[string][]int64)
	customerMLEs := make(map[string]float64)
	customerWMLEs := make(map[string]float64)

	var customerIDColIndex string
	if caiConfig.CustomerIDColIndex != "" {
		customerIDColIndex = caiConfig.CustomerIDColIndex
	} else if caiConfig.CustomerIDColName != "" {
		customerIDColIndex = dt.GetColIndexByName(caiConfig.CustomerIDColName)
	} else {
		insyra.LogWarning("mkt", "CustomerActivityIndex", "CustomerIDColIndex or CustomerIDColName must be provided, returning nil")
		return nil
	}

	var tradingDayColIndex string
	if caiConfig.TradingDayColIndex != "" {
		tradingDayColIndex = caiConfig.TradingDayColIndex
	} else if caiConfig.TradingDayColName != "" {
		tradingDayColIndex = dt.GetColIndexByName(caiConfig.TradingDayColName)
	} else {
		insyra.LogWarning("mkt", "CustomerActivityIndex", "TradingDayColIndex or TradingDayColName must be provided, returning nil")
		return nil
	}

	dateFormat := caiConfig.DateFormat
	if dateFormat == "" {
		insyra.LogInfo("mkt", "CustomerActivityIndex", "DateFormat not specified, defaulting to YYYY-MM-DD")
		dateFormat = "YYYY-MM-DD" // 預設使用 ISO 8601 格式（大寫）
	}

	timeScale := caiConfig.TimeScale
	if timeScale == "" {
		insyra.LogInfo("mkt", "CustomerActivityIndex", "TimeScale not specified, defaulting to daily")
		timeScale = TimeScaleDaily // 預設使用每日時間尺度
	}

	// 轉換為 Go 語言的日期格式
	goDateFormat := utils.ConvertDateFormat(dateFormat)

	dt.AtomicDo(func(dt *insyra.DataTable) {
		numRows, _ := dt.Size()
		for i := range numRows {
			customerID := conv.ToString(dt.GetElement(i, customerIDColIndex))
			tradingTimeStr := conv.ToString(dt.GetElement(i, tradingDayColIndex))
			if customerID == "" || tradingTimeStr == "" {
				continue
			}

			tradingTime, err := time.Parse(goDateFormat, tradingTimeStr)
			if err != nil {
				insyra.LogWarning("mkt", "CustomerActivityIndex", "Failed to parse trading time '%s' with format '%s': %v. Skipping the row", tradingTimeStr, goDateFormat, err)
				continue
			}
			customerTransactionsTime[customerID] = append(customerTransactionsTime[customerID], tradingTime)
		}
	})

	// 最後一個點加入交易紀錄中最晚的時間
	allLastTimes := []time.Time{}
	for _, times := range customerTransactionsTime {
		lenTimes := len(times)
		if lenTimes == 0 {
			continue
		}
		allLastTimes = append(allLastTimes, times[lenTimes-1])
	}
	utils.ParallelSortStableFunc(allLastTimes, func(a, b time.Time) int {
		return a.Compare(b)
	})
	latestTime := allLastTimes[len(allLastTimes)-1]

	for customerID, times := range customerTransactionsTime {
		customerTransactionsTime[customerID] = append(times, latestTime)
	}

	// 根據 timeScale 計算每個客戶的交易間隔
	// 同一單位尺度下的多次交易不算入間隔計算
	for customerID, times := range customerTransactionsTime {
		if len(times) < 2 {
			continue // 少於兩次交易無法計算間隔
		}
		// 先排序交易時間
		insyra.SortTimes(times)
		intervals := calculateIntervals(times, timeScale)
		customerTransactionsIntervals[customerID] = intervals
	}

	// 計算每個客戶的 MLE
	for customerID, intervals := range customerTransactionsIntervals {
		if len(intervals) == 0 {
			continue
		}
		l := insyra.NewDataList(intervals)
		mle := l.Mean()
		customerMLEs[customerID] = mle
	}

	// 計算WMLE
	for customerID, intervals := range customerTransactionsIntervals {
		if len(intervals) == 0 {
			continue
		}
		numIntervals := len(intervals)
		var weightDenominator float64
		for i := 1; i <= numIntervals; i++ {
			weightDenominator += float64(i)
		}
		for i, interval := range intervals {
			weight := float64(i+1) / weightDenominator
			customerWMLEs[customerID] += weight * float64(interval)
		}
	}

	// 將結果寫入新的 DataTable
	resultDT := insyra.NewDataTable(
		insyra.NewDataList().SetName("CustomerID"),
		insyra.NewDataList().SetName("MLE"),
		insyra.NewDataList().SetName("WMLE"),
		insyra.NewDataList().SetName("CAI"),
	)
	for customerID, mle := range customerMLEs {
		wmle := customerWMLEs[customerID]
		cai := (mle - wmle) / mle * 100 // 百分比表示
		resultDT.AppendRowsByColName(map[string]any{
			"CustomerID": customerID,
			"MLE":        mle,
			"WMLE":       wmle,
			"CAI":        cai,
		})
	}

	return resultDT
}

func calculateIntervals(dateTimes []time.Time, scale TimeScale) []int64 {
	if len(dateTimes) < 2 {
		return nil
	}

	var intervals []int64
	for i := 1; i < len(dateTimes); i++ {
		interval := calculateTimeDifference(dateTimes[i], dateTimes[i-1], scale)
		if interval < 1 {
			// 同一時間尺度下的多次交易不計入間隔
			continue
		}
		intervals = append(intervals, interval)
	}
	return intervals
}
