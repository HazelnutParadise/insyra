package datafetch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/datafetch/internal/limiter"
)

const (
	defaultTWStockTimeout     = 15 * time.Second
	defaultTWStockUserAgent   = "insyra-datafetch/" + insyra.Version
	defaultTWStockBackoff     = 300 * time.Millisecond
	defaultTWStockConcurrency = 6
)

// TWStockConfig configures a TWSE/TPEx stock data client. A zero value is valid.
type TWStockConfig struct {
	// Timeout is the per-request timeout. Default: 15s.
	Timeout time.Duration
	// Interval is the minimum spacing between requests. Zero disables throttling.
	Interval time.Duration
	// UserAgent is the HTTP User-Agent header.
	UserAgent string
	// Retries is the number of retry attempts after a failed request.
	Retries int
	// RetryBackoff is the base delay before a retry. Default: 300ms.
	RetryBackoff time.Duration
	// Concurrency is the normalized concurrency setting for batched fetches.
	Concurrency int
}

func (cfg TWStockConfig) normalize() (TWStockConfig, error) {
	out := cfg
	if out.Timeout <= 0 {
		out.Timeout = defaultTWStockTimeout
	}
	if out.Interval < 0 {
		return TWStockConfig{}, errors.New("datafetch: TWStock Interval must be >= 0")
	}
	if out.UserAgent == "" {
		out.UserAgent = defaultTWStockUserAgent
	}
	if out.Retries < 0 {
		return TWStockConfig{}, errors.New("datafetch: TWStock Retries must be >= 0")
	}
	if out.RetryBackoff < 0 {
		return TWStockConfig{}, errors.New("datafetch: TWStock RetryBackoff must be >= 0")
	}
	if out.RetryBackoff == 0 {
		out.RetryBackoff = defaultTWStockBackoff
	}
	if out.Concurrency < 0 {
		return TWStockConfig{}, errors.New("datafetch: TWStock Concurrency must be >= 0")
	}
	if out.Concurrency == 0 {
		out.Concurrency = defaultTWStockConcurrency
	}
	return out, nil
}

// TWMarket selects the Taiwan exchange used by a TWStock method.
type TWMarket string

const (
	TWMarketTWSE TWMarket = "TWSE" // Taiwan Stock Exchange.
	TWMarketTPEx TWMarket = "TPEx" // Taipei Exchange.
	TWMarketAuto TWMarket = "Auto" // Try TWSE, then TPEx on no data.
)

var errTWStockNoData = errors.New("datafetch: TWStock no data")

type twStock struct {
	cfg     TWStockConfig
	client  *http.Client
	limiter *limiter.IntervalLimiter

	twseBaseURL        string
	tpexBaseURL        string
	twseOpenAPIBaseURL string
	tpexOpenAPIBaseURL string
}

// TWStock creates a client for the unauthenticated TWSE and TPEx data APIs.
func TWStock(cfg TWStockConfig) (*twStock, error) {
	normalized, err := cfg.normalize()
	if err != nil {
		return nil, err
	}
	return &twStock{
		cfg:                normalized,
		client:             &http.Client{Timeout: normalized.Timeout},
		limiter:            limiter.NewIntervalLimiter(normalized.Interval),
		twseBaseURL:        "https://www.twse.com.tw",
		tpexBaseURL:        "https://www.tpex.org.tw",
		twseOpenAPIBaseURL: "https://openapi.twse.com.tw",
		tpexOpenAPIBaseURL: "https://www.tpex.org.tw",
	}, nil
}

