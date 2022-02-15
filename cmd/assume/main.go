package main

import (
	"fmt"
	"os"

	"github.com/common-fate/granted/pkg/assume"
)

func main() {
	app := assume.GetCliApp()
	err := app.Run(os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
