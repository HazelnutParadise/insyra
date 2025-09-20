package mkt

import (
	"testing"

	"github.com/HazelnutParadise/insyra"
)

func TestRFM(t *testing.T) {
	// 創建測試數據
	dt := insyra.NewDataTable()
	dt.AppendRowsByColIndex(map[string]any{
		"A": "C001",       // CustomerID
		"B": "2023-01-01", // TradingDay
		"C": 100.0,        // Amount
	})
	dt.AppendRowsByColIndex(map[string]any{
		"A": "C001",
		"B": "2023-01-05",
		"C": 200.0,
	})
	dt.AppendRowsByColIndex(map[string]any{
		"A": "C002",
		"B": "2023-01-10",
		"C": 150.0,
	})
	dt.AppendRowsByColIndex(map[string]any{
		"A": "C002",
		"B": "2023-01-15",
		"C": 250.0,
	})
	dt.AppendRowsByColIndex(map[string]any{
		"A": "C002",
		"B": "2023-01-20",
		"C": 300.0,
	})

	// 設置列名
	dt.SetColNameByIndex("A", "CustomerID")
	dt.SetColNameByIndex("B", "TradingDay")
	dt.SetColNameByIndex("C", "Amount")

	// RFM 配置
	config := RFMConfig{
		CustomerIDColName:  "CustomerID",
		TradingDayColIndex: "B",
		AmountColIndex:     "C",
		NumGroups:          5,
		DateFormat:         "2006-01-02",
	}

	// 執行 RFM
	result := RFM(dt, config)
	if result == nil {
		t.Fatal("RFM returned nil")
	}

	// 檢查結果
	numRows, numCols := result.Size()
	if numRows != 2 { // 兩個客戶
		t.Errorf("Expected 2 rows, got %d", numRows)
	}
	if numCols != 5 { // CustomerID, R_Score, F_Score, M_Score, RFM_Score
		t.Errorf("Expected 5 columns, got %d", numCols)
	}

	// 檢查列名
	colNames := result.ColNames()
	expectedCols := []string{"CustomerID", "R_Score", "F_Score", "M_Score", "RFM_Score"}
	for i, expected := range expectedCols {
		if colNames[i] != expected {
			t.Errorf("Expected column %d to be %s, got %s", i, expected, colNames[i])
		}
	}

	// 打印結果以手動檢查
	t.Logf("RFM Result:")
	for i := range numRows {
		row := result.GetRow(i)
		t.Logf("Row %d: %v", i, row)
	}
}