func (t *twStock) doJSON(rawURL string, output any) error {
	var lastErr error
	for attempt := 0; attempt <= t.cfg.Retries; attempt++ {
		if err := t.limiter.Wait(context.Background()); err != nil {
			return err
		}
		req, err := http.NewRequest(http.MethodGet, rawURL, nil)
		if err != nil {
			lastErr = err
		} else {
			req.Header.Set("User-Agent", t.cfg.UserAgent)
			resp, requestErr := t.client.Do(req)
			if requestErr != nil {
				lastErr = requestErr
			} else {
				lastErr = readJSONResponse(resp, output)
				if lastErr == nil {
					if response, ok := output.(*twStockResponse); ok {
						lastErr = responseStatus(response.Stat)
						if errors.Is(lastErr, errTWStockNoData) {
							return lastErr
						}
					}
				}
			}
		}
		if lastErr == nil {
			return nil
		}
		if attempt < t.cfg.Retries {
			t.sleepBackoff(attempt)
		}
	}
	return lastErr
}

func readJSONResponse(resp *http.Response, output any) error {
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
		return fmt.Errorf("datafetch: TWStock HTTP status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return fmt.Errorf("datafetch: TWStock read response: %w", err)
	}
	if err := json.Unmarshal(body, output); err != nil {
		return fmt.Errorf("datafetch: TWStock decode JSON: %w", err)
	}
	return nil
}

func (t *twStock) sleepBackoff(attempt int) {
	if t.cfg.RetryBackoff > 0 {
		time.Sleep(t.cfg.RetryBackoff * time.Duration(attempt+1))
	}
}

func requestURL(base, path string, values url.Values) string {
	result := strings.TrimRight(base, "/") + path
	if encoded := values.Encode(); encoded != "" {
		result += "?" + encoded
	}
	return result
}

func normalizeDate(date time.Time) time.Time {
	return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
}

func validMarket(market TWMarket) bool {
	return market == TWMarketTWSE || market == TWMarketTPEx || market == TWMarketAuto
}

type dailyPriceRecord struct {
	date             time.Time
	code, market     string
	volume, turnover any
	open, high, low  any
	close, change    any
	transactions     any
}

func (t *twStock) DailyPrices(code string, from, to time.Time, market TWMarket) (*insyra.DataTable, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, errors.New("datafetch: DailyPrices requires a non-empty code")
	}
	if normalizeDate(from).After(normalizeDate(to)) {
		return nil, errors.New("datafetch: DailyPrices from must not be after to")
	}
	if !validMarket(market) {
		return nil, fmt.Errorf("datafetch: unsupported market %q", market)
	}

	from = normalizeDate(from)
	to = normalizeDate(to)
	rows := make([]dailyPriceRecord, 0)
	month := time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, time.UTC)
	for !month.After(to) {
		monthly, err := t.dailyPricesMonth(code, month, market)
		if err != nil && !errors.Is(err, errTWStockNoData) {
			return nil, err
		}
		for _, row := range monthly {
			if !row.date.Before(from) && !row.date.After(to) {
				rows = append(rows, row)
			}
		}
		month = month.AddDate(0, 1, 0)
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].date.Before(rows[j].date) })
	return dailyPriceTable(rows), nil
}

func (t *twStock) dailyPricesMonth(code string, month time.Time, market TWMarket) ([]dailyPriceRecord, error) {
	if market == TWMarketAuto {
		rows, err := t.dailyPricesMonth(code, month, TWMarketTWSE)
		if !errors.Is(err, errTWStockNoData) {
			return rows, err
		}
		return t.dailyPricesMonth(code, month, TWMarketTPEx)
	}
	var response twStockResponse
	if market == TWMarketTWSE {
		values := url.Values{"date": {month.Format("20060102")}, "stockNo": {code}, "response": {"json"}}
		if err := t.doJSON(requestURL(t.twseBaseURL, "/rwd/zh/afterTrading/STOCK_DAY", values), &response); err != nil {
			return nil, err
		}
		if err := responseStatus(response.Stat); err != nil {
			return nil, err
		}
		if len(response.Data) == 0 {
			return nil, errTWStockNoData
		}
		indexes, err := mapHeaders(response.Fields, twseDailyHeaderAliases, []string{"Date", "Volume", "Turnover", "Open", "High", "Low", "Close", "Change", "Transactions"})
		if err != nil {
			return nil, err
		}
		return parseDailyRows(response.Data, indexes, code, "TWSE", false)
	}

	values := url.Values{"code": {code}, "date": {month.Format("2006/01/02")}, "response": {"json"}}
	if err := t.doJSON(requestURL(t.tpexBaseURL, "/www/zh-tw/afterTrading/tradingStock", values), &response); err != nil {
		return nil, err
	}
	if err := responseStatus(response.Stat); err != nil {
		return nil, err
	}
	table, err := tableFromResponse(response)
	if err != nil || len(table.Data) == 0 {
		if err == nil {
			err = errTWStockNoData
		}
		return nil, err
	}
	indexes, err := mapHeaders(table.Fields, tpexDailyHeaderAliases, []string{"Date", "Volume", "Turnover", "Open", "High", "Low", "Close", "Change", "Transactions"})
	if err != nil {
		return nil, err
	}
	return parseDailyRows(table.Data, indexes, code, "TPEx", true)
}

