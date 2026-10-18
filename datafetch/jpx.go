package datafetch

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/HazelnutParadise/insyra"
)

// JPXUnit chooses whether JPXClient.InvestorTypes reports traded value or
// traded shares.
type JPXUnit string

const (
	// JPXUnitValue is the value traded, in thousands of yen.
	JPXUnitValue JPXUnit = "Value"
	// JPXUnitShares is the number of shares traded, in thousands of shares.
	JPXUnitShares JPXUnit = "Shares"
)

// JPXClient reads the Japan Exchange Group's weekly trading by investor type
// (投資部門別売買状況). Create one with JPX. A client that did not come from JPX,
// such as a zero value, returns an error from every method.
type JPXClient struct {
	base    *TWStockClient
	baseURL string
}

// JPX creates a client for the JPX statistics pages. It takes the same settings
// as TWStock: Timeout, Interval, UserAgent, Retries and RetryBackoff. The JPX
// serves files one at a time, so an Interval of about 3 seconds is polite and
// was enough to fetch every week since 2016 without being blocked.
func JPX(cfg TWStockConfig) (*JPXClient, error) {
	base, err := TWStock(cfg)
	if err != nil {
		return nil, err
	}
	return &JPXClient{base: base, baseURL: "https://www.jpx.co.jp"}, nil
}

var errJPXNotCreated = errors.New("datafetch: JPXClient was not created with JPX")

const jpxInvestorPath = "/markets/statistics-equities/investor-type/"

var (
	// The yearly archives are options of a drop-down list: value="...archives-02.html">2024年.
	jpxArchivePattern = regexp.MustCompile(`value="([^"]*investor-type/00-00-archives-\d+\.html)">(\d{4})年`)
	jpxFilePattern    = regexp.MustCompile(`href="([^"]*/(stock_(val|vol)_1_(\d{6})\.xls|stock_1_w_(\d{8})_(\d{8})\.xlsx))"`)
)

// jpxFile is one downloadable file found on a JPX page.
type jpxFile struct {
	url    string
	legacy bool   // an .xls file, one per unit and week
	kind   string // "val" or "vol" for a legacy file
	code   string // yymmww of a legacy file
	start  time.Time
	end    time.Time // the week of an .xlsx file
}

// InvestorTypes returns the weekly trading of each investor type on the Tokyo
// and Nagoya markets for the weeks that overlap [from, to]. It is
// InvestorTypesContext with context.Background().
func (c *JPXClient) InvestorTypes(from, to time.Time, unit JPXUnit) (*insyra.DataTable, error) {
	return c.InvestorTypesContext(context.Background(), from, to, unit)
}

