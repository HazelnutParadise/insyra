package datafetch

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/HazelnutParadise/insyra"
)

var listingColumns = []string{"Code", "Name", "Market", "Listed", "Delisted", "Reason", "MovedTo"}

// tpexDelistingsFirstYear is the first year the TPEx's list of terminated
// listings is asked for; the exchange opened in 1994.
const tpexDelistingsFirstYear = 1994

// listingsNow is the clock that decides the last year of the TPEx's list.
var listingsNow = time.Now

// Listings returns every listing an exchange publishes, current and ended. It
// is ListingsContext with context.Background().
func (t *TWStockClient) Listings(market TWMarket) (*insyra.DataTable, error) {
	return t.ListingsContext(context.Background(), market)
}

// ListingsContext is Listings with a context. The limiter wait, each request
// and each retry backoff stop when ctx is done, and the call then returns
// ctx.Err() without sending another request.
//
// Each row is one company's stint on one exchange: Code, Name, Market ("TWSE"
// or "TPEx"), Listed and Delisted (time.Time), Reason and MovedTo. A company
// that moved from the TPEx to the TWSE has a TPEx row that ends on the day its
// TWSE row begins. Name is the name the exchange's list gives: the short name
// for a listed company and in the TWSE's list of ended listings, the full name
// in the TPEx's. TWMarketAuto returns both exchanges, the TWSE first, then each
// by code.
//
// Listed is the listing date of a company still listed, from the exchange's
// list of listed companies. For a company whose listing has ended, the exchange
// publishes only the end, so Listed is nil and Delisted is the date the listing
// ended. Delisted is nil for a company still listed. Reason is the TPEx's reason
// for ending a listing, a citation of its rules, and nil for the TWSE, which
// publishes none, and where the TPEx left it empty, as it did through 2006.
// MovedTo is "TWSE" when the
// TPEx ended the listing because the company moved to the TWSE, and nil
// otherwise; a move the other way cannot be told from the TWSE's list.
//
// An exchange may give a code to another company after a listing ends, so the
// stints of one code can overlap: 2301 is listed on the TWSE from 1995 and also
// has a TWSE listing that ended in 2002.
//
// The TWSE's list of ended listings starts in 2001, so a company that left
// before then is not in the table. The lists of listed companies hold companies
// only: no ETFs, which DailyQuotesOn has. To know which securities were on an
// exchange on one trading day, use DailyQuotesOn, which has every security of
// that day, including the ones that later left.
//
// The TPEx is asked for each year since 1994 twice, all ended listings and the
// moves to the TWSE, so TWMarketTPEx and TWMarketAuto make about seventy
// requests: set Interval and Retries as for any long TWStock run.
func (t *TWStockClient) ListingsContext(ctx context.Context, market TWMarket) (*insyra.DataTable, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	if !validMarket(market) {
		return nil, fmt.Errorf("datafetch: unsupported market %q", market)
	}
	if err := t.usable(); err != nil {
		return nil, err
	}
	var rows [][]any
	if market != TWMarketTPEx {
		twse, err := t.twseListings(ctx)
		if err != nil {
			return nil, err
		}
		rows = append(rows, twse...)
	}
	if market != TWMarketTWSE {
		tpex, err := t.tpexListings(ctx)
		if err != nil {
			return nil, err
		}
		rows = append(rows, tpex...)
	}
	return listingTable(rows), nil
}

