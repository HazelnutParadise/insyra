package datafetch

import (
	"context"
	"errors"
	"fmt"
	json "github.com/goccy/go-json"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/HazelnutParadise/insyra"
	"github.com/gorilla/websocket"
)

// FugleChannel is a channel of Fugle's WebSocket stream.
type FugleChannel string

const (
	// FugleChannelTrades is every trade, FugleChannelCandles the minute
	// candles, FugleChannelBooks the best five bids and asks,
	// FugleChannelAggregates the running quote of the day and
	// FugleChannelIndices the index values (stocks only).
	FugleChannelTrades     FugleChannel = "trades"
	FugleChannelCandles    FugleChannel = "candles"
	FugleChannelBooks      FugleChannel = "books"
	FugleChannelAggregates FugleChannel = "aggregates"
	FugleChannelIndices    FugleChannel = "indices"
)

// FugleStreamMessage is one message of a Fugle stream.
type FugleStreamMessage struct {
	// Channel and Symbol say what the message is about.
	Channel FugleChannel
	Symbol  string
	// Snapshot is true for the latest data Fugle sends right after a
	// subscription (again after a reconnect), and false for live updates.
	Snapshot bool
	// Time is the message's time field, or its date for a candle; zero when it
	// has neither.
	Time time.Time
	// Data holds Fugle's fields with nested ones joined by an underscore, the
	// names the REST methods use (bids_1_price, total_tradeVolume). A flag
	// Fugle sends only when true is missing when false.
	Data map[string]any
}

const (
	fugleStreamBuffer     = 1024
	fugleStreamMaxBackoff = 30 * time.Second
	fugleStreamWriteWait  = 10 * time.Second
)

var errFugleStreamClosed = errors.New("datafetch: Fugle stream is closed")

// fugleNoMessages is the closed channel a stream that did not come from
// Stream returns from Messages.
var fugleNoMessages = func() chan FugleStreamMessage {
	c := make(chan FugleStreamMessage)
	close(c)
	return c
}()

type fugleStreamKey struct {
	channel FugleChannel
	symbol  string
	flag    bool
}

type fugleStreamReply struct {
	event string
	code  int
	data  json.RawMessage
}

type fugleStreamEntry struct {
	ID      string `json:"id"`
	Channel string `json:"channel"`
	Symbol  string `json:"symbol"`
}

// fugleStream is the connection shared by FugleStockStream and
// FugleFutOptStream.
type fugleStream struct {
	url      string
	apiKey   string
	flagName string // intradayOddLot or afterHours
	channels map[FugleChannel]bool
	symbolOK func(string) (string, error)
	timeout  time.Duration
	backoff  time.Duration

	cmdMu   sync.Mutex // one command at a time, since replies carry no request id
	writeMu sync.Mutex
	mu      sync.Mutex // guards conn, subs, err and closed
	conn    *websocket.Conn
	lost    chan struct{}             // closed when conn stops being read
	subs    map[fugleStreamKey]string // the id Fugle gave each subscription
	resub   []bool                    // flags of resubscriptions whose reply is pending, oldest first
	err     error
	closed  bool

	replies chan fugleStreamReply
	msgs    chan FugleStreamMessage
	dropped atomic.Uint64
	done    chan struct{} // closed when Close is called
	exited  chan struct{} // closed when the reading goroutine returns
	once    sync.Once
}

func (c *FugleClient) openStream(ctx context.Context, product, flagName string, channels []FugleChannel, symbolOK func(string) (string, error)) (*fugleStream, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	if err := c.usable(); err != nil {
		return nil, err
	}
	url := c.baseURL + "/" + product + "/streaming"
	switch {
	case strings.HasPrefix(url, "https://"):
		url = "wss://" + strings.TrimPrefix(url, "https://")
	case strings.HasPrefix(url, "http://"):
		url = "ws://" + strings.TrimPrefix(url, "http://")
	}
	s := &fugleStream{
		url:      url,
		apiKey:   c.base.headers["X-API-KEY"],
		flagName: flagName,
		channels: map[FugleChannel]bool{},
		symbolOK: symbolOK,
		timeout:  c.base.cfg.Timeout,
		backoff:  c.base.cfg.RetryBackoff,
		subs:     map[fugleStreamKey]string{},
		replies:  make(chan fugleStreamReply, 16),
		msgs:     make(chan FugleStreamMessage, fugleStreamBuffer),
		done:     make(chan struct{}),
		exited:   make(chan struct{}),
	}
	for _, ch := range channels {
		s.channels[ch] = true
	}
	if s.timeout <= 0 {
		s.timeout = 15 * time.Second
	}
	if s.backoff <= 0 {
		s.backoff = 300 * time.Millisecond
	}
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.timeout)
		defer cancel()
	}
	conn, err := s.connect(ctx)
	if err != nil {
		return nil, err
	}
	s.conn = conn
	s.lost = make(chan struct{})
	go s.run()
	return s, nil
}

