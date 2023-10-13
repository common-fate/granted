package cfaws

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/ini.v1"
)

func TestValidateCredentialProcess(t *testing.T) {
	tests := []struct {
		name        string
		arg         string
		profileName string
		wantErr     string
	}{
		{
			name:        "valid argument with correct profile name",
			arg:         "  granted credential-process   --profile develop",
			profileName: "develop",
		},
		{
			name:        "valid argument with correct profile name and --aws-config-file flag",
			arg:         "  granted --aws-config-file example credential-process   --profile develop",
			profileName: "develop",
		},
		{
			name:        "valid argument with incorrect profile name",
			arg:         "granted credential-process --profile abc",
			profileName: "develop",
			wantErr:     "unmatched profile names. The profile name 'abc' provided to 'granted credential-process' does not match AWS profile name 'develop'",
		},
		{
			name:        "invalid argument",
			arg:         "aws-sso-util --profile abc",
			profileName: "apple",
			wantErr:     "unable to parse 'credential_process'. Looks like your credential_process isn't configured correctly. \n You need to add 'granted credential-process --profile <profile-name>'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			err := validateCredentialProcess(tt.arg, tt.profileName)
			if err != nil {
				if err.Error() != tt.wantErr {
					t.Fatal(err)
				}
			}
		})
	}
}

// tests support for aws sso-session configuration
func TestSSOSessionSupport(t *testing.T) {
	tests := []struct {
		name           string
		file           string
		profileName    string
		wantSSOSession *SSOSession
	}{
		{
			name: "valid argument with correct profile name",
			file: `[profile testing]
sso_session = testing-sso
sso_account_id     = 12345678912
sso_role_name      = Test
region             = ap-southeast-2

[sso-session testing-sso]
sso_start_url  = https://d-12345678910.awsapps.com/start
sso_region     = ap-southeast-2
`,
			profileName: "testing",
			wantSSOSession: &SSOSession{
				SSORegion:   "ap-southeast-2",
				SSOStartURL: "https://d-12345678910.awsapps.com/start",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := loader{fileString: tt.file}
			profiles, err := loadProfiles(l, nooploader{})
			if err != nil {
				t.Fatal(err)
			}
			profile, err := profiles.Profile(tt.profileName)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, tt.wantSSOSession, profile.SSOSession)
		})
	}
}

type loader struct {
	fileString string
}

func (l loader) Path() string { return "" }
func (l loader) Load() (*ini.File, error) {
	testConfigFile, err := ini.LoadSources(ini.LoadOptions{}, []byte(l.fileString))
	if err != nil {
		return nil, err
	}
	return testConfigFile, nil
}

type nooploader struct {
}

func (l nooploader) Path() string { return "" }
func (l nooploader) Load() (*ini.File, error) {
	return ini.Empty(), nil
}
