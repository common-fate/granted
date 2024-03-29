package cfaws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/aws-sdk-go-v2/service/sts/types"
	"github.com/stretchr/testify/assert"
)

func Test_getSessionTags(t *testing.T) {

	tests := []struct {
		name string

		caller       *sts.GetCallerIdentityOutput
		wantTags     []types.Tag
		wantUserName string
	}{
		{
			name: "ok",
			caller: &sts.GetCallerIdentityOutput{
				Account: aws.String("123456789012"),
				Arn:     aws.String("arn:aws:iam::123456789012:user/example-user"),
				UserId:  aws.String("XXXYYYZZZ"),
			},
			wantUserName: "example-user",
			wantTags: []types.Tag{
				{Key: aws.String("userID"), Value: aws.String("XXXYYYZZZ")},
				{Key: aws.String("account"), Value: aws.String("123456789012")},
				{Key: aws.String("principalArn"), Value: aws.String("arn:aws:iam::123456789012:user/example-user")},
				{Key: aws.String("userName"), Value: aws.String("example-user")},
			},
		},
		{
			name: "falls_back_to_user_id_if_arn_invalid",
			caller: &sts.GetCallerIdentityOutput{
				Account: aws.String("123456789012"),
				Arn:     aws.String("invalid arn"),
				UserId:  aws.String("XXXYYYZZZ"),
			},
			wantUserName: "XXXYYYZZZ",
			wantTags: []types.Tag{
				{Key: aws.String("userID"), Value: aws.String("XXXYYYZZZ")},
				{Key: aws.String("account"), Value: aws.String("123456789012")},
				{Key: aws.String("principalArn"), Value: aws.String("invalid arn")},
			},
		},
		{
			name:         "wont_panic",
			caller:       nil,
			wantUserName: "",
			wantTags:     nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTags, gotUserName := getSessionTags(tt.caller)
			assert.Equal(t, tt.wantTags, gotTags)
			if gotUserName != tt.wantUserName {
				t.Errorf("getSessionTags() gotUserName = %v, want %v", gotUserName, tt.wantUserName)
			}
		})
	}
}
