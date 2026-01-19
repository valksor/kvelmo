package main

import (
	"fmt"
	"os"

	"github.com/valksor/go-mehrhof/cmd/mehr/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		fmt.Fprint(os.Stderr, commands.FormatError(err))
		os.Exit(1)
	}
}