func parseDailyRows(rows [][]string, indexes map[string]int, code, market string, tpexUnits bool) ([]dailyPriceRecord, error) {
	result := make([]dailyPriceRecord, 0, len(rows))
	for rowIndex, row := range rows {
		date, err := parseROCDate(cell(row, indexes["Date"]))
		if err != nil {
			return nil, fmt.Errorf("datafetch: daily row %d: %w", rowIndex, err)
		}
		volume := parseInt(cell(row, indexes["Volume"]))
		turnover := parseInt(cell(row, indexes["Turnover"]))
		if tpexUnits {
			volume = multiplyInt(volume, 1000)
			turnover = multiplyInt(turnover, 1000)
		}
		result = append(result, dailyPriceRecord{
			date:         date,
			code:         code,
			market:       market,
			volume:       volume,
			turnover:     turnover,
			open:         parseFloat(cell(row, indexes["Open"])),
			high:         parseFloat(cell(row, indexes["High"])),
			low:          parseFloat(cell(row, indexes["Low"])),
			close:        parseFloat(cell(row, indexes["Close"])),
			change:       parseFloat(cell(row, indexes["Change"])),
			transactions: parseInt(cell(row, indexes["Transactions"])),
		})
	}
	return result, nil
}

func multiplyInt(value any, factor int64) any {
	if value == nil {
		return nil
	}
	return value.(int64) * factor
}

func dailyPriceTable(rows []dailyPriceRecord) *insyra.DataTable {
	dateValues := make([]any, len(rows))
	codeValues := make([]any, len(rows))
	marketValues := make([]any, len(rows))
	volumeValues := make([]any, len(rows))
	turnoverValues := make([]any, len(rows))
	openValues := make([]any, len(rows))
	highValues := make([]any, len(rows))
	lowValues := make([]any, len(rows))
	closeValues := make([]any, len(rows))
	changeValues := make([]any, len(rows))
	transactionValues := make([]any, len(rows))
	for i, row := range rows {
		dateValues[i] = row.date.UTC()
		codeValues[i], marketValues[i] = row.code, row.market
		volumeValues[i], turnoverValues[i] = row.volume, row.turnover
		openValues[i], highValues[i], lowValues[i] = row.open, row.high, row.low
		closeValues[i], changeValues[i], transactionValues[i] = row.close, row.change, row.transactions
	}
	return insyra.NewDataTable(
		newNamedCol("Date", dateValues), newNamedCol("Code", codeValues),
		newNamedCol("Volume", volumeValues), newNamedCol("Turnover", turnoverValues),
		newNamedCol("Open", openValues), newNamedCol("High", highValues), newNamedCol("Low", lowValues),
		newNamedCol("Close", closeValues), newNamedCol("Change", changeValues),
		newNamedCol("Transactions", transactionValues), newNamedCol("Market", marketValues),
	)
}

func responseStatus(stat string) error {
	if strings.EqualFold(strings.TrimSpace(stat), "ok") {
		return nil
	}
	if strings.Contains(stat, "沒有") || strings.Contains(stat, "查無") || strings.Contains(stat, "no data") {
		return errTWStockNoData
	}
	return fmt.Errorf("datafetch: TWStock payload status %q", stat)
}

