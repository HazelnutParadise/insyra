package mkt

import "github.com/HazelnutParadise/insyra"

type RFMConfig struct {
	CustomerIDCol string // The column index(A, B, C, ...) of customer ID in the data table
	TradingDayCol string // The column index(A, B, C, ...) of trading day in the data table
	AmountCol     string // The column index(A, B, C, ...) of amount in the data table
	NumGroups     uint   // The number of groups to divide the customers into
}

// todo
func RFM(dt insyra.IDataTable, rfmConfig RFMConfig) {}
