package cfaws

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
	"github.com/stretchr/testify/assert"
)

func TestGetPollingConfig(t *testing.T) {
	in := ssooidc.StartDeviceAuthorizationOutput{
		ExpiresIn: 120, // seconds
		Interval:  5,   // seconds
	}

	got := getPollingConfig(&in)

	want := PollingConfig{
		TimeoutAfter:  2 * time.Minute,
		CheckInterval: 5 * time.Second,
	}

	assert.Equal(t, want, got)
}
