package testable

import (
	"testing"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/core"
)

var isTesting = false
var nextSurveyInput func() StringOrBool = func() StringOrBool { panic("not implemented") }
var validateOutput func(key string, value string) = func(key, value string) {}

// use this type for survey inputs
type StringOrBool interface{}

// use this type for survey inputs
type SurveyInputs []StringOrBool

// configures Testable functions to utilise testing hooks
func BeginTesting() {
	isTesting = true
}

// configures Testable functions to stop utilising testing hooks
func EndTesting() {
	isTesting = false
}

// Configure this with a function that returns the next input required for a cli test
func WithNextSurveyInputFunc(next func() StringOrBool) {
	nextSurveyInput = next
}

// Configure this with a function that validates a keyvalue pair
func WithValidateOutputFunc(fn func(key string, value string)) {
	validateOutput = fn
}

// A helper which produces a next function that will call t.Fatal if all the inputs are exhausted
// position is an int representing the index in input for the next survey input
func NextFuncFromSlice(t *testing.T, inputs SurveyInputs, position *int) func() StringOrBool {
	return func() StringOrBool {
		if *position > len(inputs) {
			t.Fatal("attempted to call nextSurveyInput when no inputs remain")
		}
		v := inputs[*position]
		i := *position + 1
		position = &i
		return v
	}
}

// AskOne is a function which can be used to intercept surveys in the cli and replace the survey with input from a test input stream
// NextSurveyInput should be set to a function which returns the next string to satisfy the input
func AskOne(in survey.Prompt, out interface{}, opts ...survey.AskOpt) error {
	if isTesting {
		return core.WriteAnswer(out, "", nextSurveyInput())
	}
	return survey.AskOne(in, out, opts...)
}

// use this hook to output key value pairs which can be validated in a test
func Output(key string, value string) {
	if isTesting {
		validateOutput(key, value)
	}
}

// use this hook to output key value pairs which can be validated in a test
// expects the sequence to be key values pairs
func Outputs(kvs ...string) {
	if isTesting {
		for i := 0; i < len(kvs); i += 2 {
			validateOutput(kvs[i], kvs[i+1])
		}

	}
}

// func Fprintf(w io.Writer, key string, format string, a ...interface{}) (n int, err error) {
// 	if isTesting {
// 		validateNextOutput(format, a...)
// 		return len([]byte(fmt.Sprintf(format, a...))), nil
// 	}
// 	n, err = fmt.Fprintf(w, format, a...)
// 	return
// }

// func Fprintln(w io.Writer, a ...interface{}) (n int, err error) {
// 	if isTesting {
// 		validateNextOutput(format, a...)
// 		return len([]byte(fmt.Sprintf(format, a...))), nil
// 	}
// 	n, err = fmt.Fprintln(w, a...)
// 	return
// }
// func Fprint(w io.Writer, a ...interface{}) (n int, err error) {
// 	n, err = fmt.Fprint(w, a...)
// 	return
// }

// func Print(format string, a ...interface{}) (n int, err error) {
// 	n, err = Fprintf(os.Stdout, format, a...)
// 	return
// }
// func Printf(format string, a ...interface{}) (n int, err error) {
// 	n, err = Fprintf(os.Stdout, format, a...)
// 	return
// }

// func Println(format string, a ...interface{}) (n int, err error) {
// 	n, err = Fprintln(os.Stdout, a...)
// 	return
// }
