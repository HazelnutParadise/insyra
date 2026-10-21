package lpgen

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// ParseLingo and ParseLingoFile are the two names that return an error, and the
// only difference between them is where the text comes from. Reading the same
// model each way must produce the same model, and the same model the deprecated
// name still produces.
func TestParseLingoAndParseLingoFileReadTheSameModel(t *testing.T) {
	path := filepath.Join(t.TempDir(), "model.lng")
	if err := os.WriteFile(path, []byte(lingoModel), 0o600); err != nil {
		t.Fatalf("writing the fixture: %v", err)
	}

	fromString, err := ParseLingo(lingoModel)
	if err != nil {
		t.Fatalf("ParseLingo: %v", err)
	}
	if fromString == nil {
		t.Fatal("ParseLingo returned a nil model with a nil error")
		return
	}

	fromFile, err := ParseLingoFile(path)
	if err != nil {
		t.Fatalf("ParseLingoFile: %v", err)
	}
	if fromFile == nil {
		t.Fatal("ParseLingoFile returned a nil model with a nil error")
	}

	if !reflect.DeepEqual(fromString, fromFile) {
		t.Errorf("the text and file readers disagree:\n text: %+v\n file: %+v", fromString, fromFile)
	}
	if fromString.ObjectiveType != "Maximize" {
		t.Errorf("objective type: got %q, want \"Maximize\"", fromString.ObjectiveType)
	}
}

// A missing file used to give a nil model and a log line. The error keeps
// fs.ErrNotExist, so a caller can still tell it apart from any other failure.
func TestParseLingoFileReportsAMissingFile(t *testing.T) {
	quiet(t)

	m, err := ParseLingoFile(filepath.Join(t.TempDir(), "nope.lng"))
	if m != nil {
		t.Errorf("a missing file gave a model: %+v", m)
	}
	if err == nil {
		t.Fatal("a missing file gave a nil error")
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("errors.Is(err, fs.ErrNotExist) is false for %v", err)
	}
}

// bufio.Scanner's default 64 KiB line limit is the one thing the text reader
// cannot get past. A model with a longer line is an error now, where the old
// name returned nil.
func TestParseLingoReportsALineItCannotHold(t *testing.T) {
	text := "MODEL:\nMIN= " + strings.Repeat("X", 70000) + ";\nEND"

	m, err := ParseLingo(text)
	if m != nil {
		t.Errorf("a line the reader cannot hold gave a model: %+v", m)
	}
	if err == nil {
		t.Fatal("a line the reader cannot hold gave a nil error")
	}
}