func (t *TWStockClient) twseListings(ctx context.Context) ([][]any, error) {
	var listed []map[string]string
	if err := t.doJSON(ctx, requestURL(t.twseOpenAPIBaseURL, "/v1/opendata/t187ap03_L", nil), &listed); err != nil {
		return nil, fmt.Errorf("datafetch: TWSE listed companies: %w", err)
	}
	rows, err := currentListings(listed, "TWSE", "公司代號", "公司簡稱", "上市日期")
	if err != nil {
		return nil, err
	}
	var ended struct {
		Status string     `json:"status"`
		Fields []string   `json:"fields"`
		Data   [][]string `json:"data"`
	}
	if err := t.doJSON(ctx, requestURL(t.twseBaseURL, "/rwd/zh/company/suspendListing", url.Values{"response": {"json"}}), &ended); err != nil {
		return nil, fmt.Errorf("datafetch: TWSE ended listings: %w", err)
	}
	if ended.Status != "" && !strings.EqualFold(ended.Status, "ok") {
		return nil, fmt.Errorf("datafetch: TWSE ended listings: status %q", ended.Status)
	}
	indexes, err := mapHeaders(ended.Fields, map[string]string{
		"終止上市日期": "Date", "公司名稱": "Name", "上市編號": "Code", "證券代號": "Code",
	}, []string{"Date", "Name", "Code"})
	if err != nil {
		return nil, fmt.Errorf("datafetch: TWSE ended listings: %w", err)
	}
	for i, row := range ended.Data {
		if len(row) < len(ended.Fields) {
			return nil, fmt.Errorf("datafetch: TWSE ended listings row %d has %d fields, want %d", i, len(row), len(ended.Fields))
		}
		date, err := parseROCDate(row[indexes["Date"]])
		if err != nil {
			return nil, fmt.Errorf("datafetch: TWSE ended listings row %d: %w", i, err)
		}
		rows = append(rows, []any{strings.TrimSpace(row[indexes["Code"]]), strings.TrimSpace(row[indexes["Name"]]), "TWSE", nil, date, nil, nil})
	}
	return rows, nil
}

// tpexDelisted is the TPEx's answer for one year of ended listings.
type tpexDelisted struct {
	Stat   string `json:"stat"`
	Tables []struct {
		Fields     []string   `json:"fields"`
		Data       [][]string `json:"data"`
		TotalCount *int       `json:"totalCount"`
	} `json:"tables"`
}

func (t *TWStockClient) tpexListings(ctx context.Context) ([][]any, error) {
	var listed []map[string]string
	if err := t.doJSON(ctx, requestURL(t.tpexOpenAPIBaseURL, "/openapi/v1/mopsfin_t187ap03_O", nil), &listed); err != nil {
		return nil, fmt.Errorf("datafetch: TPEx listed companies: %w", err)
	}
	rows, err := currentListings(listed, "TPEx", "SecuritiesCompanyCode", "CompanyAbbreviation", "DateOfListing")
	if err != nil {
		return nil, err
	}
	last := listingsNow().In(time.FixedZone("CST", 8*60*60)).Year()
	for year := tpexDelistingsFirstYear; year <= last; year++ {
		ended, err := t.tpexDelistedYear(ctx, year, "-1")
		if err != nil {
			return nil, err
		}
		moved, err := t.tpexDelistedYear(ctx, year, "2")
		if err != nil {
			return nil, err
		}
		movedCodes := map[string]bool{}
		for _, row := range moved {
			movedCodes[row.code+"\x00"+row.date.Format("20060102")] = true
		}
		for _, row := range ended {
			var movedTo any
			if movedCodes[row.code+"\x00"+row.date.Format("20060102")] {
				movedTo = "TWSE"
			}
			var reason any
			if row.reason != "" {
				reason = row.reason
			}
			rows = append(rows, []any{row.code, row.name, "TPEx", nil, row.date, reason, movedTo})
		}
	}
	return rows, nil
}

type tpexDelistedRow struct {
	code, name, reason string
	date               time.Time
}