// InvestorTypesContext is InvestorTypes with a context. The limiter wait, each
// request and each retry backoff stop when ctx is done, and the call then
// returns ctx.Err() without sending another request.
//
// The table has one row per week, market and investor type: WeekStart and
// WeekEnd (time.Time, the first and last trading day of the week, so a holiday
// week is one to three days and a week may cross the year end), Market
// ("TSE Prime", "TSE Standard", "TSE Growth" and "Tokyo & Nagoya", or "TSE 1st",
// "TSE 2nd", "TSE Mothers" and "TSE JASDAQ" for the weeks before April 2022),
// Investor, and Sales, Purchases, Balance (purchases less sales) and Total
// (sales plus purchases), all int64 in thousands of yen for JPXUnitValue and
// thousands of shares for JPXUnitShares. A week with no figures for a market
// has no rows.
//
// Investor is one of fourteen groups that do not overlap and add up to the
// market: "Proprietary Cash", "Proprietary Margin", "Individuals Cash",
// "Individuals Margin", "Foreigners Institutions", "Foreigners Individuals",
// "Securities Cos.", "Investment Trusts", "Business Cos.", "Other Cos.",
// "Life & Non-Life", "City & Regional BK", "Trust BK" and "Other Financials".
// Individuals in total are the first two Individuals rows summed, and so on.
//
// The JPX publishes a week's figures on the fourth business day of the
// following week at 15:00 JST, so a backtest must lag them by at least that.
// Weeks since September 2026 come from one workbook per week; earlier weeks
// from the older files, which hold the previous week beside the report week.
// Only the report week of each file is used.
func (c *JPXClient) InvestorTypesContext(ctx context.Context, from, to time.Time, unit JPXUnit) (*insyra.DataTable, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	if c == nil || c.base == nil {
		return nil, errJPXNotCreated
	}
	if err := c.base.usable(); err != nil {
		return nil, err
	}
	if unit != JPXUnitValue && unit != JPXUnitShares {
		return nil, fmt.Errorf("datafetch: unsupported JPX unit %q", unit)
	}
	from, to = normalizeDate(from), normalizeDate(to)
	if from.IsZero() || to.IsZero() || to.Before(from) {
		return nil, errors.New("datafetch: JPX needs a from date that is not after the to date")
	}

	files, err := c.findFiles(ctx, from, to)
	if err != nil {
		return nil, err
	}
	var rows []jpxRow
	seen := map[string]bool{}
	for _, f := range selectJPXFiles(files, from, to, unit) {
		data, err := c.download(ctx, f.url)
		if err != nil {
			return nil, err
		}
		var part []jpxRow
		if f.legacy {
			part, err = parseJPXLegacy(data, unit)
		} else {
			part, err = parseJPXWeekly(data, f.start, f.end, unit)
		}
		if err != nil {
			return nil, fmt.Errorf("datafetch: JPX file %s: %w", f.url[strings.LastIndex(f.url, "/")+1:], err)
		}
		for _, r := range part {
			key := r.start.Format("20060102") + "|" + r.market + "|" + r.investor
			if r.end.Before(from) || r.start.After(to) || seen[key] {
				continue
			}
			seen[key] = true
			rows = append(rows, r)
		}
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if !rows[i].start.Equal(rows[j].start) {
			return rows[i].start.Before(rows[j].start)
		}
		return false
	})
	return jpxTable(rows), nil
}

// findFiles lists the files on the current page and on the yearly archive pages
// for the years from the year of from to the year after to, since a week that
// ends in January is filed under the new year.
func (c *JPXClient) findFiles(ctx context.Context, from, to time.Time) ([]jpxFile, error) {
	page, err := c.fetchPage(ctx, c.baseURL+jpxInvestorPath+"index.html")
	if err != nil {
		return nil, err
	}
	files := jpxFilesOn(page, c.baseURL)
	seenPage := map[string]bool{}
	for _, m := range jpxArchivePattern.FindAllStringSubmatch(page, -1) {
		year, _ := strconv.Atoi(m[2])
		if year < from.Year() || year > to.Year()+1 || seenPage[m[1]] {
			continue
		}
		seenPage[m[1]] = true
		body, err := c.fetchPage(ctx, absoluteJPX(c.baseURL, m[1]))
		if err != nil {
			return nil, err
		}
		files = append(files, jpxFilesOn(body, c.baseURL)...)
	}
	return files, nil
}

func absoluteJPX(base, href string) string {
	if strings.HasPrefix(href, "http") {
		return href
	}
	return base + href
}

func jpxFilesOn(page, base string) []jpxFile {
	var files []jpxFile
	for _, m := range jpxFilePattern.FindAllStringSubmatch(page, -1) {
		f := jpxFile{url: absoluteJPX(base, m[1])}
		if m[3] != "" {
			f.legacy, f.kind, f.code = true, m[3], m[4]
		} else {
			start, err1 := time.Parse("20060102", m[5])
			end, err2 := time.Parse("20060102", m[6])
			if err1 != nil || err2 != nil {
				continue
			}
			f.start, f.end = start, end
		}
		files = append(files, f)
	}
	return files
}

