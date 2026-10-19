package datafetch

import (
	"bytes"
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/HazelnutParadise/insyra"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/transform"
)

var (
	revenueRowPattern      = regexp.MustCompile(`(?is)<tr[^>]*>(.*?)</tr>`)
	revenueCellPattern     = regexp.MustCompile(`(?is)<t([dh])[^>]*>(.*?)</t[dh]>`)
	revenueTagPattern      = regexp.MustCompile(`(?is)<[^>]*>`)
	revenueIndustryPattern = regexp.MustCompile(`產業別：([^<\s]+)`)
	revenueDatePattern     = regexp.MustCompile(`出表日期：\s*([0-9]{2,3}/[0-9]{1,2}/[0-9]{1,2})`)
)

var revenueColumns = []string{
	"Month", "Code", "Name", "Industry", "Market", "Revenue", "PrevMonthRevenue", "LastYearRevenue",
	"MoMChange", "YoYChange", "Cumulative", "LastYearCumulative", "CumulativeChange", "Remark", "PageDate",
}

// MonthlyRevenue returns the monthly operating revenue of every listed company
// for one month. It is MonthlyRevenueContext with context.Background().
func (t *TWStockClient) MonthlyRevenue(year, month int, market TWMarket) (*insyra.DataTable, error) {
	return t.MonthlyRevenueContext(context.Background(), year, month, market)
}

// MonthlyRevenueContext is MonthlyRevenue with a context. The limiter wait,
// each request and each retry backoff stop when ctx is done, and the call then
// returns ctx.Err() without sending another request.
//
// year is the Gregorian year (2025, not 114) and month 1 to 12; the Market
// Observation Post System (MOPS) has pages from January 2003. market is
// TWMarketTWSE, TWMarketTPEx or TWMarketAuto for both. A month the site has no
// page for gives an empty table with the columns in place.
//
// One row per company, from the MOPS summary pages, which carry every company
// of a market in one page: Month (time.Time, the first of the month), Code,
// Name, Industry, Market ("TWSE" or "TPEx"), Revenue, PrevMonthRevenue,
// LastYearRevenue, Cumulative and LastYearCumulative (int64, thousands of NT
// dollars), MoMChange, YoYChange and CumulativeChange (float64, percent),
// Remark (string) and PageDate (time.Time). Companies listed abroad with
// revenue in another currency are on the pages' second part and are included.
//
// PageDate is the date the whole page was produced (出表日期), not the date
// each company announced its figure: companies report through the 10th of the
// following month, so do not treat PageDate as the moment a figure was known and
// tradable. Names are decoded from Big5, which loses a few rare characters.
func (t *TWStockClient) MonthlyRevenueContext(ctx context.Context, year, month int, market TWMarket) (*insyra.DataTable, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	if !validMarket(market) {
		return nil, fmt.Errorf("datafetch: unsupported market %q", market)
	}
	if year < 2003 || year > 9999 || month < 1 || month > 12 {
		return nil, fmt.Errorf("datafetch: monthly revenue needs a year from 2003 and a month from 1 to 12, got %d-%d", year, month)
	}
	if err := t.usable(); err != nil {
		return nil, err
	}
	var rows [][]any
	markets := []TWMarket{market}
	if market == TWMarketAuto {
		markets = []TWMarket{TWMarketTWSE, TWMarketTPEx}
	}
	for _, m := range markets {
		part, err := t.monthlyRevenueRows(ctx, year, month, m)
		if err != nil {
			return nil, err
		}
		rows = append(rows, part...)
	}
	return revenueTable(rows), nil
}

func (t *TWStockClient) monthlyRevenueRows(ctx context.Context, year, month int, market TWMarket) ([][]any, error) {
	dir, label := "sii", "TWSE"
	if market == TWMarketTPEx {
		dir, label = "otc", "TPEx"
	}
	roc := year - 1911
	first := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	base := fmt.Sprintf("/nas/t21/%s/t21sc03_%d_%d", dir, roc, month)

	// Newer months have a domestic page (_0) and a foreign-company page (_1);
	// older ones have a single page with no suffix.
	var pages [][]byte
	domestic, found, err := t.fetchRevenuePage(ctx, t.mopsBaseURL+base+"_0.html")
	if err != nil {
		return nil, err
	}
	if found {
		pages = append(pages, domestic)
		foreign, ok, err := t.fetchRevenuePage(ctx, t.mopsBaseURL+base+"_1.html")
		if err != nil {
			return nil, err
		}
		if ok {
			pages = append(pages, foreign)
		}
	} else {
		single, ok, err := t.fetchRevenuePage(ctx, t.mopsBaseURL+base+".html")
		if err != nil {
			return nil, err
		}
		if ok {
			pages = append(pages, single)
		}
	}
	var rows [][]any
	for _, body := range pages {
		part, err := parseRevenuePage(body, first, label)
		if err != nil {
			return nil, err
		}
		rows = append(rows, part...)
	}
	return rows, nil
}