func (t *TWStockClient) tpexDelistedYear(ctx context.Context, year int, reason string) ([]tpexDelistedRow, error) {
	// The site pages its answer, ten rows by default; paging-size asks for all of
	// them, and the rows are checked against totalCount.
	values := url.Values{"date": {strconv.Itoa(year)}, "reason": {reason}, "paging-size": {"10000"}, "response": {"json"}}
	var response tpexDelisted
	if err := t.doJSON(ctx, requestURL(t.tpexBaseURL, "/www/zh-tw/company/deListed", values), &response); err != nil {
		return nil, fmt.Errorf("datafetch: TPEx ended listings of %d: %w", year, err)
	}
	if response.Stat != "" && !strings.EqualFold(response.Stat, "ok") {
		return nil, fmt.Errorf("datafetch: TPEx ended listings of %d: status %q", year, response.Stat)
	}
	var rows []tpexDelistedRow
	for _, table := range response.Tables {
		if table.TotalCount != nil && *table.TotalCount != len(table.Data) {
			return nil, fmt.Errorf("datafetch: TPEx ended listings of %d: got %d of %d rows", year, len(table.Data), *table.TotalCount)
		}
		if len(table.Data) == 0 {
			continue
		}
		indexes, err := mapHeaders(table.Fields, map[string]string{
			"股票代號": "Code", "公司名稱": "Name", "終止上櫃日期": "Date", "終止上櫃原因": "Reason",
		}, []string{"Code", "Name", "Date", "Reason"})
		if err != nil {
			return nil, fmt.Errorf("datafetch: TPEx ended listings of %d: %w", year, err)
		}
		for i, row := range table.Data {
			if len(row) < len(table.Fields) {
				return nil, fmt.Errorf("datafetch: TPEx ended listings of %d row %d has %d fields, want %d", year, i, len(row), len(table.Fields))
			}
			date, err := parseROCDate(strings.ReplaceAll(row[indexes["Date"]], "-", "/"))
			if err != nil {
				return nil, fmt.Errorf("datafetch: TPEx ended listings of %d row %d: %w", year, i, err)
			}
			rows = append(rows, tpexDelistedRow{
				code:   strings.TrimSpace(row[indexes["Code"]]),
				name:   strings.TrimSpace(row[indexes["Name"]]),
				reason: strings.TrimSpace(row[indexes["Reason"]]),
				date:   date,
			})
		}
	}
	return rows, nil
}

// currentListings reads an exchange's list of listed companies.
func currentListings(records []map[string]string, market, codeKey, nameKey, dateKey string) ([][]any, error) {
	if len(records) == 0 {
		return nil, fmt.Errorf("datafetch: %s listed companies: the list is empty", market)
	}
	rows := make([][]any, 0, len(records))
	for i, record := range records {
		code := strings.TrimSpace(record[codeKey])
		if code == "" {
			return nil, fmt.Errorf("datafetch: %s listed companies record %d has no %q", market, i, codeKey)
		}
		var listed any
		if raw := strings.TrimSpace(record[dateKey]); raw != "" {
			date, err := time.Parse("20060102", raw)
			if err != nil {
				return nil, fmt.Errorf("datafetch: %s listed companies record %d has %s %q", market, i, dateKey, raw)
			}
			listed = date
		}
		rows = append(rows, []any{code, strings.TrimSpace(record[nameKey]), market, listed, nil, nil, nil})
	}
	return rows, nil
}

// listingTable orders rows by exchange (TWSE first), code and the stint's end,
// a current listing last.
func listingTable(rows [][]any) *insyra.DataTable {
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if a[2] != b[2] {
			return a[2] == "TWSE"
		}
		if a[0] != b[0] {
			return a[0].(string) < b[0].(string)
		}
		ae, aok := a[4].(time.Time)
		be, bok := b[4].(time.Time)
		if aok != bok {
			return aok
		}
		return aok && ae.Before(be)
	})
	lists := make([]*insyra.DataList, len(listingColumns))
	for c, name := range listingColumns {
		values := make([]any, len(rows))
		for r, row := range rows {
			values[r] = row[c]
		}
		lists[c] = newNamedCol(name, values)
	}
	return insyra.NewDataTable(lists...)
}
