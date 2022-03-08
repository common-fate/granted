package main

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/mattn/go-colorable"
)

func main() {
	green := color.New(color.FgGreen)
	fmt.Fprint(colorable.NewColorableStderr(), green.Sprint("hello"))
}
