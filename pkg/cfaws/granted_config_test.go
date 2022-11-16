package cfaws

import (
	"testing"
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
			name:        "valid argument without incorrect profile name",
			arg:         "granted credential-process --profile abc",
			profileName: "develop",
			wantErr:     "unmatched profile names. The profile name 'abc' provided to 'granted credential-process' doesnot match AWS profile name 'develop'",
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
