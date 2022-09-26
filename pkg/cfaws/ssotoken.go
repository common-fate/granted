package cfaws

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
)

const (
	// permission for user to read/write/execute.
	USER_READ_WRITE_PERM = 0700
)

type SSOPlainTextOut struct {
	AccessToken string `json:"accessToken"`
	ExpiresAt   string `json:"expiresAt"`
	StartUrl    string `json:"startUrl"`
	Region      string `json:"region"`
}

// CreatePlainTextSSO is currently unused. In a future version of the Granted CLI,
// we'll allow users to export a plaintext token from their keychain for compatibility
// purposes with other AWS tools.
//
// see: https://github.com/common-fate/granted/issues/155
func CreatePlainTextSSO(awsConfig config.SharedConfig, token *SSOToken) *SSOPlainTextOut {
	return &SSOPlainTextOut{
		AccessToken: token.AccessToken,
		ExpiresAt:   token.Expiry.Format(time.RFC3339),
		Region:      awsConfig.Region,
		StartUrl:    awsConfig.SSOStartURL,
	}
}

func (s *SSOPlainTextOut) DumpToCacheDirectory() error {
	jsonOut, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("unable to parse token to json with err %s", err)
	}

	err = dumpTokenFile(jsonOut, s.StartUrl)
	if err != nil {
		return err
	}

	return nil
}

func getCacheFileName(url string) (string, error) {
	hash := sha1.New()
	_, err := hash.Write([]byte(url))
	if err != nil {
		return "", err
	}
	return strings.ToLower(hex.EncodeToString(hash.Sum(nil))) + ".json", nil
}

// Write SSO token as JSON output to default cache location.
func dumpTokenFile(jsonToken []byte, url string) error {
	key, err := getCacheFileName(url)
	if err != nil {
		return err
	}

	path, err := getDefaultCacheLocation()
	if err != nil {
		return err
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.MkdirAll(path, USER_READ_WRITE_PERM)
		if err != nil {
			return fmt.Errorf("unable to create sso cache directory with err: %s", err)
		}
	}

	err = os.WriteFile(filepath.Join(path, key), jsonToken, USER_READ_WRITE_PERM)
	if err != nil {
		return err
	}

	return nil
}

// Find the ~/.aws/sso/cache absolute path based on OS.
func getDefaultCacheLocation() (string, error) {
	h, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	cachePath := filepath.Join(h, ".aws", "sso", "cache")
	return cachePath, nil
}
