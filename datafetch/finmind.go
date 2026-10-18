package datafetch

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/HazelnutParadise/insyra"
	json "github.com/goccy/go-json"
)

// ErrFinMindLimit matches (with errors.Is) an error returned because FinMind
// refused the request for reaching the request limit of the account's plan
// (HTTP 402, "Requests reach the upper limit"). It is not retried: wait, or
// use a token of a higher plan.
var ErrFinMindLimit = errors.New("datafetch: FinMind request limit reached")

// ErrFinMindLevel matches (with errors.Is) an error returned because the
// account's plan does not give access to the request, such as a free account
// asking for a dataset that needs a sponsor level, or for one that needs a
// data_id without giving it. It is not retried.
var ErrFinMindLevel = errors.New("datafetch: FinMind plan does not allow this request")

// FinMindConfig configures a FinMind client. A zero value is valid and reaches
// the free tier without a token.
type FinMindConfig struct {
	// Token is the account's API token, sent in an Authorization header. Leave it
	// empty to call without one, as an external proxy that adds it would need.
	Token string
	// Timeout is the per-request timeout. Default: 15s.
	Timeout time.Duration
	// Interval is the minimum spacing between the scheduled starts of successive
	// requests. Zero disables throttling.
	Interval time.Duration
	// UserAgent is the HTTP User-Agent header.
	UserAgent string
	// Retries is the number of retry attempts after a failed request. A request
	// limit or a plan without access is never retried.
	Retries int
	// RetryBackoff is the base delay before a retry. Default: 300ms.
	RetryBackoff time.Duration
}

// FinMindClient reads datasets from FinMind (finmindtrade.com), which serves
// Taiwan and US market data one security's full history per request. Create one
// with FinMind. A client that did not come from FinMind returns an error from
// every method.
type FinMindClient struct {
	base    *TWStockClient
	baseURL string
}

// FinMind creates a client for the FinMind v4 data API.
func FinMind(cfg FinMindConfig) (*FinMindClient, error) {
	base, err := TWStock(TWStockConfig{
		Timeout: cfg.Timeout, Interval: cfg.Interval, UserAgent: cfg.UserAgent,
		Retries: cfg.Retries, RetryBackoff: cfg.RetryBackoff,
	})
	if err != nil {
		return nil, err
	}
	if cfg.Token != "" {
		base.headers = map[string]string{"Authorization": "Bearer " + cfg.Token}
	}
	return &FinMindClient{base: base, baseURL: "https://api.finmindtrade.com"}, nil
}

var errFinMindNotCreated = errors.New("datafetch: FinMindClient was not created with FinMind")

// Dataset returns a FinMind dataset as a table. It is DatasetContext with
// context.Background().
func (c *FinMindClient) Dataset(name, dataID string, from, to time.Time) (*insyra.DataTable, error) {
	return c.DatasetContext(context.Background(), name, dataID, from, to)
}

// DatasetContext is Dataset with a context. The limiter wait, the request and
// each retry backoff stop when ctx is done, and the call then returns ctx.Err()
// without sending another request.
//
// name is a FinMind dataset such as "TaiwanStockPrice" or
// "TaiwanStockMonthRevenue" (see https://finmind.github.io/llms-full.txt for the
// list). dataID is the security it is for, such as "2330"; most datasets need
// one on the free tier. A zero from or to leaves that end open.
//
// The table has the columns of the dataset in the order FinMind sends them.
// Integers become int64, other numbers float64, text stays text, and a column
// named date becomes a time.Time when every value is a date. A dataset with no
// rows gives an empty table with no columns. FinMind's columns are not renamed
// or interpreted, so check the dataset's documentation, and see the notes in
// Docs/datafetch.md about the traps found in some of them.
func (c *FinMindClient) DatasetContext(ctx context.Context, name, dataID string, from, to time.Time) (*insyra.DataTable, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	if c == nil || c.base == nil {
		return nil, errFinMindNotCreated
	}
	if err := c.base.usable(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(name) == "" {
		return nil, errors.New("datafetch: FinMind dataset name is empty")
	}
	values := url.Values{"dataset": {name}}
	if dataID != "" {
		values.Set("data_id", dataID)
	}
	if !from.IsZero() {
		values.Set("start_date", from.Format("2006-01-02"))
	}
	if !to.IsZero() {
		values.Set("end_date", to.Format("2006-01-02"))
	}

	var columns []string
	var rows []map[string]any
	err := c.base.doGet(ctx, requestURL(c.baseURL, "/api/v4/data", values), func(resp *http.Response) error {
		if resp.StatusCode == http.StatusPaymentRequired {
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
			_ = resp.Body.Close()
			return &finalError{ErrFinMindLimit}
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 && resp.StatusCode != http.StatusBadRequest {
			snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 300))
			_ = resp.Body.Close()
			return fmt.Errorf("datafetch: FinMind HTTP status %d: %s", resp.StatusCode, strings.TrimSpace(string(snippet)))
		}
		// FinMind answers a refused request with HTTP 400 and a JSON body that
		// says why, so a 400 is read like a 200.
		body, err := io.ReadAll(io.LimitReader(resp.Body, 256<<20))
		_ = resp.Body.Close()
		if err != nil {
			return fmt.Errorf("datafetch: FinMind read response: %w", err)
		}
		columns, rows, err = parseFinMind(body)
		return err
	})
	if err != nil {
		return nil, err
	}
	return finMindTable(columns, rows), nil
}

