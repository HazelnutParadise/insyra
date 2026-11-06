package arrowipc

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"sort"

	"github.com/apache/arrow-go/v18/arrow/ipc"
	flatbuffers "github.com/google/flatbuffers/go"
)

const defaultMaxBatchBytes int64 = 256 << 20

type ipcBlock struct {
	offset     int64
	metadata   int
	body       int64
	dictionary bool
}
type pendingBody struct {
	block  ipcBlock
	record flatbuffers.Table
}

// FileReader reads each message's metadata, then its encoded body. Validate the
// body before returning it to Arrow, without additional reads or buffering.
// In particular, compressed buffers carry their decoded length in an int64
// prefix; Arrow allocates that length before it attempts decompression.
// https://github.com/apache/arrow/blob/main/format/Message.fbs
// arrow-go v18.8.0 arrow/ipc/{metadata.go,file_reader.go}
type bufferLimitedSource struct {
	source          io.ReaderAt
	size            int64
	limit           int64
	footerAt        int64
	blocks          map[int64]ipcBlock
	pending         *pendingBody
	dictionaryBytes int64
}

func (r *bufferLimitedSource) ReadAt(p []byte, off int64) (n int, err error) {
	defer func() {
		if v := recover(); v != nil {
			err = fmt.Errorf("arrowipc: invalid IPC message metadata: %v", v)
		}
		// io.ReadFull intentionally discards an error accompanying all
		// requested bytes. A rejected message must deliver no usable bytes.
		if err != nil {
			n = 0
		}
	}()
	n, err = r.source.ReadAt(p, off)
	if err != nil {
		return n, err
	}
	if off == r.size-10 && len(p) == 10 {
		length := int64(binary.LittleEndian.Uint32(p[:4]))
		if length <= 0 || length > maxFooterSize || length > r.size-18 {
			return n, fmt.Errorf("arrowipc: invalid IPC footer length %d", length)
		}
		r.footerAt = r.size - 10 - length
	} else if off == r.footerAt && r.footerAt > 0 {
		err = r.readBlocks(p)
	} else if block, ok := r.blocks[off]; ok {
		if len(p) != block.metadata {
			return n, fmt.Errorf("arrowipc: unexpected IPC metadata read length")
		}
		var record flatbuffers.Table
		record, err = recordMetadata(p, block)
		if err == nil {
			r.pending = &pendingBody{block: block, record: record}
			if block.body == 0 {
				err = r.checkBody(nil)
			}
		}
	} else if r.pending != nil && off == r.pending.block.offset+int64(r.pending.block.metadata) {
		err = r.checkBody(p)
	}
	return n, err
}

func (r *bufferLimitedSource) readBlocks(footer []byte) error {
	tab := flatbuffers.Table{Bytes: footer, Pos: flatbuffers.GetUOffsetT(footer)}
	if err := validateSchemaFooter(tab); err != nil {
		return err
	}
	var starts [2]uint64
	var counts [2]int
	total := 0
	for i, slot := range []flatbuffers.VOffsetT{8, 10} {
		start, n, err := boundedVector(tab, slot, 24)
		if err != nil {
			return err
		}
		starts[i], counts[i] = start, n
		total += n
	}
	blocks := make([]ipcBlock, 0, total)
	for vector, start := range starts {
		for i := 0; i < counts[vector]; i++ {
			b := footer[start+uint64(i)*24:]
			block := ipcBlock{offset: int64(binary.LittleEndian.Uint64(b[:8])), metadata: int(int32(binary.LittleEndian.Uint32(b[8:12]))), body: int64(binary.LittleEndian.Uint64(b[16:24])), dictionary: vector == 0}
			// Arrow refuses a block outside the file or over the size limits
			// it is given before reading it (validateFileBlock in arrow-go
			// v18.8.0 arrow/ipc/file_reader.go), but it accepts 4-byte
			// metadata, which this reader could not watch. Refuse every block
			// it cannot watch, so none reaches Arrow unchecked.
			if block.offset < 8 || block.metadata < 8 || block.body < 0 || block.offset > r.footerAt || int64(block.metadata) > r.footerAt-block.offset || block.body > r.footerAt-block.offset-int64(block.metadata) {
				return fmt.Errorf("arrowipc: invalid IPC block extent")
			}
			// Arrow refuses the same, but its message does not name the option.
			if block.body > r.limit {
				return fmt.Errorf("arrowipc: an encoded IPC message body of %d bytes exceeds MaxBatchBytes %d", block.body, r.limit)
			}
			blocks = append(blocks, block)
		}
	}
	// A body offset must never also name another block's metadata. Otherwise
	// ReadAt's metadata dispatch could bypass the pending body's byte budget.
	sort.Slice(blocks, func(i, j int) bool { return blocks[i].offset < blocks[j].offset })
	var end int64
	for _, block := range blocks {
		if block.offset < end {
			return fmt.Errorf("arrowipc: overlapping IPC blocks at offset %d", block.offset)
		}
		// The extent checks above prove this sum cannot overflow.
		end = block.offset + int64(block.metadata) + block.body
	}
	r.blocks = make(map[int64]ipcBlock, len(blocks))
	for _, block := range blocks {
		r.blocks[block.offset] = block
	}
	return nil
}