// connect dials and authenticates one connection.
func (s *fugleStream) connect(ctx context.Context) (*websocket.Conn, error) {
	conn, resp, err := websocket.DefaultDialer.DialContext(ctx, s.url, nil)
	if err != nil {
		if resp != nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusUnauthorized {
				return nil, fmt.Errorf("%w: WebSocket handshake HTTP %d", ErrFugleAuth, resp.StatusCode)
			}
			return nil, fmt.Errorf("datafetch: Fugle WebSocket handshake failed with HTTP %d: %w", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("datafetch: Fugle WebSocket connection failed: %w", err)
	}
	if err := s.writeConn(conn, map[string]any{"event": "auth", "data": map[string]any{"apikey": s.apiKey}}); err != nil {
		_ = conn.Close()
		return nil, err
	}
	stop := context.AfterFunc(ctx, func() { _ = conn.SetReadDeadline(time.Now()) })
	defer stop()
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			_ = conn.Close()
			if ctxErr := ctx.Err(); ctxErr != nil {
				return nil, fmt.Errorf("datafetch: Fugle WebSocket authentication: %w", ctxErr)
			}
			return nil, fmt.Errorf("datafetch: Fugle WebSocket closed during authentication: %w", err)
		}
		var f fugleStreamFrame
		if err := json.Unmarshal(raw, &f); err != nil {
			continue
		}
		switch f.Event {
		case "authenticated":
			if !stop() {
				// ctx ended as authentication succeeded and its deadline is
				// being set; the connection cannot be trusted to read.
				_ = conn.Close()
				return nil, fmt.Errorf("datafetch: Fugle WebSocket authentication: %w", ctx.Err())
			}
			_ = conn.SetReadDeadline(time.Time{})
			return conn, nil
		case "error":
			_ = conn.Close()
			return nil, fugleStreamError(f.Code, f.message())
		}
	}
}

func (s *fugleStream) writeConn(conn *websocket.Conn, v any) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	_ = conn.SetWriteDeadline(time.Now().Add(fugleStreamWriteWait))
	if err := conn.WriteJSON(v); err != nil {
		return fmt.Errorf("datafetch: Fugle WebSocket write failed: %w", err)
	}
	return nil
}

type fugleStreamFrame struct {
	Event   string          `json:"event"`
	Code    int             `json:"code"`
	ID      string          `json:"id"`
	Channel string          `json:"channel"`
	Data    json.RawMessage `json:"data"`
}

func (f fugleStreamFrame) message() string {
	var d struct {
		Message string `json:"message"`
	}
	_ = json.Unmarshal(f.Data, &d)
	return d.Message
}

// fugleStreamError turns an error event into an error. An invalid key is
// ErrFugleAuth and a plan without the product ErrFuglePlan.
func fugleStreamError(code int, msg string) error {
	switch {
	case code == 1000 && strings.Contains(msg, "plan"):
		return fmt.Errorf("%w: %s", ErrFuglePlan, msg)
	case code == 1000:
		return fmt.Errorf("%w: %s", ErrFugleAuth, msg)
	}
	return fmt.Errorf("datafetch: Fugle WebSocket error %d: %s", code, msg)
}

