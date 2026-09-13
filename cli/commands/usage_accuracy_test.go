package commands

import (
	"strings"
	"testing"
)

// help prints a command's Usage, so a Usage that leaves out an option the
// command parses hides it, and one that marks a required argument optional
// sends people straight into an error. pca and regression stored their result
// under `as <var>` without saying so, and count called its required value
// optional.
func TestUsageNamesWhatTheCommandParses(t *testing.T) {
	registryMu.RLock()
	defer registryMu.RUnlock()

	for _, name := range []string{"pca", "regression"} {
		if u := Registry[name].Usage; !strings.HasSuffix(u, "[as <var>]") {
			t.Errorf("%s parses `as <var>` but its Usage is %q", name, u)
		}
	}
	if u := Registry["count"].Usage; u != "count <var> <value>" {
		t.Errorf("count requires a value, but its Usage is %q", u)
	}

	ctx := newTestExecContext(t)
	if err := runCountCommand(ctx, nil); err == nil || strings.Contains(err.Error(), "[value]") {
		t.Errorf("count with no arguments: got %v, want a usage error that does not call the value optional", err)
	}
	// save … sql accepts `rownames true|false` as well as the bare flag; its
	// usage error still showed only the bare flag.
	if err := runSaveSQL(ctx, "t", nil, nil); err == nil || !strings.Contains(err.Error(), "[rownames [true|false]]") {
		t.Errorf("save sql with no arguments: got %v, want the usage to show [rownames [true|false]]", err)
	}
}
