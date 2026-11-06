package arrowipc_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra/arrowipc"
	"github.com/apache/arrow-go/v18/arrow"
	flatbuffers "github.com/google/flatbuffers/go"
)

func TestAliasedSchemaStringsAreBudgetedPerOccurrence(t *testing.T) {
	for _, kind := range []string{"schema-metadata", "field-metadata", "field-name"} {
		t.Run(kind, func(t *testing.T) {
			const count = 1025
			keys, values := make([]string, count), make([]string, count)
			for i := range keys {
				keys[i], values[i] = fmt.Sprintf("key%d", i), "x"
			}
			values[0] = strings.Repeat("x", 64<<10)
			metadata := arrow.NewMetadata(keys, values)
			fields := []arrow.Field{{Name: "id", Type: arrow.PrimitiveTypes.Int64}}
			var schema *arrow.Schema
			switch kind {
			case "schema-metadata":
				schema = arrow.NewSchema(fields, &metadata)
			case "field-metadata":
				fields[0].Metadata = metadata
				schema = arrow.NewSchema(fields, nil)
			default:
				fields = make([]arrow.Field, count)
				for i := range fields {
					fields[i] = arrow.Field{Name: values[i], Type: arrow.PrimitiveTypes.Int64}
				}
				schema = arrow.NewSchema(fields, nil)
			}
			data := schemaOnlyFixture(t, schema)
			tab := indirectTable(footerTable(data), 6)
			slot, textSlot := flatbuffers.VOffsetT(8), flatbuffers.VOffsetT(6)
			switch kind {
			case "field-metadata":
				at := tab.Vector(flatbuffers.UOffsetT(tab.Offset(6)))
				tab = flatbuffers.Table{Bytes: tab.Bytes, Pos: tab.Indirect(at)}
				slot = 16
			case "field-name":
				slot, textSlot = 6, 4
			}
			at := tab.Vector(flatbuffers.UOffsetT(tab.Offset(slot)))
			var positions []flatbuffers.UOffsetT
			var shared flatbuffers.UOffsetT
			for i := 0; i < count; i++ {
				entry := flatbuffers.Table{Bytes: tab.Bytes, Pos: tab.Indirect(at + flatbuffers.UOffsetT(i*4))}
				position := entry.Pos + flatbuffers.UOffsetT(entry.Offset(textSlot))
				target := position + flatbuffers.GetUOffsetT(entry.Bytes[position:])
				if flatbuffers.GetUOffsetT(entry.Bytes[target:]) == 64<<10 {
					shared = target
				}
				positions = append(positions, position)
			}
			for _, position := range positions {
				if shared <= position {
					t.Fatal("fixture needs a forward shared string")
				}
				binary.LittleEndian.PutUint32(tab.Bytes[position:], uint32(shared-position))
			}
			runtime.GC()
			var before, after runtime.MemStats
			runtime.ReadMemStats(&before)
			_, err := arrowipc.InspectFrom(context.Background(), bytes.NewReader(data), int64(len(data)))
			runtime.ReadMemStats(&after)
			if err == nil || !strings.Contains(err.Error(), "metadata strings") {
				t.Fatalf("%s: expected decoded metadata string budget error, got %v", kind, err)
			}
			if allocated := after.TotalAlloc - before.TotalAlloc; allocated > 2<<20 {
				t.Fatalf("%d-byte fixture allocated %d bytes", len(data), allocated)
			}
		})
	}
}

func TestSharedMetadataStringWithinBudgetRemainsReadable(t *testing.T) {
	metadata := arrow.NewMetadata([]string{"first", "second"}, []string{"shared", "short"})
	data := schemaOnlyFixture(t, arrow.NewSchema(nil, &metadata))
	schema := indirectTable(footerTable(data), 6)
	vector := schema.Vector(flatbuffers.UOffsetT(schema.Offset(8)))
	first := flatbuffers.Table{Bytes: schema.Bytes, Pos: schema.Indirect(vector)}
	second := flatbuffers.Table{Bytes: schema.Bytes, Pos: schema.Indirect(vector + 4)}
	firstValue := first.Pos + flatbuffers.UOffsetT(first.Offset(6))
	target := firstValue + flatbuffers.GetUOffsetT(first.Bytes[firstValue:])
	secondValue := second.Pos + flatbuffers.UOffsetT(second.Offset(6))
	if target <= secondValue {
		t.Fatal("fixture needs a forward shared string")
	}
	binary.LittleEndian.PutUint32(second.Bytes[secondValue:], uint32(target-secondValue))
	info, err := arrowipc.InspectFrom(context.Background(), bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	clear(data)
	if value, ok := info.Metadata["second"]; !ok || value != "shared" {
		t.Fatalf("shared metadata was not preserved: %q, %v", value, ok)
	}
}
