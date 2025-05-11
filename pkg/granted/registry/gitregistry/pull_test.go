package gitregistry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegistryWithRef(t *testing.T) {
	tests := []struct {
		name string
		opts Opts
	}{
		{
			name: "create registry without ref (backward compatibility)",
			opts: Opts{
				Name:     "test-registry",
				URL:      "https://github.com/octocat/Hello-World.git",
				Filename: "README",
				// Ref is not set, testing backward compatibility
			},
		},
		{
			name: "create registry with empty ref",
			opts: Opts{
				Name:     "test-registry",
				URL:      "https://github.com/octocat/Hello-World.git",
				Filename: "README",
				Ref:      "",
			},
		},
		{
			name: "create registry with ref",
			opts: Opts{
				Name:     "test-registry",
				URL:      "https://github.com/octocat/Hello-World.git",
				Filename: "README",
				Ref:      "master",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that we can create a registry with or without ref
			registry, err := New(tt.opts)
			assert.NoError(t, err)
			assert.NotNil(t, registry)
			assert.Equal(t, tt.opts.Ref, registry.opts.Ref)
		})
	}
}