package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/valksor/go-mehrhof/cmd/mehr/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		// If error contains newlines, it's likely pre-formatted with suggestions
		errMsg := err.Error()
		if strings.Contains(errMsg, "\n") {
			fmt.Fprintf(os.Stderr, "%s\n", errMsg)
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}
}
