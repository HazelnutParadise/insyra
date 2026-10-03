package py

import (
	"errors"
	"fmt"
	"math"
	"net"
	"reflect"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra/py/internal/ipc"
	json "github.com/goccy/go-json"
)

func asFloat(v any) float64 {
	f, _ := v.(float64)
	return f
}

// The review of py-ipc-server-errors measured a DataFrame with a missing value
// coming back as nil: Python's json.dumps writes NaN, Infinity and -Infinity,
// which go-json refused.
func TestDecodeResultMessageReadsPythonsNonFiniteFloats(t *testing.T) {
	msg := `{"execution_id": "ab12", "data": [{"a": [1.5, NaN, Infinity, -Infinity, 1e-07, 10], "s": "NaN, Infinity and -Infinity", "q": "say \"NaN\" Infinity \\"}, null]}`
	m, err := decodeResultMessage([]byte(msg))
	if err != nil {
		t.Fatal(err)
	}
	if m.ExecutionID != "ab12" || m.Data[1] != nil {
		t.Fatalf("got %+v", m)
	}
	got, ok := m.Data[0].(map[string]any)
	if !ok {
		t.Fatalf("the result is %T", m.Data[0])
	}
	a, ok := got["a"].([]any)
	if !ok || len(a) != 6 {
		t.Fatalf("a is %#v", got["a"])
	}
	if a[0] != 1.5 || !math.IsNaN(asFloat(a[1])) || !math.IsInf(asFloat(a[2]), 1) || !math.IsInf(asFloat(a[3]), -1) || a[4] != 1e-07 || a[5] != float64(10) {
		t.Errorf("a is %#v", a)
	}
	if got["s"] != "NaN, Infinity and -Infinity" {
		t.Errorf("s is %q", got["s"])
	}
	if got["q"] != `say "NaN" Infinity \` {
		t.Errorf("q is %q", got["q"])
	}
}

func TestDecodeResultMessageLeavesOrdinaryMessagesAlone(t *testing.T) {
	msg := `{"execution_id": "ab12", "data": [{"n": 0.1, "edge": 9007199254740992, "t": true, "l": [null, "x"]}, "boom"]}`
	m, err := decodeResultMessage([]byte(msg))
	if err != nil {
		t.Fatal(err)
	}
	var want resultMessage
	if err := json.Unmarshal([]byte(msg), &want); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(m, want) {
		t.Errorf("got %#v, want %#v", m, want)
	}
}

// Read again because of a NaN, the other numbers in a message come out as
// they would have without it.
func TestNumbersBesideANaNDecodeAsTheyWouldWithout(t *testing.T) {
	numbers := `[0.1, 1e+300, -2.5e-08, 12345678901234567890, 0, -0.0, 3]`
	withNaN, err := decodeResultMessage([]byte(`{"execution_id": "a1", "data": [[NaN, ` + numbers + `], null]}`))
	if err != nil {
		t.Fatal(err)
	}
	plain, err := decodeResultMessage([]byte(`{"execution_id": "a1", "data": [[null, ` + numbers + `], null]}`))
	if err != nil {
		t.Fatal(err)
	}
	got, _ := withNaN.Data[0].([]any)
	want, _ := plain.Data[0].([]any)
	if len(got) != 2 || len(want) != 2 || !reflect.DeepEqual(got[1], want[1]) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestANumberTooLargeForAFloatIsAnError(t *testing.T) {
	big := strings.Repeat("9", 400)
	for _, msg := range []string{
		`{"execution_id": "a1", "data": [` + big + `, null]}`,
		`{"execution_id": "a1", "data": [[NaN, ` + big + `], null]}`,
	} {
		if _, err := decodeResultMessage([]byte(msg)); err == nil || !strings.Contains(err.Error(), "float64") {
			t.Errorf("got %v, want an error saying the number does not fit in a float64", err)
		}
	}
}

// exchange sends msg to the IPC handler and returns its answer.
func exchange(t *testing.T, msg string) map[string]string {
	t.Helper()
	client, server := net.Pipe()
	go handleIPCConnection(server)
	defer func() { _ = client.Close() }()
	if err := ipc.WriteMessage(client, []byte(msg)); err != nil {
		t.Fatal(err)
	}
	reply, err := ipc.ReadMessage(client)
	if err != nil {
		t.Fatalf("no answer: %v", err)
	}
	var r map[string]string
	if err := json.Unmarshal(reply, &r); err != nil {
		t.Fatal(err)
	}
	return r
}

func TestTheHandlerStoresANaNResult(t *testing.T) {
	id := "feed01"
	defer resultStore.Delete(id)
	if r := exchange(t, `{"execution_id": "`+id+`", "data": [[1, NaN], null]}`); r["status"] != "ok" {
		t.Fatalf("the answer was %v", r)
	}
	v, ok := resultStore.Load(id)
	if !ok {
		t.Fatal("nothing was stored")
	}
	list, _ := v.([2]any)[0].([]any)
	if len(list) != 2 || !math.IsNaN(asFloat(list[1])) {
		t.Errorf("stored %#v", v)
	}
}

// A message the handler could not read was closed without an answer, which
// Python took for success, and the run returned nil.
func TestTheHandlerAnswersAMessageItCannotReadWithAnError(t *testing.T) {
	id := "feed02"
	defer refusedStore.Delete(id)
	r := exchange(t, `{"execution_id": "`+id+`", "data": [`+strings.Repeat("9", 400)+`, null]}`)
	if r["status"] != "error" || !strings.Contains(r["error"], "float64") {
		t.Fatalf("the answer was %v", r)
	}
	if _, delivered := resultStore.Load(id); delivered {
		t.Error("the refusal was stored as the run's result")
	}
	v, ok := refusedStore.Load(id)
	if !ok {
		t.Fatal("the reason was not kept for the run")
	}
	if !strings.Contains(fmt.Sprint(v), "float64") {
		t.Errorf("kept %#v", v)
	}
	// Without an ID to store it under, the answer still carries the reason.
	if r := exchange(t, `not json`); r["status"] != "error" || r["error"] == "" {
		t.Errorf("the answer was %v", r)
	}
}

// The review of py-results-keep-nan found the refusal stored as the run's
// result while Python was still running: code that caught the error
// insyra.Return raised and returned something else got either value, by
// timing, and the other one was left in the store.
func TestALaterResultWinsOverARefusal(t *testing.T) {
	id := "feed03"
	defer resultStore.Delete(id)
	refusedStore.Store(id, "py: the result Python sent could not be read: x")
	resultStore.Store(id, [2]any{"fallback", nil})
	processDone := make(chan struct{})
	close(processDone)
	got := waitForResult(id, processDone, make(chan error, 1))
	if got[0] != "fallback" || got[1] != nil {
		t.Errorf("got %#v, want the fallback", got)
	}
	if _, left := refusedStore.Load(id); left {
		t.Error("the refusal was left behind")
	}
}

func TestARefusalIsTheErrorWhenNothingElseArrives(t *testing.T) {
	reason := "py: the result Python sent could not be read: x"
	// The process failed, as insyra.Return raising makes it.
	id := "feed04"
	refusedStore.Store(id, reason)
	execErr := make(chan error, 1)
	execErr <- errors.New("exit status 1")
	processDone := make(chan struct{})
	close(processDone)
	if got := waitForResult(id, processDone, execErr); got[0] != nil || got[1] != reason {
		t.Errorf("after a failed process got %#v, want the refusal", got)
	}
	// The process exited cleanly, as when Python swallowed the error.
	id = "feed05"
	refusedStore.Store(id, reason)
	if got := waitForResult(id, processDone, make(chan error, 1)); got[0] != nil || got[1] != reason {
		t.Errorf("after a clean exit got %#v, want the refusal", got)
	}
	for _, id := range []string{"feed04", "feed05"} {
		if _, left := refusedStore.Load(id); left {
			t.Errorf("the refusal for %s was left behind", id)
		}
	}
}

// The review of py-nested-table-results measured 2^53 + 1 coming back as 2^53:
// every number was decoded into a float64, which does not hold every integer
// past 2^53.
func TestAnIntegerAboveTwoToThe53KeepsEveryDigit(t *testing.T) {
	msg := `{"execution_id": "a1", "data": [[9007199254740993, -9007199254740993, 9223372036854775807, -9223372036854775808, 18446744073709551615, 9007199254740992, 3, 0.5, 1e+20, 9007199254740993.0, 100000000000000000000000], null]}`
	m, err := decodeResultMessage([]byte(msg))
	if err != nil {
		t.Fatal(err)
	}
	got, _ := m.Data[0].([]any)
	want := []any{int64(9007199254740993), int64(-9007199254740993), int64(9223372036854775807), int64(-9223372036854775808), uint64(18446744073709551615), float64(9007199254740992), float64(3), 0.5, 1e+20, float64(9007199254740993.0), 1e23}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got  %#v\nwant %#v", got, want)
	}
}

// Digits in a string are not a number and change nothing.
func TestLongDigitsInAStringChangeNothing(t *testing.T) {
	m, err := decodeResultMessage([]byte(`{"execution_id": "a1", "data": [{"s": "12345678901234567890", "n": 2}, null]}`))
	if err != nil {
		t.Fatal(err)
	}
	got, _ := m.Data[0].(map[string]any)
	if got["s"] != "12345678901234567890" || got["n"] != float64(2) {
		t.Errorf("got %#v", got)
	}
}

// A decimal's digits do not send a message the longer way: Python prints most
// floats with sixteen or seventeen of them.
func TestOnlyALongIntegerTakesTheLongerDecoding(t *testing.T) {
	for msg, want := range map[string]bool{
		`[0.8444218515250481, 1.2345678901234567e+20, 12345678901234567.5, 3]`: false,
		`[9007199254740993]`:       true,
		`[-9007199254740993]`:      true,
		`["12345678901234567890"]`: false,
		`{"s": "a\"12345678901234567890", "n": 9007199254740993}`: true,
		`["\\", 9007199254740993]`:                                true,
	} {
		if got := hasLongInteger([]byte(msg)); got != want {
			t.Errorf("hasLongInteger(%s) = %v, want %v", msg, got, want)
		}
	}
}