type institutionalRecord struct {
	date, code, name                any
	foreignNet, trustNet, dealerNet any
	totalNet                        any
}

// InstitutionalTrades returns the day's foreign-investor, investment-trust,
// dealer, and total net trades for each security.
func (t *twStock) InstitutionalTrades(date time.Time, market TWMarket) (*insyra.DataTable, error) {
	date = normalizeDate(date)
	if !validMarket(market) {
		return nil, fmt.Errorf("datafetch: unsupported market %q", market)
	}
	rows, err := t.institutionalRows(date, market)
	if err != nil && !errors.Is(err, errTWStockNoData) {
		return nil, err
	}
	return institutionalTable(rows), nil
}

func (t *twStock) institutionalRows(date time.Time, market TWMarket) ([]institutionalRecord, error) {
	if market == TWMarketAuto {
		rows, err := t.institutionalRows(date, TWMarketTWSE)
		if !errors.Is(err, errTWStockNoData) {
			return rows, err
		}
		return t.institutionalRows(date, TWMarketTPEx)
	}
	var response twStockResponse
	if market == TWMarketTWSE {
		values := url.Values{"date": {date.Format("20060102")}, "selectType": {"ALL"}, "response": {"json"}}
		if err := t.doJSON(requestURL(t.twseBaseURL, "/rwd/zh/fund/T86", values), &response); err != nil {
			return nil, err
		}
		if err := responseStatus(response.Stat); err != nil {
			return nil, err
		}
		if len(response.Data) == 0 {
			return nil, errTWStockNoData
		}
		indexes, err := mapHeaders(response.Fields, institutionalHeaderAliases, []string{"Code", "Name", "ForeignNet", "TrustNet", "DealerNet", "TotalNet"})
		if err != nil {
			return nil, err
		}
		return parseInstitutionalRows(response.Data, indexes, date, false)
	}

	values := url.Values{"date": {date.Format("2006/01/02")}, "response": {"json"}, "type": {"Daily"}}
	if err := t.doJSON(requestURL(t.tpexBaseURL, "/www/zh-tw/insti/dailyTrade", values), &response); err != nil {
		return nil, err
	}
	if err := responseStatus(response.Stat); err != nil {
		return nil, err
	}
	table, err := tableFromResponse(response)
	if err != nil || len(table.Data) == 0 {
		if err == nil {
			err = errTWStockNoData
		}
		return nil, err
	}
	return parseInstitutionalRows(table.Data, nil, date, true, table.Fields...)
}

func parseInstitutionalRows(rows [][]string, indexes map[string]int, date time.Time, tpex bool, fields ...string) ([]institutionalRecord, error) {
	if tpex {
		var err error
		indexes, err = tpexInstitutionalIndexes(fields)
		if err != nil {
			return nil, err
		}
	}
	result := make([]institutionalRecord, 0, len(rows))
	for rowIndex, row := range rows {
		if len(row) <= indexes["TotalNet"] {
			return nil, fmt.Errorf("datafetch: institutional row %d is missing required cells", rowIndex)
		}
		result = append(result, institutionalRecord{
			date:       date,
			code:       strings.TrimSpace(cell(row, indexes["Code"])),
			name:       strings.TrimSpace(cell(row, indexes["Name"])),
			foreignNet: parseInt(cell(row, indexes["ForeignNet"])),
			trustNet:   parseInt(cell(row, indexes["TrustNet"])),
			dealerNet:  parseInt(cell(row, indexes["DealerNet"])),
			totalNet:   parseInt(cell(row, indexes["TotalNet"])),
		})
	}
	return result, nil
}

