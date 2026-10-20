package datafetch

import (
	"errors"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"time"

	"github.com/HazelnutParadise/insyra"
	"golang.org/x/text/encoding/traditionalchinese"
)

// taifexHistory describes one of the exchange website's CSV downloads.
type taifexHistory struct {
	path    string
	columns []column
	// commodity says the download takes an empty commodityId, which means every
	// contract.
	commodity bool
}

// ErrTAIFEXRefused matches (with errors.Is) an error returned because the
// exchange website refused a history range: the table does not reach back that
// far, or the range is longer than the website allows. A caller can move from
// forward and ask again.
var ErrTAIFEXRefused = errors.New("datafetch: TAIFEX website refused the date range")

var taifexAlert = regexp.MustCompile(`alert\("([^"]*)"\)`)

// historyTable downloads the website's CSV for each calendar month of
// [from, to], joins the rows in date order, oldest first, and builds the table
// of the same columns as the matching OpenAPI method.
func (c *TAIFEXClient) historyTable(ctx context.Context, h taifexHistory, from, to time.Time) (*insyra.DataTable, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	if err := c.usable(); err != nil {
		return nil, err
	}
	if from.IsZero() || to.IsZero() {
		return nil, errors.New("datafetch: TAIFEX history needs both from and to; a zero time is not an open end")
	}
	from, to = normalizeDate(from), normalizeDate(to)
	if to.Before(from) {
		return nil, fmt.Errorf("datafetch: TAIFEX history needs from (%s) not after to (%s)", from.Format("2006-01-02"), to.Format("2006-01-02"))
	}
	var records []map[string]string
	for start := from; !start.After(to); {
		end := time.Date(start.Year(), start.Month()+1, 0, 0, 0, 0, 0, start.Location())
		if end.After(to) {
			end = to
		}
		chunk, err := c.historyChunk(ctx, h, start, end)
		if err != nil {
			return nil, err
		}
		records = append(records, chunk...)
		start = end.AddDate(0, 0, 1)
	}
	dateKeys := h.columns[0].keys
	sort.SliceStable(records, func(i, j int) bool { return recordDate(records[i], dateKeys) < recordDate(records[j], dateKeys) })
	return taifexTable(h.path, records, h.columns)
}

func recordDate(record map[string]string, keys []string) string {
	for _, key := range keys {
		if v, ok := record[key]; ok {
			d, err := parseTAIFEXDate(v)
			if err != nil {
				return v
			}
			return d.Format("20060102")
		}
	}
	return ""
}

func (c *TAIFEXClient) historyChunk(ctx context.Context, h taifexHistory, from, to time.Time) ([]map[string]string, error) {
	values := url.Values{"queryStartDate": {from.Format("2006/01/02")}, "queryEndDate": {to.Format("2006/01/02")}}
	if h.commodity {
		values.Set("commodityId", "")
	}
	var records []map[string]string
	err := c.base.doGet(ctx, requestURL(c.websiteURL, h.path, values), func(resp *http.Response) error {
		body, err := readBodyResponse(resp, 64<<20)
		if err != nil {
			return err
		}
		if bytes.HasPrefix(bytes.TrimSpace(body), []byte("<!DOCTYPE")) || bytes.Contains(body[:min(len(body), 2048)], []byte("<html")) {
			message := "the website answered with a page instead of a CSV"
			if m := taifexAlert.FindSubmatch(body); m != nil {
				message = string(m[1])
			}
			return &finalError{err: fmt.Errorf("%w: %s from %s to %s: %s; the website serves each table only back to a limit and sets a longest query range",
				ErrTAIFEXRefused, h.path, from.Format("2006-01-02"), to.Format("2006-01-02"), message)}
		}
		text, err := traditionalchinese.Big5.NewDecoder().Bytes(body)
		if err != nil {
			return &finalError{err: fmt.Errorf("datafetch: TAIFEX decode Big5: %w", err)}
		}
		records, err = parseTAIFEXRecords(text)
		if err != nil {
			return &finalError{err: err}
		}
		return nil
	})
	return records, err
}

// InstitutionalFuturesHistory is InstitutionalFutures for every trading day from
// from to to, read from the exchange website's CSV download. It is
// InstitutionalFuturesHistoryContext with context.Background().
func (c *TAIFEXClient) InstitutionalFuturesHistory(from, to time.Time) (*insyra.DataTable, error) {
	return c.InstitutionalFuturesHistoryContext(context.Background(), from, to)
}

