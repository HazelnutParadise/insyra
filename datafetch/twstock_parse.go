package datafetch

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type twStockTable struct {
	Fields []string   `json:"fields"`
	Data   [][]string `json:"data"`
}

type twStockResponse struct {
	Stat   string         `json:"stat"`
	Date   string         `json:"date"`
	Fields []string       `json:"fields"`
	Data   [][]string     `json:"data"`
	Tables []twStockTable `json:"tables"`
}

var institutionalHeaderAliases = map[string]string{
	"證券代號": "Code",
	"證券名稱": "Name",
	"外陸資買賣超股數(不含外資自營商)": "ForeignNet",
	"投信買賣超股數":           "TrustNet",
	"自營商買賣超股數":          "DealerNet",
	"三大法人買賣超股數":         "TotalNet",
}

var twseDailyHeaderAliases = map[string]string{
	"日期":   "Date",
	"成交股數": "Volume",
	"成交金額": "Turnover",
	"開盤價":  "Open",
	"最高價":  "High",
	"最低價":  "Low",
	"收盤價":  "Close",
	"漲跌價差": "Change",
	"成交筆數": "Transactions",
}

var tpexDailyHeaderAliases = map[string]string{
	"日 期":  "Date",
	"成交張數": "Volume",
	"成交仟元": "Turnover",
	"開盤":   "Open",
	"最高":   "High",
	"最低":   "Low",
	"收盤":   "Close",
	"漲跌":   "Change",
	"筆數":   "Transactions",
}

var exRightsHeaderAliases = map[string]string{
	"資料日期":    "Date",
	"股票代號":    "Code",
	"股票名稱":    "Name",
	"除權息前收盤價": "PrevClose",
	"除權息參考價":  "RefPrice",
	"權值+息值":   "Distribution",
	"權/息":     "Kind",
}

// exRightsKind maps the exchange's 權/息 column to a stable English label.
// Anything the exchange has not published before is passed through as-is
// rather than silently folded into one of the known kinds.
func exRightsKind(value string) string {
	switch strings.TrimSpace(value) {
	case "息":
		return "dividend"
	case "權":
		return "rights"
	case "權息":
		return "both"
	default:
		return strings.TrimSpace(value)
	}
}

var twseQuoteHeaderAliases = map[string]string{
	"Date":         "Date",
	"Code":         "Code",
	"Name":         "Name",
	"TradeVolume":  "Volume",
	"TradeValue":   "Turnover",
	"OpeningPrice": "Open",
	"HighestPrice": "High",
	"LowestPrice":  "Low",
	"ClosingPrice": "Close",
	"Change":       "Change",
	"Transaction":  "Transactions",
}

var tpexQuoteHeaderAliases = map[string]string{
	"Date":                  "Date",
	"SecuritiesCompanyCode": "Code",
	"CompanyName":           "Name",
	"TradingShares":         "Volume",
	"TransactionAmount":     "Turnover",
	"Open":                  "Open",
	"High":                  "High",
	"Low":                   "Low",
	"Close":                 "Close",
	"Change":                "Change",
	"TransactionNumber":     "Transactions",
}

func parseROCDate(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	var year, month, day int
	switch {
	case strings.Contains(value, "年"):
		yearPart, rest, hasYear := strings.Cut(value, "年")
		monthPart, rest, hasMonth := strings.Cut(rest, "月")
		dayPart, trailing, hasDay := strings.Cut(rest, "日")
		if !hasYear || !hasMonth || !hasDay || strings.TrimSpace(trailing) != "" {
			return time.Time{}, fmt.Errorf("datafetch: invalid ROC date %q", value)
		}
		var err error
		year, err = strconv.Atoi(strings.TrimSpace(yearPart))
		if err != nil {
			return time.Time{}, fmt.Errorf("datafetch: invalid ROC date %q: %w", value, err)
		}
		month, err = strconv.Atoi(strings.TrimSpace(monthPart))
		if err != nil {
			return time.Time{}, fmt.Errorf("datafetch: invalid ROC date %q: %w", value, err)
		}
		day, err = strconv.Atoi(strings.TrimSpace(dayPart))
		if err != nil {
			return time.Time{}, fmt.Errorf("datafetch: invalid ROC date %q: %w", value, err)
		}
	case strings.Contains(value, "/"):
		parts := strings.Split(value, "/")
		if len(parts) != 3 {
			return time.Time{}, fmt.Errorf("datafetch: invalid ROC date %q", value)
		}
		var err error
		year, err = strconv.Atoi(parts[0])
		if err != nil {
			return time.Time{}, fmt.Errorf("datafetch: invalid ROC date %q: %w", value, err)
		}
		month, err = strconv.Atoi(parts[1])
		if err != nil {
			return time.Time{}, fmt.Errorf("datafetch: invalid ROC date %q: %w", value, err)
		}
		day, err = strconv.Atoi(parts[2])
		if err != nil {
			return time.Time{}, fmt.Errorf("datafetch: invalid ROC date %q: %w", value, err)
		}
	case len(value) == 7:
		var err error
		year, err = strconv.Atoi(value[:3])
		if err != nil {
			return time.Time{}, fmt.Errorf("datafetch: invalid ROC date %q: %w", value, err)
		}
		month, err = strconv.Atoi(value[3:5])
		if err != nil {
			return time.Time{}, fmt.Errorf("datafetch: invalid ROC date %q: %w", value, err)
		}
		day, err = strconv.Atoi(value[5:])
		if err != nil {
			return time.Time{}, fmt.Errorf("datafetch: invalid ROC date %q: %w", value, err)
		}
	default:
		return time.Time{}, fmt.Errorf("datafetch: invalid ROC date %q", value)
	}
	if year < 1 {
		return time.Time{}, fmt.Errorf("datafetch: invalid ROC date %q", value)
	}
	result := time.Date(year+1911, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	if result.Year() != year+1911 || result.Month() != time.Month(month) || result.Day() != day {
		return time.Time{}, fmt.Errorf("datafetch: invalid ROC date %q", value)
	}
	return result, nil
}

func parseNumber(value string) (float64, bool) {
	value = strings.TrimSpace(strings.ReplaceAll(value, ",", ""))
	if value == "" || value == "--" || strings.EqualFold(value, "X") {
		return 0, false
	}
	result, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, false
	}
	return result, true
}

func parseInt(value string) any {
	number, ok := parseNumber(value)
	if !ok {
		return nil
	}
	return int64(number)
}

func parseFloat(value string) any {
	number, ok := parseNumber(value)
	if !ok {
		return nil
	}
	return number
}

func mapHeaders(fields []string, aliases map[string]string, required []string) (map[string]int, error) {
	indexes := make(map[string]int, len(fields))
	for index, field := range fields {
		if name, ok := aliases[field]; ok {
			if _, exists := indexes[name]; !exists {
				indexes[name] = index
			}
		}
	}
	for _, name := range required {
		if _, ok := indexes[name]; !ok {
			return nil, fmt.Errorf("datafetch: required header %q is missing", requiredHeaderName(name, aliases))
		}
	}
	return indexes, nil
}

func requiredHeaderName(name string, aliases map[string]string) string {
	for header, alias := range aliases {
		if alias == name {
			return header
		}
	}
	return name
}

func cell(row []string, index int) string {
	if index < 0 || index >= len(row) {
		return ""
	}
	return row[index]
}

func tableFromResponse(response twStockResponse) (twStockTable, error) {
	if len(response.Tables) == 0 {
		return twStockTable{}, errTWStockNoData
	}
	return response.Tables[0], nil
}