func tpexInstitutionalIndexes(fields []string) (map[string]int, error) {
	if len(fields) < 24 {
		return nil, fmt.Errorf("datafetch: TPEx institutional table is missing required header %q", "三大法人買賣超股數合計")
	}
	if fields[0] != "代號" {
		return nil, fmt.Errorf("datafetch: required header %q is missing", "代號")
	}
	if fields[1] != "名稱" {
		return nil, fmt.Errorf("datafetch: required header %q is missing", "名稱")
	}
	// TPEx publishes the institutional table as 買進/賣出/買賣超 triplets in a
	// fixed order: 外資及陸資(不含外資自營商) 2-4, 外資自營商 5-7, 外資及陸資合計
	// 8-10, 投信 11-13, 自營商(自行買賣) 14-16, 自營商(避險) 17-19, 自營商合計
	// 20-22, then 三大法人合計 at 23. ForeignNet uses the ex-dealer foreign
	// figure to match TWSE's 外陸資買賣超股數(不含外資自營商); DealerNet is the
	// dealer total.
	for _, index := range []int{4, 13, 22} {
		if fields[index] != "買賣超股數" {
			return nil, fmt.Errorf("datafetch: required header %q is missing", "買賣超股數")
		}
	}
	if fields[23] != "三大法人買賣超股數合計" {
		return nil, fmt.Errorf("datafetch: required header %q is missing", "三大法人買賣超股數合計")
	}
	return map[string]int{"Code": 0, "Name": 1, "ForeignNet": 4, "TrustNet": 13, "DealerNet": 22, "TotalNet": 23}, nil
}

func institutionalTable(rows []institutionalRecord) *insyra.DataTable {
	dateValues := make([]any, len(rows))
	codeValues := make([]any, len(rows))
	nameValues := make([]any, len(rows))
	foreignValues := make([]any, len(rows))
	trustValues := make([]any, len(rows))
	dealerValues := make([]any, len(rows))
	totalValues := make([]any, len(rows))
	for i, row := range rows {
		dateValues[i], codeValues[i], nameValues[i] = row.date, row.code, row.name
		foreignValues[i], trustValues[i] = row.foreignNet, row.trustNet
		dealerValues[i], totalValues[i] = row.dealerNet, row.totalNet
	}
	return insyra.NewDataTable(
		newNamedCol("Date", dateValues), newNamedCol("Code", codeValues), newNamedCol("Name", nameValues),
		newNamedCol("ForeignNet", foreignValues), newNamedCol("TrustNet", trustValues),
		newNamedCol("DealerNet", dealerValues), newNamedCol("TotalNet", totalValues),
	)
}

type marginRecord struct {
	date, code, name, marginBalance, shortBalance any
}

// MarginBalance returns the day's margin-buy and short-sale balances by security.
func (t *twStock) MarginBalance(date time.Time, market TWMarket) (*insyra.DataTable, error) {
	date = normalizeDate(date)
	if !validMarket(market) {
		return nil, fmt.Errorf("datafetch: unsupported market %q", market)
	}
	rows, err := t.marginRows(date, market)
	if err != nil && !errors.Is(err, errTWStockNoData) {
		return nil, err
	}
	return marginTable(rows), nil
}

func (t *twStock) marginRows(date time.Time, market TWMarket) ([]marginRecord, error) {
	if market == TWMarketAuto {
		rows, err := t.marginRows(date, TWMarketTWSE)
		if !errors.Is(err, errTWStockNoData) {
			return rows, err
		}
		return t.marginRows(date, TWMarketTPEx)
	}
	var response twStockResponse
	if market == TWMarketTWSE {
		values := url.Values{"date": {date.Format("20060102")}, "selectType": {"ALL"}, "response": {"json"}}
		if err := t.doJSON(requestURL(t.twseBaseURL, "/rwd/zh/marginTrading/MI_MARGN", values), &response); err != nil {
			return nil, err
		}
		if err := responseStatus(response.Stat); err != nil {
			return nil, err
		}
		if len(response.Tables) < 2 || len(response.Tables[1].Data) == 0 {
			return nil, errTWStockNoData
		}
		return parseTWSEMarginRows(response.Tables[1], date)
	}

	values := url.Values{"date": {date.Format("2006/01/02")}, "response": {"json"}}
	if err := t.doJSON(requestURL(t.tpexBaseURL, "/www/zh-tw/margin/balance", values), &response); err != nil {
		return nil, err
	}
	if err := responseStatus(response.Stat); err != nil {
		return nil, err
	}
	table, err := tableFromResponse(response)
	if err != nil || len(table.Data) == 0 {
		if err == nil {
			err = errTWStockNoData
		}
		return nil, err
	}
	indexes, err := mapHeaders(table.Fields, map[string]string{"代號": "Code", "名稱": "Name", "資餘額": "MarginBalance", "券餘額": "ShortBalance"}, []string{"Code", "Name", "MarginBalance", "ShortBalance"})
	if err != nil {
		return nil, err
	}
	return parseMarginRows(table.Data, indexes, date)
}

