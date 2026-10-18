package datafetch

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type statusSequence struct {
	statuses []int
	calls    int
}

func (s *statusSequence) RoundTrip(*http.Request) (*http.Response, error) {
	status := s.statuses[min(s.calls, len(s.statuses)-1)]
	s.calls++
	body := "<html>security</html>"
	if status == http.StatusOK {
		body = `{"stat":"OK"}`
	}
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{}}, nil
}

func TestTWStockReportsThrottling(t *testing.T) {
	for _, status := range []int{http.StatusTemporaryRedirect, http.StatusTooManyRequests} {
		stock, err := TWStock(TWStockConfig{})
		if err != nil {
			t.Fatal(err)
		}
		stock.client.Transport = &statusSequence{statuses: []int{status}}
		var out map[string]any
		err = stock.doJSON(context.Background(), "https://twse.test/x", &out)
		if !errors.Is(err, ErrTWStockThrottled) {
			t.Fatalf("status %d: want ErrTWStockThrottled, got %v", status, err)
		}
		if !strings.Contains(err.Error(), "Interval") || !strings.Contains(err.Error(), "Retries") {
			t.Fatalf("status %d: the message should say what to raise: %v", status, err)
		}
	}
}

func TestTWStockOtherStatusIsNotThrottling(t *testing.T) {
	stock, _ := TWStock(TWStockConfig{})
	stock.client.Transport = &statusSequence{statuses: []int{http.StatusInternalServerError}}
	var out map[string]any
	if err := stock.doJSON(context.Background(), "https://twse.test/x", &out); err == nil || errors.Is(err, ErrTWStockThrottled) {
		t.Fatalf("a 500 is not throttling: %v", err)
	}
}

func TestTWStockRetriesPastAThrottledAnswer(t *testing.T) {
	stock, _ := TWStock(TWStockConfig{Retries: 2, RetryBackoff: 1})
	transport := &statusSequence{statuses: []int{307, 307, 200}}
	stock.client.Transport = transport
	var out map[string]any
	if err := stock.doJSON(context.Background(), "https://twse.test/x", &out); err != nil {
		t.Fatalf("two 307s then a 200 should succeed with Retries 2: %v", err)
	}
	if transport.calls != 3 {
		t.Fatalf("want 3 requests, got %d", transport.calls)
	}
}
