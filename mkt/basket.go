package mkt

import (
	"sort"

	"github.com/HazelnutParadise/Go-Utils/conv"
	"github.com/HazelnutParadise/insyra"
)

// BasketConfig 是 BasketAnalysis 的設定。
// 輸入資料形式與 RFM / CAI 類似：每一列代表「某張訂單包含的某項商品」。
type BasketConfig struct {
	OrderIDColIndex   string // The column index(A, B, C, ...) of order ID in the data table
	OrderIDColName    string // The column name of order ID in the data table (if both index and name are provided, index takes precedence)
	ProductIDColIndex string // The column index(A, B, C, ...) of product ID in the data table
	ProductIDColName  string // The column name of product ID in the data table (if both index and name are provided, index takes precedence)
}

// BasketResult holds the three matrices returned by BasketAnalysis.
// 三個矩陣皆為 商品 × 商品 的方陣，列代表前項商品 A、欄代表後項商品 B。
type BasketResult struct {
	Support    insyra.IDataTable // 支援度矩陣: P(A ∩ B) = 同時購買 A 和 B 的訂單數 / 總訂單數
	Confidence insyra.IDataTable // 置信度矩陣: P(B | A) = 同時購買 A 和 B 的訂單數 / 購買 A 的訂單數
	Lift       insyra.IDataTable // 提升度矩陣: Support(A,B) / (P(A) * P(B))
}

// BasketAnalysis performs market basket analysis on the given transaction data.
// It returns a BasketResult containing the support, confidence, and lift matrices,
// each stored as a DataTable where rows represent the antecedent item A and
// columns represent the consequent item B.
//
//   - Support(A, B)    = orders containing both A and B / total orders
//   - Confidence(A→B)  = orders containing both A and B / orders containing A
//   - Lift(A→B)        = Support(A, B) / (P(A) * P(B))
//
// Each order is treated as a set of items, so duplicated product IDs within the
// same order are counted only once.
func BasketAnalysis(dt insyra.IDataTable, config BasketConfig) *BasketResult {
	var orderIDColIndex string
	if config.OrderIDColIndex != "" {
		orderIDColIndex = config.OrderIDColIndex
	} else if config.OrderIDColName != "" {
		orderIDColIndex = dt.GetColIndexByName(config.OrderIDColName)
	} else {
		insyra.LogWarning("mkt", "BasketAnalysis", "OrderIDColIndex or OrderIDColName must be provided, returning nil")
		return nil
	}

	var productIDColIndex string
	if config.ProductIDColIndex != "" {
		productIDColIndex = config.ProductIDColIndex
	} else if config.ProductIDColName != "" {
		productIDColIndex = dt.GetColIndexByName(config.ProductIDColName)
	} else {
		insyra.LogWarning("mkt", "BasketAnalysis", "ProductIDColIndex or ProductIDColName must be provided, returning nil")
		return nil
	}

	// orderItems[orderID] = set of productIDs in that order
	orderItems := make(map[string]map[string]bool)
	dt.AtomicDo(func(dt *insyra.DataTable) {
		numRows, _ := dt.Size()
		for i := range numRows {
			orderID := conv.ToString(dt.GetElement(i, orderIDColIndex))
			productID := conv.ToString(dt.GetElement(i, productIDColIndex))
			if orderID == "" || productID == "" {
				continue
			}
			if _, ok := orderItems[orderID]; !ok {
				orderItems[orderID] = make(map[string]bool)
			}
			orderItems[orderID][productID] = true
		}
	})

	totalOrders := len(orderItems)
	if totalOrders == 0 {
		insyra.LogWarning("mkt", "BasketAnalysis", "No valid orders found, returning nil")
		return nil
	}

	// 統計每個商品出現的訂單數，與每對商品的共現訂單數
	itemCounts := make(map[string]int)
	pairCounts := make(map[string]map[string]int)
	for _, items := range orderItems {
		productList := make([]string, 0, len(items))
		for p := range items {
			productList = append(productList, p)
		}
		for _, p := range productList {
			itemCounts[p]++
		}
		for _, a := range productList {
			row, ok := pairCounts[a]
			if !ok {
				row = make(map[string]int)
				pairCounts[a] = row
			}
			for _, b := range productList {
				row[b]++ // 包含對角線 (A, A)，其值會等於 itemCounts[A]
			}
		}
	}

	// 收集所有商品，依字典序排序，作為矩陣的列／欄名稱
	products := make([]string, 0, len(itemCounts))
	for p := range itemCounts {
		products = append(products, p)
	}
	sort.Strings(products)

	nFloat := float64(totalOrders)

	supportTable := insyra.NewDataTable()
	confidenceTable := insyra.NewDataTable()
	liftTable := insyra.NewDataTable()

	// 外層 b 為欄（後項），內層 a 為列（前項）
	for _, b := range products {
		supportCol := insyra.NewDataList().SetName(b)
		confidenceCol := insyra.NewDataList().SetName(b)
		liftCol := insyra.NewDataList().SetName(b)

		countB := itemCounts[b]
		for _, a := range products {
			countA := itemCounts[a]
			countAB := 0
			if row, ok := pairCounts[a]; ok {
				countAB = row[b]
			}

			support := float64(countAB) / nFloat

			var confidence float64
			if countA > 0 {
				confidence = float64(countAB) / float64(countA)
			}

			var lift float64
			if countA > 0 && countB > 0 {
				lift = support / ((float64(countA) / nFloat) * (float64(countB) / nFloat))
			}

			supportCol.Append(support)
			confidenceCol.Append(confidence)
			liftCol.Append(lift)
		}

		supportTable.AppendCols(supportCol)
		confidenceTable.AppendCols(confidenceCol)
		liftTable.AppendCols(liftCol)
	}

	supportTable.SetRowNames(products)
	confidenceTable.SetRowNames(products)
	liftTable.SetRowNames(products)

	return &BasketResult{
		Support:    supportTable,
		Confidence: confidenceTable,
		Lift:       liftTable,
	}
}
