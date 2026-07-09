package ipc

import (
	"encoding/binary"
	"fmt"
	"io"
)

// maxMessageSize bounds the length prefix so a malformed/hostile peer cannot
// trigger a multi-gigabyte allocation from a single 4-byte header.
const maxMessageSize = 256 << 20 // 256 MiB

// ReadMessage reads a single length-prefixed message (uint32 little-endian)
// and returns the payload bytes.
func ReadMessage(r io.Reader) ([]byte, error) {
	var lenbuf [4]byte
	if _, err := io.ReadFull(r, lenbuf[:]); err != nil {
		return nil, err
	}
	l := binary.LittleEndian.Uint32(lenbuf[:])
	if l > maxMessageSize {
		return nil, fmt.Errorf("ipc: message length %d exceeds maximum %d", l, maxMessageSize)
	}
	buf := make([]byte, l)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

// WriteMessage writes a single length-prefixed message.
func WriteMessage(w io.Writer, b []byte) error {
	var lenbuf [4]byte
	binary.LittleEndian.PutUint32(lenbuf[:], uint32(len(b)))
	if _, err := w.Write(lenbuf[:]); err != nil {
		return err
	}
	_, err := w.Write(b)
	return err
}
