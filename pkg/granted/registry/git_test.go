package registry

import (
	"fmt"
	"testing"
)

func TestParseGitURL(t *testing.T) {
	tests := []struct {
		name    string
		want    GitURL
		wantErr error
		url     string
	}{
		{
			name: "https with personal repo slash and not .git",
			url:  "https://github.com/personal/repo-name-with-slash/",
			want: GitURL{
				ProvidedURL: "https://github.com/personal/repo-name-with-slash/",
				Host:        "github.com",
				Org:         "personal",
				Repo:        "repo-name-with-slash",
			},
			wantErr: nil,
		},
		{
			name: "https bitbucket",
			url:  "https://owner@bitbucket.org/owner/name",
			want: GitURL{
				ProvidedURL: "https://owner@bitbucket.org/owner/name",
				Host:        "owner@bitbucket.org",
				Org:         "owner",
				Repo:        "name",
			},
			wantErr: nil,
		},
		{
			name:    "invalid git url",
			url:     "owner@github.com/abc/xyz",
			want:    GitURL{},
			wantErr: fmt.Errorf("unable to parse the provided git url '%s'", "owner@github.com/abc/xyz"),
		},
		{
			name: "ssh github with slash org name and number",
			url:  "ssh://git@bitbucket.company.com/owner-name-with-slash10001/1234repo.git",
			want: GitURL{
				ProvidedURL: "ssh://git@bitbucket.company.com/owner-name-with-slash10001/1234repo.git",
				Host:        "bitbucket.company.com",
				Org:         "owner-name-with-slash10001",
				Repo:        "1234repo",
			},
			wantErr: fmt.Errorf("unable to parse the provided git url '%s'", "owner@github.com/abc/xyz"),
		},
		{
			name: "ssh bitbucket",
			url:  "ssh://git@bitbucket.company.com/owner/name.git",
			want: GitURL{
				ProvidedURL: "ssh://git@bitbucket.company.com/owner/name.git",
				Host:        "bitbucket.company.com",
				Org:         "owner",
				Repo:        "name",
			},
			wantErr: nil,
		}, {
			name:    "https personal org; not supported",
			url:     "https://owner@organization.git.cloudforge.com/name.git",
			want:    GitURL{},
			wantErr: fmt.Errorf("unable to parse the provided git url '%s'", "https://owner@organization.git.cloudforge.com/name.git"),
		}, {
			name: "https github.com with .git",
			url:  "https://github.com/owner/name",
			want: GitURL{
				ProvidedURL: "https://github.com/owner/name",
				Host:        "github.com",
				Org:         "owner",
				Repo:        "name",
			},
			wantErr: nil,
		}, {
			name: "http github.com with not .git",
			url:  "http://github.com/Eddie023/granted-registry/",
			want: GitURL{
				ProvidedURL: "http://github.com/Eddie023/granted-registry/",
				Host:        "github.com",
				Org:         "Eddie023",
				Repo:        "granted-registry",
			},
			wantErr: nil,
		},
		{
			name: "http github.com with subpath",
			url:  "http://github.com/Eddie023/granted-registry.git/team_a",
			want: GitURL{
				ProvidedURL: "http://github.com/Eddie023/granted-registry.git/team_a",
				Host:        "github.com",
				Org:         "Eddie023",
				Repo:        "granted-registry",
				Subpath:     "team_a",
			},
			wantErr: nil,
		},
		{
			name: "http github.com with subpath and specific file",
			url:  "http://github.com/Eddie023/granted-registry.git/team_a/team_b/config.yml",
			want: GitURL{
				ProvidedURL: "http://github.com/Eddie023/granted-registry.git/team_a/team_b/config.yml",
				Host:        "github.com",
				Org:         "Eddie023",
				Repo:        "granted-registry",
				Subpath:     "team_a/team_b",
				Filename:    "config.yml",
			},
			wantErr: nil,
		},
		{
			name: "http github.com with subpath and last slash",
			url:  "http://github.com/Eddie023/granted-registry.git/team_a/team_b/",
			want: GitURL{
				ProvidedURL: "http://github.com/Eddie023/granted-registry.git/team_a/team_b/",
				Host:        "github.com",
				Org:         "Eddie023",
				Repo:        "granted-registry",
				Subpath:     "team_a/team_b/",
			},
			wantErr: nil,
		},
		{
			name: "http github.com with subpath and granted.yml file",
			url:  "http://github.com/Eddie023/granted-registry.git/team_a/team_b/granted.yml",
			want: GitURL{
				ProvidedURL: "http://github.com/Eddie023/granted-registry.git/team_a/team_b/granted.yml",
				Host:        "github.com",
				Org:         "Eddie023",
				Repo:        "granted-registry",
				Subpath:     "team_a/team_b",
				Filename:    "granted.yml",
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseGitURL(tt.url)
			if err != nil {

				if err.Error() == tt.wantErr.Error() {
					return
				}

				t.Error(err)
			}

			want := tt.want

			if want != got {
				t.Errorf("got %v, want %v", got, want)
			}
		})
	}

}

