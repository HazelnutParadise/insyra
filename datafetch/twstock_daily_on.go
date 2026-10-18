package datafetch

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/HazelnutParadise/insyra"
)

var htmlTagText = regexp.MustCompile(`>([^<]*)<`)

// tagText returns the text inside the markup the TWSE wraps its up and down
// marks in, such as <p style= color:red>+</p>, or s itself when there is none.
func tagText(s string) string {
	if m := htmlTagText.FindStringSubmatch(s); m != nil {
		return strings.TrimSpace(m[1])
	}
	return strings.TrimSpace(s)
}

// dailyQuoteRow is one security's close on one day. sign is "+" (up), "-"
// (down), "=" (unchanged) or "X" (no reference price, as on the first day of a
// listing); change is signed and nil under "X".
type dailyQuoteRow struct {
	code, name                       string
	volume, transactions, turnover   any
	open, high, low, closePrice, chg any
	sign                             string
	date                             time.Time
}

var dailyQuoteColumns = []string{"Date", "Code", "Name", "Volume", "Turnover", "Open", "High", "Low", "Close", "Change", "Sign", "Transactions"}

// DailyQuotesOn returns every security's close for one trading day. It is
// DailyQuotesOnContext with context.Background().
func (t *TWStockClient) DailyQuotesOn(date time.Time, market TWMarket) (*insyra.DataTable, error) {
	return t.DailyQuotesOnContext(context.Background(), date, market)
}

// DailyQuotesOnContext is DailyQuotesOn with a context. The limiter wait, each
// request and each retry backoff stop when ctx is done, and the call then
// returns ctx.Err() without sending another request.
//
// Unlike AllDailyQuotes, which only has the latest day, any past day is
// available, and a security that was later delisted is on its days. The table
// has the columns of AllDailyQuotes plus Sign: "+" up, "-" down, "=" unchanged
// and "X" when the exchange gave no reference price, in which case Change is
// nil. Change is signed, in price units, and is measured from the exchange's own
// reference price, which already accounts for ex-rights and ex-dividend days.
// Prices are as traded.
//
// A day the exchange has no data for, such as a weekend or holiday, gives an
// empty table with the columns in place and no error. TWMarketAuto tries the
// TWSE first and the TPEx when the TWSE has no data. The TWSE has data from
// 2004-02-11.
func (t *TWStockClient) DailyQuotesOnContext(ctx context.Context, date time.Time, market TWMarket) (*insyra.DataTable, error) {
	rows, _, err := t.dailyQuotesOn(ctx, date, market)
	if err != nil {
		return nil, err
	}
	return dailyQuoteTable(rows), nil
}

// AdvanceDecline returns how many securities rose, fell and stood still on one
// day. It is AdvanceDeclineContext with context.Background().
func (t *TWStockClient) AdvanceDecline(date time.Time, market TWMarket, commonStockOnly bool) (*insyra.DataTable, error) {
	return t.AdvanceDeclineContext(context.Background(), date, market, commonStockOnly)
}

// AdvanceDeclineContext is AdvanceDecline with a context.
//
// The table has one row with Date, Advancing, AdvancingLimit, Declining,
// DecliningLimit, Unchanged, NoTrade, NoReference (all int64) and Source
// ("official" or "computed"). When the TWSE published its 漲跌證券數合計 table
// for the day, which it has only for recent years, that table is used: the
// 股票 column when commonStockOnly is set and the 整體市場 column otherwise, with
// the limit-up and limit-down counts, and Source is "official". Otherwise the
// counts are made from the day's closes using each security's up or down mark,
// the limit counts are nil, and Source is "computed". In that case commonStockOnly
// keeps four-digit codes starting with 1 to 9 and drops those starting with
// 91, the depositary receipts; on 2026-10-16 this gave the same 684 and 286 as
// the official 股票 column. NoTrade counts securities that did not trade and
// NoReference those with no reference price.
//
// A day with no data gives an empty table with the columns in place.
func (t *TWStockClient) AdvanceDeclineContext(ctx context.Context, date time.Time, market TWMarket, commonStockOnly bool) (*insyra.DataTable, error) {
	rows, official, err := t.dailyQuotesOn(ctx, date, market)
	if err != nil {
		return nil, err
	}
	date = normalizeDate(date)
	if len(rows) == 0 {
		return advanceDeclineTable(date, nil, ""), nil
	}
	if official != nil {
		counts := official.overall
		if commonStockOnly {
			counts = official.stock
		}
		return advanceDeclineTable(date, counts, "official"), nil
	}
	counts := &advanceDeclineCounts{advancingLimit: -1, decliningLimit: -1}
	for _, r := range rows {
		if commonStockOnly && !isCommonStockCode(r.code) {
			continue
		}
		volume, _ := r.volume.(int64)
		switch {
		case volume == 0:
			counts.noTrade++
		case r.sign == "X":
			counts.noReference++
		case r.sign == "+":
			counts.advancing++
		case r.sign == "-":
			counts.declining++
		default:
			counts.unchanged++
		}
	}
	return advanceDeclineTable(date, counts, "computed"), nil
}

