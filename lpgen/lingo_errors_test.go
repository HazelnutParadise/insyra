package lpgen

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
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
	if fromDeprecated := ParseLingoModel_str(lingoModel); !reflect.DeepEqual(fromString, fromDeprecated) {
		t.Errorf("ParseLingo and the deprecated name disagree:\n new:  %+v\n old: %+v", fromString, fromDeprecated)
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

	// The deprecated name keeps its old meaning: nil plus a warning.
	quiet(t)
	if got := ParseLingoModel_str(text); got != nil {
		t.Errorf("the deprecated name gave %+v, want nil", got)
	}
}

// A Deprecated name has to say what replaced it, and the sentence has to be in
// the doc comment rather than only in the changelog, so it is read from the
// source.
func TestDeprecatedLingoNamesSayWhatReplacedThem(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "lingo.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("parsing lingo.go: %v", err)
	}

	docs := make(map[string]string)
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Doc == nil {
			continue
		}
		switch fn.Name.Name {
		case "ParseLingoModel_str", "ParseLingoModel_txt":
			// Doc.Text() keeps the comment's own line breaks, so collapse the
			// wrapped paragraph before looking for a sentence in it.
			docs[fn.Name.Name] = strings.Join(strings.Fields(fn.Doc.Text()), " ")
		}
	}

	want := map[string][]string{
		"ParseLingoModel_str": {
			"Deprecated: use ParseLingo, which returns the failure as an error.",
			"Removed in the release after the one that deprecated it.",
		},
		"ParseLingoModel_txt": {
			"Deprecated: use ParseLingoFile, which returns the failure as an error.",
			"Removed in the release after the one that deprecated it.",
		},
	}
	for name, sentences := range want {
		doc, ok := docs[name]
		if !ok {
			t.Errorf("%s has no doc comment at all", name)
			continue
		}
		for _, sentence := range sentences {
			if !strings.Contains(doc, sentence) {
				t.Errorf("%s does not say %q:\n%s", name, sentence, doc)
			}
		}
	}
}
