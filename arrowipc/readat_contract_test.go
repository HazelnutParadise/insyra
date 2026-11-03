package arrowipc_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/HazelnutParadise/insyra/arrowipc"
)

type eofAtEnd struct{ *bytes.Reader }

func (r eofAtEnd) ReadAt(p []byte, off int64) (int, error) {
	n, err := r.Reader.ReadAt(p, off)
	if off+int64(n) == r.Size() {
		return n, io.EOF
	}
	return n, err
}
func TestReaderAtMayReturnEOFWithFullRead(t *testing.T) {
	data := scalarFixture(t)
	src := eofAtEnd{bytes.NewReader(data)}
	if _, err := arrowipc.InspectFrom(context.Background(), src, int64(len(data))); err != nil {
		t.Fatal(err)
	}
	tables, errs := streamFrom(context.Background(), src, int64(len(data)), arrowipc.ReadOptions{}, 2)
	got, err := drain(t, tables, errs)
	if err != nil || len(got) == 0 {
		t.Fatalf("full read + EOF: %d tables, %v", len(got), err)
	}
}

type shortReadNoError struct{ *bytes.Reader }

func (r shortReadNoError) ReadAt(p []byte, off int64) (int, error) {
	return r.Reader.ReadAt(p[:len(p)-1], off)
}
func TestShortReadCannotFabricateMetadata(t *testing.T) {
	data := scalarFixture(t)
	_, err := arrowipc.InspectFrom(context.Background(), shortReadNoError{bytes.NewReader(data)}, int64(len(data)))
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("short read: %v", err)
	}
}

type checksumEOFSource struct{ *bytes.Reader }

var errChecksum = errors.New("checksum failed")

func (r checksumEOFSource) ReadAt(p []byte, off int64) (int, error) {
	n, err := r.Reader.ReadAt(p, off)
	if off+int64(n) == r.Size() {
		return n, errors.Join(io.EOF, errChecksum)
	}
	return n, err
}
func TestFullReadDoesNotHideJoinedSourceError(t *testing.T) {
	data := scalarFixture(t)
	src := checksumEOFSource{bytes.NewReader(data)}
	_, err := arrowipc.InspectFrom(context.Background(), src, int64(len(data)))
	if !errors.Is(err, errChecksum) {
		t.Fatalf("source error was lost: %v", err)
	}
	tables, errs := streamFrom(context.Background(), src, int64(len(data)), arrowipc.ReadOptions{}, 1)
	got, err := drain(t, tables, errs)
	if len(got) != 0 || !errors.Is(err, errChecksum) {
		t.Fatalf("source error lost in stream: %d, %v", len(got), err)
	}
}
