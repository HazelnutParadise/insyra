package datafetch

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/HazelnutParadise/insyra"
	json "github.com/goccy/go-json"
)

// TAIFEXClient reads the Taiwan Futures Exchange's open data (OpenAPI), which
// needs no key. Create one with TAIFEX. A client that did not come from TAIFEX,
// such as a zero value, returns an error from every method.
//
// The OpenAPI gives only the latest trading day for most tables, and the last
// 20 days for the put/call ratio. Longer histories come from the exchange's
// website, which this client does not read.
type TAIFEXClient struct {
	base    *TWStockClient
	baseURL string
}

// TAIFEX creates a client for the exchange's OpenAPI. It takes the same
// settings as TWStock: Timeout, Interval, UserAgent, Retries and RetryBackoff.
func TAIFEX(cfg TWStockConfig) (*TAIFEXClient, error) {
	base, err := TWStock(cfg)
	if err != nil {
		return nil, err
	}
	return &TAIFEXClient{base: base, baseURL: "https://openapi.taifex.com.tw"}, nil
}

var errTAIFEXNotCreated = errors.New("datafetch: TAIFEXClient was not created with TAIFEX")

func (c *TAIFEXClient) usable() error {
	if c == nil || c.base == nil {
		return errTAIFEXNotCreated
	}
	return c.base.usable()
}

// column describes one output column: its name, the keys the source may call it
// (the JSON key, and the Chinese header the exchange uses in its CSV replies),
// and how to read it.
type column struct {
	name string
	keys []string
	kind columnKind
}

type columnKind int

const (
	kindText columnKind = iota
	kindInt
	kindFloat
	kindDate
)

// fetchTable requests path and builds a table from its records.
func (c *TAIFEXClient) fetchTable(ctx context.Context, path string, columns []column) (*insyra.DataTable, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	if err := c.usable(); err != nil {
		return nil, err
	}
	var records []map[string]string
	err := c.base.doGet(ctx, requestURL(c.baseURL, path, nil), func(resp *http.Response) error {
		body, err := readBodyResponse(resp, 64<<20)
		if err != nil {
			return err
		}
		records, err = parseTAIFEXRecords(body)
		return err
	})
	if err != nil {
		return nil, err
	}
	return taifexTable(path, records, columns)
}

// parseTAIFEXRecords reads a reply that is a JSON array of objects or, as
// some of the exchange's tables are whatever the Accept header says, a CSV with
// a header row. Both give string-keyed records.
func parseTAIFEXRecords(body []byte) ([]map[string]string, error) {
	body = bytes.TrimSpace(bytes.TrimPrefix(body, []byte("\xef\xbb\xbf")))
	if len(body) == 0 {
		return nil, nil
	}
	if body[0] == '[' {
		var records []map[string]string
		if err := json.Unmarshal(body, &records); err != nil {
			return nil, fmt.Errorf("datafetch: TAIFEX decode JSON: %w", err)
		}
		return records, nil
	}
	reader := csv.NewReader(bytes.NewReader(body))
	reader.FieldsPerRecord = -1
	var header []string
	var records []map[string]string
	for {
		row, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("datafetch: TAIFEX decode CSV: %w", err)
		}
		if header == nil {
			header = row
			continue
		}
		record := make(map[string]string, len(header))
		for i, name := range header {
			if i < len(row) {
				record[strings.TrimSpace(name)] = row[i]
			}
		}
		records = append(records, record)
	}
	return records, nil
}

