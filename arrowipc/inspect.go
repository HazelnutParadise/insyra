package arrowipc

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/apache/arrow-go/v18/arrow/ipc"
	flatbuffers "github.com/google/flatbuffers/go"
)

const maxFooterSize = 64 << 20

func inspectFile(ctx context.Context, path string) (info FileInfo, err error) {
	if err = checkContext(ctx); err != nil {
		return FileInfo{}, err
	}
	f, err := os.Open(path)
	if err != nil {
		return FileInfo{}, err
	}
	defer func() { err = errors.Join(err, f.Close()) }()
	stat, err := f.Stat()
	if err != nil {
		return FileInfo{}, err
	}
	return InspectFrom(ctx, f, stat.Size())
}

func inspectSource(ctx context.Context, r io.ReaderAt, size int64) (info FileInfo, err error) {
	// FlatBuffers and Arrow can panic on malformed offsets. The library boundary
	// converts those failures to read errors, as the Parquet reader does.
	defer func() {
		if v := recover(); v != nil {
			info = FileInfo{}
			err = fmt.Errorf("arrowipc: invalid file metadata: %v", v)
		}
	}()
	if err := checkSource(ctx, r, size); err != nil {
		return FileInfo{}, err
	}
	src := contextReaderAt{ctx: ctx, source: r}
	header := make([]byte, 6)
	if _, err := src.ReadAt(header, 0); err != nil {
		return FileInfo{}, err
	}
	if string(header) != "ARROW1" {
		return FileInfo{}, errors.New("arrowipc: not an Arrow IPC file (expected ARROW1 header)")
	}
	trailer := make([]byte, 10)
	if _, err := src.ReadAt(trailer, size-10); err != nil {
		return FileInfo{}, err
	}
	if string(trailer[4:]) != "ARROW1" {
		return FileInfo{}, errors.New("arrowipc: invalid or truncated Arrow IPC trailer")
	}
	length := int64(binary.LittleEndian.Uint32(trailer[:4]))
	if length <= 0 || length > size-18 || length > maxFooterSize {
		return FileInfo{}, fmt.Errorf("arrowipc: invalid footer length %d (file size %d, limit %d)", length, size, maxFooterSize)
	}
	footer := make([]byte, int(length))
	if _, err := src.ReadAt(footer, size-10-length); err != nil {
		return FileInfo{}, err
	}

	// Arrow v18 FileReader eagerly loads every dictionary body even for Schema.
	// Its schema decoder is reusable if the copied footer has no dictionary
	// blocks. Dictionary types themselves live in Schema and stay untouched.
	// Footer field 2 is dictionaries, field 3 is recordBatches (24-byte Block).
	// https://github.com/apache/arrow/blob/main/format/File.fbs
	tab := flatbuffers.Table{Bytes: footer, Pos: flatbuffers.GetUOffsetT(footer)}
	if err := validateSchemaFooter(tab); err != nil {
		return FileInfo{}, err
	}
	for _, slot := range []flatbuffers.VOffsetT{8, 10} {
		o := flatbuffers.UOffsetT(tab.Offset(slot))
		if o == 0 {
			continue
		}
		start, n := uint64(tab.Vector(o)), uint64(tab.VectorLen(o))
		if start < 4 || start > uint64(len(footer)) || n > (uint64(len(footer))-start)/24 {
			return FileInfo{}, errors.New("arrowipc: invalid footer block vector")
		}
		if slot == 8 {
			binary.LittleEndian.PutUint32(footer[start-4:start], 0)
		}
	}
	metadataFile := make([]byte, 8+len(footer)+10)
	copy(metadataFile, "ARROW1")
	copy(metadataFile[8:], footer)
	copy(metadataFile[8+len(footer):], trailer)
	reader, err := ipc.NewFileReader(bytes.NewReader(metadataFile))
	if err != nil {
		return FileInfo{}, fmt.Errorf("arrowipc: decode schema: %w", err)
	}
	defer func() { err = errors.Join(err, reader.Close()) }()
	if err := ctx.Err(); err != nil {
		return FileInfo{}, err
	}
	return FileInfo{Schema: reader.Schema(), NumBatches: reader.NumRecords(), Version: reader.Version().String()}, nil
}

func checkContext(ctx context.Context) error {
	if ctx == nil {
		return errors.New("arrowipc: nil context")
	}
	return ctx.Err()
}
func checkSource(ctx context.Context, r io.ReaderAt, size int64) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	if r == nil {
		return errors.New("arrowipc: nil ReaderAt")
	}
	if size < 18 {
		return fmt.Errorf("arrowipc: file too small: %d bytes", size)
	}
	return nil
}

type contextReaderAt struct {
	ctx    context.Context
	source io.ReaderAt
}

func (r contextReaderAt) ReadAt(p []byte, off int64) (n int, err error) {
	defer func() {
		if v := recover(); v != nil {
			n = 0
			err = fmt.Errorf("arrowipc: ReaderAt panic: %v", v)
		}
	}()
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	n, err = r.source.ReadAt(p, off)
	if ctxErr := r.ctx.Err(); ctxErr != nil {
		return n, ctxErr
	}
	if n != len(p) && err == nil {
		err = io.ErrUnexpectedEOF
	}
	if n == len(p) && err == io.EOF { //nolint:errorlint // Wrapped or joined EOF may carry another error that must survive.
		err = nil
	}
	return n, err
}
