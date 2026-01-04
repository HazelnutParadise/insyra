package ipc

import (
	"encoding/binary"
	"io"
)

// ReadMessage reads a single length-prefixed message (uint32 little-endian)
// and returns the payload bytes.
func ReadMessage(r io.Reader) ([]byte, error) {
	var lenbuf [4]byte
	if _, err := io.ReadFull(r, lenbuf[:]); err != nil {
		return nil, err
	}
	l := binary.LittleEndian.Uint32(lenbuf[:])
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
