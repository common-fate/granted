package cfgcp

import (
	"context"
)

// implements same assumer style interface as the AWS implementation
type GCPAssumer interface {
	// AssumeTerminal should follow the required process for it implemetation and return credentials in byes format to be handled individually
	AssumeTerminal(context.Context, *ServiceAccount) ([]byte, error)
	// AssumeConsole should follow any console specific credentials processes, this may be the same as AssumeTerminal under the hood
	AssumeConsole(context.Context, *ServiceAccount) ([]byte, error)
	// A unique key which identifies this assumer
	Type() string
}

// List of assumers should be ordered by how they match type
var assumers []GCPAssumer = []GCPAssumer{&GCPServiceAccountAssumer{}}

// RegisterAssumer allows assumers to be registered when using this library as a package in other projects
// position = -1 will append the assumer
// position to insert assumer
func RegisterAssumer(a GCPAssumer, position int) {
	if position < 0 || position > len(assumers)-1 {
		assumers = append(assumers, a)
	} else {
		newAssumers := append([]GCPAssumer{}, assumers[:position]...)
		newAssumers = append(newAssumers, a)
		assumers = append(newAssumers, assumers[position:]...)
	}
}

func AssumerFromType(t string) GCPAssumer {
	for _, a := range assumers {
		if a.Type() == t {
			return a
		}
	}
	return nil
}
