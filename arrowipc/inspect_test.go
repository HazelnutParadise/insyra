package arrowipc_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"reflect"
	"testing"

	"github.com/HazelnutParadise/insyra/arrowipc"
	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/ipc"
	"github.com/apache/arrow-go/v18/arrow/memory"
)

type footerOnlySource struct {
	data    []byte
	footer  int64
	fetched int
}

func (s *footerOnlySource) ReadAt(p []byte, off int64) (int, error) {
	end := off + int64(len(p))
	if off < 0 || (off < s.footer && end > 8) {
		return 0, fmt.Errorf("inspection touched data: [%d,%d)", off, end)
	}
	s.fetched += len(p)
	return bytes.NewReader(s.data).ReadAt(p, off)
}

func TestInspectDictionaryAndNestedSchemaWithoutValues(t *testing.T) {
	dict := &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Int8, ValueType: arrow.BinaryTypes.String, Ordered: true}
	md := arrow.NewMetadata([]string{"source"}, []string{"schema-only proof"})
	schema := arrow.NewSchema([]arrow.Field{
		{Name: "status", Type: dict, Nullable: true, Metadata: md},
		{Name: "ids", Type: arrow.ListOf(arrow.PrimitiveTypes.Int64), Nullable: true},
	}, &md)
	b := array.NewRecordBuilder(memory.DefaultAllocator, schema)
	defer b.Release()
	var buf bytes.Buffer
	w, err := ipc.NewFileWriter(&buf, ipc.WithSchema(schema))
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if err := b.Field(0).(*array.BinaryDictionaryBuilder).AppendString("台灣"); err != nil {
			t.Fatal(err)
		}
		lb := b.Field(1).(*array.ListBuilder)
		lb.Append(true)
		lb.ValueBuilder().(*array.Int64Builder).Append(9007199254740993)
		rec := b.NewRecordBatch()
		err = w.Write(rec)
		rec.Release()
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	data := buf.Bytes()
	original := bytes.Clone(data)
	footerLen := int64(binary.LittleEndian.Uint32(data[len(data)-10:]))
	src := &footerOnlySource{data: data, footer: int64(len(data)) - 10 - footerLen}
	info, err := arrowipc.InspectFrom(context.Background(), src, int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	if !info.Schema.Equal(schema) || !reflect.DeepEqual(info.Schema.Metadata(), schema.Metadata()) {
		t.Fatalf("schema changed: %s", info.Schema)
	}
	if info.NumBatches != 3 || info.Version != "V5" {
		t.Fatalf("metadata: %#v", info)
	}
	if src.fetched > int(footerLen)+18 {
		t.Fatalf("fetched %d bytes for footer %d", src.fetched, footerLen)
	}
	if !bytes.Equal(data, original) {
		t.Fatal("inspection mutated source")
	}
	// The schema stays valid after the backing source is overwritten.
	clear(data)
	if !info.Schema.Equal(schema) {
		t.Fatal("schema borrowed source memory")
	}
}

func TestInspectBadReaderAtDoesNotPanic(t *testing.T) {
	_, err := arrowipc.InspectFrom(context.Background(), panicReaderAt{}, 100)
	if err == nil {
		t.Fatal("expected error for panicking source")
	}
}

type panicReaderAt struct{}

func (panicReaderAt) ReadAt([]byte, int64) (int, error) { panic(io.ErrUnexpectedEOF) }
