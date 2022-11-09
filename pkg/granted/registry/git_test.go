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
			name: "https with personal repo slash",
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
			name: "https github.com",
			url:  "https://github.com/owner/name",
			want: GitURL{
				ProvidedURL: "https://github.com/owner/name",
				Host:        "github.com",
				Org:         "owner",
				Repo:        "name",
			},
			wantErr: nil,
		}, {
			name: "http github.com",
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
			name: "http github.com withs subpath",
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
