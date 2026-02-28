package commands

import (
	"fmt"

	insyra "github.com/HazelnutParadise/insyra"
)

func init() {
	_ = Register(&CommandHandler{
		Name:        "version",
		Usage:       "version",
		Description: "Show insyra version",
		Run: func(ctx *ExecContext, args []string) error {
			_, _ = fmt.Fprintf(ctx.Output, "insyra v%s (%s)\n", insyra.Version, insyra.VersionName)
			return nil
		},
	})
}
