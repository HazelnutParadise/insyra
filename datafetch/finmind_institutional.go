package datafetch

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/HazelnutParadise/insyra"
)

// InstitutionalFutures returns the three institutional trader groups' daily
// trading and open interest in one futures contract, from FinMind, with the
// columns of TAIFEXClient.InstitutionalFutures so that it can be joined to
// TAIFEXClient.InstitutionalFuturesHistory. It is InstitutionalFuturesContext
// with context.Background().
func (c *FinMindClient) InstitutionalFutures(contract string, from, to time.Time) (*insyra.DataTable, error) {
	return c.InstitutionalFuturesContext(context.Background(), contract, from, to)
}

// InstitutionalFuturesContext is InstitutionalFutures with a context. The
// limiter wait, each request and each retry backoff stop when ctx is done.
//
// Unlike TAIFEXClient.InstitutionalFutures, which gives the latest day of every
// contract, this gives one contract over a range of days. contract is the
// exchange's product code, such as "TX" (臺股期貨), "MTX" or "TMF", and is
// required; FinMind keys the data by it, so another source's alias, such as
// "TXF" for TX, returns an empty table. A zero from or to leaves that end open. FinMind has this data from
// 2018-06-05 (measured for TX), which reaches back before the three years the
// exchange's website serves.
//
// The rows are in date order, oldest first, and within a day in the
// exchange's order: 自營商, 投信, 外資及陸資. Contract holds the exchange's
// name for the contract (臺股期貨 for TX), Trader the exchange's name for the
// group (FinMind's 外資 becomes 外資及陸資), and the Net columns are Long minus
// Short. The counts match the exchange's tables exactly, and the ...Value
// columns, in thousands of NT dollars, match them except TradeNetValue and
// OINetValue, which can differ by 1: the exchange subtracts before rounding to
// thousands, and FinMind sends only the rounded sides.
//
// The first call on a client makes two requests to FinMind, one for the
// contract names and one for the data; later calls make one. Both count toward
// the plan's request limit.
func (c *FinMindClient) InstitutionalFuturesContext(ctx context.Context, contract string, from, to time.Time) (*insyra.DataTable, error) {
	return c.institutional(ctx, false, contract, from, to)
}

// InstitutionalOptions returns the three institutional trader groups' daily
// trading and open interest in one options contract, call and put separately,
// from FinMind, with the columns of TAIFEXClient.InstitutionalOptions. It is
// InstitutionalOptionsContext with context.Background().
func (c *FinMindClient) InstitutionalOptions(contract string, from, to time.Time) (*insyra.DataTable, error) {
	return c.InstitutionalOptionsContext(context.Background(), contract, from, to)
}

// InstitutionalOptionsContext is InstitutionalOptions with a context. It works
// as InstitutionalFuturesContext does, for an options contract such as "TXO"
// (臺指選擇權). CallPut is "CALL" or "PUT", as the exchange writes it, and
// within a day the calls come before the puts.
func (c *FinMindClient) InstitutionalOptionsContext(ctx context.Context, contract string, from, to time.Time) (*insyra.DataTable, error) {
	return c.institutional(ctx, true, contract, from, to)
}

// finMindTraders maps FinMind's names for the trader groups to the exchange's,
// in the exchange's order.
var finMindTraders = []struct{ finMind, taifex string }{
	{"自營商", "自營商"},
	{"投信", "投信"},
	{"外資", "外資及陸資"},
}

