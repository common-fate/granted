package main

import (
	"fmt"
	"os"

	"github.com/common-fate/granted/pkg/assume"
	"github.com/fatih/color"
)

func main() {
	app := assume.GetCliApp()

	err := app.Run(os.Args)
	if err != nil {
		fmt.Fprintf(color.Error, "%s\n", err)
		os.Exit(1)
	}
}