// run reads the connection until Close or an error a reconnect cannot cure,
// reconnecting after the drops Fugle says to retry.
func (s *fugleStream) run() {
	defer close(s.exited)
	defer close(s.msgs)
	for {
		s.mu.Lock()
		conn, lost := s.conn, s.lost
		s.mu.Unlock()
		fatal := s.read(conn)
		_ = conn.Close()
		close(lost)
		if s.isClosed() {
			return
		}
		if fatal != nil {
			s.fail(fatal)
			return
		}
		if !s.reconnect() {
			return
		}
	}
}

// read handles frames until the connection fails. It returns an error when
// the failure must end the stream, and nil when a reconnect may help.
func (s *fugleStream) read(conn *websocket.Conn) error {
	var lastErr error
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			if lastErr != nil {
				return lastErr
			}
			var ce *websocket.CloseError
			if errors.As(err, &ce) {
				switch {
				case ce.Code == 1001 && strings.Contains(ce.Text, "plan"):
					return fmt.Errorf("%w: %s", ErrFuglePlan, ce.Text)
				case ce.Code == 1013, ce.Code == 1008:
					return fmt.Errorf("datafetch: Fugle WebSocket closed the connection (%d %s)", ce.Code, ce.Text)
				}
			}
			return nil
		}
		var f fugleStreamFrame
		if err := json.Unmarshal(raw, &f); err != nil {
			continue
		}
		switch f.Event {
		case "snapshot", "data":
			s.deliver(f)
		case "subscribed", "unsubscribed", "error":
			if f.Event == "error" {
				msg := f.message()
				if f.Code == 1000 || strings.Contains(msg, "Maximum number of connections") {
					// Fugle closes the connection next; report why.
					lastErr = fugleStreamError(f.Code, msg)
				}
			}
			if f.Event != "unsubscribed" && s.resubscribed(f) {
				continue
			}
			s.reply(fugleStreamReply{event: f.Event, code: f.Code, data: f.Data})
		}
	}
}

// resubscribed records the reply to a resubscription sent after a reconnect,
// and reports whether the frame was one.
func (s *fugleStream) resubscribed(f fugleStreamFrame) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.resub) == 0 {
		return false
	}
	flag := s.resub[0]
	s.resub = s.resub[1:]
	if f.Event == "error" {
		insyra.LogWarning("datafetch", "FugleStream", "a subscription could not be renewed after a reconnect: %s", f.message())
		return true
	}
	for _, e := range fugleStreamEntries(f.Data) {
		k := fugleStreamKey{FugleChannel(e.Channel), e.Symbol, flag}
		if _, ok := s.subs[k]; ok {
			s.subs[k] = e.ID
		}
	}
	return true
}

func (s *fugleStream) reply(r fugleStreamReply) {
	select {
	case s.replies <- r:
	default: // nobody is waiting for it
	}
}

// reconnect dials again with growing pauses, authenticates and subscribes
// everything again. It returns false when the stream ended meanwhile.
func (s *fugleStream) reconnect() bool {
	wait := s.backoff
	full := 0 // reconnects refused in a row because the plan's connections are all in use
	for {
		select {
		case <-s.done:
			return false
		case <-time.After(wait):
		}
		ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
		go func() {
			select {
			case <-s.done:
				cancel()
			case <-ctx.Done():
			}
		}()
		conn, err := s.connect(ctx)
		cancel()
		if err == nil {
			// Hold commands back until the resubscriptions are on the wire, so
			// their replies come first.
			s.cmdMu.Lock()
			defer s.cmdMu.Unlock()
			s.mu.Lock()
			if s.closed {
				s.mu.Unlock()
				_ = conn.Close()
				return false
			}
			s.conn = conn
			s.lost = make(chan struct{})
			type group struct {
				ch   FugleChannel
				flag bool
			}
			groups := map[group][]string{}
			for k := range s.subs {
				g := group{k.channel, k.flag}
				groups[g] = append(groups[g], k.symbol)
				s.subs[k] = ""
			}
			s.resub = nil
			order := make([]group, 0, len(groups))
			for g := range groups {
				order = append(order, g)
				s.resub = append(s.resub, g.flag)
			}
			s.mu.Unlock()
			for i, g := range order {
				symbols := groups[g]
				sort.Strings(symbols)
				if err := s.writeConn(conn, s.subscribeFrame(g.ch, symbols, g.flag)); err != nil {
					s.mu.Lock()
					s.resub = s.resub[:i]
					s.mu.Unlock()
					break
				}
			}
			insyra.LogWarning("datafetch", "FugleStream", "the connection dropped and was reopened; subscriptions were renewed and send their snapshots again")
			return true
		}
		if strings.Contains(err.Error(), "Maximum number of connections") {
			// The dropped connection may hold its place for a moment, so a few
			// refusals are expected; more mean the limit is really reached.
			if full++; full >= 5 {
				s.fail(err)
				return false
			}
		} else {
			full = 0
		}
		if errors.Is(err, ErrFugleAuth) || errors.Is(err, ErrFuglePlan) {
			s.fail(err)
			return false
		}
		wait *= 2
		if wait > fugleStreamMaxBackoff {
			wait = fugleStreamMaxBackoff
		}
	}
}

