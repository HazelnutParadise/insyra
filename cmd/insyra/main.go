package main

import (
	"fmt"
	"os"

	"github.com/HazelnutParadise/insyra/cli"
	"github.com/HazelnutParadise/insyra/cli/style"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, style.ErrorText(err.Error()))
		os.Exit(1)
	}
}
