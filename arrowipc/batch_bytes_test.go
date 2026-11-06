package arrowipc_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra/arrowipc"
	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	flatbuffers "github.com/google/flatbuffers/go"
)

func TestDecodedBatchByteBudgetPrecedesDecompression(t *testing.T) {
	data := streamFixture(t)
	footer := footerTable(data)
	block := footer.Vector(flatbuffers.UOffsetT(footer.Offset(10)))
	offset := int(binary.LittleEndian.Uint64(footer.Bytes[block : block+8]))
	metaLen := int(binary.LittleEndian.Uint32(footer.Bytes[block+8 : block+12]))
	metadata := data[offset+8 : offset+metaLen]
	message := flatbuffers.Table{Bytes: metadata, Pos: flatbuffers.GetUOffsetT(metadata)}
	record := indirectTable(message, 8)
	buffers := record.Vector(flatbuffers.UOffsetT(record.Offset(8)))
	body := data[offset+metaLen:]
	changed := false
	for i := 0; i < record.VectorLen(flatbuffers.UOffsetT(record.Offset(8))); i++ {
		buf := buffers + flatbuffers.UOffsetT(i*16)
		start := int(binary.LittleEndian.Uint64(metadata[buf : buf+8]))
		size := int(binary.LittleEndian.Uint64(metadata[buf+8 : buf+16]))
		if size >= 8 {
			binary.LittleEndian.PutUint64(body[start:start+8], 1025)
			changed = true
			break
		}
	}
	if !changed {
		t.Fatal("fixture has no compressed buffers")
	}
	tables, errs := streamFrom(context.Background(), bytes.NewReader(data), int64(len(data)), arrowipc.ReadOptions{Columns: namedColumns([]string{"id"}), MaxBatchBytes: 1024}, 1)
	got, err := drain(t, tables, errs)
	if len(got) != 0 || err == nil || !strings.Contains(err.Error(), "MaxBatchBytes") {
		t.Fatalf("decoded budget: %d tables, %v", len(got), err)
	}
}
func TestNativeByteBudgetAndNegativeOption(t *testing.T) {
	data := scalarFixture(t)
	for _, limit := range []int64{-1, 1} {
		tables, errs := streamFrom(context.Background(), bytes.NewReader(data), int64(len(data)), arrowipc.ReadOptions{MaxBatchBytes: limit}, 1)
		got, err := drain(t, tables, errs)
		if len(got) != 0 || err == nil || !strings.Contains(err.Error(), "MaxBatchBytes") {
			t.Fatalf("budget %d: %d tables, %v", limit, len(got), err)
		}
	}
}

func TestRetainedDictionaryConsumesDecodedBudget(t *testing.T) {
	typ := &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Int8, ValueType: arrow.BinaryTypes.String}
	schema := arrow.NewSchema([]arrow.Field{{Name: "category", Type: typ}}, nil)
	data := writeRecords(t, schema, func(b *array.RecordBuilder) {
		db := b.Field(0).(*array.BinaryDictionaryBuilder)
		for i := 0; i < 1500; i++ {
			if err := db.AppendString(strings.Repeat("x", 1024)); err != nil {
				t.Fatal(err)
			}
		}
	})
	for _, limit := range []int64{2000, 3000} {
		tables, errs := streamFrom(context.Background(), bytes.NewReader(data), int64(len(data)), arrowipc.ReadOptions{MaxBatchBytes: limit}, 100)
		got, err := drain(t, tables, errs)
		if limit == 2000 {
			if len(got) != 0 || err == nil || !strings.Contains(err.Error(), "MaxBatchBytes") {
				t.Fatalf("dictionary budget: %d tables %v", len(got), err)
			}
		} else if err != nil || len(got) != 15 {
			t.Fatalf("sufficient budget: %d tables %v", len(got), err)
		}
	}
}
