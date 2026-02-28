package commands

import (
	"encoding/json"
	"fmt"

	"github.com/HazelnutParadise/insyra/cli/env"
)

func init() {
	_ = Register(&CommandHandler{
		Name:        "config",
		Usage:       "config [key] [value]",
		Description: "Read or update global CLI config",
		Run:         runConfigCommand,
	})
}

func runConfigCommand(ctx *ExecContext, args []string) error {
	if len(args) == 0 {
		cfg, err := env.LoadGlobalConfig()
		if err != nil {
			return err
		}
		bytes, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(ctx.Output, "%s\n", string(bytes))
		return nil
	}

	if len(args) < 2 {
		return fmt.Errorf("usage: config [key] [value]")
	}

	cfg, err := env.UpdateGlobalConfig(args[0], args[1])
	if err != nil {
		return err
	}
	bytes, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(ctx.Output, "%s\n", string(bytes))
	return nil
}
