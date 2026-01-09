package ipc

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestListenAndDial(t *testing.T) {
	var addr string
	if runtime.GOOS == "windows" {
		addr = `\\.\\pipe\\insyra_test_pipe`
	} else {
		addr = filepath.Join(os.TempDir(), "insyra_test.sock")
		// remove leftover
		_ = os.Remove(addr)
	}

	ln, err := Listen(addr)
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	defer func() {
		if cerr := ln.Close(); cerr != nil {
			t.Errorf("ln.Close error: %v", cerr)
		}
	}()

	// server
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() {
			if cerr := c.Close(); cerr != nil {
				t.Logf("c.Close error: %v", cerr)
			}
		}()
		b, err := ReadMessage(c)
		if err != nil {
			return
		}
		// echo back a JSON object
		var obj map[string]interface{}
		if err := json.Unmarshal(b, &obj); err != nil {
			t.Logf("json.Unmarshal error: %v", err)
			return
		}
		resp, merr := json.Marshal(map[string]string{"status": "ok"})
		if merr != nil {
			t.Logf("json.Marshal error: %v", merr)
			return
		}
		if werr := WriteMessage(c, resp); werr != nil {
			t.Logf("WriteMessage error: %v", werr)
			return
		}
	}()

	// client
	var conn io.ReadWriteCloser
	connRaw, err := Dial(addr)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	conn = connRaw
	defer func() {
		if cerr := conn.Close(); cerr != nil {
			t.Errorf("conn.Close error: %v", cerr)
		}
	}()

	msg, _ := json.Marshal(map[string]string{"hello": "world"})
	if err := WriteMessage(conn, msg); err != nil {
		t.Fatalf("WriteMessage failed: %v", err)
	}

	resp, err := ReadMessage(conn)
	if err != nil {
		t.Fatalf("ReadMessage failed: %v", err)
	}
	var r map[string]string
	if err := json.Unmarshal(resp, &r); err != nil {
		t.Fatalf("unmarshal resp: %v", err)
	}
	if r["status"] != "ok" {
		t.Fatalf("unexpected resp: %v", r)
	}
	// wait a bit to ensure socket file cleanup on unix
	time.Sleep(10 * time.Millisecond)
}
