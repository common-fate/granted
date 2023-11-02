package cfgcp

import (
	"context"
	"fmt"

	"google.golang.org/api/iam/v1"
)

type ServiceAccount struct {
	Name          string
	Type          string
	ProjectId     string
	PolicyVersion string
}

func LoadServiceAccounts(ctx context.Context, projectId string) ([]*iam.ServiceAccount, error) {
	service, err := iam.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("iam.NewService: %w", err)
	}

	response, err := service.Projects.ServiceAccounts.List("projects/" + projectId).Do()
	if err != nil {
		return nil, err
	}

	return response.Accounts, nil
}

func (sa *ServiceAccount) AssumeConsole(ctx context.Context) (GCPCredentials, error) {
	return AssumerFromType(sa.Type).AssumeConsole(ctx, sa)
}

func (sa *ServiceAccount) AssumeTerminal(ctx context.Context) (GCPCredentials, error) {
	return AssumerFromType(sa.Type).AssumeTerminal(ctx, sa)
}
