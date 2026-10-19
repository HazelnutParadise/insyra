package datafetch

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/HazelnutParadise/insyra"
)

// TWOddLotSession selects which odd-lot session OddLotQuotes reads.
type TWOddLotSession string

const (
	// TWOddLotIntraday is the odd-lot trading during the regular session, which
	// began on 2020-10-26. Earlier dates have no data.
	TWOddLotIntraday TWOddLotSession = "Intraday"
	// TWOddLotAfterHours is the odd-lot session after the close, published since
	// 2005.
	TWOddLotAfterHours TWOddLotSession = "AfterHours"
)

var oddLotHeaderAliases = map[string]string{
	"證券代號":      "Code",
	"證券名稱":      "Name",
	"成交股數":      "Volume",
	"成交筆數":      "Transactions",
	"成交金額":      "Turnover",
	"當日第一次成交價":  "Open",
	"當日最後一次成交價": "Close",
	"當日最高價":     "High",
	"當日最低價":     "Low",
	"成交價":       "Price",
	"最後揭示買價":    "BidPrice",
	"最後揭示買量":    "BidVolume",
	"最後揭示賣價":    "AskPrice",
	"最後揭示賣量":    "AskVolume",
}

// oddLotColumns lists, in order, the columns of each session's table after Date
// and the kind of value each holds.
var oddLotColumns = map[TWOddLotSession][]string{
	TWOddLotIntraday:   {"Code", "Name", "Volume", "Transactions", "Turnover", "Open", "Close", "High", "Low", "BidPrice", "BidVolume", "AskPrice", "AskVolume"},
	TWOddLotAfterHours: {"Code", "Name", "Volume", "Transactions", "Turnover", "Price", "BidPrice", "BidVolume", "AskPrice", "AskVolume"},
}

var oddLotPaths = map[TWOddLotSession]string{
	TWOddLotIntraday:   "/rwd/zh/afterTrading/TWTC7U",
	TWOddLotAfterHours: "/rwd/zh/afterTrading/TWT53U",
}

// OddLotQuotes returns the day's odd-lot (less than one board lot of 1,000
// shares) trading of every security on the chosen market for the chosen
// session. It is OddLotQuotesContext with context.Background().
func (t *TWStockClient) OddLotQuotes(date time.Time, session TWOddLotSession, market TWMarket) (*insyra.DataTable, error) {
	return t.OddLotQuotesContext(context.Background(), date, session, market)
}

// OddLotQuotesContext is OddLotQuotes with a context. The limiter wait, the
// request and each retry backoff stop when ctx is done, and the call then
// returns ctx.Err() without sending another request.
//
// The table has one row per security with Date, Code, Name, Volume (shares),
// Transactions, Turnover, the last best bid and ask price and volume, and the
// prices of the session: Open, Close (the last trade), High and Low for
// TWOddLotIntraday, and Price for TWOddLotAfterHours. Prices are as traded, not
// adjusted for ex-rights. The two markets give the same columns, so their tables
// can be stacked.
//
// For TWMarketTWSE a price shown as "--" is nil and the exchange's 合計 totals
// row is left out. For TWMarketTPEx the intraday report's change column is left
// out and a price shown as 0.00 (no trade or no quote) is nil. As with
// DailyQuotesOn, TWMarketAuto reads the TWSE and, only if it has no data for the
// date, the TPEx. A date with no data, such as a holiday or one before the
// session existed (intraday began on 2020-10-26; the TWSE's after-hours reports
// start in 2005 and the TPEx's in January 2011), gives an empty table.
func (t *TWStockClient) OddLotQuotesContext(ctx context.Context, date time.Time, session TWOddLotSession, market TWMarket) (*insyra.DataTable, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	if _, ok := oddLotPaths[session]; !ok {
		return nil, fmt.Errorf("datafetch: unsupported odd-lot session %q", session)
	}
	if !validMarket(market) {
		return nil, fmt.Errorf("datafetch: unsupported market %q", market)
	}
	date = normalizeDate(date)
	if err := t.usable(); err != nil {
		return nil, err
	}
	switch market {
	case TWMarketTPEx:
		return t.tpexOddLotQuotes(ctx, date, session)
	case TWMarketAuto:
		table, err := t.twseOddLotQuotes(ctx, date, session)
		if err != nil || table.NumRows() > 0 {
			return table, err
		}
		return t.tpexOddLotQuotes(ctx, date, session)
	}
	return t.twseOddLotQuotes(ctx, date, session)
}