// parseFinMind reads the envelope {"msg", "status", "data": [...]} and the rows
// of data, keeping the order of the columns as they first appear.
func parseFinMind(body []byte) ([]string, []map[string]any, error) {
	var envelope struct {
		Msg    string          `json:"msg"`
		Status int             `json:"status"`
		Data   json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, nil, fmt.Errorf("datafetch: FinMind decode JSON: %w", err)
	}
	switch {
	case envelope.Status == http.StatusPaymentRequired:
		return nil, nil, &finalError{fmt.Errorf("%w: %s", ErrFinMindLimit, envelope.Msg)}
	case envelope.Status == http.StatusBadRequest && strings.Contains(strings.ToLower(envelope.Msg), "level"):
		return nil, nil, &finalError{fmt.Errorf("%w: %s", ErrFinMindLevel, envelope.Msg)}
	case envelope.Status != 0 && envelope.Status != http.StatusOK:
		return nil, nil, fmt.Errorf("datafetch: FinMind status %d: %s", envelope.Status, envelope.Msg)
	}
	if len(bytes.TrimSpace(envelope.Data)) == 0 || bytes.Equal(bytes.TrimSpace(envelope.Data), []byte("null")) {
		return nil, nil, nil
	}

	dec := json.NewDecoder(bytes.NewReader(envelope.Data))
	dec.UseNumber()
	if tok, err := dec.Token(); err != nil || tok != json.Delim('[') {
		return nil, nil, errors.New("datafetch: FinMind data is not an array")
	}
	var columns []string
	seen := map[string]bool{}
	var rows []map[string]any
	for dec.More() {
		if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
			return nil, nil, errors.New("datafetch: FinMind data row is not an object")
		}
		row := map[string]any{}
		for dec.More() {
			keyTok, err := dec.Token()
			if err != nil {
				return nil, nil, fmt.Errorf("datafetch: FinMind decode JSON: %w", err)
			}
			key, ok := keyTok.(string)
			if !ok {
				return nil, nil, errors.New("datafetch: FinMind data row has a bad key")
			}
			var value any
			if err := dec.Decode(&value); err != nil {
				return nil, nil, fmt.Errorf("datafetch: FinMind decode JSON: %w", err)
			}
			row[key] = value
			if !seen[key] {
				seen[key] = true
				columns = append(columns, key)
			}
		}
		if _, err := dec.Token(); err != nil { // the closing brace
			return nil, nil, fmt.Errorf("datafetch: FinMind decode JSON: %w", err)
		}
		rows = append(rows, row)
	}
	return columns, rows, nil
}

func finMindTable(columns []string, rows []map[string]any) *insyra.DataTable {
	lists := make([]*insyra.DataList, len(columns))
	for c, name := range columns {
		values := make([]any, len(rows))
		for r, row := range rows {
			values[r] = finMindValue(row[name])
		}
		if name == "date" {
			values = finMindDates(values)
		}
		lists[c] = newNamedCol(name, values)
	}
	return insyra.NewDataTable(lists...)
}

// finMindValue turns a decoded JSON value into an int64, float64, string, bool
// or nil.
func finMindValue(v any) any {
	n, ok := v.(json.Number)
	if !ok {
		return v
	}
	text := n.String()
	if !strings.ContainsAny(text, ".eE") {
		if i, err := n.Int64(); err == nil {
			return i
		}
	}
	if f, err := n.Float64(); err == nil {
		return f
	}
	return text
}

// finMindDates converts a column of "YYYY-MM-DD" strings to times, or returns
// it unchanged when any value is something else.
func finMindDates(values []any) []any {
	out := make([]any, len(values))
	for i, v := range values {
		s, ok := v.(string)
		if !ok {
			return values
		}
		d, err := time.Parse("2006-01-02", s)
		if err != nil {
			return values
		}
		out[i] = d
	}
	return out
}
