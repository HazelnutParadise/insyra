package insyra

import (
	"os"
	"strings"
	"testing"
)

// TestChangelogUnreleasedLayout keeps the layout that lets "## Unreleased" be
// promoted to a release note as-is: every entry sits inside it, and each
// package heading appears there once. Both rules have been broken twice. Two
// PRs that each opened a package heading in an empty Unreleased and merged the
// same day left the heading twice (and v0.3.1 shipped "### `nn`" twice), and
// entries cherry-picked between the lines landed above "## Unreleased", where
// the release note would have dropped them.
func TestChangelogUnreleasedLayout(t *testing.T) {
	for _, name := range []string{"CHANGELOG.md", "CHANGELOG_TW.md"} {
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		lines := strings.Split(string(data), "\n")
		unreleased := -1
		for i, line := range lines {
			if line == "## Unreleased" {
				unreleased = i
				break
			}
			if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "### ") {
				t.Errorf("%s:%d: %q sits above \"## Unreleased\"; move it to the end of its package section under Unreleased", name, i+1, truncate(line))
			}
		}
		if unreleased < 0 {
			t.Errorf("%s: no \"## Unreleased\" section", name)
			continue
		}
		seen := map[string]int{}
		for i := unreleased + 1; i < len(lines); i++ {
			line := lines[i]
			if strings.HasPrefix(line, "## ") {
				break
			}
			if !strings.HasPrefix(line, "### ") {
				continue
			}
			if first, ok := seen[line]; ok {
				t.Errorf("%s:%d: %q is already open at line %d; move these entries to the end of that section and delete this heading", name, i+1, line, first)
				continue
			}
			seen[line] = i + 1
		}
	}
}

func truncate(s string) string {
	if len(s) > 60 {
		return s[:60] + "…"
	}
	return s
}
