package arrowipc_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"runtime"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra/arrowipc"
	flatbuffers "github.com/google/flatbuffers/go"
)

func TestAggregateFooterBlockEntryLimit(t *testing.T) {
	for _, shared := range []bool{false, true} {
		name := "records"
		count := 65537
		if shared {
			name = "shared-dictionary-records"
			count = 32769
		}
		t.Run(name, func(t *testing.T) {
			original := scalarFixture(t)
			oldFooter := footerTable(original)
			recordVector := oldFooter.Vector(flatbuffers.UOffsetT(oldFooter.Offset(10)))
			block := bytes.Clone(oldFooter.Bytes[recordVector : recordVector+24])
			bodyAt := int64(binary.LittleEndian.Uint64(block[:8])) + int64(binary.LittleEndian.Uint32(block[8:12]))
			bodyLen := int64(binary.LittleEndian.Uint64(block[16:24]))
			footer := bytes.Clone(oldFooter.Bytes)
			for len(footer)%8 != 4 {
				footer = append(footer, 0)
			}
			lengthAt := len(footer)
			footer = binary.LittleEndian.AppendUint32(footer, uint32(count))
			for i := 0; i < count; i++ {
				footer = append(footer, block...)
			}
			slots := []flatbuffers.VOffsetT{10}
			if shared {
				slots = append(slots, 8)
			}
			for _, slot := range slots {
				at := int(oldFooter.Pos) + int(oldFooter.Offset(slot))
				if lengthAt <= at {
					t.Fatal("fixture requires forward vector")
				}
				binary.LittleEndian.PutUint32(footer[at:], uint32(lengthAt-at))
			}
			footerAt := len(original) - 10 - len(oldFooter.Bytes)
			data := append(bytes.Clone(original[:footerAt]), footer...)
			data = binary.LittleEndian.AppendUint32(data, uint32(len(footer)))
			data = append(data, "ARROW1"...)
			source := &watchedIPCBody{Reader: bytes.NewReader(data), start: bodyAt, end: bodyAt + bodyLen}
			runtime.GC()
			var before, after runtime.MemStats
			runtime.ReadMemStats(&before)
			tables, errs := arrowipc.StreamFrom(context.Background(), source, int64(len(data)), arrowipc.ReadOptions{}, 1)
			got, err := drain(t, tables, errs)
			runtime.ReadMemStats(&after)
			if allocated := after.TotalAlloc - before.TotalAlloc; allocated > uint64(8*len(footer)+(1<<20)) {
				t.Fatalf("%d-byte footer allocated %d bytes before rejection", len(footer), allocated)
			}
			if len(got) != 0 || err == nil || !strings.Contains(err.Error(), "block entry limit") || source.reads != 0 {
				t.Fatalf("%d tables, %d body reads, error %v", len(got), source.reads, err)
			}
		})
	}
}
