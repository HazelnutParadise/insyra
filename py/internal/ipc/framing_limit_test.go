package ipc

import (
	"bytes"
	"strings"
	"testing"
)

// IN-19 of #339: WriteMessage put len(b) into a uint32 prefix without checking
// it. A payload over 256 MiB wrote fine and the reader refused it; a payload
// over 4 GiB truncated the prefix, so the reader would take the wrong number of
// bytes and every message after it would be misframed.
func TestWriteMessage_RefusesWhatCannotBeRead(t *testing.T) {
	var buf bytes.Buffer

	err := WriteMessage(&buf, make([]byte, maxMessageSize+1))
	if err == nil {
		t.Fatal("a payload over the maximum was written")
	}
	if !strings.Contains(err.Error(), "exceeds maximum") {
		t.Errorf("the error %q does not say the message is too large", err)
	}
	if buf.Len() != 0 {
		t.Errorf("%d bytes were written before the refusal, which would misframe the stream", buf.Len())
	}
}

// What the writer accepts, the reader takes back unchanged.
func TestWriteMessage_RoundTrip(t *testing.T) {
	for _, size := range []int{0, 1, 1024, 1 << 20} {
		var buf bytes.Buffer
		payload := bytes.Repeat([]byte{0xAB}, size)

		if err := WriteMessage(&buf, payload); err != nil {
			t.Fatalf("size %d: %v", size, err)
		}
		got, err := ReadMessage(&buf)
		if err != nil {
			t.Fatalf("size %d: %v", size, err)
		}
		if !bytes.Equal(got, payload) {
			t.Errorf("size %d came back different", size)
		}
	}
}

// Anything the writer accepts must be inside what the reader accepts, or the
// two ends disagree about the frame.
func TestWriteAndReadLimitsMatch(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteMessage(&buf, make([]byte, maxMessageSize)); err != nil {
		t.Fatalf("a payload at exactly the maximum was refused: %v", err)
	}
	if _, err := ReadMessage(&buf); err != nil {
		t.Errorf("the reader refused a message the writer accepted: %v", err)
	}
}
