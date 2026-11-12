package datafetch

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/HazelnutParadise/insyra"
)

// TWCentralBankClient reads interest rates from the open data of the Central
// Bank of the Republic of China (Taiwan), which needs no key. Create one with
// TWCentralBank. A client that did not come from TWCentralBank, such as a zero
// value, returns an error from every method.
type TWCentralBankClient struct {
	base    *TWStockClient
	baseURL string
}

// TWCentralBank creates a client for the central bank's open data. It takes the
// same settings as TWStock: Timeout, Interval, UserAgent, Retries and
// RetryBackoff.
func TWCentralBank(cfg TWStockConfig) (*TWCentralBankClient, error) {
	base, err := TWStock(cfg)
	if err != nil {
		return nil, err
	}
	return &TWCentralBankClient{base: base, baseURL: "https://www.cbc.gov.tw"}, nil
}

var errTWCentralBankNotCreated = errors.New("datafetch: TWCentralBankClient was not created with TWCentralBank")

const (
	// twcbOvernightPath is 金融業拆款利率 (日資料), data.gov.tw dataset 10808.
	twcbOvernightPath = "/public/data/OpenData/經研處/EG37D01.csv"
	// twcbFiveBankPath is 五大銀行存放款利率歷史月資料, data.gov.tw dataset 10359.
	twcbFiveBankPath = "/public/data/OpenData/A13Rate.csv"
)

// OvernightRate returns the interbank overnight call-loan rate (金融業隔夜拆款
// 利率) for every day from from to to. It is OvernightRateContext with
// context.Background().
func (c *TWCentralBankClient) OvernightRate(from, to time.Time) (*insyra.DataTable, error) {
	return c.OvernightRateContext(context.Background(), from, to)
}

// OvernightRateContext is OvernightRate with a context. The limiter wait, the
// request and each retry backoff stop when ctx is done.
//
// The table has one row per day, oldest first: Date (time.Time), Rate, the
// weighted average, and High and Low, the highest and lowest rate traded that
// day (float64, percent per year as the central bank publishes them: 0.812
// means 0.812%). A day without overnight trades, as on some days before the
// Lunar New Year, has nil rates. The series starts on 1991-10-01. The bank
// updates it about once a month, so the last month or so may not be there yet.
// A zero from or to leaves that end open.
//
// quant takes a risk-free rate per period as a fraction; for daily returns, one
// day's rate is math.Pow(1+Rate/100, 1.0/252) - 1, or Rate/100/252 as a simple
// approximation.
func (c *TWCentralBankClient) OvernightRateContext(ctx context.Context, from, to time.Time) (*insyra.DataTable, error) {
	from, to, err := c.checkRange(ctx, from, to)
	if err != nil {
		return nil, err
	}
	records, err := c.download(ctx, twcbOvernightPath)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, errors.New("datafetch: TWCentralBank overnight rates: the file is empty")
	}
	indexes, err := mapHeaders(records[0], map[string]string{
		"日": "Date", "隔夜-加權平均-原始值(%)": "Rate", "隔夜-最高-原始值(%)": "High", "隔夜-最低-原始值(%)": "Low",
	}, []string{"Date", "Rate", "High", "Low"})
	if err != nil {
		return nil, fmt.Errorf("datafetch: TWCentralBank overnight rates: %w", err)
	}
	var dates, rates, highs, lows []any
	for i, row := range records[1:] {
		raw := strings.TrimSpace(row[indexes["Date"]])
		if strings.HasSuffix(raw, "00") && len(raw) == 8 {
			continue // a month's summary row, such as 20131000
		}
		date, err := time.Parse("20060102", raw)
		if err != nil {
			return nil, fmt.Errorf("datafetch: TWCentralBank overnight rates row %d has date %q", i+1, raw)
		}
		if !inRange(date, from, to) {
			continue
		}
		dates = append(dates, date)
		rates = append(rates, parseFloat(row[indexes["Rate"]]))
		highs = append(highs, parseFloat(row[indexes["High"]]))
		lows = append(lows, parseFloat(row[indexes["Low"]]))
	}
	return insyra.NewDataTable(
		newNamedCol("Date", dates), newNamedCol("Rate", rates),
		newNamedCol("High", highs), newNamedCol("Low", lows),
	), nil
}

// DepositRate returns the five major banks' average posted rate for a one-year
// fixed-rate time deposit (五大銀行一年期定期存款固定利率牌告利率平均), monthly,
// for the months from from to to. It is DepositRateContext with
// context.Background().
func (c *TWCentralBankClient) DepositRate(from, to time.Time) (*insyra.DataTable, error) {
	return c.DepositRateContext(context.Background(), from, to)
}

