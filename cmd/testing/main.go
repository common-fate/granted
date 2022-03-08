package main

import (
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/common-fate/granted/pkg/testable"
)

func main() {
	withStdio := survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)
	in := survey.Input{
		Message: "Please select the profile you would like to assume:",
	}
	var p string
	testable.AskOne(&in, &p, withStdio)
}