func (s *fugleStream) subscribeFrame(ch FugleChannel, symbols []string, flag bool) map[string]any {
	data := map[string]any{"channel": string(ch), "symbols": symbols}
	if flag {
		data[s.flagName] = true
	}
	return map[string]any{"event": "subscribe", "data": data}
}

func (s *fugleStream) isClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}

func (s *fugleStream) fail(err error) {
	s.mu.Lock()
	if s.err == nil {
		s.err = err
	}
	s.mu.Unlock()
}

// deliver turns a snapshot or data frame into messages. A candles snapshot
// carries today's candles in an array and becomes one message per candle.
func (s *fugleStream) deliver(f fugleStreamFrame) {
	var body any
	if err := json.Unmarshal(f.Data, &body); err != nil {
		return
	}
	snapshot := f.Event == "snapshot"
	var items []map[string]any
	switch b := body.(type) {
	case []any:
		for _, it := range b {
			if m, ok := it.(map[string]any); ok {
				items = append(items, m)
			}
		}
	case map[string]any:
		if arr, ok := b["data"].([]any); ok {
			for _, it := range arr {
				m, ok := it.(map[string]any)
				if !ok {
					continue
				}
				merged := map[string]any{}
				for k, v := range b {
					if k != "data" {
						merged[k] = v
					}
				}
				for k, v := range m {
					merged[k] = v
				}
				items = append(items, merged)
			}
		} else {
			items = append(items, b)
		}
	}
	for _, it := range items {
		flat := map[string]any{}
		flattenFugle("", it, flat)
		symbol, _ := flat["symbol"].(string)
		s.push(FugleStreamMessage{
			Channel:  FugleChannel(f.Channel),
			Symbol:   symbol,
			Snapshot: snapshot,
			Time:     fugleStreamTime(flat),
			Data:     flat,
		})
	}
}

func fugleStreamTime(m map[string]any) time.Time {
	if v, ok := m["time"].(float64); ok && v > 0 {
		return time.UnixMicro(int64(v))
	}
	if d, ok := m["date"].(string); ok {
		if t, err := time.Parse(time.RFC3339Nano, d); err == nil {
			return t
		}
		if t, err := time.Parse("2006-01-02", d); err == nil {
			return t
		}
	}
	return time.Time{}
}

// push queues a message, dropping the oldest when the consumer has fallen
// behind: blocking here would stop the replies to Fugle's pings and get the
// connection dropped.
func (s *fugleStream) push(m FugleStreamMessage) {
	for {
		select {
		case s.msgs <- m:
			return
		default:
		}
		select {
		case <-s.msgs:
			if s.dropped.Add(1) == 1 {
				insyra.LogWarning("datafetch", "FugleStream", "messages are arriving faster than they are read; the oldest are dropped (see Dropped)")
			}
		default:
		}
	}
}

func (s *fugleStream) checkSymbols(ch FugleChannel, symbols []string) ([]string, error) {
	if !s.channels[ch] {
		names := make([]string, 0, len(s.channels))
		for c := range s.channels {
			names = append(names, string(c))
		}
		sort.Strings(names)
		return nil, fmt.Errorf("datafetch: Fugle stream channel %q is not one of %s", string(ch), strings.Join(names, ", "))
	}
	if len(symbols) == 0 {
		return nil, errors.New("datafetch: Fugle stream needs at least one symbol")
	}
	out := make([]string, len(symbols))
	for i, sym := range symbols {
		v, err := s.symbolOK(sym)
		if err != nil {
			return nil, err
		}
		out[i] = v
	}
	return out, nil
}

