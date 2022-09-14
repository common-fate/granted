package cfaws

import (
	"time"

	"github.com/common-fate/granted/pkg/debug"
	"github.com/common-fate/granted/pkg/tokenstore"
	"github.com/fatih/color"
	"github.com/pkg/errors"
)

type SSOToken struct {
	AccessToken string
	Expiry      time.Time
}

// GetValidCachedToken returns nil if no token was found, or if it is expired
func GetValidCachedToken(profileKey string) *SSOToken {
	var t SSOToken
	err := tokenstore.Retrieve(profileKey, &t)
	if err != nil {
		debug.Fprintf(debug.VerbosityDebug, color.Error, "%s\n", errors.Wrap(err, "GetValidCachedToken").Error())
	}
	if t.Expiry.Before(time.Now()) {
		return nil
	}
	return &t
}

// Attempts to store the token, any errors will be logged to debug logging
func StoreSSOToken(profileKey string, ssoTokenValue SSOToken) {
	err := tokenstore.Store(profileKey, ssoTokenValue)
	if err != nil {
		debug.Fprintf(debug.VerbosityDebug, color.Error, "%s\n", errors.Wrap(err, "writing sso token to credentials cache").Error())
	}

}

// Attempts to clear the token, any errors will be logged to debug logging
func ClearSSOToken(profileKey string) {
	err := tokenstore.Clear(profileKey)
	if err != nil {
		debug.Fprintf(debug.VerbosityDebug, color.Error, "%s\n", errors.Wrap(err, "clearing sso token from the credentials cache").Error())
	}
}
