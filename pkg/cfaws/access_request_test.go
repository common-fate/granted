package cfaws

import (
	"testing"

	"github.com/bigkevmcd/go-configparser"
	grantedConfig "github.com/common-fate/granted/pkg/config"
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
			got, err := parseURLFlagFromConfig(tt.args.rawConfig)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseURLFlagFromConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
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
		want    string
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
			want: `You need to request access to this role:
https://example.com/access?accountId=123456789012&permissionSetArn.label=test&type=commonfate%2Faws-sso`,
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
			want: `You need to request access to this role:
https://override.example.com/access?accountId=123456789012&permissionSetArn.label=test&type=commonfate%2Faws-sso`,
		},
		{
			name: "display prompt if no URL is set",
			args: args{
				gConf: grantedConfig.Config{},
			},
			want: `Granted Approvals URL not configured. 
Set up a URL to request access to this role with 'granted settings request-url set <YOUR_GRANTED_APPROVALS_URL'`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetGrantedApprovalsURL(tt.args.rawConfig, tt.args.gConf, tt.args.SSORoleName, tt.args.SSOAccountId)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetGrantedApprovalsURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetGrantedApprovalsURL() = %v, want %v", got, tt.want)
			}
		})
	}
}