func TestIsSameURL(t *testing.T) {
	tests := []struct {
		name      string
		firstURL  string
		secondURL string
		want      bool
	}{
		{
			name:      "Same url",
			firstURL:  "https://github.com/octo/repo.git",
			secondURL: "https://github.com/octo/repo",
			want:      true,
		},
		{
			name:      "Different organization",
			firstURL:  "https://github.com/octo/repo.git",
			secondURL: "https://gitlab.com/octo/repo",
			want:      false,
		},
		{
			name:      "Same organization but one has subfolder",
			firstURL:  "https://github.com/octo/repo/team_a/team_b",
			secondURL: "https://gitlab.com/octo/repo.git",
			want:      false,
		},
		{
			name:      "Same organization but different protocol",
			firstURL:  "https://github.com/octo/repo-name.git",
			secondURL: "git@github.com:octo/repo-name.git",
			want:      true,
		},
		{
			name:      "Same organization but different protocol but one with subfolder",
			firstURL:  "https://github.com/octo/repo-name.git/team_a",
			secondURL: "git@github.com:octo/repo-name.git",
			want:      false,
		},
		{
			name:      "Same organization, same protocol but one with config.yml file",
			firstURL:  "https://github.com/octo/repo-name.git/team_a/config.yml",
			secondURL: "https://github.com/octo/repo-name.git/team_a",
			want:      false,
		},
		{
			name:      "Same organization, same protocol but different folder",
			firstURL:  "https://github.com/octo/repo-name.git/team_a/config.yml",
			secondURL: "https://github.com/octo/repo-name.git/team_b/config.yml",
			want:      false,
		},
		{
			name:      "Same organization, different protocol but same subfolder and filename",
			firstURL:  "https://github.com/octo/repo-name.git/team_a/config.yml",
			secondURL: "git@github.com:octo/repo-name.git/team_a/config.yml",
			want:      true,
		},
		{
			name:      "Same organization, different protocol but different subfolder",
			firstURL:  "https://github.com/octo/repo-name.git/team_a/config.yml",
			secondURL: "git@github.com:octo/repo-name.git/team_b/config.yml",
			want:      false,
		},
		{
			name:      "Same organization, different protocol but same subfolder and different filename",
			firstURL:  "https://github.com/octo/repo-name.git/team_a/config.yml",
			secondURL: "git@github.com:octo/repo-name.git/team_b/granted.yml",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			first, err := parseGitURL(tt.firstURL)
			if err != nil {
				t.Error(err)
			}

			second, err := parseGitURL(tt.secondURL)
			if err != nil {
				t.Error(err)
			}

			got := IsSameGitURL(first, second)
			want := tt.want

			if want != got {
				t.Errorf("got git url %v, want url %v", first, second)
			}
		})
	}
}
