package commands

import (
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// CLI-15 of #324: a bad numeric argument returned strconv's own text —
// `strconv.ParseFloat: parsing "abc": invalid syntax` — which names neither the
// command nor the argument the caller got wrong.
func TestNumericArgumentErrorsNameTheCommandAndField(t *testing.T) {
	tests := []struct {
		name  string
		cmd   string
		args  []string
		field string
	}{
		{name: "ttest mu", cmd: "ttest", args: []string{"single", "x", "abc"}, field: "mu"},
		{name: "ztest mu", cmd: "ztest", args: []string{"single", "x", "abc", "1"}, field: "mu"},
		{name: "ztest sigma", cmd: "ztest", args: []string{"single", "x", "1", "abc"}, field: "sigma"},
		{name: "movavg window", cmd: "movavg", args: []string{"x", "abc"}, field: "window"},
		{name: "expsmooth alpha", cmd: "expsmooth", args: []string{"x", "abc"}, field: "alpha"},
		{name: "quartile q", cmd: "quartile", args: []string{"x", "abc"}, field: "q"},
		{name: "percentile p", cmd: "percentile", args: []string{"x", "abc"}, field: "p"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := newTestExecContext(t)
			ctx.Vars["x"] = insyra.NewDataList(1.0, 2.0, 3.0, 4.0)

			err := Dispatch(ctx, tt.cmd, tt.args)
			if err == nil {
				t.Fatal("a non-numeric argument gave no error")
			}
			msg := err.Error()
			if strings.Contains(msg, "strconv.") {
				t.Errorf("the message leaks strconv's text: %q", msg)
			}
			if !strings.Contains(msg, tt.cmd) {
				t.Errorf("the message %q does not name the command", msg)
			}
			if !strings.Contains(msg, tt.field) {
				t.Errorf("the message %q does not name the field %q", msg, tt.field)
			}
		})
	}
}

// CLI-16: option keys are matched without regard to case, and an unknown one
// says what is supported.
func TestKMeansAndKNNOptionParsing(t *testing.T) {
	t.Run("kmeans keys are case-insensitive", func(t *testing.T) {
		opts, err := parseKMeansOptions([]string{"NSTART", "3", "IterMax", "10"})
		if err != nil {
			t.Fatalf("uppercase option keys were refused: %v", err)
		}
		if opts.NStart != 3 || opts.IterMax != 10 {
			t.Errorf("got NStart=%d IterMax=%d, want 3 and 10", opts.NStart, opts.IterMax)
		}
	})

	t.Run("kmeans unknown option lists what is supported", func(t *testing.T) {
		_, err := parseKMeansOptions([]string{"nope", "1"})
		if err == nil {
			t.Fatal("an unknown option was accepted")
		}
		for _, want := range []string{"kmeans", "nstart", "itermax", "seed"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("the message %q does not mention %q", err, want)
			}
		}
	})

	t.Run("knn keys are case-insensitive", func(t *testing.T) {
		opts, err := parseKNNCommandOptions([]string{"WEIGHTING", "Distance", "Algorithm", "KD_TREE"})
		if err != nil {
			t.Fatalf("uppercase option keys were refused: %v", err)
		}
		if opts.Weighting != "distance" || opts.Algorithm != "kd_tree" {
			t.Errorf("got weighting=%q algorithm=%q", opts.Weighting, opts.Algorithm)
		}
	})

	// A misspelled value used to be converted straight into the enum and handed
	// to the library.
	t.Run("knn rejects an unknown value", func(t *testing.T) {
		if _, err := parseKNNCommandOptions([]string{"weighting", "inverse"}); err == nil {
			t.Error("an unknown weighting was accepted")
		}
		if _, err := parseKNNCommandOptions([]string{"algorithm", "quadtree"}); err == nil {
			t.Error("an unknown algorithm was accepted")
		}
	})
}

// CLI-18 of #326: the help table used a fixed 12-character column, so
// knn_neighbors (13) pushed its description out of line.
func TestHelpTableIsAligned(t *testing.T) {
	ctx := newTestExecContext(t)
	if err := Dispatch(ctx, "help", nil); err != nil {
		t.Fatalf("help: %v", err)
	}

	var descriptionColumn = -1
	for _, line := range strings.Split(ctx.Output.(interface{ String() string }).String(), "\n") {
		if !strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "available") || strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		at := strings.Index(line, fields[1])
		if descriptionColumn == -1 {
			descriptionColumn = at
			continue
		}
		if at != descriptionColumn {
			t.Fatalf("descriptions start at different columns (%d and %d):\n%s", descriptionColumn, at, line)
		}
	}
	if descriptionColumn == -1 {
		t.Fatal("help printed no commands")
	}
}

// read and env had no Forms, and env has nine subcommands.
func TestReadAndEnvHaveForms(t *testing.T) {
	for _, name := range []string{"read", "env"} {
		h, ok := LookupCommand(name)
		if !ok {
			t.Fatalf("%s is not registered", name)
		}
		if len(h.Forms) == 0 {
			t.Errorf("%s has no Forms", name)
		}
		if len(h.Examples) == 0 {
			t.Errorf("%s has no Examples", name)
		}
	}
}