// command sends one subscribe or unsubscribe and waits for its reply. build
// makes the frame and apply records the reply; both run under cmdMu, so a reconnect cannot come between
// them.
func (s *fugleStream) command(build func() (map[string]any, error), want string, apply func(json.RawMessage)) error {
	s.cmdMu.Lock()
	defer s.cmdMu.Unlock()
	s.mu.Lock()
	if s.closed || s.err != nil {
		err := s.err
		s.mu.Unlock()
		if err == nil {
			err = errFugleStreamClosed
		}
		return err
	}
	conn, lost := s.conn, s.lost
	s.mu.Unlock()
	frame, err := build()
	if err != nil {
		return err
	}
	for len(s.replies) > 0 {
		<-s.replies
	}
	if err := s.writeConn(conn, frame); err != nil {
		return err
	}
	timer := time.NewTimer(s.timeout)
	defer timer.Stop()
	for {
		select {
		case r := <-s.replies:
			if r.event == "error" {
				var d struct {
					Message string `json:"message"`
				}
				_ = json.Unmarshal(r.data, &d)
				return fugleStreamError(r.code, d.Message)
			}
			if r.event == want {
				apply(r.data)
				return nil
			}
		case <-lost:
			if s.isClosed() {
				return errFugleStreamClosed
			}
			if err := s.Err(); err != nil {
				return err
			}
			return fmt.Errorf("datafetch: Fugle connection dropped before the %s request was answered; the stream is reconnecting, try again", frame["event"])
		case <-timer.C:
			return fmt.Errorf("datafetch: Fugle did not answer the %s request within %s; try again", frame["event"], s.timeout)
		}
	}
}

func fugleStreamEntries(data json.RawMessage) []fugleStreamEntry {
	var list []fugleStreamEntry
	if json.Unmarshal(data, &list) == nil {
		return list
	}
	var one fugleStreamEntry
	if json.Unmarshal(data, &one) == nil && one.ID != "" {
		return []fugleStreamEntry{one}
	}
	return nil
}

func (s *fugleStream) subscribe(ch FugleChannel, symbols []string, flags []bool) error {
	flag, err := oneFlag(flags, s.flagName)
	if err != nil {
		return err
	}
	symbols, err = s.checkSymbols(ch, symbols)
	if err != nil {
		return err
	}
	return s.command(func() (map[string]any, error) { return s.subscribeFrame(ch, symbols, flag), nil }, "subscribed",
		func(data json.RawMessage) {
			s.mu.Lock()
			defer s.mu.Unlock()
			for _, e := range fugleStreamEntries(data) {
				s.subs[fugleStreamKey{FugleChannel(e.Channel), e.Symbol, flag}] = e.ID
			}
		})
}

func (s *fugleStream) unsubscribe(ch FugleChannel, symbols []string, flags []bool) error {
	flag, err := oneFlag(flags, s.flagName)
	if err != nil {
		return err
	}
	symbols, err = s.checkSymbols(ch, symbols)
	if err != nil {
		return err
	}
	build := func() (map[string]any, error) {
		s.mu.Lock()
		defer s.mu.Unlock()
		var ids, missing, renewing []string
		for _, sym := range symbols {
			id, ok := s.subs[fugleStreamKey{ch, sym, flag}]
			switch {
			case !ok:
				missing = append(missing, sym)
			case id == "":
				renewing = append(renewing, sym)
			default:
				ids = append(ids, id)
			}
		}
		if len(missing) > 0 {
			return nil, fmt.Errorf("datafetch: Fugle stream is not subscribed to %s for %s", string(ch), strings.Join(missing, ", "))
		}
		if len(renewing) > 0 {
			return nil, fmt.Errorf("datafetch: Fugle stream is renewing the %s subscription of %s after a reconnect; try again", string(ch), strings.Join(renewing, ", "))
		}
		return map[string]any{"event": "unsubscribe", "data": map[string]any{"ids": ids}}, nil
	}
	return s.command(build, "unsubscribed", func(json.RawMessage) {
		s.mu.Lock()
		defer s.mu.Unlock()
		for _, sym := range symbols {
			delete(s.subs, fugleStreamKey{ch, sym, flag})
		}
	})
}