// InstitutionalFuturesHistoryContext is InstitutionalFuturesHistory with a
// context. The columns are those of InstitutionalFutures and the rows are in date
// order, oldest first. The website, which was measured on 2026-10-20 to serve this
// table only from 2024, refuses an earlier date: the call then returns an error
// naming the range, not an empty table. A range is requested month by month, so
// a long one makes many requests under the client's Interval.
func (c *TAIFEXClient) InstitutionalFuturesHistoryContext(ctx context.Context, from, to time.Time) (*insyra.DataTable, error) {
	return c.historyTable(ctx, taifexHistory{"/cht/3/futContractsDateDown", institutionalColumns(false), true}, from, to)
}

// InstitutionalOptionsHistory is InstitutionalOptions for every trading day from
// from to to. It is InstitutionalOptionsHistoryContext with context.Background().
func (c *TAIFEXClient) InstitutionalOptionsHistory(from, to time.Time) (*insyra.DataTable, error) {
	return c.InstitutionalOptionsHistoryContext(context.Background(), from, to)
}

// InstitutionalOptionsHistoryContext is InstitutionalOptionsHistory with a
// context. It has the columns of InstitutionalOptions and the same limit as
// InstitutionalFuturesHistoryContext.
func (c *TAIFEXClient) InstitutionalOptionsHistoryContext(ctx context.Context, from, to time.Time) (*insyra.DataTable, error) {
	return c.historyTable(ctx, taifexHistory{"/cht/3/callsAndPutsDateDown", institutionalColumns(true), true}, from, to)
}

// LargeTradersFuturesHistory is LargeTradersFutures for every trading day from
// from to to. It is LargeTradersFuturesHistoryContext with context.Background().
func (c *TAIFEXClient) LargeTradersFuturesHistory(from, to time.Time) (*insyra.DataTable, error) {
	return c.LargeTradersFuturesHistoryContext(context.Background(), from, to)
}

// LargeTradersFuturesHistoryContext is LargeTradersFuturesHistory with a
// context. It has the columns of LargeTradersFutures. The website was measured
// on 2026-10-20 to serve it back to about 2004, in files of roughly 100 KB a day
// for all contracts, so a long range is slow.
func (c *TAIFEXClient) LargeTradersFuturesHistoryContext(ctx context.Context, from, to time.Time) (*insyra.DataTable, error) {
	return c.historyTable(ctx, taifexHistory{"/cht/3/largeTraderFutDown", largeTraderColumns(false), true}, from, to)
}

// LargeTradersOptionsHistory is LargeTradersOptions for every trading day from
// from to to. It is LargeTradersOptionsHistoryContext with context.Background().
func (c *TAIFEXClient) LargeTradersOptionsHistory(from, to time.Time) (*insyra.DataTable, error) {
	return c.LargeTradersOptionsHistoryContext(context.Background(), from, to)
}

// LargeTradersOptionsHistoryContext is LargeTradersOptionsHistory with a
// context. It has the columns of LargeTradersOptions.
func (c *TAIFEXClient) LargeTradersOptionsHistoryContext(ctx context.Context, from, to time.Time) (*insyra.DataTable, error) {
	return c.historyTable(ctx, taifexHistory{"/cht/3/largeTraderOptDown", largeTraderColumns(true), true}, from, to)
}

// PutCallRatioHistory is PutCallRatio for every trading day from from to to,
// oldest first, where PutCallRatio gives only the last 20 days newest first. It is
// PutCallRatioHistoryContext with context.Background().
func (c *TAIFEXClient) PutCallRatioHistory(from, to time.Time) (*insyra.DataTable, error) {
	return c.PutCallRatioHistoryContext(context.Background(), from, to)
}

// PutCallRatioHistoryContext is PutCallRatioHistory with a context. It has the
// columns of PutCallRatio. The exchange's longest query range for this table is
// about a month, so it is requested month by month.
func (c *TAIFEXClient) PutCallRatioHistoryContext(ctx context.Context, from, to time.Time) (*insyra.DataTable, error) {
	return c.historyTable(ctx, taifexHistory{"/cht/3/pcRatioDown", putCallColumns(), false}, from, to)
}
