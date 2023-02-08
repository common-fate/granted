package profilegen

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/common-fate/awsconfigfile"
	"github.com/stretchr/testify/assert"
	"gopkg.in/ini.v1"
)

// testSource implements the Source interface
// and provides mock AWS profiles
type testSource struct {
	Profiles []awsconfigfile.SSOProfile
}

func (s testSource) GetProfiles(ctx context.Context) ([]awsconfigfile.SSOProfile, error) {
	return s.Profiles, nil
}

func TestGenerator_Generate(t *testing.T) {
	tests := []struct {
		name                string
		profiles            []awsconfigfile.SSOProfile
		config              string
		noCredentialProcess bool
		sectionNameTemplate string
		prefix              string
		want                string
		wantErr             bool
	}{
		{
			name: "ok",
			profiles: []awsconfigfile.SSOProfile{
				{
					SSOStartURL:   "https://example.awsapps.com/start",
					SSORegion:     "ap-southeast-2",
					AccountID:     "123456789012",
					AccountName:   "prod",
					RoleName:      "DevRole",
					GeneratedFrom: "aws-sso",
				},
			},
			config: `
[profile example]
test = 1
`,
			want: `
[profile example]
test = 1

[profile prod/DevRole]
granted_sso_start_url      = https://example.awsapps.com/start
granted_sso_region         = ap-southeast-2
granted_sso_account_id     = 123456789012
granted_sso_role_name      = DevRole
common_fate_generated_from = aws-sso
credential_process         = granted credential-process --profile prod/DevRole
`,
		},
		{
			name:   "ok with prefix",
			prefix: "myprefix-",
			profiles: []awsconfigfile.SSOProfile{
				{
					SSOStartURL:   "https://example.awsapps.com/start",
					SSORegion:     "ap-southeast-2",
					AccountID:     "123456789012",
					AccountName:   "prod",
					RoleName:      "DevRole",
					GeneratedFrom: "aws-sso",
				},
			},
			config: `
[profile example]
test = 1
`,
			want: `
[profile example]
test = 1

[profile myprefix-prod/DevRole]
granted_sso_start_url      = https://example.awsapps.com/start
granted_sso_region         = ap-southeast-2
granted_sso_account_id     = 123456789012
granted_sso_role_name      = DevRole
common_fate_generated_from = aws-sso
credential_process         = granted credential-process --profile myprefix-prod/DevRole
`,
		},
		{
			name:                "invalid template fails whitespace",
			sectionNameTemplate: "{{ .AccountName }}. ",
			wantErr:             true,
		},
		{
			name:                "invalid template fails ;",
			sectionNameTemplate: "{{ .AccountName }}.;",
			wantErr:             true,
		},
		{
			name:                "valid template fails ;",
			sectionNameTemplate: "{{ .AccountName }}.hello",
			profiles: []awsconfigfile.SSOProfile{
				{
					SSOStartURL:   "https://example.awsapps.com/start",
					SSORegion:     "ap-southeast-2",
					AccountID:     "123456789012",
					AccountName:   "prod",
					RoleName:      "DevRole",
					GeneratedFrom: "aws-sso",
				},
			},
			want: `
[profile prod.hello]
granted_sso_start_url      = https://example.awsapps.com/start
granted_sso_region         = ap-southeast-2
granted_sso_account_id     = 123456789012
granted_sso_role_name      = DevRole
common_fate_generated_from = aws-sso
credential_process         = granted credential-process --profile prod.hello
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			var output bytes.Buffer

			cfg, err := ini.Load([]byte(tt.config))
			if err != nil {
				t.Fatal(err)
			}

			g := &Generator{
				output:              &output,
				sources:             []Source{testSource{Profiles: tt.profiles}},
				config:              cfg,
				noCredentialProcess: tt.noCredentialProcess,
				profileNameTemplate: tt.sectionNameTemplate,
				prefix:              tt.prefix,
			}
			if err := g.Generate(ctx); (err != nil) != tt.wantErr {
				t.Errorf("Generator.Generate() error = %v, wantErr %v", err, tt.wantErr)
			}
			// ignore leading/trailing whitespace so it's easier to format the 'want' section in the test table
			got := strings.TrimSpace(output.String())
			want := strings.TrimSpace(tt.want)
			assert.Equal(t, want, got)
		})
	}
}
