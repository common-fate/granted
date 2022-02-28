package cfaws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/bigkevmcd/go-configparser"
)

type Assumer interface {
	// AssumeTerminal should follow the required process for it implemetation and return aws credentials ready to be exported to the terminal environment
	AssumeTerminal(context.Context, *CFSharedConfig) (aws.Credentials, error)
	// AssumeConsole should follow any console specific credentials processes, this may be the same as AssumeTerminal under the hood
	AssumeConsole(context.Context, *CFSharedConfig) (aws.Credentials, error)
	// A unique key which identifies this assumer e.g AWS-SSO or GOOGLE-AWS-AUTH
	Type() string
	// ProfileMatchesType takes a list of strings which are the lines in an aw config profile and returns true if this profile is the assumers type
	ProfileMatchesType(configparser.Dict, config.SharedConfig) bool
}

// List of assumers should be ordered by how they match type
// specific types should be first, generic types like IAM should be last / the (default)
var assumers []Assumer = []Assumer{&AwsGoogleAuthAssumer{}, &Saml2AwsAssumer{}, &AwsSsoAssumer{}, &AwsIamAssumer{}}

// RegisterAssumer allows assumers to be registered when using this library as a package in other projects
// position = -1 will append the assumer
// position to insert assumer
func RegisterAssumer(a Assumer, position int) {
	if position < 0 || position > len(assumers)-1 {
		assumers = append(assumers, a)
	} else {
		newAssumers := append([]Assumer{}, assumers[:position]...)
		newAssumers = append(newAssumers, a)
		assumers = append(newAssumers, assumers[position:]...)
	}
}

func AssumerFromType(t string) Assumer {
	for _, a := range assumers {
		if a.Type() == t {
			return a
		}
	}
	return nil
}