// Err returns the error that ended the stream, or nil while it runs and after
// a Close that nothing preceded.
func (s *fugleStream) Err() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.err
}

func (s *fugleStream) close() error {
	s.once.Do(func() {
		s.mu.Lock()
		s.closed = true
		conn := s.conn
		s.mu.Unlock()
		close(s.done)
		s.writeMu.Lock()
		_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
		s.writeMu.Unlock()
		_ = conn.Close()
		<-s.exited
	})
	return nil
}

func (s *fugleStream) collect(ctx context.Context, ch FugleChannel, n int) (*insyra.DataTable, error) {
	if ctx == nil {
		return nil, errors.New("datafetch: Fugle Collect needs a context")
	}
	var rows []map[string]any
	table := func() *insyra.DataTable {
		return fugleTable(rows, []string{"time", "symbol", "snapshot"}, []string{"time", "symbol", "snapshot"})
	}
	for n <= 0 || len(rows) < n {
		select {
		case <-ctx.Done():
			return table(), nil
		case m, ok := <-s.msgs:
			if !ok {
				return table(), s.Err()
			}
			if m.Channel != ch {
				continue
			}
			row := make(map[string]any, len(m.Data)+3)
			for k, v := range m.Data {
				row[k] = v
			}
			row["symbol"] = m.Symbol
			row["snapshot"] = m.Snapshot
			if m.Time.IsZero() {
				row["time"] = nil
			} else {
				row["time"] = m.Time
			}
			rows = append(rows, row)
		}
	}
	return table(), nil
}

// FugleStockStream is a WebSocket stream of stock market data. Create one
// with client.Stock.Stream. It reconnects and resubscribes by itself after the
// drops Fugle says to retry.
type FugleStockStream struct{ s *fugleStream }

// FugleFutOptStream is a WebSocket stream of futures and options market data.
// Create one with client.FutOpt.Stream.
type FugleFutOptStream struct{ s *fugleStream }

// Stream connects to Fugle's stock stream and authenticates. It is
// StreamContext with context.Background(), waiting at most FugleConfig.Timeout.
func (s FugleStock) Stream() (*FugleStockStream, error) {
	return s.StreamContext(context.Background())
}

// StreamContext is Stream with a context that bounds connecting and
// authenticating only; the stream then runs until Close. Without a deadline
// FugleConfig.Timeout applies.
func (s FugleStock) StreamContext(ctx context.Context) (*FugleStockStream, error) {
	st, err := s.c.openStream(ctx, "stock", "intradayOddLot",
		[]FugleChannel{FugleChannelTrades, FugleChannelCandles, FugleChannelBooks, FugleChannelAggregates, FugleChannelIndices}, fugleSymbol)
	if err != nil {
		return nil, err
	}
	return &FugleStockStream{st}, nil
}

// Stream connects to Fugle's futures and options stream and authenticates. It
// is StreamContext with context.Background(), waiting at most
// FugleConfig.Timeout.
func (f FugleFutOpt) Stream() (*FugleFutOptStream, error) {
	return f.StreamContext(context.Background())
}

// StreamContext is Stream with a context that bounds connecting and
// authenticating only; the stream then runs until Close.
func (f FugleFutOpt) StreamContext(ctx context.Context) (*FugleFutOptStream, error) {
	st, err := f.c.openStream(ctx, "futopt", "afterHours",
		[]FugleChannel{FugleChannelTrades, FugleChannelCandles, FugleChannelBooks, FugleChannelAggregates}, fugleFutOptSymbol)
	if err != nil {
		return nil, err
	}
	return &FugleFutOptStream{st}, nil
}

// Subscribe asks for a channel of the symbols and waits for Fugle to confirm.
// oddLot subscribes to the intraday odd-lot market; more than one flag is an
// error. Fugle first sends each symbol's latest data as a snapshot message.
func (st *FugleStockStream) Subscribe(ch FugleChannel, symbols []string, oddLot ...bool) error {
	if st == nil || st.s == nil {
		return errFugleStreamClosed
	}
	return st.s.subscribe(ch, symbols, oddLot)
}