// fetchRevenuePage gets one MOPS page and decodes it from Big5. found is false
// for a 404, which is how the site says a page does not exist.
func (t *TWStockClient) fetchRevenuePage(ctx context.Context, rawURL string) (text []byte, found bool, err error) {
	err = t.doGet(ctx, rawURL, func(resp *http.Response) error {
		if resp.StatusCode == http.StatusNotFound {
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
			_ = resp.Body.Close()
			return nil
		}
		body, err := readBodyResponse(resp, 64<<20)
		if err != nil {
			return err
		}
		decoded, _, err := transform.Bytes(traditionalchinese.Big5.NewDecoder(), body)
		if err != nil {
			return fmt.Errorf("datafetch: MOPS page is not Big5: %w", err)
		}
		text, found = decoded, true
		return nil
	})
	return text, found, err
}

func revenueCellText(cell string) string {
	cell = strings.ReplaceAll(cell, "<br>", "")
	cell = revenueTagPattern.ReplaceAllString(cell, "")
	return strings.TrimSpace(strings.ReplaceAll(html.UnescapeString(cell), " ", " "))
}

// parseRevenuePage reads the company rows of one MOPS revenue page.
func parseRevenuePage(page []byte, month time.Time, market string) ([][]any, error) {
	text := string(bytes.ReplaceAll(page, []byte("\r"), nil))
	var pageDate any
	if m := revenueDatePattern.FindStringSubmatch(text); m != nil {
		if d, err := parseROCDate(m[1]); err == nil {
			pageDate = d
		}
	}
	// Industry headers come before their tables, so walk the text in order.
	type marker struct {
		at       int
		industry string
	}
	var markers []marker
	for _, m := range revenueIndustryPattern.FindAllStringSubmatchIndex(text, -1) {
		markers = append(markers, marker{m[0], text[m[2]:m[3]]})
	}
	industryAt := func(pos int) string {
		name := ""
		for _, m := range markers {
			if m.at > pos {
				break
			}
			name = m.industry
		}
		return name
	}

	var rows [][]any
	for _, loc := range revenueRowPattern.FindAllStringSubmatchIndex(text, -1) {
		cells := revenueCellPattern.FindAllStringSubmatch(text[loc[2]:loc[3]], -1)
		// A company row is a td code, a td name and eight td figures, with a
		// remark after them on newer pages. Headers and totals use th cells.
		if len(cells) < 10 || cells[0][1] != "d" && cells[0][1] != "D" {
			continue
		}
		code := revenueCellText(cells[0][2])
		if code == "" || strings.ContainsAny(code, " \t") || len(code) > 8 {
			continue
		}
		if _, ok := parseNumber(revenueCellText(cells[2][2])); !ok && revenueCellText(cells[2][2]) != "" && revenueCellText(cells[2][2]) != "-" {
			continue
		}
		remark := ""
		if len(cells) > 10 {
			remark = revenueCellText(cells[10][2])
			if remark == "-" {
				remark = ""
			}
		}
		fig := func(i int) string { return revenueCellText(cells[i][2]) }
		rows = append(rows, []any{
			month, code, fig(1), industryAt(loc[0]), market,
			parseInt(fig(2)), parseInt(fig(3)), parseInt(fig(4)),
			parseFloat(fig(5)), parseFloat(fig(6)),
			parseInt(fig(7)), parseInt(fig(8)), parseFloat(fig(9)),
			remark, pageDate,
		})
	}
	return rows, nil
}

func revenueTable(rows [][]any) *insyra.DataTable {
	cols := make([]*insyra.DataList, len(revenueColumns))
	for c, name := range revenueColumns {
		values := make([]any, len(rows))
		for r, row := range rows {
			values[r] = row[c]
		}
		cols[c] = newNamedCol(name, values)
	}
	return insyra.NewDataTable(cols...)
}
