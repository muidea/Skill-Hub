package main

import (
	"fmt"
	"os"

	"github.com/muidea/skill-hub/internal/cli"
	hubmodule "github.com/muidea/skill-hub/internal/modules/kernel/hub"
)

func main() {
	if err := hubmodule.New().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", cli.FormatError(err))
		os.Exit(1)
	}
}
