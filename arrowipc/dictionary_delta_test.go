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
	"github.com/apache/arrow-go/v18/arrow/ipc"
	"github.com/apache/arrow-go/v18/arrow/memory"
	flatbuffers "github.com/google/flatbuffers/go"
)

func deltaFixture(t *testing.T, text bool) []byte {
	t.Helper()
	valueType := arrow.PrimitiveTypes.Int64
	if text {
		valueType = arrow.BinaryTypes.String
	}
	typ := &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Int8, ValueType: valueType, Ordered: true}
	schema := arrow.NewSchema([]arrow.Field{{Name: "id", Type: arrow.PrimitiveTypes.Int64}, {Name: "category", Type: typ}}, nil)
	var base, stream bytes.Buffer
	fw, err := ipc.NewFileWriter(&base, ipc.WithSchema(schema))
	if err != nil {
		t.Fatal(err)
	}
	sw := ipc.NewWriter(&stream, ipc.WithSchema(schema), ipc.WithDictionaryDeltas(true))
	b := array.NewRecordBuilder(memory.DefaultAllocator, schema)
	defer b.Release()
	for i := 0; i < 2; i++ {
		b.Field(0).(*array.Int64Builder).Append(int64(i + 1))
		if text {
			values := []string{"first", "second"}
			vb := array.NewStringBuilder(memory.DefaultAllocator)
			vb.AppendValues(values[:i+1], nil)
			v := vb.NewStringArray()
			db := b.Field(1).(*array.BinaryDictionaryBuilder)
			err = db.InsertStringDictValues(v)
			v.Release()
			vb.Release()
			if err == nil {
				err = db.AppendString(values[i])
			}
		} else {
			vb := array.NewInt64Builder(memory.DefaultAllocator)
			vb.AppendValues([]int64{11, 12}[:i+1], nil)
			v := vb.NewInt64Array()
			db := b.Field(1).(*array.Int64DictionaryBuilder)
			err = db.InsertDictValues(v)
			v.Release()
			vb.Release()
			if err == nil {
				err = db.Append(int64(i + 11))
			}
		}
		if err != nil {
			t.Fatal(err)
		}
		rec := b.NewRecordBatch()
		if i == 0 {
			err = fw.Write(rec)
		}
		if err == nil {
			err = sw.Write(rec)
		}
		rec.Release()
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := fw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := sw.Close(); err != nil {
		t.Fatal(err)
	}
	// Arrow's file writer refuses dictionary updates. Use the actual delta
	// message emitted by its stream writer with the file writer's footer/schema.
	var deltaFrame []byte
	var deltaMetadata int
	var deltaBody int64
	encoded := stream.Bytes()
	for off := 0; off+8 <= len(encoded); {
		if binary.LittleEndian.Uint32(encoded[off:]) != 0xffffffff {
			t.Fatal("expected continuation frame")
		}
		metadata := int(binary.LittleEndian.Uint32(encoded[off+4:]))
		if metadata == 0 {
			break
		}
		msgBytes := encoded[off+8 : off+8+metadata]
		msg := flatbuffers.Table{Bytes: msgBytes, Pos: flatbuffers.GetUOffsetT(msgBytes)}
		body := int64(0)
		if o := msg.Offset(10); o != 0 {
			body = msg.GetInt64(msg.Pos + flatbuffers.UOffsetT(o))
		}
		size := 8 + metadata + int(body)
		if o := msg.Offset(6); o != 0 && msg.GetByte(msg.Pos+flatbuffers.UOffsetT(o)) == 2 {
			dict := indirectTable(msg, 8)
			if o := dict.Offset(8); o != 0 && dict.GetByte(dict.Pos+flatbuffers.UOffsetT(o)) != 0 {
				deltaFrame = bytes.Clone(encoded[off : off+size])
				deltaMetadata = 8 + metadata
				deltaBody = body
			}
		}
		off += size
	}
	if len(deltaFrame) == 0 {
		t.Fatal("stream writer did not emit a delta")
	}
	original := base.Bytes()
	oldFooter := footerTable(original)
	footer := bytes.Clone(oldFooter.Bytes)
	tab := flatbuffers.Table{Bytes: footer, Pos: oldFooter.Pos}
	records := tab.Vector(flatbuffers.UOffsetT(tab.Offset(10)))
	recordAt := int(binary.LittleEndian.Uint64(footer[records:]))
	binary.LittleEndian.PutUint64(footer[records:], uint64(recordAt+len(deltaFrame)))
	dictionaries := tab.Vector(flatbuffers.UOffsetT(tab.Offset(8)))
	initial := bytes.Clone(footer[dictionaries : dictionaries+24])
	for len(footer)%8 != 4 {
		footer = append(footer, 0)
	}
	vectorAt := len(footer)
	footer = binary.LittleEndian.AppendUint32(footer, 2)
	footer = append(footer, initial...)
	block := make([]byte, 24)
	binary.LittleEndian.PutUint64(block, uint64(recordAt))
	binary.LittleEndian.PutUint32(block[8:], uint32(deltaMetadata))
	binary.LittleEndian.PutUint64(block[16:], uint64(deltaBody))
	footer = append(footer, block...)
	dictSlot := int(tab.Pos) + int(tab.Offset(8))
	binary.LittleEndian.PutUint32(footer[dictSlot:], uint32(vectorAt-dictSlot))
	footerAt := len(original) - 10 - len(oldFooter.Bytes)
	data := append(bytes.Clone(original[:recordAt]), deltaFrame...)
	data = append(data, original[recordAt:footerAt]...)
	data = append(data, footer...)
	data = binary.LittleEndian.AppendUint32(data, uint32(len(footer)))
	return append(data, "ARROW1"...)
}

