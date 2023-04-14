package cfaws

import (
	"testing"

	"github.com/common-fate/clio/clierr"
	grantedConfig "github.com/common-fate/granted/pkg/config"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"gopkg.in/ini.v1"
)

func Test_parseURLFlagFromConfig(t *testing.T) {
	testFileContents := `[profile test1]
credential_process = granted credential-process --url https://example.com

[profile test2]
credential_process =  granted    credential-process   --url   https://example.com  

[profile test3]
credential_process = some-other-cli --url https://example.com

[profile test4]
`
	testConfigFile, err := ini.LoadSources(ini.LoadOptions{}, []byte(testFileContents))
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		profile string
		want    string
		wantErr bool
	}{
		{
			name:    "ok",
			profile: "profile test1",
			want:    "https://example.com",
		},
		{
			name:    "multiple spaces",
			profile: "profile test2",
			want:    "https://example.com",
		},
		{
			name:    "other credential process",
			profile: "profile test3",
			want:    "",
		},
		{
			name:    "no credential process entry",
			profile: "profile test4",
			want:    "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			section, err := testConfigFile.GetSection(tt.profile)
			if err != nil {
				t.Fatal(err)
			}
			got := parseURLFlagFromConfig(section)
			if got != tt.want {
				t.Errorf("parseURLFlagFromConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetGrantedApprovalsURL(t *testing.T) {
	type args struct {
		rawConfig    *ini.Section
		gConf        grantedConfig.Config
		SSORoleName  string
		SSOAccountId string
	}
	testFile := ini.Empty()

	emptySection, err := testFile.NewSection("empty")
	if err != nil {
		t.Fatal(err)
	}
	section, err := testFile.NewSection("test")
	if err != nil {
		t.Fatal(err)
	}
	_, err = section.NewKey("credential_process", "granted credential-process --url https://override.example.com")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		args    args
		want    *clierr.Err
		wantErr bool
	}{
		{
			name: "ok",
			args: args{
				gConf: grantedConfig.Config{
					AccessRequestURL: "https://example.com",
				},
				SSORoleName:  "test",
				SSOAccountId: "123456789012",
				rawConfig:    emptySection,
			},

			want: &clierr.Err{
				Err: "test error",
				Messages: []clierr.Printer{
					clierr.Warn("You need to request access to this role:"),
					clierr.Warn("https://example.com/access?accountId=123456789012&permissionSetArn.label=test&type=commonfate%2Faws-sso"),
					clierr.Warn("or run: 'granted exp request latest'"),
				},
			},
		},
		{
			name: "url flag precedence",
			args: args{
				gConf: grantedConfig.Config{
					AccessRequestURL: "https://example.com",
				},
				rawConfig:    section,
				SSORoleName:  "test",
				SSOAccountId: "123456789012",
			},
			want: &clierr.Err{
				Err: "test error",
				Messages: []clierr.Printer{
					clierr.Warn("You need to request access to this role:"),
					clierr.Warn("https://override.example.com/access?accountId=123456789012&permissionSetArn.label=test&type=commonfate%2Faws-sso"),
					clierr.Warn("or run: 'granted exp request latest'"),
				},
			},
		},
		{
			name: "display prompt if no URL is set",
			args: args{
				gConf:     grantedConfig.Config{},
				rawConfig: emptySection,
			},
			want: &clierr.Err{
				Err: "test error",
				Messages: []clierr.Printer{
					clierr.Info("It looks like you don't have the right permissions to access this role"),
					clierr.Info("If you are using Common Fate to manage this role you can configure the Granted CLI with a request URL so that you can be directed to your Granted Approvals instance to make a new access request the next time you have this error"),
					clierr.Info("To configure a URL to request access to this role with 'granted settings request-url set <YOUR_GRANTED_APPROVALS_URL'"),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := FormatAWSErrorWithGrantedApprovalsURL(errors.New("test error"), tt.args.rawConfig, tt.args.gConf, tt.args.SSORoleName, tt.args.SSOAccountId)
			assert.Equal(t, tt.want, err)
		})
	}
}