func parseTWSEMarginRows(table twStockTable, date time.Time) ([]marginRecord, error) {
	codeIndex := findHeader(table.Fields, "代號")
	nameIndex := findHeader(table.Fields, "名稱")
	balances := headerIndexes(table.Fields, "今日餘額")
	if codeIndex < 0 {
		return nil, fmt.Errorf("datafetch: required header %q is missing", "代號")
	}
	if nameIndex < 0 {
		return nil, fmt.Errorf("datafetch: required header %q is missing", "名稱")
	}
	if len(balances) < 2 {
		return nil, fmt.Errorf("datafetch: required header %q is missing twice", "今日餘額")
	}
	return parseMarginRows(table.Data, map[string]int{"Code": codeIndex, "Name": nameIndex, "MarginBalance": balances[0], "ShortBalance": balances[1]}, date)
}

func parseMarginRows(rows [][]string, indexes map[string]int, date time.Time) ([]marginRecord, error) {
	result := make([]marginRecord, 0, len(rows))
	for rowIndex, row := range rows {
		if len(row) <= indexes["ShortBalance"] {
			return nil, fmt.Errorf("datafetch: margin row %d is missing required cells", rowIndex)
		}
		result = append(result, marginRecord{
			date: date, code: strings.TrimSpace(cell(row, indexes["Code"])), name: strings.TrimSpace(cell(row, indexes["Name"])),
			marginBalance: parseInt(cell(row, indexes["MarginBalance"])), shortBalance: parseInt(cell(row, indexes["ShortBalance"])),
		})
	}
	return result, nil
}

func findHeader(fields []string, wanted string) int {
	for index, field := range fields {
		if field == wanted {
			return index
		}
	}
	return -1
}

func headerIndexes(fields []string, wanted string) []int {
	result := make([]int, 0, 2)
	for index, field := range fields {
		if field == wanted {
			result = append(result, index)
		}
	}
	return result
}

func marginTable(rows []marginRecord) *insyra.DataTable {
	dateValues := make([]any, len(rows))
	codeValues := make([]any, len(rows))
	nameValues := make([]any, len(rows))
	marginValues := make([]any, len(rows))
	shortValues := make([]any, len(rows))
	for i, row := range rows {
		dateValues[i], codeValues[i], nameValues[i] = row.date, row.code, row.name
		marginValues[i], shortValues[i] = row.marginBalance, row.shortBalance
	}
	return insyra.NewDataTable(
		newNamedCol("Date", dateValues), newNamedCol("Code", codeValues), newNamedCol("Name", nameValues),
		newNamedCol("MarginBalance", marginValues), newNamedCol("ShortBalance", shortValues),
	)
}

type quoteRecord struct {
	date, code, name                  any
	volume, turnover, open, high, low any
	close, change, transactions       any
}

// AllDailyQuotes returns the latest full-market daily quote table from an exchange.
func (t *twStock) AllDailyQuotes(market TWMarket) (*insyra.DataTable, error) {
	if !validMarket(market) {
		return nil, fmt.Errorf("datafetch: unsupported market %q", market)
	}
	rows, err := t.quoteRows(market)
	if err != nil && !errors.Is(err, errTWStockNoData) {
		return nil, err
	}
	return quoteTable(rows), nil
}

