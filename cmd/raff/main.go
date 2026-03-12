package main

import (
	"os"

	"github.com/rafftechnologies/raff-cli/internal/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		os.Exit(1)
	}
}