func (c *FinMindClient) institutional(ctx context.Context, options bool, contract string, from, to time.Time) (*insyra.DataTable, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	if c == nil || c.base == nil {
		return nil, errFinMindNotCreated
	}
	if err := c.base.usable(); err != nil {
		return nil, err
	}
	contract = strings.TrimSpace(contract)
	if contract == "" {
		return nil, errors.New("datafetch: FinMind institutional data needs a contract code, such as \"TX\"")
	}
	if !from.IsZero() && !to.IsZero() && normalizeDate(to).Before(normalizeDate(from)) {
		return nil, fmt.Errorf("datafetch: FinMind needs from (%s) not after to (%s)", from.Format("2006-01-02"), to.Format("2006-01-02"))
	}
	dataset, idKey, kind := "TaiwanFuturesInstitutionalInvestors", "futures_id", "TaiwanFuturesDaily"
	if options {
		dataset, idKey, kind = "TaiwanOptionInstitutionalInvestors", "option_id", "TaiwanOptionDaily"
	}
	name, err := c.contractName(ctx, kind, contract)
	if err != nil {
		return nil, err
	}
	_, rows, err := c.rows(ctx, dataset, contract, from, to)
	if err != nil {
		return nil, err
	}

	columns := institutionalColumns(options)
	type record struct {
		date    time.Time
		callPut int
		trader  int
		values  []any
	}
	records := make([]record, 0, len(rows))
	for i, row := range rows {
		if id, _ := row[idKey].(string); id != "" && id != contract {
			return nil, fmt.Errorf("datafetch: FinMind %s row %d is for %q, not %q", dataset, i, id, contract)
		}
		dateText, _ := row["date"].(string)
		date, err := time.Parse("2006-01-02", dateText)
		if err != nil {
			return nil, fmt.Errorf("datafetch: FinMind %s row %d has date %v", dataset, i, row["date"])
		}
		traderText, _ := row["institutional_investors"].(string)
		trader := -1
		for t, names := range finMindTraders {
			if traderText == names.finMind || traderText == names.taifex {
				trader = t
			}
		}
		if trader < 0 {
			return nil, fmt.Errorf("datafetch: FinMind %s row %d has trader group %q", dataset, i, traderText)
		}
		rec := record{date: date, trader: trader}
		values := []any{date, name}
		if options {
			switch row["call_put"] {
			case "買權":
				values = append(values, "CALL")
			case "賣權":
				rec.callPut = 1
				values = append(values, "PUT")
			default:
				return nil, fmt.Errorf("datafetch: FinMind %s row %d has call_put %v", dataset, i, row["call_put"])
			}
		}
		values = append(values, finMindTraders[trader].taifex)
		for _, side := range []string{"deal", "open_interest_balance"} {
			long, err := finMindInt(row, "long_"+side+"_volume")
			if err != nil {
				return nil, fmt.Errorf("datafetch: FinMind %s row %d: %w", dataset, i, err)
			}
			longValue, err := finMindInt(row, "long_"+side+"_amount")
			if err != nil {
				return nil, fmt.Errorf("datafetch: FinMind %s row %d: %w", dataset, i, err)
			}
			short, err := finMindInt(row, "short_"+side+"_volume")
			if err != nil {
				return nil, fmt.Errorf("datafetch: FinMind %s row %d: %w", dataset, i, err)
			}
			shortValue, err := finMindInt(row, "short_"+side+"_amount")
			if err != nil {
				return nil, fmt.Errorf("datafetch: FinMind %s row %d: %w", dataset, i, err)
			}
			values = append(values, long, longValue, short, shortValue, long-short, longValue-shortValue)
		}
		rec.values = values
		records = append(records, rec)
	}
	sort.SliceStable(records, func(i, j int) bool {
		a, b := records[i], records[j]
		if !a.date.Equal(b.date) {
			return a.date.Before(b.date)
		}
		if a.callPut != b.callPut {
			return a.callPut < b.callPut
		}
		return a.trader < b.trader
	})

	lists := make([]*insyra.DataList, len(columns))
	for col := range columns {
		values := make([]any, len(records))
		for r, rec := range records {
			values[r] = rec.values[col]
		}
		lists[col] = newNamedCol(columns[col].name, values)
	}
	return insyra.NewDataTable(lists...), nil
}

// finMindInt reads an integer field of a FinMind row.
func finMindInt(row map[string]any, key string) (int64, error) {
	value, ok := row[key]
	if !ok {
		return 0, fmt.Errorf("missing field %q", key)
	}
	n, ok := finMindValue(value).(int64)
	if !ok {
		return 0, fmt.Errorf("field %q is %v, not an integer", key, value)
	}
	return n, nil
}

// taifexContractNames corrects FinMind's TaiwanFutOptDailyInfo where its name
// for a contract is not the exchange's, so that Contract joins with the TAIFEX
// tables. Every other futures and options code had the exchange's name when
// the two were compared on 2026-11-07.
var taifexContractNames = map[string]string{
	"TaiwanFuturesDaily\x00GTF": "櫃買指數期貨", // FinMind: 櫃買期貨
}

// contractName returns the exchange's name for a futures or options product
// code. The names come from FinMind's TaiwanFutOptDailyInfo, read on a
// client's first call and again when a code is not in the names read before,
// such as a contract listed since.
func (c *FinMindClient) contractName(ctx context.Context, kind, code string) (string, error) {
	select {
	case c.namesLock <- struct{}{}:
	case <-ctx.Done():
		return "", ctx.Err()
	}
	defer func() { <-c.namesLock }()
	key := kind + "\x00" + code
	name, ok := c.names[key]
	if !ok {
		_, rows, err := c.rows(ctx, "TaiwanFutOptDailyInfo", "", time.Time{}, time.Time{})
		if err != nil {
			return "", fmt.Errorf("datafetch: FinMind contract names: %w", err)
		}
		names := map[string]string{}
		for _, row := range rows {
			rowKind, _ := row["type"].(string)
			rowCode, _ := row["code"].(string)
			rowName, _ := row["name"].(string)
			if rowKind != "" && rowCode != "" && rowName != "" {
				names[rowKind+"\x00"+rowCode] = rowName
			}
		}
		for k, v := range taifexContractNames {
			if _, listed := names[k]; listed {
				names[k] = v
			}
		}
		c.names = names
		if name, ok = names[key]; !ok {
			return "", fmt.Errorf("datafetch: FinMind has no %s contract with code %q", strings.TrimPrefix(strings.TrimSuffix(kind, "Daily"), "Taiwan"), code)
		}
	}
	return name, nil
}
