package commands

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// The CLI is documented in three hand-written places, and nothing compared
// them with the commands: accel was missing from all three, read had lost two
// options, plot still advertised options batch 7 removed. This compares every
// registered command's Usage with each document, so the next drift fails here
// instead of reaching someone reading the docs.
func TestCLIDocsMatchRegistry(t *testing.T) {
	docs := []struct {
		name   string
		path   string
		parse  func(string) map[string]string
		extras map[string]bool
	}{
		{"Docs/cli-dsl.md command index", "../../Docs/cli-dsl.md", parseCommandIndex,
			map[string]bool{"completion": true}}, // Cobra's own, not in the registry
		{"cli-command-usage.md", "../../skills/use-insyra-cli/references/cli-command-usage.md",
			sectionUsages("## "), nil},
		{"cli-command-guide.md", "../../skills/use-insyra-cli/references/cli-command-guide.md",
			sectionUsages("### "), map[string]bool{"load sql": true, "save sql": true}},
	}

	registryMu.RLock()
	defer registryMu.RUnlock()

	for _, d := range docs {
		raw, err := os.ReadFile(d.path)
		if err != nil {
			t.Fatalf("%s: %v", d.name, err)
		}
		documented := d.parse(string(raw))
		// Without this a broken parser would find nothing to compare and pass.
		if len(documented) < 100 {
			t.Fatalf("%s: parsed only %d entries; the parser no longer matches the document", d.name, len(documented))
		}
		for name, h := range Registry {
			usage, ok := documented[name]
			if !ok {
				t.Errorf("%s does not document %q", d.name, name)
				continue
			}
			if usage != h.Usage {
				t.Errorf("%s documents %q as\n\t%s\nbut its Usage is\n\t%s", d.name, name, usage, h.Usage)
			}
		}
		for name := range documented {
			if _, ok := Registry[name]; !ok && !d.extras[name] {
				t.Errorf("%s documents %q, which is not a command", d.name, name)
			}
		}
	}
}

// Two more pages list the commands by topic, without usage lines: the command
// groups in cli-dsl.md and the skill's cli-commands.md. A command missing from
// them is invisible to anyone looking for it by topic, which is how accel went
// unnoticed.
func TestCLITopicListsNameEveryCommand(t *testing.T) {
	pages := []struct {
		name, path, section string
	}{
		{"Docs/cli-dsl.md command groups", "../../Docs/cli-dsl.md", "## Command Groups"},
		{"cli-commands.md", "../../skills/use-insyra-cli/references/cli-commands.md", ""},
	}

	registryMu.RLock()
	defer registryMu.RUnlock()

	for _, p := range pages {
		raw, err := os.ReadFile(p.path)
		if err != nil {
			t.Fatalf("%s: %v", p.name, err)
		}
		doc := string(raw)
		if p.section != "" {
			start := strings.Index(doc, p.section)
			if start < 0 {
				t.Fatalf("%s: section %q not found", p.name, p.section)
			}
			doc = doc[start:]
			if end := strings.Index(doc[1:], "\n## "); end >= 0 {
				doc = doc[:end+1]
			}
		}
		// The first word of every backtick span on a list line: `db connect
		// <name> <dsn>` names db, `quant sharpe` names quant.
		named := map[string]bool{}
		for _, line := range strings.Split(doc, "\n") {
			if !strings.HasPrefix(strings.TrimLeft(line, " \t"), "- ") {
				continue
			}
			for _, span := range backtickSpan.FindAllStringSubmatch(line, -1) {
				if fields := strings.Fields(span[1]); len(fields) > 0 {
					named[fields[0]] = true
				}
			}
		}
		if len(named) < 100 {
			t.Fatalf("%s: found only %d command names; the parser no longer matches the page", p.name, len(named))
		}
		for name := range Registry {
			if !named[name] {
				t.Errorf("%s does not list %q", p.name, name)
			}
		}
	}
}

var backtickSpan = regexp.MustCompile("`([^`]*)`")

// usageOf returns the usage a document gives: the text of its one backtick
// span. Anything else — no span, or several — is returned as written so it
// cannot compare equal to a Usage string by accident.
func usageOf(s string) string {
	spans := backtickSpan.FindAllStringSubmatch(s, -1)
	if len(spans) != 1 {
		return strings.TrimSpace(s)
	}
	return spans[0][1]
}

// parseCommandIndex reads the "| `name` | `usage` | description |" rows of
// cli-dsl.md's command index. A "|" inside a cell is written "\|".
func parseCommandIndex(doc string) map[string]string {
	out := map[string]string{}
	start := strings.Index(doc, "## Full Command Index (Appendix)")
	if start < 0 {
		return out
	}
	section := doc[start:]
	if end := strings.Index(section[1:], "\n## "); end >= 0 {
		section = section[:end+1]
	}
	for _, line := range strings.Split(section, "\n") {
		if !strings.HasPrefix(line, "| `") {
			continue
		}
		cells := splitTableRow(line)
		if len(cells) < 2 {
			continue
		}
		name := strings.Trim(strings.TrimSpace(cells[0]), "`")
		out[name] = strings.ReplaceAll(usageOf(cells[1]), `\|`, "|")
	}
	return out
}

// splitTableRow splits a markdown table row on "|" that is not escaped.
func splitTableRow(line string) []string {
	var cells []string
	var cur strings.Builder
	for i := 0; i < len(line); i++ {
		if line[i] == '\\' && i+1 < len(line) && line[i+1] == '|' {
			cur.WriteString(`\|`)
			i++
			continue
		}
		if line[i] == '|' {
			cells = append(cells, cur.String())
			cur.Reset()
			continue
		}
		cur.WriteByte(line[i])
	}
	// The leading "|" produces an empty first cell; the trailing one leaves an
	// empty remainder. Keep only the cells between them.
	if len(cells) > 0 && strings.TrimSpace(cells[0]) == "" {
		cells = cells[1:]
	}
	return cells
}

// sectionUsages reads sections that start with "<hdr>`name`" (anything may
// follow the name, such as "(deprecated)") and returns each section's
// "- Usage:" line.
func sectionUsages(hdr string) func(string) map[string]string {
	head := regexp.MustCompile("(?m)^" + regexp.QuoteMeta(hdr) + "`([^`]+)`")
	return func(doc string) map[string]string {
		out := map[string]string{}
		idx := head.FindAllStringSubmatchIndex(doc, -1)
		for k, ix := range idx {
			name := doc[ix[2]:ix[3]]
			end := len(doc)
			if k+1 < len(idx) {
				end = idx[k+1][0]
			}
			usage := ""
			for _, line := range strings.Split(doc[ix[1]:end], "\n") {
				if strings.HasPrefix(line, "- Usage: ") {
					usage = usageOf(strings.TrimPrefix(line, "- Usage: "))
					break
				}
			}
			out[name] = usage
		}
		return out
	}
}
