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
	defer ln.Close()

	// server
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		b, err := ReadMessage(c)
		if err != nil {
			return
		}
		// echo back a JSON object
		var obj map[string]interface{}
		json.Unmarshal(b, &obj)
		resp, _ := json.Marshal(map[string]string{"status": "ok"})
		WriteMessage(c, resp)
	}()

	// client
	var conn io.ReadWriteCloser
	connRaw, err := Dial(addr)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	conn = connRaw
	defer conn.Close()

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
