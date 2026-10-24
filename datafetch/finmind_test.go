package datafetch

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type headerRecorder struct {
	next          http.RoundTripper
	authorization string
}

func (h *headerRecorder) RoundTrip(req *http.Request) (*http.Response, error) {
	h.authorization = req.Header.Get("Authorization")
	return h.next.RoundTrip(req)
}

func newFixtureFinMind(t *testing.T, cfg FinMindConfig) (*FinMindClient, *fixtureTransport, *headerRecorder) {
	t.Helper()
	client, err := FinMind(cfg)
	if err != nil {
		t.Fatalf("FinMind returned error: %v", err)
	}
	transport := newFixtureTransport()
	recorder := &headerRecorder{next: transport}
	client.base.client.Transport = recorder
	client.baseURL = "https://finmind.test"
	return client, transport, recorder
}

func finMindKey(values ...string) string {
	return fixtureKey("/api/v4/data", query(values...))
}

func TestFinMindDataset(t *testing.T) {
	client, transport, recorder := newFixtureFinMind(t, FinMindConfig{Token: "secret-token"})
	body, err := os.ReadFile(filepath.Join("testdata", "finmind", "price_2330.json"))
	if err != nil {
		t.Fatal(err)
	}
	key := finMindKey("dataset", "TaiwanStockPrice", "data_id", "2330", "start_date", "2026-10-12", "end_date", "2026-10-14")
	transport.replies[key] = []fixtureReply{{status: http.StatusOK, body: body}}
	dt, err := client.Dataset("TaiwanStockPrice", "2330", dateUTC(2026, time.October, 12), dateUTC(2026, time.October, 14))
	if err != nil {
		t.Fatalf("Dataset error: %v", err)
	}
	names := dt.ColNames()
	want := []string{"date", "stock_id", "Trading_Volume", "Trading_money", "open", "max", "min", "close", "spread", "Trading_turnover"}
	if len(names) != len(want) {
		t.Fatalf("columns = %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("columns = %v, want %v", names, want)
		}
	}
	if dt.NumRows() != 3 {
		t.Fatalf("rows = %d, want 3", dt.NumRows())
	}
	if got := dt.GetColByName("date").Data()[0]; got != dateUTC(2026, time.October, 12) {
		t.Errorf("date = %v (%T)", got, got)
	}
	if got := dt.GetColByName("stock_id").Data()[0]; got != "2330" {
		t.Errorf("stock_id = %v", got)
	}
	if got := dt.GetColByName("Trading_Volume").Data()[0]; got != int64(23555768) {
		t.Errorf("Trading_Volume = %v (%T), want an int64", got, got)
	}
	if got := dt.GetColByName("close").Data()[0]; got != 2540.0 {
		t.Errorf("close = %v (%T)", got, got)
	}
	if recorder.authorization != "Bearer secret-token" {
		t.Errorf("Authorization = %q", recorder.authorization)
	}
}

func TestFinMindErrorsAreRecognisableAndNotRetried(t *testing.T) {
	client, transport, _ := newFixtureFinMind(t, FinMindConfig{Retries: 3, RetryBackoff: time.Millisecond})
	level := finMindKey("dataset", "TaiwanStockPrice", "start_date", "2026-10-12")
	transport.replies[level] = []fixtureReply{{status: http.StatusBadRequest, body: []byte(`{"msg":"Your level is free. Please update your user level.","status":400}`)}}
	if _, err := client.Dataset("TaiwanStockPrice", "", dateUTC(2026, time.October, 12), time.Time{}); !errors.Is(err, ErrFinMindLevel) {
		t.Errorf("level error = %v, want ErrFinMindLevel", err)
	}
	if transport.requestCount() != 1 {
		t.Errorf("a plan error was sent %d times, want 1", transport.requestCount())
	}

	limit := finMindKey("dataset", "TaiwanStockPrice", "data_id", "2330")
	transport.replies[limit] = []fixtureReply{{status: http.StatusPaymentRequired, body: []byte(`{"msg":"Requests reach the upper limit","status":402}`)}}
	before := transport.requestCount()
	if _, err := client.Dataset("TaiwanStockPrice", "2330", time.Time{}, time.Time{}); !errors.Is(err, ErrFinMindLimit) {
		t.Errorf("HTTP 402 = %v, want ErrFinMindLimit", err)
	}
	if transport.requestCount()-before != 1 {
		t.Errorf("a request limit was retried")
	}
	transport.replies[limit] = []fixtureReply{{status: http.StatusOK, body: []byte(`{"msg":"Requests reach the upper limit","status":402}`)}}
	if _, err := client.Dataset("TaiwanStockPrice", "2330", time.Time{}, time.Time{}); !errors.Is(err, ErrFinMindLimit) {
		t.Errorf("status 402 in the body = %v, want ErrFinMindLimit", err)
	}

	transport.replies[limit] = []fixtureReply{{status: http.StatusUnprocessableEntity, body: []byte(`{"detail":"unknown dataset"}`)}}
	if _, err := client.Dataset("TaiwanStockPrice", "2330", time.Time{}, time.Time{}); err == nil || errors.Is(err, ErrFinMindLimit) || errors.Is(err, ErrFinMindLevel) {
		t.Errorf("HTTP 422 = %v, want a plain error", err)
	}
}

func TestFinMindEmptyAndBadCalls(t *testing.T) {
	client, transport, _ := newFixtureFinMind(t, FinMindConfig{})
	key := finMindKey("dataset", "TaiwanStockDelisting", "data_id", "9915")
	transport.replies[key] = []fixtureReply{{status: http.StatusOK, body: []byte(`{"msg":"success","status":200,"data":[]}`)}}
	dt, err := client.Dataset("TaiwanStockDelisting", "9915", time.Time{}, time.Time{})
	if err != nil || dt.NumRows() != 0 {
		t.Errorf("an empty dataset: %v, %v", dt, err)
	}
	transport.replies[key] = []fixtureReply{{status: http.StatusOK, body: []byte(`{"msg":"success","status":200,"data":[{"date":"2026-10-12","value":1},{"date":"not a date","value":null,"extra":"x"}]}`)}}
	mixed, err := client.Dataset("TaiwanStockDelisting", "9915", time.Time{}, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if got := mixed.GetColByName("date").Data()[0]; got != "2026-10-12" {
		t.Errorf("a date column with a non-date stays text, got %v (%T)", got, got)
	}
	if got := mixed.GetColByName("value").Data()[1]; got != nil {
		t.Errorf("null = %v", got)
	}
	if got := mixed.GetColByName("extra").Data()[0]; got != nil {
		t.Errorf("a column absent from a row = %v, want nil", got)
	}
	if _, err := client.Dataset("  ", "", time.Time{}, time.Time{}); err == nil {
		t.Error("an empty dataset name should be an error")
	}
	var zero FinMindClient
	if _, err := zero.Dataset("TaiwanStockPrice", "2330", time.Time{}, time.Time{}); err == nil {
		t.Error("a zero client should be an error")
	}
	transport.replies[key] = []fixtureReply{{status: http.StatusOK, body: []byte(`not json`)}}
	if _, err := client.Dataset("TaiwanStockDelisting", "9915", time.Time{}, time.Time{}); err == nil {
		t.Error("a body that is not JSON should be an error")
	}
}
