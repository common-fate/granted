package integration_testing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAssumeCommandE2E tests the full assume command end-to-end
func TestAssumeCommandE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Only run if explicitly enabled via environment variable
	if os.Getenv("GRANTED_E2E_TESTING") != "true" {
		t.Skip("Skipping E2E test: set GRANTED_E2E_TESTING=true to enable")
	}

	// Check if there's a pre-built binary to use (for CI)
	grantedBinary := os.Getenv("GRANTED_BINARY_PATH")

	if grantedBinary == "" {
		// Build the granted binary which includes assume functionality
		projectRoot := filepath.Join("..", "..", "..")
		grantedBinary = filepath.Join(t.TempDir(), "dgranted")

		// Build with the dgranted name to trigger assume CLI
		cmd := exec.Command("go", "build", "-o", grantedBinary, "./cmd/granted")
		cmd.Dir = projectRoot
		cmd.Env = append(os.Environ(), "CGO_ENABLED=1") // Ensure CGO is enabled for keychain
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to build granted binary: %v\nOutput: %s", err, output)
		}

		// Make binary executable
		err = os.Chmod(grantedBinary, 0755)
		require.NoError(t, err)
	}

	// Start mock AWS server
	mockServer := NewAssumeE2EMockServer()
	defer mockServer.Close()

	// Setup test environment
	tempDir := t.TempDir()
	homeDir := filepath.Join(tempDir, "home")
	awsDir := filepath.Join(homeDir, ".aws")
	// Use XDG_CONFIG_HOME to set custom config directory
	xdgConfigHome := filepath.Join(tempDir, "config")
	grantedDir := filepath.Join(xdgConfigHome, "granted")

	for _, dir := range []string{awsDir, grantedDir} {
		err := os.MkdirAll(dir, 0755)
		require.NoError(t, err)
	}

	// Create AWS config with a simple IAM profile for testing
	awsConfig := `[profile test-iam]
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
region = us-east-1
`
	awsConfigPath := filepath.Join(awsDir, "config")
	err := os.WriteFile(awsConfigPath, []byte(awsConfig), 0644)
	require.NoError(t, err)

	// Create granted config with all necessary fields to avoid interactive prompts
	// Set CustomBrowserPath to "stdout" to satisfy the UserHasDefaultBrowser check
	grantedConfig := `DefaultBrowser = "STDOUT"
CustomBrowserPath = "stdout"
Ordering = "Alphabetical"
[Keyring]
Backend = "file"
FileBackend = ""
`
	grantedConfigPath := filepath.Join(grantedDir, "config")
	err = os.WriteFile(grantedConfigPath, []byte(grantedConfig), 0644)
	require.NoError(t, err)

	t.Run("AssumeProfileWithIAM", func(t *testing.T) {
		// Set up environment
		env := []string{
			fmt.Sprintf("HOME=%s", homeDir),
			fmt.Sprintf("AWS_CONFIG_FILE=%s", awsConfigPath),
			fmt.Sprintf("XDG_CONFIG_HOME=%s", xdgConfigHome),
			"GRANTED_QUIET=true",        // Suppress output messages
			"FORCE_NO_ALIAS=true",       // Skip alias configuration
			"FORCE_ASSUME_CLI=true",     // Force assume mode
			"PATH=" + os.Getenv("PATH"), // Preserve PATH
		}

		// Run assume command with IAM profile
		cmd := exec.Command(grantedBinary, "test-iam")
		cmd.Env = env

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()
		if err != nil {
			t.Fatalf("Assume command failed: %v\nStdout: %s\nStderr: %s", err, stdout.String(), stderr.String())
		}

		// Parse output
		output := stdout.String()
		t.Logf("Assume output: %s", output)

		// The assume command outputs credentials in a specific format
		assert.Contains(t, output, "GrantedAssume")

		// Extract credentials from output
		parts := strings.Fields(output)
		if len(parts) >= 4 {
			accessKey := parts[1]
			secretKey := parts[2]

			// For IAM profiles, we expect the actual keys to be output
			assert.Equal(t, "AKIAIOSFODNN7EXAMPLE", accessKey)
			assert.NotEqual(t, "None", secretKey)

			// Session token should be "None" for IAM profiles
			sessionToken := parts[3]
			assert.Equal(t, "None", sessionToken)
		} else {
			t.Errorf("Unexpected output format: %s", output)
		}
	})
}

// AssumeE2EMockServer is a specialized mock server for assume command testing
type AssumeE2EMockServer struct {
	*http.Server
	URL         string
	accessToken string
	accessCount int
}

func NewAssumeE2EMockServer() *AssumeE2EMockServer {
	server := &AssumeE2EMockServer{
		accessToken: "default-test-token",
	}

	mux := http.NewServeMux()
	server.Server = &http.Server{Handler: mux}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		server.accessCount++

		// Log the request for debugging
		fmt.Printf("Mock server received: %s %s %s\n", r.Method, r.URL.Path, r.Header.Get("X-Amz-Target"))

		// Handle SSO operations
		target := r.Header.Get("X-Amz-Target")
		switch target {
		case "AWSSSSOPortalService.GetRoleCredentials":
			server.handleGetRoleCredentials(w, r)
		case "AWSSSSOPortalService.ListAccounts":
			server.handleListAccounts(w, r)
		case "SSOOIDCService.CreateToken":
			server.handleCreateToken(w, r)
		default:
			// For unexpected requests, return a generic response
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"message": "Mock response",
			})
		}
	})

	// Start server on a random port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}

	serverURL := fmt.Sprintf("http://%s", listener.Addr().String())
	server.URL = serverURL

	go server.Server.Serve(listener)

	return server
}

func (s *AssumeE2EMockServer) Close() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s.Server.Shutdown(ctx)
}

func (s *AssumeE2EMockServer) SetAccessToken(token string) {
	s.accessToken = token
}

func (s *AssumeE2EMockServer) GetAccessCount() int {
	return s.accessCount
}

func (s *AssumeE2EMockServer) handleGetRoleCredentials(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"roleCredentials": map[string]interface{}{
			"accessKeyId":     "ASIAMOCKEXAMPLE",
			"secretAccessKey": "mock-secret-key",
			"sessionToken":    "mock-session-token",
			"expiration":      time.Now().Add(1*time.Hour).Unix() * 1000,
		},
	}

	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	json.NewEncoder(w).Encode(response)
}

func (s *AssumeE2EMockServer) handleListAccounts(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"accountList": []map[string]interface{}{
			{
				"accountId":    "123456789012",
				"accountName":  "Test Account",
				"emailAddress": "test@example.com",
			},
		},
	}

	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	json.NewEncoder(w).Encode(response)
}

func (s *AssumeE2EMockServer) handleCreateToken(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"accessToken":  s.accessToken,
		"tokenType":    "Bearer",
		"expiresIn":    3600,
		"refreshToken": "mock-refresh-token",
	}

	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	json.NewEncoder(w).Encode(response)
}
