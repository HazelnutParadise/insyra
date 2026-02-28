package commands

func init() {
	_ = Register(&CommandHandler{
		Name:        "exit",
		Aliases:     []string{"quit"},
		Usage:       "exit",
		Description: "Exit REPL",
		Run: func(ctx *ExecContext, args []string) error {
			return nil
		},
	})
}
