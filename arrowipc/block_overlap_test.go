package arrowipc_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra/arrowipc"
	flatbuffers "github.com/google/flatbuffers/go"
)

type watchedIPCBody struct {
	*bytes.Reader
	start, end int64
	reads      int
}

func (r *watchedIPCBody) ReadAt(p []byte, off int64) (int, error) {
	if off >= r.start && off < r.end {
		r.reads++
	}
	return r.Reader.ReadAt(p, off)
}

func TestOverlappingIPCBlocksFailBeforeBodyReads(t *testing.T) {
	for _, kind := range []string{"metadata", "body", "duplicate", "dictionary-body"} {
		t.Run(kind, func(t *testing.T) {
			data := scalarFixture(t)
			footer := footerTable(data)
			records := footer.Vector(flatbuffers.UOffsetT(footer.Offset(10)))
			first := footer.Bytes[records : records+24]
			second := footer.Bytes[records+24 : records+48]
			offset := int64(binary.LittleEndian.Uint64(first[:8]))
			metadata := int64(binary.LittleEndian.Uint32(first[8:12]))
			body := int64(binary.LittleEndian.Uint64(first[16:24]))
			if kind == "dictionary-body" {
				dictionaries := footer.Vector(flatbuffers.UOffsetT(footer.Offset(8)))
				dict := footer.Bytes[dictionaries : dictionaries+24]
				offset = int64(binary.LittleEndian.Uint64(dict[:8]))
				metadata = int64(binary.LittleEndian.Uint32(dict[8:12]))
				body = int64(binary.LittleEndian.Uint64(dict[16:24]))
			}
			if metadata <= 8 || body < 8 {
				t.Fatal("fixture needs metadata and a nonempty body")
			}
			bodyAt := offset + metadata
			switch kind {
			case "duplicate":
				copy(second, first)
			case "metadata":
				binary.LittleEndian.PutUint64(second[:8], uint64(offset+8))
				binary.LittleEndian.PutUint32(second[8:12], uint32(metadata-8))
				binary.LittleEndian.PutUint64(second[16:24], 0)
			default:
				binary.LittleEndian.PutUint64(second[:8], uint64(bodyAt))
				binary.LittleEndian.PutUint32(second[8:12], uint32(body))
				binary.LittleEndian.PutUint64(second[16:24], 0)
			}
			source := &watchedIPCBody{Reader: bytes.NewReader(data), start: bodyAt, end: bodyAt + body}
			tables, errs := streamFrom(context.Background(), source, int64(len(data)), arrowipc.ReadOptions{}, 3)
			got, err := drain(t, tables, errs)
			if len(got) != 0 || err == nil || !strings.Contains(err.Error(), "overlapping IPC blocks") || source.reads != 0 {
				t.Fatalf("%s: %d tables, %d body reads, error %v", kind, len(got), source.reads, err)
			}
		})
	}
}
