package style

import (
	"fmt"

	insyra "github.com/HazelnutParadise/insyra"
)

const (
	ansiReset  = "\x1b[0m"
	ansiRed    = "\x1b[31m"
	ansiYellow = "\x1b[33m"
)

func ErrorText(message string) string {
	return withColor("error: "+message, ansiRed)
}

func WarningText(message string) string {
	return withColor("warn: "+message, ansiYellow)
}

func withColor(text, color string) string {
	if !insyra.Config.GetDoesUseColoredOutput() {
		return text
	}
	return fmt.Sprintf("%s%s%s", color, text, ansiReset)
}
