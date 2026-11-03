package arrowipc_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra/arrowipc"
	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	flatbuffers "github.com/google/flatbuffers/go"
)

func TestUnselectedViewVariadicCountsFailBeforeBodyReads(t *testing.T) {
	for _, count := range []int64{-1, 32768, 0} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			schema := arrow.NewSchema([]arrow.Field{{Name: "id", Type: arrow.PrimitiveTypes.Int64}, {Name: "view", Type: arrow.BinaryTypes.StringView}}, nil)
			data := writeRecords(t, schema, func(b *array.RecordBuilder) {
				b.Field(0).(*array.Int64Builder).Append(7)
				b.Field(1).(*array.StringViewBuilder).Append("x")
			})
			footer := footerTable(data)
			block := footer.Vector(flatbuffers.UOffsetT(footer.Offset(10)))
			offset := int(binary.LittleEndian.Uint64(footer.Bytes[block:]))
			metadata := int(binary.LittleEndian.Uint32(footer.Bytes[block+8:]))
			body := int64(binary.LittleEndian.Uint64(footer.Bytes[block+16:]))
			msg := data[offset+8 : offset+metadata]
			message := flatbuffers.Table{Bytes: msg, Pos: flatbuffers.GetUOffsetT(msg)}
			record := indirectTable(message, 8)
			slot := flatbuffers.UOffsetT(record.Offset(12))
			if slot == 0 || record.VectorLen(slot) != 1 {
				t.Fatal("fixture needs one view count")
			}
			at := record.Vector(slot)
			binary.LittleEndian.PutUint64(msg[at:], uint64(count))
			source := &watchedIPCBody{Reader: bytes.NewReader(data), start: int64(offset + metadata), end: int64(offset+metadata) + body}
			tables, errs := arrowipc.StreamFrom(context.Background(), source, int64(len(data)), arrowipc.ReadOptions{Columns: []string{"id"}}, 1)
			got, err := drain(t, tables, errs)
			if count == 0 {
				if err != nil || len(got) != 1 || got[0].GetColByNumber(0).Get(0) != int64(7) {
					t.Fatalf("valid unselected view: %d tables, %v", len(got), err)
				}
				return
			}
			if len(got) != 0 || err == nil || !strings.Contains(err.Error(), "variadic") || source.reads != 0 {
				t.Fatalf("count %d: %d tables, %d body reads, error %v", count, len(got), source.reads, err)
			}
		})
	}
}
