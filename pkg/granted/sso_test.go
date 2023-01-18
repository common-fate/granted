package granted

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSSOGenerateParseFlags(t *testing.T) {

	type testcase struct {
		name         string
		giveTemplate string
		wantErr      error
	}

	testcases := []testcase{
		{
			name:         "default passes",
			giveTemplate: defaultProfileNameTemplate,
		},
		{
			name:         "valid template passes",
			giveTemplate: "{{ .AccountName }}.hello",
		},
		{
			name:         "invalid template fails whitespace",
			giveTemplate: "{{ .AccountName }}. ",
			wantErr:      errors.New(`--profile-template flag must not contain any of these illegal characters ( \][;'")`),
		},
		{
			name:         "invalid template fails ;",
			giveTemplate: "{{ .AccountName }}.;",
			wantErr:      errors.New(`--profile-template flag must not contain any of these illegal characters ( \][;'")`),
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			c := cliOptions{
				profileTemplate: tc.giveTemplate,
				args:            []string{"demo"},
			}
			_, err := parseCliOptions(c)
			if tc.wantErr == nil {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.wantErr.Error())
			}
		})
	}
}