// isCommonStockCode reports whether a TWSE or TPEx code is an ordinary stock:
// four digits with a first digit from 1 to 9, excluding 91, the depositary
// receipts.
func isCommonStockCode(code string) bool {
	if len(code) != 4 {
		return false
	}
	for _, r := range code {
		if r < '0' || r > '9' {
			return false
		}
	}
	return code[0] >= '1' && code[0] <= '9' && !strings.HasPrefix(code, "91")
}

// advanceDeclineCounts holds one column of counts. A limit field is -1 when the
// source gave none.
type advanceDeclineCounts struct {
	advancing, advancingLimit, declining, decliningLimit, unchanged, noTrade, noReference int64
}

type officialAdvanceDecline struct {
	overall, stock *advanceDeclineCounts
}

func (t *TWStockClient) dailyQuotesOn(ctx context.Context, date time.Time, market TWMarket) ([]dailyQuoteRow, *officialAdvanceDecline, error) {
	if err := contextErr(ctx); err != nil {
		return nil, nil, err
	}
	if !validMarket(market) {
		return nil, nil, fmt.Errorf("datafetch: unsupported market %q", market)
	}
	if err := t.usable(); err != nil {
		return nil, nil, err
	}
	date = normalizeDate(date)
	switch market {
	case TWMarketAuto:
		rows, official, err := t.dailyQuotesOn(ctx, date, TWMarketTWSE)
		if err != nil || len(rows) > 0 {
			return rows, official, err
		}
		return t.dailyQuotesOn(ctx, date, TWMarketTPEx)
	case TWMarketTPEx:
		rows, err := t.tpexDailyQuotesOn(ctx, date)
		return rows, nil, err
	}
	return t.twseDailyQuotesOn(ctx, date)
}

