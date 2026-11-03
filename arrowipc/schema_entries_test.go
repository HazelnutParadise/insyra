package arrowipc_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra/arrowipc"
	"github.com/apache/arrow-go/v18/arrow"
	flatbuffers "github.com/google/flatbuffers/go"
)

func TestExpandedSchemaEntryLimit(t *testing.T) {
	for _, kind := range []string{"fields", "metadata"} {
		t.Run(kind, func(t *testing.T) {
			const count = 65537
			fields := []arrow.Field{{Type: arrow.PrimitiveTypes.Int64}}
			var md *arrow.Metadata
			if kind == "fields" {
				fields = make([]arrow.Field, count)
				for i := range fields {
					fields[i].Type = arrow.PrimitiveTypes.Int64
				}
			} else {
				v := arrow.NewMetadata(make([]string, count), make([]string, count))
				md = &v
			}
			data := schemaOnlyFixture(t, arrow.NewSchema(fields, md))
			schema := indirectTable(footerTable(data), 6)
			slot := flatbuffers.VOffsetT(6)
			if kind == "metadata" {
				slot = 8
			}
			vector := schema.Vector(flatbuffers.UOffsetT(schema.Offset(slot)))
			var target flatbuffers.UOffsetT
			for i := 0; i < count; i++ {
				at := vector + flatbuffers.UOffsetT(i*4)
				if to := schema.Indirect(at); to > target {
					target = to
				}
			}
			for i := 0; i < count; i++ {
				at := vector + flatbuffers.UOffsetT(i*4)
				if target <= at {
					t.Fatal("fixture requires forward references")
				}
				binary.LittleEndian.PutUint32(schema.Bytes[at:], uint32(target-at))
			}
			_, err := arrowipc.InspectFrom(context.Background(), bytes.NewReader(data), int64(len(data)))
			if err == nil || !strings.Contains(err.Error(), "entry limit") {
				t.Fatalf("%s: expected expanded entry limit error, got %v", kind, err)
			}
		})
	}
}
