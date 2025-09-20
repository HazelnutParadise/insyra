package mkt

import (
	"fmt"
	"time"

	"github.com/HazelnutParadise/Go-Utils/conv"
	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/internal/utils"
	"github.com/HazelnutParadise/insyra/parallel"
)

type RFMConfig struct {
	CustomerIDColIndex string // The column index(A, B, C, ...) of customer ID in the data table
	CustomerIDColName  string // The column name of customer ID in the data table (if both index and name are provided, index takes precedence)
	TradingDayColIndex string // The column index(A, B, C, ...) of trading day in the data table
	TradingDayColName  string // The column name of trading day in the data table (if both index and name are provided, index takes precedence)
	AmountColIndex     string // The column index(A, B, C, ...) of amount in the data table
	AmountColName      string // The column name of amount in the data table (if both index and name are provided, index takes precedence)
	NumGroups          uint   // The number of groups to divide the customers into
	DateFormat         string // The format of the date string (e.g., "YYYY-MM-DD", "DD/MM/YYYY", "yyyy-mm-dd")
}

// RFM performs RFM analysis on the given data table based on the provided configuration.
// It returns a new data table containing the R, F, M scores and the combined RFM score for each customer.
func RFM(dt insyra.IDataTable, rfmConfig RFMConfig) insyra.IDataTable {
	var customerIDColIndex string
	if rfmConfig.CustomerIDColIndex != "" {
		customerIDColIndex = rfmConfig.CustomerIDColIndex
	} else if rfmConfig.CustomerIDColName != "" {
		customerIDColIndex = dt.GetColIndexByName(rfmConfig.CustomerIDColName)
	} else {
		insyra.LogWarning("mkt", "RFM", "CustomerIDColIndex or CustomerIDColName must be provided, returning nil")
		return nil
	}

	var tradingDayColIndex string
	if rfmConfig.TradingDayColIndex != "" {
		tradingDayColIndex = rfmConfig.TradingDayColIndex
	} else if rfmConfig.TradingDayColName != "" {
		tradingDayColIndex = dt.GetColIndexByName(rfmConfig.TradingDayColName)
	} else {
		insyra.LogWarning("mkt", "RFM", "TradingDayColIndex or TradingDayColName must be provided, returning nil")
		return nil
	}

	var amountColIndex string
	if rfmConfig.AmountColIndex != "" {
		amountColIndex = rfmConfig.AmountColIndex
	} else if rfmConfig.AmountColName != "" {
		amountColIndex = dt.GetColIndexByName(rfmConfig.AmountColName)
	} else {
		insyra.LogWarning("mkt", "RFM", "AmountColIndex or AmountColName must be provided, returning nil")
		return nil
	}

	numGroups := rfmConfig.NumGroups
	dateFormat := rfmConfig.DateFormat

	// 如果沒有指定日期格式，使用預設格式
	if dateFormat == "" {
		dateFormat = "YYYY-MM-DD" // 預設使用 ISO 8601 格式（大寫）
	}

	// 轉換為 Go 語言的日期格式
	goDateFormat := utils.ConvertDateFormat(dateFormat)

	customerLastTradingDayMap := make(map[string]int64)  // map[customerID]lastTradingDay
	customerTradingFrequencyMap := make(map[string]uint) // map[customerID]tradingFrequency
	customerTotalAmountMap := make(map[string]float64)   // map[customerID]totalAmount

	fail := false
	dt.AtomicDo(func(dt *insyra.DataTable) {
		// 找出每個客戶的最後交易日
		numRows, _ := dt.Size()
		for i := range numRows {
			lastTradingDayStr := conv.ToString(dt.GetElement(i, tradingDayColIndex))
			customerID := conv.ToString(dt.GetElement(i, customerIDColIndex))
			amount := conv.ParseF64(dt.GetElement(i, amountColIndex))

			// 跳過無效的資料
			if lastTradingDayStr == "" || customerID == "" {
				continue
			}

			// 計算交易頻率
			if _, exists := customerTradingFrequencyMap[customerID]; !exists {
				customerTradingFrequencyMap[customerID] = 0
			}
			customerTradingFrequencyMap[customerID]++

			// 解析日期字串
			lastTradingDay, err := time.Parse(goDateFormat, lastTradingDayStr)
			if err != nil {
				insyra.LogWarning("mkt", "RFM", "Failed to parse date: %s, returning nil", lastTradingDayStr)
				fail = true
				return
			}
			lastTradingDayUnix := lastTradingDay.Unix()

			// 計算最後交易日
			if existingLastTradingDay, exists := customerLastTradingDayMap[customerID]; !exists || lastTradingDayUnix > existingLastTradingDay {
				customerLastTradingDayMap[customerID] = lastTradingDayUnix
			}

			// 計算總交易金額
			if _, exists := customerTotalAmountMap[customerID]; !exists {
				customerTotalAmountMap[customerID] = 0.0
			}
			customerTotalAmountMap[customerID] += amount
		}
	})
	if fail {
		return nil
	}

	rThresholds := make([]float64, numGroups-1)
	fThresholds := make([]float64, numGroups-1)
	mThresholds := make([]float64, numGroups-1)
	customerRMap := make(map[string]int) // map[customerID]R_value (days since last trade)
	parallel.GroupUp(func() {
		// 計算當前時間
		now := time.Now().Unix()

		// 計算R值（天數差異）
		for customerID, lastTradingDay := range customerLastTradingDayMap {
			rValue := int((now - lastTradingDay) / (24 * 60 * 60)) // 轉換為天數
			customerRMap[customerID] = rValue
		}

		// 全體客戶的R值
		allRValues := insyra.NewDataList()
		for _, rValue := range customerRMap {
			allRValues.Append(rValue)
		}

		// 計算R分組門檻（反向：小值高分）
		for i := 0; i < int(numGroups)-1; i++ {
			percentile := float64(i+1) / float64(numGroups) * 100
			rThresholds[i] = allRValues.Percentile(percentile)
		}
	}, func() {
		// 全體客戶的交易頻率
		allTradingFrequencies := insyra.NewDataList()
		for _, tradingFrequency := range customerTradingFrequencyMap {
			allTradingFrequencies.Append(tradingFrequency)
		}

		// 計算F分組門檻（正向：大值高分）
		for i := 0; i < int(numGroups)-1; i++ {
			percentile := float64(i+1) / float64(numGroups) * 100
			fThresholds[i] = allTradingFrequencies.Percentile(percentile)
		}
	}, func() {
		// 全體客戶的總交易金額
		allTotalAmounts := insyra.NewDataList()
		for _, totalAmount := range customerTotalAmountMap {
			allTotalAmounts.Append(totalAmount)
		}
		// 計算M分組門檻（正向：大值高分）
		for i := 0; i < int(numGroups)-1; i++ {
			percentile := float64(i+1) / float64(numGroups) * 100
			mThresholds[i] = allTotalAmounts.Percentile(percentile)
		}
	}).Run().AwaitResult()

	// 創建RFM表
	rfmTable := insyra.NewDataTable()

	// 為每個客戶計算分數
	for customerID := range customerLastTradingDayMap {
		rValue := customerRMap[customerID]
		fValue := customerTradingFrequencyMap[customerID]
		mValue := customerTotalAmountMap[customerID]

		// 計算R分數（反向）
		rScore := calculateScore(float64(rValue), rThresholds, false)

		// 計算F分數（正向）
		fScore := calculateScore(float64(fValue), fThresholds, true)

		// 計算M分數（正向）
		mScore := calculateScore(mValue, mThresholds, true)

		// RFM組合分數
		rfmScore := fmt.Sprintf("%d%d%d", rScore, fScore, mScore)

		// 添加到表中
		rowData := map[string]any{
			"A": customerID,
			"B": rScore,
			"C": fScore,
			"D": mScore,
			"E": rfmScore,
		}
		rfmTable.AppendRowsByColIndex(rowData)
	}
	rfmTable.SetColNameByIndex("A", "CustomerID")
	rfmTable.SetColNameByIndex("B", "R_Score")
	rfmTable.SetColNameByIndex("C", "F_Score")
	rfmTable.SetColNameByIndex("D", "M_Score")
	rfmTable.SetColNameByIndex("E", "RFM_Score")

	return rfmTable
}

// calculateScore 根據值和門檻計算分數
func calculateScore(value float64, thresholds []float64, ascending bool) int {
	score := 1
	for _, threshold := range thresholds {
		if ascending {
			if value > threshold {
				score++
			}
		} else {
			if value <= threshold {
				score++
			}
		}
	}
	return score
}
