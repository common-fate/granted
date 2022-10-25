package cfaws

import (
	"testing"

	"github.com/bigkevmcd/go-configparser"
	"github.com/common-fate/clio"
	grantedConfig "github.com/common-fate/granted/pkg/config"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func Test_parseURLFlagFromConfig(t *testing.T) {
	type args struct {
		rawConfig configparser.Dict
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "ok",
			args: args{
				rawConfig: configparser.Dict{
					"credential_process": "granted credential-process --url https://example.com",
				},
			},
			want: "https://example.com",
		},
		{
			name: "multiple spaces",
			args: args{
				rawConfig: configparser.Dict{
					"credential_process": " granted    credential-process   --url   https://example.com  ",
				},
			},
			want: "https://example.com",
		},
		{
			name: "other credential process",
			args: args{
				rawConfig: configparser.Dict{
					"credential_process": "some-other-cli --url https://example.com",
				},
			},
			want: "",
		},
		{
			name: "no credential process entry",
			args: args{
				rawConfig: configparser.Dict{},
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseURLFlagFromConfig(tt.args.rawConfig)
			if got != tt.want {
				t.Errorf("parseURLFlagFromConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetGrantedApprovalsURL(t *testing.T) {
	type args struct {
		rawConfig    configparser.Dict
		gConf        grantedConfig.Config
		SSORoleName  string
		SSOAccountId string
	}
	tests := []struct {
		name    string
		args    args
		want    *clio.CLIError
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
			},
			want: &clio.CLIError{
				Err: "test error",
				Messages: []clio.Printer{
					clio.WarnMsg("You need to request access to this role:"),
					clio.WarnlnMsg("https://example.com/access?accountId=123456789012&permissionSetArn.label=test&type=commonfate%2Faws-sso"),
				},
			},
		},
		{
			name: "url flag precedence",
			args: args{
				gConf: grantedConfig.Config{
					AccessRequestURL: "https://example.com",
				},
				rawConfig: configparser.Dict{
					// we should show the overridden --url flag, rather than the global setting.
					"credential_process": "granted credential-process --url https://override.example.com",
				},
				SSORoleName:  "test",
				SSOAccountId: "123456789012",
			},
			want: &clio.CLIError{
				Err: "test error",
				Messages: []clio.Printer{
					clio.WarnMsg("You need to request access to this role:"),
					clio.WarnlnMsg("https://override.example.com/access?accountId=123456789012&permissionSetArn.label=test&type=commonfate%2Faws-sso"),
				},
			},
		},
		{
			name: "display prompt if no URL is set",
			args: args{
				gConf: grantedConfig.Config{},
			},
			want: &clio.CLIError{
				Err: "test error",
				Messages: []clio.Printer{
					clio.InfoMsg("It looks like you don't have the right permissions to access this role"),
					clio.InfoMsg("If you are using Granted Approvals to manage this role you can configure the Granted CLI with a request URL so that you can be directed to your Granted Approvals instance to make a new access request the next time you have this error"),
					clio.InfoMsg("To configure a URL to request access to this role with 'granted settings request-url set <YOUR_GRANTED_APPROVALS_URL'"),
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
