package main

import (
	"fmt"
	"os"

	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/fatih/color"
	"github.com/mgutz/ansi"
)

func main() {
	// withStdio := survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)
	// in := survey.Input{
	// 	Message: "Please select the profile you would like to assume:",
	// }
	// var p string
	// testable.AskOne(&in, &p, withStdio)
	// // a := "\x1b[1;92m? hello world \x1b[0m"
	// c := ansi.ColorCode("green+fb")

	// fmt.Printf("c: %v\n", c)
	// green := color.New(color.FgHiGreen)
	// // c := color.New(color.Attribute(1), color.Attribute(92))
	// 	color.Bold
	// format := []string{}
	// for i := 0; i < 255; i++ {
	// 	format = append(format, strconv.Itoa(i))
	// }
	// fmt.Printf("format: %v\n", format)
	// s := strings.Join(format, ";")
	// need to work out where the m comes from, can I create the correct attribute? to match the working green string
	// c.Fprintln(colorable.NewColorableStdout(), "mhello\x1b[0m")
	// green.Fprintf(os.Stdout, "hello")
	// a := green.Sprintln("hello")
	// _ = a
	// // working
	color.NoColor = false
	fmt.Fprintln(terminal.NewAnsiStdout(os.Stdout), ansi.ColorFunc("green")("hello"))
	fmt.Fprintln(terminal.NewAnsiStdout(os.Stdout), color.GreenString("hello"))
}