func (t *TWStockClient) twseOddLotQuotes(ctx context.Context, date time.Time, session TWOddLotSession) (*insyra.DataTable, error) {
	path := oddLotPaths[session]
	columns := oddLotColumns[session]

	var response twStockResponse
	values := url.Values{"date": {date.Format("20060102")}, "selectType": {"ALL"}, "response": {"json"}}
	if err := t.doJSON(ctx, requestURL(t.twseBaseURL, path, values), &response); err != nil {
		// doJSON has already read the reply, so its status is in response. A day
		// with no report, or one before the report existed, is an empty table.
		if errors.Is(err, errTWStockNoData) || strings.Contains(response.Stat, "查詢日期小於") {
			return oddLotTable(columns, nil), nil
		}
		return nil, err
	}
	if len(response.Data) == 0 {
		return oddLotTable(columns, nil), nil
	}
	indexes, err := mapHeaders(response.Fields, oddLotHeaderAliases, columns)
	if err != nil {
		return nil, err
	}
	rows := make([][]any, 0, len(response.Data))
	for rowIndex, row := range response.Data {
		for _, name := range columns {
			if indexes[name] >= len(row) {
				return nil, fmt.Errorf("datafetch: odd-lot row %d is missing required cells", rowIndex)
			}
		}
		if strings.TrimSpace(cell(row, indexes["Code"])) == "" {
			continue // the exchange's 合計 totals row
		}
		out := make([]any, len(columns)+1)
		out[0] = date
		for i, name := range columns {
			raw := cell(row, indexes[name])
			switch name {
			case "Code", "Name":
				out[i+1] = strings.TrimSpace(raw)
			case "Volume", "Transactions", "Turnover", "BidVolume", "AskVolume":
				out[i+1] = parseInt(raw)
			default:
				out[i+1] = parseFloat(raw)
			}
		}
		rows = append(rows, out)
	}
	return oddLotTable(columns, rows), nil
}

var tpexOddLotHeaderAliases = map[TWOddLotSession]map[string]string{
	TWOddLotIntraday: {
		"代號": "Code", "名稱": "Name", "最後成交價": "Close", "首筆成交價": "Open", "最高": "High", "最低": "Low",
		"成交股數": "Volume", "成交金額(元)": "Turnover", "成交筆數": "Transactions",
		"最後買價": "BidPrice", "最後買量(股)": "BidVolume", "最後賣價": "AskPrice", "最後賣量(股)": "AskVolume",
	},
	TWOddLotAfterHours: {
		"代號": "Code", "名稱": "Name", "成交股數": "Volume", "成交筆數": "Transactions", "成交金額": "Turnover",
		"成交價格(元)": "Price", "未成交買價": "BidPrice", "未成交買量": "BidVolume", "未成交賣價": "AskPrice", "未成交賣量": "AskVolume",
	},
}

var tpexOddLotPaths = map[TWOddLotSession]string{
	TWOddLotIntraday:   "/www/zh-tw/afterTrading/oddQuote",
	TWOddLotAfterHours: "/www/zh-tw/afterTrading/odd",
}

func (t *TWStockClient) tpexOddLotQuotes(ctx context.Context, date time.Time, session TWOddLotSession) (*insyra.DataTable, error) {
	path := tpexOddLotPaths[session]
	columns := oddLotColumns[session]

	var response twStockResponse
	values := url.Values{"date": {date.Format("2006/01/02")}, "type": {"Daily"}, "response": {"json"}}
	if err := t.doJSON(ctx, requestURL(t.tpexBaseURL, path, values), &response); err != nil {
		if errors.Is(err, errTWStockNoData) {
			return oddLotTable(columns, nil), nil
		}
		return nil, err
	}
	if len(response.Tables) == 0 || len(response.Tables[0].Data) == 0 {
		return oddLotTable(columns, nil), nil
	}
	table := response.Tables[0]
	fields := make([]string, len(table.Fields))
	for i, f := range table.Fields {
		fields[i] = strings.TrimSpace(f)
	}
	indexes, err := mapHeaders(fields, tpexOddLotHeaderAliases[session], columns)
	if err != nil {
		return nil, err
	}
	rows := make([][]any, 0, len(table.Data))
	for rowIndex, row := range table.Data {
		for _, name := range columns {
			if indexes[name] >= len(row) {
				return nil, fmt.Errorf("datafetch: TPEx odd-lot row %d is missing required cells", rowIndex)
			}
		}
		if strings.TrimSpace(cell(row, indexes["Code"])) == "" {
			continue
		}
		out := make([]any, len(columns)+1)
		out[0] = date
		for i, name := range columns {
			raw := cell(row, indexes[name])
			switch name {
			case "Code", "Name":
				out[i+1] = strings.TrimSpace(raw)
			case "Volume", "Transactions", "Turnover", "BidVolume", "AskVolume":
				out[i+1] = parseInt(raw)
			default:
				// A price the TPEx shows as 0.00 is no trade or no quote.
				if price := parseFloat(raw); price != 0.0 {
					out[i+1] = price
				}
			}
		}
		rows = append(rows, out)
	}
	return oddLotTable(columns, rows), nil
}

func oddLotTable(columns []string, rows [][]any) *insyra.DataTable {
	names := append([]string{"Date"}, columns...)
	cols := make([]*insyra.DataList, len(names))
	for c, name := range names {
		values := make([]any, len(rows))
		for r, row := range rows {
			values[r] = row[c]
		}
		cols[c] = newNamedCol(name, values)
	}
	return insyra.NewDataTable(cols...)
}