func recordMetadata(data []byte, block ipcBlock) (flatbuffers.Table, error) {
	prefix := 4
	if len(data) < 8 {
		return flatbuffers.Table{}, fmt.Errorf("arrowipc: IPC metadata too short")
	}
	if binary.LittleEndian.Uint32(data[:4]) == 0xffffffff {
		prefix = 8
	}
	if len(data) < prefix+4 || int64(binary.LittleEndian.Uint32(data[prefix-4:prefix])) != int64(len(data)-prefix) {
		return flatbuffers.Table{}, fmt.Errorf("arrowipc: invalid IPC metadata length prefix")
	}
	meta := data[prefix:]
	message := flatbuffers.Table{Bytes: meta, Pos: flatbuffers.GetUOffsetT(meta)}
	headerSlot := message.Offset(6)
	if headerSlot == 0 {
		return flatbuffers.Table{}, fmt.Errorf("arrowipc: missing IPC message header type")
	}
	kind := message.GetByte(message.Pos + flatbuffers.UOffsetT(headerSlot))
	expected := byte(3) // MessageHeader.RecordBatch
	if block.dictionary {
		expected = 2
	} // MessageHeader.DictionaryBatch
	if kind != expected {
		return flatbuffers.Table{}, fmt.Errorf("arrowipc: unexpected IPC message type %v", ipc.MessageType(kind))
	}
	bodyLen := int64(0)
	if o := message.Offset(10); o != 0 {
		bodyLen = message.GetInt64(message.Pos + flatbuffers.UOffsetT(o))
	}
	if bodyLen != block.body {
		return flatbuffers.Table{}, fmt.Errorf("arrowipc: IPC body length differs from footer")
	}
	stringsBudget := int64(maxFooterSize)
	if err := validateKeyValueStrings(message, 12, &stringsBudget); err != nil {
		return flatbuffers.Table{}, err
	}
	record, err := offsetTable(message, 8)
	if err != nil {
		return flatbuffers.Table{}, err
	}
	if block.dictionary {
		// Arrow concatenates delta dictionaries before per-table output checks.
		// Refuse this encoding before its body can reach the native decoder.
		if o := record.Offset(8); o != 0 && record.GetBool(record.Pos+flatbuffers.UOffsetT(o)) {
			return flatbuffers.Table{}, fmt.Errorf("arrowipc: dictionary delta messages are unsupported")
		}
		record, err = offsetTable(record, 6)
		if err != nil {
			return flatbuffers.Table{}, err
		}
	}
	if _, _, err := boundedVector(record, 6, 16); err != nil {
		return flatbuffers.Table{}, err
	} // FieldNode
	_, buffers, err := boundedVector(record, 8, 16)
	if err != nil {
		return flatbuffers.Table{}, err
	}
	start, counts, err := boundedVector(record, 12, 8)
	if err != nil {
		return flatbuffers.Table{}, err
	}
	if counts > maxMetadataEntries {
		return flatbuffers.Table{}, fmt.Errorf("arrowipc: variadic buffer count vector exceeds entry limit")
	}
	var total int64
	for i := 0; i < counts; i++ {
		count := int64(binary.LittleEndian.Uint64(record.Bytes[start+uint64(i)*8:]))
		if count < 0 || count > int64(buffers)-total || count > int64(maxMetadataEntries)-total {
			return flatbuffers.Table{}, fmt.Errorf("arrowipc: invalid or excessive variadic buffer count %d", count)
		}
		total += count
	}
	return record, nil
}

func (r *bufferLimitedSource) checkBody(body []byte) error {
	pending := r.pending
	r.pending = nil
	if int64(len(body)) != pending.block.body {
		return fmt.Errorf("arrowipc: IPC body read length differs from footer")
	}
	record := pending.record
	compressed := record.Offset(10) != 0
	if compressed {
		if _, err := offsetTable(record, 10); err != nil {
			return err
		}
	}
	start, n, err := boundedVector(record, 8, 16)
	if err != nil {
		return err
	}
	available := r.limit - r.dictionaryBytes
	var total int64
	for i := 0; i < n; i++ {
		b := record.Bytes[start+uint64(i)*16:]
		offset, length := int64(binary.LittleEndian.Uint64(b[:8])), int64(binary.LittleEndian.Uint64(b[8:16]))
		if offset < 0 || length < 0 || offset > int64(len(body)) || length > int64(len(body))-offset {
			return fmt.Errorf("arrowipc: IPC buffer exceeds message body")
		}
		decoded := length
		if compressed && length != 0 {
			if length < 8 {
				return fmt.Errorf("arrowipc: compressed IPC buffer has no length prefix")
			}
			decoded = int64(binary.LittleEndian.Uint64(body[offset : offset+8]))
			if decoded == -1 {
				decoded = length - 8
			} else if decoded < 0 {
				return fmt.Errorf("arrowipc: invalid decoded IPC buffer length")
			}
		}
		if decoded > available-total {
			return fmt.Errorf("arrowipc: decoded batch and dictionary buffers exceed MaxBatchBytes %d", r.limit)
		}
		total += decoded
	}
	if pending.block.dictionary {
		r.dictionaryBytes += total
	}
	return nil
}

func limitedSource(ctx context.Context, source io.ReaderAt, size, limit int64) *bufferLimitedSource {
	return &bufferLimitedSource{source: contextReaderAt{ctx: ctx, source: source}, size: size, limit: limit, footerAt: -1}
}