// selectJPXFiles picks the files that can hold a week overlapping [from, to]:
// every workbook whose week overlaps, and the legacy files of the right unit
// whose month is within a month of the range. Duplicates are dropped.
func selectJPXFiles(files []jpxFile, from, to time.Time, unit JPXUnit) []jpxFile {
	kind := "val"
	if unit == JPXUnitShares {
		kind = "vol"
	}
	lo := from.AddDate(0, -1, 0)
	hi := to.AddDate(0, 1, 0)
	seen := map[string]bool{}
	var out []jpxFile
	for _, f := range files {
		if seen[f.url] {
			continue
		}
		if f.legacy {
			if f.kind != kind || len(f.code) != 6 {
				continue
			}
			yy, _ := strconv.Atoi(f.code[:2])
			mm, _ := strconv.Atoi(f.code[2:4])
			month := time.Date(2000+yy, time.Month(mm), 1, 0, 0, 0, 0, time.UTC)
			if month.Before(time.Date(lo.Year(), lo.Month(), 1, 0, 0, 0, 0, time.UTC)) || month.After(hi) {
				continue
			}
		} else if f.end.Before(from) || f.start.After(to) {
			continue
		}
		seen[f.url] = true
		out = append(out, f)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].url < out[j].url })
	return out
}

func (c *JPXClient) fetchPage(ctx context.Context, rawURL string) (string, error) {
	data, err := c.download(ctx, rawURL)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (c *JPXClient) download(ctx context.Context, rawURL string) ([]byte, error) {
	var body []byte
	err := c.base.doGet(ctx, rawURL, func(resp *http.Response) error {
		b, err := readBodyResponse(resp, 64<<20)
		body = b
		return err
	})
	return body, err
}

// jpxRow is one week, market and investor group.
type jpxRow struct {
	start, end                       time.Time
	market, investor                 string
	sales, purchases, balance, total int64
}

func jpxTable(rows []jpxRow) *insyra.DataTable {
	names := []string{"WeekStart", "WeekEnd", "Market", "Investor", "Sales", "Purchases", "Balance", "Total"}
	cols := make([][]any, len(names))
	for c := range cols {
		cols[c] = make([]any, len(rows))
	}
	for i, r := range rows {
		cols[0][i], cols[1][i], cols[2][i], cols[3][i] = r.start, r.end, r.market, r.investor
		cols[4][i], cols[5][i], cols[6][i], cols[7][i] = r.sales, r.purchases, r.balance, r.total
	}
	lists := make([]*insyra.DataList, len(names))
	for c, name := range names {
		lists[c] = newNamedCol(name, cols[c])
	}
	return insyra.NewDataTable(lists...)
}

// jpxInvestors lists the fourteen groups in the order of the weekly workbook.
var jpxInvestors = []string{
	"Proprietary Cash", "Proprietary Margin", "Individuals Cash", "Individuals Margin",
	"Foreigners Institutions", "Foreigners Individuals", "Securities Cos.", "Investment Trusts",
	"Business Cos.", "Other Cos.", "Life & Non-Life", "City & Regional BK", "Trust BK", "Other Financials",
}

// jpxNumber reads a figure the JPX writes as text with thousands separators, or
// as a number. ok is false for a blank cell.
func jpxNumber(text string) (int64, bool) {
	text = strings.TrimSpace(strings.ReplaceAll(text, ",", ""))
	if text == "" || text == "-" {
		return 0, false
	}
	if n, err := strconv.ParseInt(text, 10, 64); err == nil {
		return n, true
	}
	if f, err := strconv.ParseFloat(text, 64); err == nil {
		return int64(f), true
	}
	return 0, false
}

func newJPXRow(start, end time.Time, market, investor string, sales, purchases int64) jpxRow {
	return jpxRow{start: start, end: end, market: market, investor: investor,
		sales: sales, purchases: purchases, balance: purchases - sales, total: sales + purchases}
}