func (t *TWStockClient) twseDailyQuotesOn(ctx context.Context, date time.Time) ([]dailyQuoteRow, *officialAdvanceDecline, error) {
	var response twStockResponse
	values := url.Values{"date": {date.Format("20060102")}, "type": {"ALLBUT0999"}, "response": {"json"}}
	if err := t.doJSON(ctx, requestURL(t.twseBaseURL, "/rwd/zh/afterTrading/MI_INDEX", values), &response); err != nil {
		if errors.Is(err, errTWStockNoData) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	var quotes, breadth *twStockTable
	for i := range response.Tables {
		switch {
		case strings.Contains(response.Tables[i].Title, "每日收盤行情"):
			quotes = &response.Tables[i]
		case strings.Contains(response.Tables[i].Title, "漲跌證券數合計"):
			breadth = &response.Tables[i]
		}
	}
	if quotes == nil || len(quotes.Data) == 0 {
		return nil, nil, nil
	}
	indexes, err := mapHeaders(quotes.Fields, twseDailyOnHeaderAliases, []string{"Code", "Name", "Volume", "Transactions", "Turnover", "Open", "High", "Low", "Close", "Sign", "Diff"})
	if err != nil {
		return nil, nil, err
	}
	rows := make([]dailyQuoteRow, 0, len(quotes.Data))
	for rowIndex, row := range quotes.Data {
		for _, name := range []string{"Code", "Diff"} {
			if indexes[name] >= len(row) {
				return nil, nil, fmt.Errorf("datafetch: daily quote row %d is missing required cells", rowIndex)
			}
		}
		sign, diff := tagText(cell(row, indexes["Sign"])), parseFloat(cell(row, indexes["Diff"]))
		r := dailyQuoteRow{
			date: date, code: strings.TrimSpace(cell(row, indexes["Code"])), name: strings.TrimSpace(cell(row, indexes["Name"])),
			volume: parseInt(cell(row, indexes["Volume"])), transactions: parseInt(cell(row, indexes["Transactions"])),
			turnover: parseInt(cell(row, indexes["Turnover"])),
			open:     parseFloat(cell(row, indexes["Open"])), high: parseFloat(cell(row, indexes["High"])),
			low: parseFloat(cell(row, indexes["Low"])), closePrice: parseFloat(cell(row, indexes["Close"])),
		}
		switch sign {
		case "+", "-":
			r.sign = sign
			if d, ok := diff.(float64); ok && sign == "-" {
				diff = -d
			}
			r.chg = diff
		case "X":
			r.sign = "X"
		default:
			r.sign = "="
			r.chg = diff
		}
		rows = append(rows, r)
	}
	var official *officialAdvanceDecline
	if breadth != nil {
		official = parseOfficialAdvanceDecline(breadth.Data)
	}
	return rows, official, nil
}

var twseDailyOnHeaderAliases = map[string]string{
	"證券代號":    "Code",
	"證券名稱":    "Name",
	"成交股數":    "Volume",
	"成交筆數":    "Transactions",
	"成交金額":    "Turnover",
	"開盤價":     "Open",
	"最高價":     "High",
	"最低價":     "Low",
	"收盤價":     "Close",
	"漲跌(+/-)": "Sign",
	"漲跌價差":    "Diff",
}

// parseOfficialAdvanceDecline reads the 漲跌證券數合計 table, whose rows are 上漲(漲停),
// 下跌(跌停), 持平, 未成交 and 無比價 and whose columns are the whole market and stocks.
// A cell such as "9,016(68)" is the count with its limit-up or limit-down part.
func parseOfficialAdvanceDecline(rows [][]string) *officialAdvanceDecline {
	out := &officialAdvanceDecline{overall: &advanceDeclineCounts{advancingLimit: -1, decliningLimit: -1}, stock: &advanceDeclineCounts{advancingLimit: -1, decliningLimit: -1}}
	found := 0
	for _, row := range rows {
		if len(row) < 3 {
			continue
		}
		label := strings.TrimSpace(row[0])
		for col, dst := range []*advanceDeclineCounts{out.overall, out.stock} {
			count, limit := splitCountWithLimit(row[col+1])
			switch {
			case strings.HasPrefix(label, "上漲"):
				dst.advancing, dst.advancingLimit = count, limit
			case strings.HasPrefix(label, "下跌"):
				dst.declining, dst.decliningLimit = count, limit
			case strings.HasPrefix(label, "持平"):
				dst.unchanged = count
			case strings.HasPrefix(label, "未成交"):
				dst.noTrade = count
			case strings.HasPrefix(label, "無比價"):
				dst.noReference = count
			default:
				continue
			}
		}
		if strings.HasPrefix(label, "上漲") || strings.HasPrefix(label, "下跌") || strings.HasPrefix(label, "持平") || strings.HasPrefix(label, "未成交") || strings.HasPrefix(label, "無比價") {
			found++
		}
	}
	if found < 5 {
		return nil
	}
	return out
}

func splitCountWithLimit(cell string) (count, limit int64) {
	limit = -1
	cell = strings.TrimSpace(cell)
	if open := strings.Index(cell, "("); open >= 0 {
		if l, err := strconv.ParseInt(strings.TrimSuffix(strings.TrimSpace(cell[open+1:]), ")"), 10, 64); err == nil {
			limit = l
		}
		cell = cell[:open]
	}
	n, _ := parseNumber(cell)
	return int64(n), limit
}

func (t *TWStockClient) tpexDailyQuotesOn(ctx context.Context, date time.Time) ([]dailyQuoteRow, error) {
	var response twStockResponse
	values := url.Values{"date": {date.Format("2006/01/02")}, "type": {"EW"}, "response": {"json"}}
	if err := t.doJSON(ctx, requestURL(t.tpexBaseURL, "/www/zh-tw/afterTrading/otc", values), &response); err != nil {
		if errors.Is(err, errTWStockNoData) {
			return nil, nil
		}
		return nil, err
	}
	if len(response.Tables) == 0 || len(response.Tables[0].Data) == 0 {
		return nil, nil
	}
	table := response.Tables[0]
	fields := make([]string, len(table.Fields))
	for i, f := range table.Fields {
		fields[i] = strings.TrimSpace(f)
	}
	indexes, err := mapHeaders(fields, tpexDailyOnHeaderAliases, []string{"Code", "Name", "Close", "Diff", "Open", "High", "Low", "Volume", "Turnover", "Transactions"})
	if err != nil {
		return nil, err
	}
	rows := make([]dailyQuoteRow, 0, len(table.Data))
	for rowIndex, row := range table.Data {
		if indexes["Transactions"] >= len(row) || indexes["Diff"] >= len(row) {
			return nil, fmt.Errorf("datafetch: daily quote row %d is missing required cells", rowIndex)
		}
		r := dailyQuoteRow{
			date: date, code: strings.TrimSpace(cell(row, indexes["Code"])), name: strings.TrimSpace(cell(row, indexes["Name"])),
			volume: parseInt(cell(row, indexes["Volume"])), transactions: parseInt(cell(row, indexes["Transactions"])),
			turnover: parseInt(cell(row, indexes["Turnover"])),
			open:     parseFloat(cell(row, indexes["Open"])), high: parseFloat(cell(row, indexes["High"])),
			low: parseFloat(cell(row, indexes["Low"])), closePrice: parseFloat(cell(row, indexes["Close"])),
		}
		diff := strings.TrimSpace(cell(row, indexes["Diff"]))
		if chg, ok := parseNumber(strings.TrimPrefix(diff, "+")); ok {
			r.chg = chg
			switch {
			case chg > 0:
				r.sign = "+"
			case chg < 0:
				r.sign = "-"
			default:
				r.sign = "="
			}
		} else {
			// "---" and the like: no reference price to compare with.
			r.sign = "X"
		}
		rows = append(rows, r)
	}
	return rows, nil
}

var tpexDailyOnHeaderAliases = map[string]string{
	"代號":      "Code",
	"名稱":      "Name",
	"收盤":      "Close",
	"漲跌":      "Diff",
	"開盤":      "Open",
	"最高":      "High",
	"最低":      "Low",
	"成交股數":    "Volume",
	"成交金額(元)": "Turnover",
	"成交筆數":    "Transactions",
}

func dailyQuoteTable(rows []dailyQuoteRow) *insyra.DataTable {
	cols := make(map[string][]any, len(dailyQuoteColumns))
	for _, name := range dailyQuoteColumns {
		cols[name] = make([]any, len(rows))
	}
	for i, r := range rows {
		cols["Date"][i], cols["Code"][i], cols["Name"][i] = r.date, r.code, r.name
		cols["Volume"][i], cols["Turnover"][i], cols["Transactions"][i] = r.volume, r.turnover, r.transactions
		cols["Open"][i], cols["High"][i], cols["Low"][i], cols["Close"][i] = r.open, r.high, r.low, r.closePrice
		cols["Change"][i], cols["Sign"][i] = r.chg, r.sign
	}
	lists := make([]*insyra.DataList, len(dailyQuoteColumns))
	for i, name := range dailyQuoteColumns {
		lists[i] = newNamedCol(name, cols[name])
	}
	return insyra.NewDataTable(lists...)
}

func advanceDeclineTable(date time.Time, c *advanceDeclineCounts, source string) *insyra.DataTable {
	names := []string{"Date", "Advancing", "AdvancingLimit", "Declining", "DecliningLimit", "Unchanged", "NoTrade", "NoReference", "Source"}
	var values []any
	if c != nil {
		limit := func(v int64) any {
			if v < 0 {
				return nil
			}
			return v
		}
		values = []any{date, c.advancing, limit(c.advancingLimit), c.declining, limit(c.decliningLimit), c.unchanged, c.noTrade, c.noReference, source}
	}
	cols := make([]*insyra.DataList, len(names))
	for i, name := range names {
		var col []any
		if values != nil {
			col = []any{values[i]}
		}
		cols[i] = newNamedCol(name, col)
	}
	return insyra.NewDataTable(cols...)
}
