package registry

import (
	"context"
	"testing"

	"github.com/common-fate/granted/pkg/config"
	"github.com/common-fate/granted/pkg/granted/registry/gitregistry"
	"github.com/stretchr/testify/assert"
)

func TestRegistryWithRef(t *testing.T) {
	tests := []struct {
		name        string
		registryOpts gitregistry.Opts
		wantErr     bool
	}{
		{
			name: "registry with master branch ref",
			registryOpts: gitregistry.Opts{
				Name:     "test-registry",
				URL:      "https://github.com/octocat/Hello-World.git",
				Ref:      "master",
				Filename: "granted.yml",
			},
			wantErr: false,
		},
		{
			name: "registry with specific commit ref",
			registryOpts: gitregistry.Opts{
				Name:     "test-registry-commit",
				URL:      "https://github.com/octocat/Hello-World.git",
				Ref:      "7fd1a60",
				Filename: "granted.yml",
			},
			wantErr: false,
		},
		{
			name: "registry with invalid ref",
			registryOpts: gitregistry.Opts{
				Name:     "test-registry-invalid",
				URL:      "https://github.com/octocat/Hello-World.git",
				Ref:      "invalid-branch",
				Filename: "granted.yml",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test config
			cfg := &config.Config{
				ProfileRegistry: &config.ProfileRegistry{
					Registries: []config.Registry{},
				},
			}

			// Create the registry
			registry, err := gitregistry.New(tt.registryOpts)
			assert.NoError(t, err)

			// Try to get AWS profiles (this will trigger the clone/pull with ref)
			_, err = registry.AWSProfiles(context.Background(), false)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}