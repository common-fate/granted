package assume_test

import (
	"context"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/bigkevmcd/go-configparser"
	"github.com/common-fate/granted/pkg/assume"
	"github.com/common-fate/granted/pkg/cfaws"
	"github.com/common-fate/granted/pkg/testable"
)

type mockAssumer struct {
	credentials aws.Credentials
}

func (ma *mockAssumer) AssumeTerminal(ctx context.Context, c *cfaws.CFSharedConfig) (aws.Credentials, error) {
	return ma.credentials, nil
}
func (ma *mockAssumer) AssumeConsole(ctx context.Context, c *cfaws.CFSharedConfig) (aws.Credentials, error) {
	return ma.AssumeTerminal(ctx, c)
}
func (ma *mockAssumer) Type() string                                                   { return "MOCK_ASSUMER" }
func (ma *mockAssumer) ProfileMatchesType(configparser.Dict, config.SharedConfig) bool { return true }

// Some notes on this, we will probably want to make the config file path configurable for testing, and also as a side feature to allow
// users to specify a custom path for their aws profiles
// AWS sdk supports this by allowing a list of config file paths to be provided

func Test_OssGranted(t *testing.T) {
	app := assume.GetCliApp()
	inputStrings := testable.SurveyInputs{"cf-dev"}

	position := 0
	testable.BeginTesting()
	testable.WithNextSurveyInputFunc(testable.NextFuncFromSlice(t, inputStrings, &position))
	os.Setenv("FORCE_NO_ALIAS", "true")
	os.Setenv("GRANTED_DISABLE_UPDATE_CHECK", "true")
	os.Args = []string{"assume"}

	// register mock assume which takes precedence over the existing ones
	// all profiles will register as this
	cfaws.RegisterAssumer(&mockAssumer{credentials: aws.Credentials{AccessKeyID: "1234", SecretAccessKey: "abcd", SessionToken: "efgh"}}, 0)
	err := app.Run(os.Args)
	if err != nil {
		t.Fatal(err)
	}
}