// DepositRateContext is DepositRate with a context. The limiter wait, the
// request and each retry backoff stop when ctx is done.
//
// The table has one row per month, oldest first: Month (time.Time, the first
// day of the month) and Rate (float64, percent per year), the mean of the five
// banks' 定存利率-一年期-固定 in the central bank's five-bank file. The banks are
// those the bank names: 臺灣銀行, 合作金庫, 第一銀行, 華南銀行 and 彰化銀行 to
// October 2008, and 臺灣土地銀行 in place of 彰化銀行 from November 2008. The
// central bank's own 一年期存款牌告利率 is this mean rounded to two decimals.
//
// Rate is nil for a month the file does not give all five banks for. Measured on
// 2026-11-12, the file gave all five through November 2025 and only 臺灣土地銀行
// after it, so those later months are nil rather than one bank's rate. The
// series starts in January 2001. A zero from or to leaves that end open; a month
// is included when its first day is in the range of from's month to to.
//
// quant takes a risk-free rate per period as a fraction; for monthly returns,
// one month's rate is Rate/100/12.
func (c *TWCentralBankClient) DepositRateContext(ctx context.Context, from, to time.Time) (*insyra.DataTable, error) {
	from, to, err := c.checkRange(ctx, from, to)
	if err != nil {
		return nil, err
	}
	if !from.IsZero() {
		from = time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, time.UTC)
	}
	records, err := c.download(ctx, twcbFiveBankPath)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, errors.New("datafetch: TWCentralBank deposit rates: the file is empty")
	}
	indexes, err := mapHeaders(records[0], map[string]string{
		"銀行": "Bank", "年月": "Month", "定存利率-一年期-固定": "Rate",
	}, []string{"Bank", "Month", "Rate"})
	if err != nil {
		return nil, fmt.Errorf("datafetch: TWCentralBank deposit rates: %w", err)
	}
	banks := map[time.Time]map[string]float64{}
	for i, row := range records[1:] {
		raw := strings.TrimSpace(row[indexes["Month"]])
		month, err := parseROCMonth(raw)
		if err != nil {
			return nil, fmt.Errorf("datafetch: TWCentralBank deposit rates row %d has month %q", i+1, raw)
		}
		if !inRange(month, from, to) {
			continue
		}
		if banks[month] == nil {
			banks[month] = map[string]float64{}
		}
		if rate, ok := parseFloat(row[indexes["Rate"]]).(float64); ok {
			banks[month][strings.TrimSpace(row[indexes["Bank"]])] = rate
		}
	}
	months := make([]time.Time, 0, len(banks))
	for month := range banks {
		months = append(months, month)
	}
	sort.Slice(months, func(i, j int) bool { return months[i].Before(months[j]) })
	monthValues := make([]any, len(months))
	rateValues := make([]any, len(months))
	for i, month := range months {
		monthValues[i] = month
		rateValues[i] = fiveBankMean(month, banks[month])
	}
	return insyra.NewDataTable(newNamedCol("Month", monthValues), newNamedCol("Rate", rateValues)), nil
}

// fiveBankMean is the mean of the five banks the central bank averages for a
// month, or nil when the file lacks one of them.
func fiveBankMean(month time.Time, rates map[string]float64) any {
	members := []string{"臺灣銀行", "合作金庫", "第一銀行", "華南銀行", "彰化銀行"}
	if !month.Before(time.Date(2008, time.November, 1, 0, 0, 0, 0, time.UTC)) {
		members[4] = "土地銀行"
	}
	sum := 0.0
	for _, bank := range members {
		rate, ok := rates[bank]
		if !ok {
			return nil
		}
		sum += rate
	}
	// The banks post three decimals, so four keep the mean exact without the
	// float noise of the division.
	return math.Round(sum/float64(len(members))*1e4) / 1e4
}

// parseROCMonth reads the five-bank file's month, a ROC year and month such as
// 11409 for September 2025.
func parseROCMonth(raw string) (time.Time, error) {
	if len(raw) < 4 || len(raw) > 5 {
		return time.Time{}, fmt.Errorf("invalid ROC month %q", raw)
	}
	year, err := strconv.Atoi(raw[:len(raw)-2])
	if err != nil {
		return time.Time{}, err
	}
	month, err := strconv.Atoi(raw[len(raw)-2:])
	if err != nil || month < 1 || month > 12 || year < 1 {
		return time.Time{}, fmt.Errorf("invalid ROC month %q", raw)
	}
	return time.Date(year+1911, time.Month(month), 1, 0, 0, 0, 0, time.UTC), nil
}

func inRange(date, from, to time.Time) bool {
	return (from.IsZero() || !date.Before(from)) && (to.IsZero() || !date.After(to))
}

func (c *TWCentralBankClient) checkRange(ctx context.Context, from, to time.Time) (time.Time, time.Time, error) {
	if err := contextErr(ctx); err != nil {
		return from, to, err
	}
	if c == nil || c.base == nil {
		return from, to, errTWCentralBankNotCreated
	}
	if err := c.base.usable(); err != nil {
		return from, to, err
	}
	if !from.IsZero() {
		from = normalizeDate(from)
	}
	if !to.IsZero() {
		to = normalizeDate(to)
	}
	if !from.IsZero() && !to.IsZero() && to.Before(from) {
		return from, to, fmt.Errorf("datafetch: TWCentralBank needs from (%s) not after to (%s)", from.Format("2006-01-02"), to.Format("2006-01-02"))
	}
	return from, to, nil
}

// download reads one of the bank's UTF-8 CSV files.
func (c *TWCentralBankClient) download(ctx context.Context, path string) ([][]string, error) {
	var records [][]string
	err := c.base.doGet(ctx, requestURL(c.baseURL, path, nil), func(resp *http.Response) error {
		body, err := readBodyResponse(resp, 64<<20)
		if err != nil {
			return err
		}
		reader := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(body, []byte("\xef\xbb\xbf"))))
		reader.FieldsPerRecord = -1
		records = nil
		for {
			row, err := reader.Read()
			if errors.Is(err, io.EOF) {
				return nil
			}
			if err != nil {
				return &finalError{err: fmt.Errorf("datafetch: TWCentralBank %s: %w", path, err)}
			}
			if records != nil && len(row) < len(records[0]) {
				return &finalError{err: fmt.Errorf("datafetch: TWCentralBank %s row %d has %d fields, want %d", path, len(records), len(row), len(records[0]))}
			}
			records = append(records, row)
		}
	})
	return records, err
}