func (t *twStock) quoteRows(market TWMarket) ([]quoteRecord, error) {
	if market == TWMarketAuto {
		rows, err := t.quoteRows(TWMarketTWSE)
		if !errors.Is(err, errTWStockNoData) {
			return rows, err
		}
		return t.quoteRows(TWMarketTPEx)
	}
	if market == TWMarketTWSE {
		var response []map[string]string
		if err := t.doJSON(requestURL(t.twseOpenAPIBaseURL, "/v1/exchangeReport/STOCK_DAY_ALL", nil), &response); err != nil {
			return nil, err
		}
		if len(response) == 0 {
			return nil, errTWStockNoData
		}
		return parseQuoteRows(response, twseQuoteHeaderAliases)
	}

	var response []map[string]string
	if err := t.doJSON(requestURL(t.tpexOpenAPIBaseURL, "/openapi/v1/tpex_mainboard_daily_close_quotes", nil), &response); err != nil {
		return nil, err
	}
	if len(response) == 0 {
		return nil, errTWStockNoData
	}
	return parseQuoteRows(response, tpexQuoteHeaderAliases)
}

func parseQuoteRows(rows []map[string]string, aliases map[string]string) ([]quoteRecord, error) {
	required := []string{"Date", "Code", "Name", "Volume", "Turnover", "Open", "High", "Low", "Close", "Change", "Transactions"}
	result := make([]quoteRecord, 0, len(rows))
	for rowIndex, row := range rows {
		values := make(map[string]string, len(aliases))
		for source, destination := range aliases {
			value, exists := row[source]
			if !exists {
				return nil, fmt.Errorf("datafetch: quote row %d is missing required field %q", rowIndex, source)
			}
			values[destination] = value
		}
		for _, name := range required {
			if _, exists := values[name]; !exists {
				return nil, fmt.Errorf("datafetch: quote row %d is missing required field %q", rowIndex, requiredHeaderName(name, aliases))
			}
		}
		date, err := parseROCDate(values["Date"])
		if err != nil {
			return nil, fmt.Errorf("datafetch: quote row %d: %w", rowIndex, err)
		}
		result = append(result, quoteRecord{
			date: date, code: strings.TrimSpace(values["Code"]), name: strings.TrimSpace(values["Name"]),
			volume: parseInt(values["Volume"]), turnover: parseInt(values["Turnover"]),
			open: parseFloat(values["Open"]), high: parseFloat(values["High"]), low: parseFloat(values["Low"]),
			close: parseFloat(values["Close"]), change: parseFloat(values["Change"]), transactions: parseInt(values["Transactions"]),
		})
	}
	return result, nil
}

func quoteTable(rows []quoteRecord) *insyra.DataTable {
	dateValues := make([]any, len(rows))
	codeValues := make([]any, len(rows))
	nameValues := make([]any, len(rows))
	volumeValues := make([]any, len(rows))
	turnoverValues := make([]any, len(rows))
	openValues := make([]any, len(rows))
	highValues := make([]any, len(rows))
	lowValues := make([]any, len(rows))
	closeValues := make([]any, len(rows))
	changeValues := make([]any, len(rows))
	transactionValues := make([]any, len(rows))
	for i, row := range rows {
		dateValues[i], codeValues[i], nameValues[i] = row.date, row.code, row.name
		volumeValues[i], turnoverValues[i] = row.volume, row.turnover
		openValues[i], highValues[i], lowValues[i] = row.open, row.high, row.low
		closeValues[i], changeValues[i], transactionValues[i] = row.close, row.change, row.transactions
	}
	return insyra.NewDataTable(
		newNamedCol("Date", dateValues), newNamedCol("Code", codeValues), newNamedCol("Name", nameValues),
		newNamedCol("Volume", volumeValues), newNamedCol("Turnover", turnoverValues),
		newNamedCol("Open", openValues), newNamedCol("High", highValues), newNamedCol("Low", lowValues),
		newNamedCol("Close", closeValues), newNamedCol("Change", changeValues), newNamedCol("Transactions", transactionValues),
	)
}
