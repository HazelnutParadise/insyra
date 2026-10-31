package datafetch

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// fakeFugleStream is a WebSocket server speaking Fugle's streaming protocol.
// session runs once per connection after a successful auth.
type fakeFugleStream struct {
	*httptest.Server
	conns   atomic.Int32
	mu      sync.Mutex
	frames  []map[string]any // every frame received after auth
	session func(n int32, c *websocket.Conn, read func() map[string]any)
	badKey  bool
}

func newFakeFugleStream(t *testing.T, session func(n int32, c *websocket.Conn, read func() map[string]any)) *fakeFugleStream {
	t.Helper()
	f := &fakeFugleStream{session: session}
	up := websocket.Upgrader{}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/stock/streaming" && r.URL.Path != "/futopt/streaming" {
			http.NotFound(w, r)
			return
		}
		c, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = c.Close() }()
		n := f.conns.Add(1)
		var auth map[string]any
		if err := c.ReadJSON(&auth); err != nil || auth["event"] != "auth" {
			return
		}
		if f.badKey {
			_ = c.WriteJSON(map[string]any{"event": "error", "code": 1000, "data": map[string]any{"message": "Invalid authentication credentials"}})
			return
		}
		_ = c.WriteJSON(map[string]any{"event": "authenticated", "data": map[string]any{"message": "Authenticated successfully"}})
		read := func() map[string]any {
			var m map[string]any
			if err := c.ReadJSON(&m); err != nil {
				return nil
			}
			f.mu.Lock()
			f.frames = append(f.frames, m)
			f.mu.Unlock()
			return m
		}
		f.session(n, c, read)
	}))
	t.Cleanup(f.Close)
	return f
}

