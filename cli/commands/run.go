package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func init() {
	_ = Register(&CommandHandler{Name: "run", Usage: "run <script.isr>", Description: "Run DSL script file", Run: runScriptCommand})
}

func runScriptCommand(ctx *ExecContext, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: run <script.isr>")
	}
	file, err := os.Open(args[0])
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		tokens := splitScriptTokens(line)
		if len(tokens) == 0 {
			continue
		}
		if err := Dispatch(ctx, tokens[0], tokens[1:]); err != nil {
			_, _ = fmt.Fprintf(ctx.Output, "line %d: %v\n", lineNumber, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	_, _ = fmt.Fprintln(ctx.Output, "script complete")
	return nil
}

func splitScriptTokens(line string) []string {
	result := []string{}
	var builder strings.Builder
	quote := rune(0)
	escaped := false
	flush := func() {
		if builder.Len() == 0 {
			return
		}
		result = append(result, builder.String())
		builder.Reset()
	}

	for _, ch := range line {
		if escaped {
			builder.WriteRune(ch)
			escaped = false
			continue
		}
		if ch == '\\' {
			escaped = true
			continue
		}
		if quote != 0 {
			if ch == quote {
				quote = 0
				continue
			}
			builder.WriteRune(ch)
			continue
		}
		if ch == '"' || ch == '\'' {
			quote = ch
			continue
		}
		if ch == ' ' || ch == '\t' {
			flush()
			continue
		}
		builder.WriteRune(ch)
	}
	flush()
	return result
}
