package py

import (
	stdjson "encoding/json"
	"runtime"
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/py/internal/ipc"
)

// A DataFrame or Series result that is a table or a list itself used to go
// into any struct field named DataTable or DataList, embedded or not; the
// per-field decoder honored only the embedded one. A dict is still decoded
// key by key beside such a field.
func TestAWholeTableOrListFieldTakesTheResult(t *testing.T) {
	type tableHolder struct {
		DataTable *insyra.DataTable
	}
	var table tableHolder
	if err := bindJSON(t, &table, frame); err != nil {
		t.Fatal(err)
	}
	checkFrame(t, "a named DataTable field", table.DataTable)

	type listHolder struct {
		DataList *insyra.DataList
	}
	var list listHolder
	if err := bindJSON(t, &list, series); err != nil {
		t.Fatal(err)
	}
	if list.DataList == nil || list.DataList.Len() != 3 || list.DataList.GetName() != "s" {
		t.Errorf("a named DataList field came back as %v", list.DataList)
	}

	type keyed struct {
		Score float64
		Table *insyra.DataTable
	}
	var byKey keyed
	if err := bindJSON(t, &byKey, `{"Score":7,"Table":`+frame+`}`); err != nil {
		t.Fatal(err)
	}
	if byKey.Score != 7 {
		t.Errorf("Score is %v, want 7", byKey.Score)
	}
	checkFrame(t, "a table under its own key beside a score", byKey.Table)
}

// A key is matched to a field the way encoding/json matches it, with Unicode
// simple folding. A package of its own used unicode.ToUpper(unicode.ToLower),
// which folded the dotless i to I where encoding/json does not.
func TestFieldNamesAreMatchedWithEqualFold(t *testing.T) {
	type withTable struct {
		I     int
		Table *insyra.DataTable
	}
	var got withTable
	if err := bindJSON(t, &got, `{"ı": 7}`); err != nil {
		t.Fatal(err)
	}
	var want struct {
		I int
	}
	if err := stdjson.Unmarshal([]byte(`{"ı": 7}`), &want); err != nil {
		t.Fatal(err)
	}
	if got.I != want.I {
		t.Errorf("I is %d, encoding/json gives %d", got.I, want.I)
	}
}

// A listener whose accept loop stopped used to leave the shared server marked
// usable, so the next call got the dead address. A run still using it keeps
// its count; the next acquire opens a fresh server.
func TestAnIPCListenerThatStopsIsReplaced(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("named pipe names do not depend on TMPDIR")
	}
	t.Setenv("TMPDIR", "")
	t.Cleanup(func() {
		serverMu.Lock()
		if serverLn != nil {
			_ = serverLn.Close()
		}
		serverLn, serverAddr, serverUsers = nil, "", 0
		serverMu.Unlock()
	})

	first, err := acquireIPCServer()
	if err != nil {
		t.Fatal(err)
	}
	serverMu.Lock()
	ln := serverLn
	serverMu.Unlock()
	if ln == nil {
		t.Fatal("the IPC server has no listener")
	}
	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(5 * time.Second)
	for {
		serverMu.Lock()
		stopped := serverLn == nil
		serverMu.Unlock()
		if stopped {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the stopped listener's server was still marked usable")
		}
		time.Sleep(time.Millisecond)
	}

	second, err := acquireIPCServer()
	if err != nil {
		t.Fatalf("acquiring a server after the listener stopped: %v", err)
	}
	defer releaseIPCServer()
	defer releaseIPCServer()
	if second == first {
		t.Errorf("the stopped server's address %q was handed out again", second)
	}
	conn, err := ipc.Dial(second)
	if err != nil {
		t.Fatalf("dialling the replacement server: %v", err)
	}
	_ = conn.Close()
}