func (f *fakeFugleStream) client(t *testing.T) *FugleClient {
	t.Helper()
	c, err := Fugle(FugleConfig{APIKey: "k", BaseURL: f.URL, Timeout: 3 * time.Second, RetryBackoff: 10 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// subscribed answers a subscribe frame with ids made from the connection
// number, so a renewed subscription has a new id.
func subscribed(n int32, c *websocket.Conn, m map[string]any) {
	data := m["data"].(map[string]any)
	var out []map[string]any
	for _, s := range data["symbols"].([]any) {
		out = append(out, map[string]any{"id": fmt.Sprintf("c%d-%s-%s", n, data["channel"], s), "channel": data["channel"], "symbol": s})
	}
	_ = c.WriteJSON(map[string]any{"event": "subscribed", "data": out})
}

func next(t *testing.T, ch <-chan FugleStreamMessage) FugleStreamMessage {
	t.Helper()
	select {
	case m, ok := <-ch:
		if !ok {
			t.Fatal("Messages closed early")
		}
		return m
	case <-time.After(3 * time.Second):
		t.Fatal("no message")
	}
	return FugleStreamMessage{}
}

func TestFugleStreamSubscribeAndReceive(t *testing.T) {
	f := newFakeFugleStream(t, func(n int32, c *websocket.Conn, read func() map[string]any) {
		for m := read(); m != nil; m = read() {
			switch m["event"] {
			case "subscribe":
				subscribed(n, c, m)
				if m["data"].(map[string]any)["channel"] == "trades" {
					_ = c.WriteJSON(map[string]any{"event": "snapshot", "channel": "trades", "id": "x",
						"data": map[string]any{"symbol": "2330", "price": 1050, "size": 3, "time": 1792560600000000, "isClose": true}})
					_ = c.WriteJSON(map[string]any{"event": "data", "channel": "trades", "id": "x",
						"data": map[string]any{"symbol": "2317", "price": 200.5, "time": 1792560601000000}})
				} else {
					_ = c.WriteJSON(map[string]any{"event": "snapshot", "channel": "candles", "id": "y",
						"data": map[string]any{"symbol": "2330", "timeframe": "1", "data": []any{
							map[string]any{"date": "2026-10-21T09:00:00.000+08:00", "open": 1, "close": 2},
							map[string]any{"date": "2026-10-21T09:01:00.000+08:00", "open": 2, "close": 3}}}})
					_ = c.WriteJSON(map[string]any{"event": "data", "channel": "books", "id": "z",
						"data": map[string]any{"symbol": "2330", "time": 1792560602000000, "bids": []any{map[string]any{"price": 1049, "size": 10}}}})
				}
			case "unsubscribe":
				ids := m["data"].(map[string]any)["ids"].([]any)
				_ = c.WriteJSON(map[string]any{"event": "unsubscribed", "data": []any{map[string]any{"id": ids[0], "channel": "trades", "symbol": "2317"}}})
			}
		}
	})
	st, err := f.client(t).Stock.Stream()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	if err := st.Subscribe(FugleChannelTrades, []string{"2330", "2317"}, true); err != nil {
		t.Fatal(err)
	}
	m := next(t, st.Messages())
	if m.Channel != FugleChannelTrades || m.Symbol != "2330" || !m.Snapshot || m.Data["price"] != float64(1050) || m.Data["isClose"] != true {
		t.Errorf("first message %+v", m)
	}
	if want := time.UnixMicro(1792560600000000); !m.Time.Equal(want) {
		t.Errorf("Time = %v, want %v", m.Time, want)
	}
	m = next(t, st.Messages())
	if m.Snapshot || m.Symbol != "2317" {
		t.Errorf("second message %+v", m)
	}
	f.mu.Lock()
	sub := f.frames[0]["data"].(map[string]any)
	f.mu.Unlock()
	if sub["intradayOddLot"] != true || sub["channel"] != "trades" || len(sub["symbols"].([]any)) != 2 {
		t.Errorf("subscribe frame %v", sub)
	}

	if err := st.Subscribe(FugleChannelCandles, []string{"2330"}); err != nil {
		t.Fatal(err)
	}
	c1, c2 := next(t, st.Messages()), next(t, st.Messages())
	if c1.Data["timeframe"] != "1" || c2.Data["close"] != float64(3) || !c1.Snapshot || c1.Time.IsZero() || !c2.Time.After(c1.Time) {
		t.Errorf("candles %+v %+v", c1, c2)
	}
	b := next(t, st.Messages())
	if b.Channel != FugleChannelBooks || b.Data["bids_1_price"] != float64(1049) {
		t.Errorf("books %+v", b)
	}

	if err := st.Unsubscribe(FugleChannelTrades, []string{"2317"}, true); err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	unsub := f.frames[len(f.frames)-1]
	f.mu.Unlock()
	if ids := unsub["data"].(map[string]any)["ids"].([]any); len(ids) != 1 || ids[0] != "c1-trades-2317" {
		t.Errorf("unsubscribe frame %v", unsub)
	}
	if err := st.Unsubscribe(FugleChannelTrades, []string{"2317"}, true); err == nil {
		t.Error("a second unsubscribe of the same symbol was accepted")
	}
	if err := st.Unsubscribe(FugleChannelTrades, []string{"2330"}); err == nil {
		t.Error("unsubscribing the board-lot market of an odd-lot subscription was accepted")
	}
}

func TestFugleStreamReconnectsAndResubscribes(t *testing.T) {
	f := newFakeFugleStream(t, func(n int32, c *websocket.Conn, read func() map[string]any) {
		for m := read(); m != nil; m = read() {
			switch m["event"] {
			case "subscribe":
				subscribed(n, c, m)
				_ = c.WriteJSON(map[string]any{"event": "snapshot", "channel": "trades", "data": map[string]any{"symbol": "2330", "price": float64(n)}})
				if n == 1 {
					// Drop the first connection without a close frame.
					_ = c.UnderlyingConn().Close()
					return
				}
			case "unsubscribe":
				ids := m["data"].(map[string]any)["ids"].([]any)
				_ = c.WriteJSON(map[string]any{"event": "unsubscribed", "data": map[string]any{"id": ids[0], "channel": "trades", "symbol": "2330"}})
			}
		}
	})
	st, err := f.client(t).Stock.Stream()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	if err := st.Subscribe(FugleChannelTrades, []string{"2330"}); err != nil {
		t.Fatal(err)
	}
	if m := next(t, st.Messages()); m.Data["price"] != float64(1) {
		t.Fatalf("first connection's snapshot %+v", m)
	}
	if m := next(t, st.Messages()); m.Data["price"] != float64(2) || !m.Snapshot {
		t.Fatalf("after the reconnect %+v", m)
	}
	if f.conns.Load() != 2 {
		t.Fatalf("%d connections, want 2", f.conns.Load())
	}
	if err := st.Unsubscribe(FugleChannelTrades, []string{"2330"}); err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	last := f.frames[len(f.frames)-1]
	f.mu.Unlock()
	if ids := last["data"].(map[string]any)["ids"].([]any); ids[0] != "c2-trades-2330" {
		t.Errorf("unsubscribe after the reconnect used id %v, want the renewed one", ids[0])
	}
	if st.Err() != nil {
		t.Errorf("Err = %v while running", st.Err())
	}
}

func TestFugleStreamEndsOnAuthAndPlanErrors(t *testing.T) {
	f := newFakeFugleStream(t, nil)
	f.badKey = true
	if _, err := f.client(t).Stock.Stream(); !errors.Is(err, ErrFugleAuth) {
		t.Fatalf("bad key: %v, want ErrFugleAuth", err)
	}

	g := newFakeFugleStream(t, func(n int32, c *websocket.Conn, read func() map[string]any) {
		m := read()
		subscribed(n, c, m)
		_ = c.WriteJSON(map[string]any{"event": "error", "code": 1000, "data": map[string]any{"message": "Your plan does not support this product type"}})
		_ = c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(1001, "Your plan does not support this product type"))
		time.Sleep(100 * time.Millisecond)
	})
	st, err := g.client(t).Stock.Stream()
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Subscribe(FugleChannelTrades, []string{"2330"}); err != nil {
		t.Fatal(err)
	}
	for range st.Messages() {
	}
	if !errors.Is(st.Err(), ErrFuglePlan) {
		t.Errorf("Err = %v, want ErrFuglePlan", st.Err())
	}
	if g.conns.Load() != 1 {
		t.Errorf("reconnected %d times after a plan error", g.conns.Load()-1)
	}
	_ = st.Close()
	if !errors.Is(st.Err(), ErrFuglePlan) {
		t.Errorf("Close cleared the error: %v", st.Err())
	}
	if err := st.Subscribe(FugleChannelTrades, []string{"2330"}); err == nil {
		t.Error("Subscribe after the stream ended was accepted")
	}
}

func TestFugleStreamDropsOldestAndCollects(t *testing.T) {
	const sent = fugleStreamBuffer + 500
	release := make(chan struct{})
	f := newFakeFugleStream(t, func(n int32, c *websocket.Conn, read func() map[string]any) {
		m := read()
		subscribed(n, c, m)
		for i := 0; i < sent; i++ {
			_ = c.WriteJSON(map[string]any{"event": "data", "channel": "trades", "data": map[string]any{"symbol": "2330", "serial": i}})
		}
		_ = c.WriteJSON(map[string]any{"event": "data", "channel": "books", "data": map[string]any{"symbol": "2330"}})
		<-release
		for i := 0; i < 3; i++ {
			_ = c.WriteJSON(map[string]any{"event": "data", "channel": "trades", "data": map[string]any{"symbol": "2317", "price": 10 + i, "time": 1792560600000000 + i}})
		}
		for read() != nil {
		}
	})
	st, err := f.client(t).Stock.Stream()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	if err := st.Subscribe(FugleChannelTrades, []string{"2330"}); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for st.Dropped() < sent+1-fugleStreamBuffer && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if got := st.Dropped(); got != sent+1-fugleStreamBuffer {
		t.Fatalf("Dropped = %d, want %d", got, sent+1-fugleStreamBuffer)
	}
	if m := next(t, st.Messages()); m.Data["serial"] != float64(sent+1-fugleStreamBuffer) {
		t.Errorf("oldest kept serial = %v, want %d", m.Data["serial"], sent+1-fugleStreamBuffer)
	}
	// Drain what is queued, then collect the three that follow.
	for len(st.Messages()) > 0 {
		<-st.Messages()
	}
	close(release)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	dt, err := st.Collect(ctx, FugleChannelTrades, 3)
	if err != nil {
		t.Fatal(err)
	}
	if dt.NumRows() != 3 || dt.ColNames()[0] != "time" || dt.GetColByName("price").Data()[2] != float64(12) || dt.GetColByName("snapshot").Data()[0] != false {
		t.Errorf("collected %v rows %d", dt.ColNames(), dt.NumRows())
	}
}

func TestFugleStreamCloseAndBadInput(t *testing.T) {
	f := newFakeFugleStream(t, func(n int32, c *websocket.Conn, read func() map[string]any) {
		for read() != nil {
		}
	})
	c := f.client(t)
	st, err := c.Stock.Stream()
	if err != nil {
		t.Fatal(err)
	}
	bad := map[string]error{
		"unknown channel": st.Subscribe("ticks", []string{"2330"}),
		"no symbols":      st.Subscribe(FugleChannelTrades, nil),
		"empty symbol":    st.Subscribe(FugleChannelTrades, []string{""}),
		"two flags":       st.Subscribe(FugleChannelTrades, []string{"2330"}, true, false),
		"not subscribed":  st.Unsubscribe(FugleChannelTrades, []string{"2330"}),
	}
	for name, err := range bad {
		if err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
	f.mu.Lock()
	if len(f.frames) != 0 {
		t.Errorf("refused calls sent %d frames", len(f.frames))
	}
	f.mu.Unlock()
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	if _, ok := <-st.Messages(); ok {
		t.Error("Messages still open after Close")
	}
	if st.Err() != nil {
		t.Errorf("Err after a clean Close = %v", st.Err())
	}
	if err := st.Subscribe(FugleChannelTrades, []string{"2330"}); err == nil {
		t.Error("Subscribe after Close was accepted")
	}

	fo, err := c.FutOpt.Stream()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = fo.Close() }()
	if err := fo.Subscribe(FugleChannelIndices, []string{"TXFA6"}); err == nil {
		t.Error("indices on the futures and options stream was accepted")
	}

	var zero FugleStockStream
	if _, ok := <-zero.Messages(); ok || zero.Close() != nil || zero.Subscribe(FugleChannelTrades, []string{"2330"}) == nil {
		t.Error("a stream that did not come from Stream misbehaved")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.Stock.StreamContext(ctx); !errors.Is(err, context.Canceled) {
		t.Errorf("canceled context: %v", err)
	}
}
