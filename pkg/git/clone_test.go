package git

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCloneWithRef(t *testing.T) {
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "git-clone-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	// Define test cases
	tests := []struct {
		name     string
		repoURL  string
		ref      string
		wantErr  bool
	}{
		{
			name:    "clone with main branch",
			repoURL: "https://github.com/octocat/Hello-World.git",
			ref:     "master",
			wantErr: false,
		},
		{
			name:    "clone with specific commit",
			repoURL: "https://github.com/octocat/Hello-World.git",
			ref:     "7fd1a60", // First 7 chars of a commit hash
			wantErr: false,
		},
		{
			name:    "clone with empty ref",
			repoURL: "https://github.com/octocat/Hello-World.git",
			ref:     "",
			wantErr: false,
		},
		{
			name:    "clone with non-existent ref",
			repoURL: "https://github.com/octocat/Hello-World.git",
			ref:     "non-existent-branch",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a unique directory for each test
			cloneDir := filepath.Join(tempDir, tt.name)
			
			err := CloneWithRef(tt.repoURL, cloneDir, tt.ref)
			if (err != nil) != tt.wantErr {
				t.Errorf("CloneWithRef() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// If we expected success and got it, verify the clone
			if !tt.wantErr && err == nil {
				// Check if the directory exists
				if _, err := os.Stat(cloneDir); os.IsNotExist(err) {
					t.Errorf("Clone directory does not exist: %s", cloneDir)
				}

				// Check if .git directory exists
				gitDir := filepath.Join(cloneDir, ".git")
				if _, err := os.Stat(gitDir); os.IsNotExist(err) {
					t.Errorf(".git directory does not exist in clone: %s", gitDir)
				}
			}
		})
	}
}