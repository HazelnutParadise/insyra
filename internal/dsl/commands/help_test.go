package commands

import (
	"bytes"
	"strings"
	"testing"
)

func TestHelp_SimpleCommand_NoFormsOrExamples(t *testing.T) {
	ctx := newTestExecContext(t)
	if err := runHelpCommand(ctx, []string{"mean"}); err != nil {
		t.Fatalf("help mean failed: %v", err)
	}
	got := ctx.Output.(*bytes.Buffer).String()
	if !strings.Contains(got, "DataList mean") {
		t.Errorf("expected description, got %q", got)
	}
	if !strings.Contains(got, "usage: mean <var>") {
		t.Errorf("expected usage line, got %q", got)
	}
	if strings.Contains(got, "Forms:") {
		t.Errorf("expected no Forms section for `mean`, got %q", got)
	}
	if strings.Contains(got, "Examples:") {
		t.Errorf("expected no Examples section for `mean`, got %q", got)
	}
}

func TestHelp_ComplexCommand_RendersFormsAndExamples(t *testing.T) {
	ctx := newTestExecContext(t)
	if err := runHelpCommand(ctx, []string{"ttest"}); err != nil {
		t.Fatalf("help ttest failed: %v", err)
	}
	got := ctx.Output.(*bytes.Buffer).String()
	for _, want := range []string{
		"T-test commands",
		"usage: ttest single|two|paired ...",
		"Forms:",
		"ttest single <var> <mu>",
		"ttest two <var1> <var2> [equal|unequal]",
		"ttest paired <var1> <var2>",
		"Examples:",
		"insyra ttest single weights 70",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in help output, got:\n%s", want, got)
		}
	}
}

func TestHelp_UnknownCommand_Errors(t *testing.T) {
	ctx := newTestExecContext(t)
	err := runHelpCommand(ctx, []string{"definitely-not-a-command"})
	if err == nil || !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("expected unknown-command error, got %v", err)
	}
}

func TestHelp_NoArgs_ListsAllCommands(t *testing.T) {
	ctx := newTestExecContext(t)
	if err := runHelpCommand(ctx, nil); err != nil {
		t.Fatalf("help (no args) failed: %v", err)
	}
	got := ctx.Output.(*bytes.Buffer).String()
	if !strings.HasPrefix(got, "available commands:") {
		t.Errorf("expected listing header, got %q", got)
	}
	for _, name := range []string{"mean", "ttest", "load", "save", "groupby", "db"} {
		if !strings.Contains(got, name) {
			t.Errorf("expected %q in command listing, got:\n%s", name, got)
		}
	}
}