func TestDictionaryDeltasFailBeforeTheirBodies(t *testing.T) {
	for _, kind := range []string{"fixed", "strings", "fixed-length", "binary-offset", "zero-body", "malformed-flag"} {
		for _, selected := range []string{"category", "id"} {
			t.Run(kind+"/"+selected, func(t *testing.T) {
				text := kind == "strings" || kind == "binary-offset"
				data := deltaFixture(t, text)
				footer := footerTable(data)
				slot := flatbuffers.UOffsetT(footer.Offset(8))
				if footer.VectorLen(slot) != 2 {
					t.Fatal("fixture needs base and delta")
				}
				at := footer.Vector(slot) + 24
				block := footer.Bytes[at : at+24]
				offset := int(binary.LittleEndian.Uint64(block[:8]))
				metadata := int(binary.LittleEndian.Uint32(block[8:12]))
				body := int64(binary.LittleEndian.Uint64(block[16:24]))
				messageBytes := data[offset+8 : offset+metadata]
				message := flatbuffers.Table{Bytes: messageBytes, Pos: flatbuffers.GetUOffsetT(messageBytes)}
				delta := indirectTable(message, 8)
				if o := delta.Offset(8); o == 0 || delta.GetByte(delta.Pos+flatbuffers.UOffsetT(o)) == 0 {
					t.Fatal("writer did not emit a delta")
				}
				record := indirectTable(delta, 6)
				nodes := record.Vector(flatbuffers.UOffsetT(record.Offset(6)))
				buffers := record.Vector(flatbuffers.UOffsetT(record.Offset(8)))
				bodyAt := offset + metadata
				switch kind {
				case "malformed-flag":
					vtable := int(delta.Pos) - int(flatbuffers.GetSOffsetT(delta.Bytes[delta.Pos:]))
					binary.LittleEndian.PutUint16(delta.Bytes[vtable+8:], 65535)
				case "fixed-length", "zero-body":
					rows := uint64(65536)
					if kind == "zero-body" {
						rows = 0
					}
					binary.LittleEndian.PutUint64(record.Bytes[nodes:], rows)
					for i := 0; i < record.VectorLen(flatbuffers.UOffsetT(record.Offset(8))); i++ {
						binary.LittleEndian.PutUint64(record.Bytes[buffers+flatbuffers.UOffsetT(i*16+8):], 0)
					}
					if kind == "zero-body" {
						binary.LittleEndian.PutUint64(block[16:24], 0)
						if o := message.Offset(10); o != 0 {
							message.MutateInt64Slot(10, 0)
						}
					}
				case "binary-offset":
					offsets := int(binary.LittleEndian.Uint64(record.Bytes[buffers+16:]))
					binary.LittleEndian.PutUint32(data[bodyAt+offsets+4:], 32768)
				}
				info, err := arrowipc.InspectFrom(context.Background(), bytes.NewReader(data), int64(len(data)))
				if err != nil || info.NumBatches != 1 {
					t.Fatalf("delta schema inspection: %v", err)
				}
				if _, ok := info.Schema.Field(1).Type.(*arrow.DictionaryType); !ok {
					t.Fatal("dictionary type lost")
				}
				source := &watchedIPCBody{Reader: bytes.NewReader(data), start: int64(bodyAt), end: int64(bodyAt) + body}
				tables, errs := streamFrom(context.Background(), source, int64(len(data)), arrowipc.ReadOptions{Columns: namedColumns([]string{selected}), MaxBatchBytes: 1024}, 1)
				got, err := drain(t, tables, errs)
				wantError := "dictionary delta"
				if kind == "malformed-flag" {
					wantError = "metadata"
				}
				if len(got) != 0 || err == nil || !strings.Contains(err.Error(), wantError) || source.reads != 0 {
					t.Fatalf("%d tables, %d delta body reads, error %v", len(got), source.reads, err)
				}
			})
		}
	}
}
