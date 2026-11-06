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

func TestAliasedTimestampTimezonesAreBudgetedPerOccurrence(t *testing.T) {
	const count = 1025
	fields := make([]arrow.Field, count)
	for i := range fields {
		zone := "x"
		if i == 0 {
			zone = strings.Repeat("x", 64<<10)
		}
		fields[i] = arrow.Field{Name: fmt.Sprintf("t%d", i), Type: &arrow.TimestampType{Unit: arrow.Second, TimeZone: zone}, Nullable: i%2 == 0}
	}
	data := schemaOnlyFixture(t, arrow.NewSchema(fields, nil))
	schema := indirectTable(footerTable(data), 6)
	vector := schema.Vector(flatbuffers.UOffsetT(schema.Offset(6)))
	var positions []flatbuffers.UOffsetT
	var shared flatbuffers.UOffsetT
	for i := range fields {
		field := flatbuffers.Table{Bytes: schema.Bytes, Pos: schema.Indirect(vector + flatbuffers.UOffsetT(i*4))}
		timestamp := indirectTable(field, 10)
		at := timestamp.Pos + flatbuffers.UOffsetT(timestamp.Offset(6))
		target := at + flatbuffers.GetUOffsetT(timestamp.Bytes[at:])
		if flatbuffers.GetUOffsetT(timestamp.Bytes[target:]) == 64<<10 {
			shared = target
		}
		positions = append(positions, at)
	}
	for _, at := range positions {
		if shared <= at {
			t.Fatal("fixture needs forward shared timezone")
		}
		binary.LittleEndian.PutUint32(schema.Bytes[at:], uint32(shared-at))
	}
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	_, err := arrowipc.InspectFrom(context.Background(), bytes.NewReader(data), int64(len(data)))
	runtime.ReadMemStats(&after)
	if err == nil || !strings.Contains(err.Error(), "metadata strings") {
		t.Fatalf("expected timezone budget error, got %v", err)
	}
	if allocated := after.TotalAlloc - before.TotalAlloc; allocated > 2<<20 {
		t.Fatalf("timezone rejection allocated %d bytes", allocated)
	}
}

func TestTimestampTimezonesSurviveInspection(t *testing.T) {
	fields := []arrow.Field{
		{Name: "a", Type: &arrow.TimestampType{Unit: arrow.Second, TimeZone: "UTC"}},
		{Name: "b", Type: &arrow.TimestampType{Unit: arrow.Nanosecond, TimeZone: "Asia/Taipei"}, Nullable: true},
	}
	data := schemaOnlyFixture(t, arrow.NewSchema(fields, nil))
	info, err := arrowipc.InspectFrom(context.Background(), bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	clear(data)
	for i, want := range []string{"UTC", "Asia/Taipei"} {
		if got := info.Columns[i].Type; !strings.Contains(got, "tz="+want) {
			t.Fatalf("timezone %d: %q", i, got)
		}
	}
}