// Unsubscribe stops a channel of the symbols; each must have been subscribed
// with the same oddLot flag.
func (st *FugleStockStream) Unsubscribe(ch FugleChannel, symbols []string, oddLot ...bool) error {
	if st == nil || st.s == nil {
		return errFugleStreamClosed
	}
	return st.s.unsubscribe(ch, symbols, oddLot)
}

// Messages returns the channel the stream's messages arrive on, closed when
// the stream ends. When it is read too slowly the oldest messages are dropped
// (see Dropped) rather than holding up the connection.
func (st *FugleStockStream) Messages() <-chan FugleStreamMessage {
	if st == nil || st.s == nil {
		return fugleNoMessages
	}
	return st.s.msgs
}

// Collect reads messages of one channel into a table until n have arrived or
// ctx ends (n <= 0 reads until ctx ends). Messages of other channels read
// meanwhile are discarded. The columns are time, symbol, snapshot and Fugle's
// fields. When the stream ends first it returns what it has with Err.
func (st *FugleStockStream) Collect(ctx context.Context, ch FugleChannel, n int) (*insyra.DataTable, error) {
	if st == nil || st.s == nil {
		return nil, errFugleStreamClosed
	}
	return st.s.collect(ctx, ch, n)
}

// Dropped is the number of messages dropped because they were not read in
// time.
func (st *FugleStockStream) Dropped() uint64 {
	if st == nil || st.s == nil {
		return 0
	}
	return st.s.dropped.Load()
}

// Err returns the error that ended the stream: ErrFugleAuth, ErrFuglePlan or
// another error a reconnect cannot cure. It is nil while the stream runs and
// after a Close that nothing preceded.
func (st *FugleStockStream) Err() error {
	if st == nil || st.s == nil {
		return nil
	}
	return st.s.Err()
}

// Close ends the stream and closes Messages. It is safe to call more than
// once.
func (st *FugleStockStream) Close() error {
	if st == nil || st.s == nil {
		return nil
	}
	return st.s.close()
}

// Subscribe asks for a channel of the contracts and waits for Fugle to
// confirm. afterHours subscribes to the after-hours session; more than one
// flag is an error.
func (st *FugleFutOptStream) Subscribe(ch FugleChannel, symbols []string, afterHours ...bool) error {
	if st == nil || st.s == nil {
		return errFugleStreamClosed
	}
	return st.s.subscribe(ch, symbols, afterHours)
}

// Unsubscribe stops a channel of the contracts; each must have been
// subscribed with the same afterHours flag.
func (st *FugleFutOptStream) Unsubscribe(ch FugleChannel, symbols []string, afterHours ...bool) error {
	if st == nil || st.s == nil {
		return errFugleStreamClosed
	}
	return st.s.unsubscribe(ch, symbols, afterHours)
}

// Messages is FugleStockStream.Messages for futures and options.
func (st *FugleFutOptStream) Messages() <-chan FugleStreamMessage {
	if st == nil || st.s == nil {
		return fugleNoMessages
	}
	return st.s.msgs
}

// Collect is FugleStockStream.Collect for futures and options.
func (st *FugleFutOptStream) Collect(ctx context.Context, ch FugleChannel, n int) (*insyra.DataTable, error) {
	if st == nil || st.s == nil {
		return nil, errFugleStreamClosed
	}
	return st.s.collect(ctx, ch, n)
}

// Dropped is FugleStockStream.Dropped for futures and options.
func (st *FugleFutOptStream) Dropped() uint64 {
	if st == nil || st.s == nil {
		return 0
	}
	return st.s.dropped.Load()
}

// Err is FugleStockStream.Err for futures and options.
func (st *FugleFutOptStream) Err() error {
	if st == nil || st.s == nil {
		return nil
	}
	return st.s.Err()
}

// Close is FugleStockStream.Close for futures and options.
func (st *FugleFutOptStream) Close() error {
	if st == nil || st.s == nil {
		return nil
	}
	return st.s.close()
}