func taifexTable(path string, records []map[string]string, columns []column) (*insyra.DataTable, error) {
	values := make([][]any, len(columns))
	for c := range values {
		values[c] = make([]any, len(records))
	}
	for r, record := range records {
		for c, col := range columns {
			raw, found := "", false
			for _, key := range col.keys {
				if v, ok := record[key]; ok {
					raw, found = v, true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("datafetch: TAIFEX %s record %d is missing field %q", path, r, col.keys[0])
			}
			switch col.kind {
			case kindInt:
				values[c][r] = parseInt(raw)
			case kindFloat:
				values[c][r] = parseFloat(raw)
			case kindDate:
				d, err := time.Parse("20060102", strings.TrimSpace(raw))
				if err != nil {
					return nil, fmt.Errorf("datafetch: TAIFEX %s record %d has date %q", path, r, raw)
				}
				values[c][r] = d
			default:
				values[c][r] = strings.TrimSpace(raw)
			}
		}
	}
	lists := make([]*insyra.DataList, len(columns))
	for c, col := range columns {
		lists[c] = newNamedCol(col.name, values[c])
	}
	return insyra.NewDataTable(lists...), nil
}

// institutionalColumns are the columns of the institutional traders' tables.
func institutionalColumns(withCallPut bool) []column {
	cols := []column{
		{"Date", []string{"Date", "日期"}, kindDate},
		{"Contract", []string{"ContractCode"}, kindText},
	}
	if withCallPut {
		cols = append(cols, column{"CallPut", []string{"CallPut"}, kindText})
	}
	return append(cols,
		column{"Trader", []string{"Item"}, kindText},
		column{"TradeLong", []string{"TradingVolume(Long)"}, kindInt},
		column{"TradeLongValue", []string{"TradingValue(Long)(Thousands)"}, kindInt},
		column{"TradeShort", []string{"TradingVolume(Short)"}, kindInt},
		column{"TradeShortValue", []string{"TradingValue(Short)(Thousands)"}, kindInt},
		column{"TradeNet", []string{"TradingVolume(Net)"}, kindInt},
		column{"TradeNetValue", []string{"TradingValue(Net)(Thousands)"}, kindInt},
		column{"OILong", []string{"OpenInterest(Long)"}, kindInt},
		column{"OILongValue", []string{"ContractValueofOpenInterest(Long)(Thousands)"}, kindInt},
		column{"OIShort", []string{"OpenInterest(Short)"}, kindInt},
		column{"OIShortValue", []string{"ContractValueofOpenInterest(Short)(Thousands)"}, kindInt},
		column{"OINet", []string{"OpenInterest(Net)"}, kindInt},
		column{"OINetValue", []string{"ContractValueofOpenInterest(Net)(Thousands)"}, kindInt},
	)
}

// InstitutionalFutures returns the latest day's trading and open interest of
// the three institutional trader groups in each futures contract. It is
// InstitutionalFuturesContext with context.Background().
func (c *TAIFEXClient) InstitutionalFutures() (*insyra.DataTable, error) {
	return c.InstitutionalFuturesContext(context.Background())
}

// InstitutionalFuturesContext is InstitutionalFutures with a context.
//
// One row per contract and trader group (自營商 dealers, 投信 investment trusts,
// 外資及陸資 foreign investors). Date is a time.Time, Contract and Trader are
// strings, the Trade and OI columns for Long, Short and Net are contract counts
// (int64), and the matching ...Value columns are in thousands of NT dollars.
func (c *TAIFEXClient) InstitutionalFuturesContext(ctx context.Context) (*insyra.DataTable, error) {
	return c.fetchTable(ctx, "/v1/MarketDataOfMajorInstitutionalTradersDetailsOfFuturesContractsBytheDate", institutionalColumns(false))
}

// InstitutionalOptions returns the latest day's trading and open interest of
// the three institutional trader groups in each options contract, call and put
// separately. It is InstitutionalOptionsContext with context.Background().
func (c *TAIFEXClient) InstitutionalOptions() (*insyra.DataTable, error) {
	return c.InstitutionalOptionsContext(context.Background())
}

// InstitutionalOptionsContext is InstitutionalOptions with a context. The
// columns are those of InstitutionalFutures plus CallPut, "CALL" or "PUT".
func (c *TAIFEXClient) InstitutionalOptionsContext(ctx context.Context) (*insyra.DataTable, error) {
	return c.fetchTable(ctx, "/v1/MarketDataOfMajorInstitutionalTradersDetailsOfCallsAndPutsBytheDate", institutionalColumns(true))
}

func largeTraderColumns(withCallPut bool) []column {
	cols := []column{
		{"Date", []string{"Date", "日期"}, kindDate},
		{"Contract", []string{"Contract", "契約"}, kindText},
		{"ContractName", []string{"ContractName", "商品名稱(契約名稱)"}, kindText},
	}
	if withCallPut {
		cols = append(cols, column{"CallPut", []string{"CallPut", "買賣權"}, kindText})
	}
	return append(cols,
		column{"SettlementMonth", []string{"SettlementMonth", "到期月份(週別)"}, kindText},
		column{"TraderType", []string{"TypeOfTraders", "交易人類別"}, kindText},
		column{"Top5Buy", []string{"Top5Buy", "前五大交易人買方", "前五大交易人買方數量"}, kindInt},
		column{"Top5Sell", []string{"Top5Sell", "前五大交易人賣方", "前五大交易人賣方數量"}, kindInt},
		column{"Top10Buy", []string{"Top10Buy", "前十大交易人買方", "前十大交易人買方數量"}, kindInt},
		column{"Top10Sell", []string{"Top10Sell", "前十大交易人賣方", "前十大交易人賣方數量"}, kindInt},
		column{"MarketOI", []string{"OIOfMarket", "全市場未沖銷部位數"}, kindInt},
	)
}

// LargeTradersFutures returns the latest day's open positions of the largest
// traders in each futures contract. It is LargeTradersFuturesContext with
// context.Background().
func (c *TAIFEXClient) LargeTradersFutures() (*insyra.DataTable, error) {
	return c.LargeTradersFuturesContext(context.Background())
}

// LargeTradersFuturesContext is LargeTradersFutures with a context.
//
// One row per contract, settlement month and trader type. Top5Buy, Top5Sell,
// Top10Buy and Top10Sell are the open long and short positions held by the 5 and
// 10 largest traders, and MarketOI the whole market's open interest, all in
// contracts (int64). SettlementMonth is kept as the exchange publishes it, a
// string such as "202610", and "999912" is its total over all months.
// TraderType is "0" for all traders and "1" for the specific institutional
// traders.
func (c *TAIFEXClient) LargeTradersFuturesContext(ctx context.Context) (*insyra.DataTable, error) {
	return c.fetchTable(ctx, "/v1/OpenInterestOfLargeTradersFutures", largeTraderColumns(false))
}

// LargeTradersOptions is LargeTradersFutures for options. It is
// LargeTradersOptionsContext with context.Background().
func (c *TAIFEXClient) LargeTradersOptions() (*insyra.DataTable, error) {
	return c.LargeTradersOptionsContext(context.Background())
}

// LargeTradersOptionsContext is LargeTradersOptions with a context. The columns
// are those of LargeTradersFutures plus CallPut, "買權" (call) or "賣權" (put).
func (c *TAIFEXClient) LargeTradersOptionsContext(ctx context.Context) (*insyra.DataTable, error) {
	return c.fetchTable(ctx, "/v1/OpenInterestOfLargeTradersOptions", largeTraderColumns(true))
}

// PutCallRatio returns the TAIEX options' put/call ratios of the last 20
// trading days, newest first. It is PutCallRatioContext with
// context.Background().
func (c *TAIFEXClient) PutCallRatio() (*insyra.DataTable, error) {
	return c.PutCallRatioContext(context.Background())
}

// PutCallRatioContext is PutCallRatio with a context. Columns: Date
// (time.Time), PutVolume, CallVolume, PutOI, CallOI (int64) and
// VolumeRatio, OIRatio (float64, in percent: 88.75 means the put volume is 88.75%
// of the call volume).
func (c *TAIFEXClient) PutCallRatioContext(ctx context.Context) (*insyra.DataTable, error) {
	return c.fetchTable(ctx, "/v1/PutCallRatio", []column{
		{"Date", []string{"Date", "日期"}, kindDate},
		{"PutVolume", []string{"PutVolume"}, kindInt},
		{"CallVolume", []string{"CallVolume"}, kindInt},
		{"VolumeRatio", []string{"PutCallVolumeRatio%"}, kindFloat},
		{"PutOI", []string{"PutOI"}, kindInt},
		{"CallOI", []string{"CallOI"}, kindInt},
		{"OIRatio", []string{"PutCallOIRatio%"}, kindFloat},
	})
}
